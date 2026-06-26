package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var testenv env.Environment

func TestMain(m *testing.M) {
	testenv, _ = env.NewFromFlags()
	provider := kind.NewProvider()

	testenv.Setup(
		// Fresh cluster every run: delete the one left behind by the previous
		// run (no-op if none exists), then create a new one.
		func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
			_ = provider.SetDefaults().WithName(clusterName).Destroy(ctx)
			return ctx, nil
		},
		envfuncs.CreateClusterWithConfig(provider, clusterName, kindConfigFile, kind.WithImage(nodeImage)),
		envfuncs.CreateNamespace(testNamespace),
		installCRDs,
		installKRO,
		applyRGDs,
	)

	// Leave the cluster behind by default for troubleshooting; destroy only
	// when KRO_E2E_DESTROY_CLUSTER is set truthy. Logs are always exported.
	finish := []env.Func{envfuncs.ExportClusterLogs(clusterName, logsDir)}
	if shouldDestroy() {
		finish = append(finish, envfuncs.DestroyCluster(clusterName))
	}
	testenv.Finish(finish...)

	os.Exit(testenv.Run(m))
}

// shouldDestroy reports whether the suite should tear things down (cluster and
// Workload instances). Default is false so everything lingers for inspection.
func shouldDestroy() bool {
	destroy, _ := strconv.ParseBool(os.Getenv(destroyEnvVar))
	return destroy
}

// installCRDs fetches and applies the Istio + Prometheus CRDs the Workload RGD
// references, so KRO can create those kinds when capabilities are enabled.
func installCRDs(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	r := cfg.Client().Resources()
	for _, spec := range []manifestSpec{istioManifest, promManifest} {
		if err := applyManifest(ctx, r, spec); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// installKRO creates the kro-system namespace (the bundle doesn't), applies the
// KRO install bundle, then waits for the KRO controller to become Available.
func installKRO(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	r := cfg.Client().Resources()
	if err := ensureNamespace(ctx, r, kroNamespace); err != nil {
		return ctx, err
	}
	if err := applyManifest(ctx, r, kroManifest); err != nil {
		return ctx, err
	}
	// KRO's default RBAC only covers kro.run/apiextensions. Its per-RGD dynamic
	// controller must list/watch the generated kind (workloads.kalypso.io) and
	// create the child resources, so grant the controller SA cluster-admin.
	// This is acceptable in an ephemeral, local e2e cluster.
	if err := grantKROAdmin(ctx, r); err != nil {
		return ctx, fmt.Errorf("granting KRO controller RBAC: %w", err)
	}
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: kroDeploymentName, Namespace: kroNamespace}}
	if err := wait.For(
		conditions.New(r).DeploymentConditionMatch(dep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(5*time.Second),
	); err != nil {
		return ctx, fmt.Errorf("waiting for KRO controller to become available: %w", err)
	}
	return ctx, nil
}

// applyRGDs applies the Workload RGD first (the ClusterWorkload RGD's graph
// references the generated Workload CRD), then the ClusterWorkload RGD.
func applyRGDs(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
	r := cfg.Client().Resources()
	if err := applyAndWaitRGD(ctx, r, rgdPath, rgdName, workloadCRDName); err != nil {
		return ctx, err
	}
	if err := applyAndWaitRGD(ctx, r, clusterRGDPath, clusterRGDName, clusterWorkloadCRDName); err != nil {
		return ctx, err
	}
	return ctx, nil
}

// applyAndWaitRGD applies an RGD manifest then waits for its generated CRD to be
// Established and the RGD itself to reach state Active (dynamic controller up).
func applyAndWaitRGD(ctx context.Context, r *resources.Resources, path, name, crdName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading RGD %s: %w", path, err)
	}
	if err := applyYAML(ctx, r, data); err != nil {
		return fmt.Errorf("applying RGD %s: %w", path, err)
	}

	crd := &unstructured.Unstructured{}
	crd.SetGroupVersionKind(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})
	crd.SetName(crdName)
	if err := wait.For(
		conditions.New(r).ResourceMatch(crd, crdEstablished),
		wait.WithTimeout(2*time.Minute),
		wait.WithInterval(3*time.Second),
	); err != nil {
		return fmt.Errorf("waiting for %s to be Established: %w", crdName, err)
	}

	// The CRD being Established isn't enough: KRO must also bring up the per-RGD
	// dynamic controller (status.state == Active) before instances reconcile.
	rgd := &unstructured.Unstructured{}
	rgd.SetGroupVersionKind(schema.GroupVersionKind{Group: "kro.run", Version: "v1alpha1", Kind: "ResourceGraphDefinition"})
	rgd.SetName(name)
	if err := wait.For(
		conditions.New(r).ResourceMatch(rgd, rgdActive),
		wait.WithTimeout(2*time.Minute),
		wait.WithInterval(3*time.Second),
	); err != nil {
		return fmt.Errorf("waiting for RGD %s to become Active: %w", name, err)
	}
	return nil
}

func rgdActive(obj k8s.Object) bool {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return false
	}
	state, _, _ := unstructured.NestedString(u.Object, "status", "state")
	return state == "Active"
}

// grantKROAdmin binds the KRO controller ServiceAccount to cluster-admin so its
// dynamic controllers can watch generated kinds and manage their children.
func grantKROAdmin(ctx context.Context, r *resources.Resources) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "kro-e2e-cluster-admin"},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      kroDeploymentName, // SA shares the controller's name
			Namespace: kroNamespace,
		}},
	}
	if err := r.Create(ctx, crb); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func crdEstablished(obj k8s.Object) bool {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return false
	}
	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if m["type"] == "Established" && m["status"] == "True" {
			return true
		}
	}
	return false
}

// --- manifest helpers --------------------------------------------------------

// applyManifest ensures the manifest is cached locally then applies it.
func applyManifest(ctx context.Context, r *resources.Resources, spec manifestSpec) error {
	path, err := ensureManifest(spec)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := applyYAML(ctx, r, data); err != nil {
		return fmt.Errorf("applying %s: %w", spec.filename, err)
	}
	return nil
}

// ensureManifest downloads spec into crdManifestsDir if not already cached and
// returns the local path. Caching keeps reruns (and CI) from re-downloading.
func ensureManifest(spec manifestSpec) (string, error) {
	path := filepath.Join(crdManifestsDir, spec.filename)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(crdManifestsDir, 0o755); err != nil {
		return "", err
	}
	resp, err := http.Get(spec.url)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", spec.url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: unexpected status %s", spec.url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Write atomically so a partial download never poisons the cache.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

// applyYAML splits a (possibly multi-document) manifest and creates each object
// as unstructured, so no Go types need to be registered for the various CRDs.
func applyYAML(ctx context.Context, r *resources.Resources, data []byte) error {
	reader := apiyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		if err := yaml.Unmarshal(doc, &obj.Object); err != nil {
			return fmt.Errorf("unmarshalling manifest document: %w", err)
		}
		if obj.GetKind() == "" {
			continue
		}
		if err := r.Create(ctx, obj); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("creating %s %q: %w", obj.GetKind(), obj.GetName(), err)
		}
	}
	return nil
}

func ensureNamespace(ctx context.Context, r *resources.Resources, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := r.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
