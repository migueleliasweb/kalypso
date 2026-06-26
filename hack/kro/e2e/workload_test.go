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

// Child-resource GVKs the Workload RGD manages.
var (
	serviceGVK         = schema.GroupVersionKind{Version: "v1", Kind: "Service"}
	hpaGVK             = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}
	virtualServiceGVK  = schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1", Kind: "VirtualService"}
	destinationRuleGVK = schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1", Kind: "DestinationRule"}
	serviceMonitorGVK  = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"}
)

// TestWorkloadMinimal applies a compute-only Workload and asserts the RGD
// creates only a Deployment + Service — every disabled capability's resource
// must be absent, validating includeWhen's default-off behavior.
func TestWorkloadMinimal(t *testing.T) {
	const name = "hello"
	feat := features.New("workload/minimal").
		Setup(applyWorkload("testdata/workload-minimal.yaml")).
		Assess("deployment available", assertDeploymentAvailable(name)).
		Assess("service present", assertExists(serviceGVK, name)).
		Assess("hpa absent", assertAbsent(hpaGVK, name)).
		Assess("virtualservice absent", assertAbsent(virtualServiceGVK, name)).
		Assess("destinationrule absent", assertAbsent(destinationRuleGVK, name)).
		Assess("servicemonitor absent", assertAbsent(serviceMonitorGVK, name)).
		Teardown(deleteWorkload(name)).
		Feature()
	testenv.Test(t, feat)
}

// TestWorkloadFull enables every capability and asserts the full resource set
// is created, validating includeWhen's on-path.
func TestWorkloadFull(t *testing.T) {
	const name = "payments"
	feat := features.New("workload/full").
		Setup(applyWorkload("testdata/workload-full.yaml")).
		Assess("deployment available", assertDeploymentAvailable(name)).
		Assess("service present", assertExists(serviceGVK, name)).
		Assess("hpa present", assertExists(hpaGVK, name)).
		Assess("virtualservice present", assertExists(virtualServiceGVK, name)).
		Assess("destinationrule present", assertExists(destinationRuleGVK, name)).
		Assess("servicemonitor present", assertExists(serviceMonitorGVK, name)).
		Teardown(deleteWorkload(name)).
		Feature()
	testenv.Test(t, feat)
}

// --- feature helpers ---------------------------------------------------------

func applyWorkload(path string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		obj := &unstructured.Unstructured{Object: map[string]interface{}{}}
		if err := yaml.Unmarshal(data, &obj.Object); err != nil {
			t.Fatalf("unmarshalling %s: %v", path, err)
		}
		obj.SetNamespace(cfg.Namespace())
		if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
			t.Fatalf("creating Workload from %s: %v", path, err)
		}
		return ctx
	}
}

func assertDeploymentAvailable(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()
		dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cfg.Namespace()}}
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
			t.Fatalf("deployment %q never became available: %v", name, err)
		}
		return ctx
	}
}

// assertExists waits (briefly) for the object to appear, since KRO creates
// children asynchronously after the Workload is applied.
func assertExists(gvk schema.GroupVersionKind, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(name)
		obj.SetNamespace(cfg.Namespace())
		if err := wait.For(
			conditions.New(r).ResourceMatch(obj, func(k8s.Object) bool { return true }),
			wait.WithTimeout(1*time.Minute),
			wait.WithInterval(2*time.Second),
		); err != nil {
			t.Fatalf("expected %s %q to exist: %v", gvk.Kind, name, err)
		}
		return ctx
	}
}

// assertAbsent verifies the object does not exist. It runs after the Deployment
// is Available, so KRO's reconcile (which decides includeWhen) has completed.
func assertAbsent(gvk schema.GroupVersionKind, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		err := cfg.Client().Resources().Get(ctx, name, cfg.Namespace(), obj)
		if err == nil {
			t.Fatalf("expected %s %q to be absent, but it exists", gvk.Kind, name)
		}
		if !apierrors.IsNotFound(err) {
			t.Fatalf("unexpected error checking %s %q: %v", gvk.Kind, name, err)
		}
		return ctx
	}
}

func deleteWorkload(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		// Leave the Workload (and its KRO-managed children) in place by default
		// so failures can be inspected; only clean up when tearing down.
		if !shouldDestroy() {
			t.Logf("leaving Workload %q in place for troubleshooting (set %s=true to clean up)", name, destroyEnvVar)
			return ctx
		}
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(schema.GroupVersionKind{Group: workloadGroup, Version: workloadVersion, Kind: workloadKind})
		obj.SetName(name)
		obj.SetNamespace(cfg.Namespace())
		if err := cfg.Client().Resources().Delete(ctx, obj); err != nil {
			t.Logf("deleting Workload %q: %v", name, err)
		}
		return ctx
	}
}
