package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// Resource GVKs the RGDs manage.
var (
	namespaceGVK       = schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}
	serviceGVK         = schema.GroupVersionKind{Version: "v1", Kind: "Service"}
	hpaGVK             = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}
	virtualServiceGVK  = schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1", Kind: "VirtualService"}
	destinationRuleGVK = schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1", Kind: "DestinationRule"}
	serviceMonitorGVK  = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"}
	workloadGVK        = schema.GroupVersionKind{Group: workloadGroup, Version: workloadVersion, Kind: workloadKind}
	clusterWorkloadGVK = schema.GroupVersionKind{Group: workloadGroup, Version: workloadVersion, Kind: clusterWorkloadKind}
)

// TestWorkloadMinimal applies a compute-only Workload and asserts the RGD
// creates only a Deployment + Service — every disabled capability's resource
// must be absent, validating includeWhen's default-off behavior.
func TestWorkloadMinimal(t *testing.T) {
	const name = "hello"
	feat := features.New("workload/minimal").
		Setup(applyNamespaced("testdata/workload-minimal.yaml")).
		Assess("deployment available", assertDeploymentAvailable(testNamespace, name)).
		Assess("service present", assertExists(serviceGVK, testNamespace, name)).
		Assess("hpa absent", assertAbsent(hpaGVK, testNamespace, name)).
		Assess("virtualservice absent", assertAbsent(virtualServiceGVK, testNamespace, name)).
		Assess("destinationrule absent", assertAbsent(destinationRuleGVK, testNamespace, name)).
		Assess("servicemonitor absent", assertAbsent(serviceMonitorGVK, testNamespace, name)).
		Teardown(deleteInstance(workloadGVK, testNamespace, name)).
		Feature()
	testenv.Test(t, feat)
}

// TestWorkloadFull enables every capability and asserts the full resource set
// is created, validating includeWhen's on-path.
func TestWorkloadFull(t *testing.T) {
	const name = "payments"
	feat := features.New("workload/full").
		Setup(applyNamespaced("testdata/workload-full.yaml")).
		Assess("deployment available", assertDeploymentAvailable(testNamespace, name)).
		Assess("service present", assertExists(serviceGVK, testNamespace, name)).
		Assess("hpa present", assertExists(hpaGVK, testNamespace, name)).
		Assess("virtualservice present", assertExists(virtualServiceGVK, testNamespace, name)).
		Assess("destinationrule present", assertExists(destinationRuleGVK, testNamespace, name)).
		Assess("servicemonitor present", assertExists(serviceMonitorGVK, testNamespace, name)).
		Teardown(deleteInstance(workloadGVK, testNamespace, name)).
		Feature()
	testenv.Test(t, feat)
}

// TestClusterWorkload applies a cluster-scoped ClusterWorkload and asserts it
// creates and owns a namespace, then (via RGD chaining) a Workload inside it
// whose Deployment + Service come up — validating the namespace-owning variant.
func TestClusterWorkload(t *testing.T) {
	const name = "tenant-a" // namespace defaults to the instance name
	feat := features.New("clusterworkload").
		Setup(applyClusterScoped("testdata/clusterworkload.yaml")).
		Assess("namespace created", assertExists(namespaceGVK, "", name)).
		Assess("workload created in owned ns", assertExists(workloadGVK, name, name)).
		Assess("deployment available in owned ns", assertDeploymentAvailable(name, name)).
		Assess("service present in owned ns", assertExists(serviceGVK, name, name)).
		Teardown(deleteInstance(clusterWorkloadGVK, "", name)).
		Feature()
	testenv.Test(t, feat)
}

// --- feature helpers ---------------------------------------------------------

// applyNamespaced creates a namespaced instance in the suite's test namespace.
func applyNamespaced(path string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		obj := decodeManifest(t, path)
		obj.SetNamespace(cfg.Namespace())
		if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
			t.Fatalf("creating %s from %s: %v", obj.GetKind(), path, err)
		}
		return ctx
	}
}

// applyClusterScoped creates a cluster-scoped instance (no namespace).
func applyClusterScoped(path string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		obj := decodeManifest(t, path)
		if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
			t.Fatalf("creating %s from %s: %v", obj.GetKind(), path, err)
		}
		return ctx
	}
}

func decodeManifest(t *testing.T, path string) *unstructured.Unstructured {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err := yaml.Unmarshal(data, &obj.Object); err != nil {
		t.Fatalf("unmarshalling %s: %v", path, err)
	}
	return obj
}

func assertDeploymentAvailable(namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()
		dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
		// ResourceMatch re-polls on NotFound (KRO creates the Deployment
		// asynchronously), unlike DeploymentConditionMatch which aborts on it.
		if err := wait.For(
			conditions.New(r).ResourceMatch(dep, func(o k8s.Object) bool {
				for _, c := range o.(*appsv1.Deployment).Status.Conditions {
					if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			}),
			wait.WithTimeout(3*time.Minute),
			wait.WithInterval(3*time.Second),
		); err != nil {
			t.Fatalf("deployment %s/%s never became available: %v", namespace, name, err)
		}
		return ctx
	}
}

// assertExists waits (briefly) for the object to appear, since KRO creates
// resources asynchronously. Pass namespace "" for cluster-scoped kinds.
func assertExists(gvk schema.GroupVersionKind, namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(name)
		obj.SetNamespace(namespace)
		if err := wait.For(
			conditions.New(r).ResourceMatch(obj, func(k8s.Object) bool { return true }),
			wait.WithTimeout(2*time.Minute),
			wait.WithInterval(2*time.Second),
		); err != nil {
			t.Fatalf("expected %s %s/%s to exist: %v", gvk.Kind, namespace, name, err)
		}
		return ctx
	}
}

// assertAbsent verifies the object does not exist. It runs after the Deployment
// is Available, so KRO's reconcile (which decides includeWhen) has completed.
func assertAbsent(gvk schema.GroupVersionKind, namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		err := cfg.Client().Resources().Get(ctx, name, namespace, obj)
		if err == nil {
			t.Fatalf("expected %s %s/%s to be absent, but it exists", gvk.Kind, namespace, name)
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("unexpected error checking %s %s/%s: %v", gvk.Kind, namespace, name, err)
		}
		return ctx
	}
}

// deleteInstance removes an instance on teardown — but only when tearing down.
// By default instances (and their KRO-managed children) linger for inspection.
func deleteInstance(gvk schema.GroupVersionKind, namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		if !shouldDestroy() {
			t.Logf("leaving %s %q in place for troubleshooting (set %s=true to clean up)", gvk.Kind, name, destroyEnvVar)
			return ctx
		}
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(name)
		obj.SetNamespace(namespace)
		if err := cfg.Client().Resources().Delete(ctx, obj); err != nil {
			t.Logf("deleting %s %q: %v", gvk.Kind, name, err)
		}
		return ctx
	}
}
