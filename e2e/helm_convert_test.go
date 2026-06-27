package e2e

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestHelmConvertCoreDNS(t *testing.T) {
	// 1. Download CoreDNS helm chart tgz
	const chartURL = "https://github.com/coredns/helm/releases/download/coredns-1.46.0/coredns-1.46.0.tgz"
	tempDir := t.TempDir()
	tgzPath := filepath.Join(tempDir, "coredns-1.46.0.tgz")

	resp, err := http.Get(chartURL)
	if err != nil {
		t.Fatalf("failed to download coredns chart: %v", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(tgzPath)
	if err != nil {
		t.Fatalf("failed to create tgz file: %v", err)
	}
	if _, err = io.Copy(out, resp.Body); err != nil {
		out.Close()
		t.Fatalf("failed to save tgz file: %v", err)
	}
	out.Close()

	// 2. Untar the tgz file
	chartDir := filepath.Join(tempDir, "chart")
	if err := untar(tgzPath, chartDir); err != nil {
		t.Fatalf("failed to untar coredns chart: %v", err)
	}

	// 3. Compile the kalypso binary
	binaryPath := filepath.Join(tempDir, "kalypso")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build kalypso binary: %v", err)
	}

	// 4. Run helm-convert
	convertedCRPath := filepath.Join(tempDir, "coredns-compute.yaml")
	convertCmd := exec.Command(binaryPath, "helm-convert", filepath.Join(chartDir, "coredns"), "-o", convertedCRPath, "--release-name", "coredns-test", "--namespace", testNamespace)
	if err := convertCmd.Run(); err != nil {
		t.Fatalf("failed to run helm-convert: %v", err)
	}

	// 5. Use E2E framework to apply the generated Compute CR and assert creation
	const name = "coredns-test"

	feat := features.New("helm-convert/coredns").
		Setup(grantCoreDNSAdmin).
		Setup(applyNamespaced(convertedCRPath)).
		Assess("deployment available", assertDeploymentAvailable(testNamespace, name)).
		Assess("serviceaccount present", assertExists(
			serviceAccountGVK,
			testNamespace,
			name,
		)).
		Assess("configmap present", assertExists(
			configMapGVK,
			testNamespace,
			name,
		)).
		Feature()

	testenv.Test(t, feat)
}

func grantCoreDNSAdmin(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	r := cfg.Client().Resources()
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "coredns-test-admin"},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "coredns-test",
			Namespace: testNamespace,
		}},
	}

	if err := r.Create(ctx, crb); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("failed to grant admin to coredns-test: %v", err)
	}

	return ctx
}

func untar(tarball, targetDir string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	tarReader := tar.NewReader(archive)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}
	return nil
}
