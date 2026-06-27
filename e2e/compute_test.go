package e2e

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	serviceAccountGVK = schema.GroupVersionKind{Version: "v1", Kind: "ServiceAccount"}
	configMapGVK      = schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
	secretGVK         = schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
	deploymentGVK     = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	statefulSetGVK    = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	daemonSetGVK      = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}
	hpaGVK            = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}
	pdbGVK            = schema.GroupVersionKind{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"}
	computeGVK        = schema.GroupVersionKind{Group: workloadGroup, Version: workloadVersion, Kind: workloadKind}
)

func TestComputeMinimal(t *testing.T) {
	const name = "compute-minimal"

	feat := features.New("compute/minimal").
		Setup(applyNamespaced("testdata/compute-minimal.yaml")).
		Assess("deployment available", assertDeploymentAvailable(testNamespace, name)).
		Assess("serviceaccount present", assertExists(
			serviceAccountGVK,
			testNamespace,
			name,
		)).
		Assess("pdb present", assertExists(
			pdbGVK,
			testNamespace,
			name,
		)).
		Assess("hpa absent", assertAbsent(
			hpaGVK,
			testNamespace,
			name,
		)).
		Assess("configmap absent", assertAbsent(
			configMapGVK,
			testNamespace,
			name,
		)).
		Assess("secret absent", assertAbsent(
			secretGVK,
			testNamespace,
			name,
		)).
		Assess("statefulset absent", assertAbsent(
			statefulSetGVK,
			testNamespace,
			name,
		)).
		Assess("daemonset absent", assertAbsent(
			daemonSetGVK,
			testNamespace,
			name,
		)).
		Feature()

	testenv.Test(t, feat)
}

func TestComputeFull(t *testing.T) {
	const name = "compute-full"

	feat := features.New("compute/full").
		Setup(applyNamespaced("testdata/compute-full.yaml")).
		Assess("deployment available", assertDeploymentAvailable(testNamespace, name)).
		Assess("serviceaccount present", assertExists(
			serviceAccountGVK,
			testNamespace,
			name,
		)).
		Assess("pdb present", assertExists(
			pdbGVK,
			testNamespace,
			name,
		)).
		Assess("hpa present", assertExists(
			hpaGVK,
			testNamespace,
			name,
		)).
		Assess("configmap present", assertExists(
			configMapGVK,
			testNamespace,
			name,
		)).
		Assess("secret present", assertExists(
			secretGVK,
			testNamespace,
			name,
		)).
		Assess("statefulset absent", assertAbsent(
			statefulSetGVK,
			testNamespace,
			name,
		)).
		Assess("daemonset absent", assertAbsent(
			daemonSetGVK,
			testNamespace,
			name,
		)).
		Feature()

	testenv.Test(t, feat)
}

func TestComputeStatefulSet(t *testing.T) {
	const name = "compute-statefulset"

	feat := features.New("compute/statefulset").
		Setup(applyNamespaced("testdata/compute-statefulset.yaml")).
		Assess("statefulset available", assertStatefulSetAvailable(testNamespace, name)).
		Assess("serviceaccount present", assertExists(
			serviceAccountGVK,
			testNamespace,
			name,
		)).
		Assess("pdb present", assertExists(
			pdbGVK,
			testNamespace,
			name,
		)).
		Assess("deployment absent", assertAbsent(
			deploymentGVK,
			testNamespace,
			name,
		)).
		Assess("daemonset absent", assertAbsent(
			daemonSetGVK,
			testNamespace,
			name,
		)).
		Feature()

	testenv.Test(t, feat)
}

func TestComputeDaemonSet(t *testing.T) {
	const name = "compute-daemonset"

	feat := features.New("compute/daemonset").
		Setup(applyNamespaced("testdata/compute-daemonset.yaml")).
		Assess("daemonset available", assertDaemonSetAvailable(testNamespace, name)).
		Assess("serviceaccount present", assertExists(
			serviceAccountGVK,
			testNamespace,
			name,
		)).
		Assess("pdb present", assertExists(
			pdbGVK,
			testNamespace,
			name,
		)).
		Assess("deployment absent", assertAbsent(
			deploymentGVK,
			testNamespace,
			name,
		)).
		Assess("statefulset absent", assertAbsent(
			statefulSetGVK,
			testNamespace,
			name,
		)).
		Feature()

	testenv.Test(t, feat)
}
