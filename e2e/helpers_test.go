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

// applyNamespaced creates a namespaced instance in the suite's test namespace.
func applyNamespaced(path string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		obj := decodeManifest(t, path)
		obj.SetNamespace(cfg.Namespace())

		if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
			t.Fatalf(
				"creating %s from %s: %v",
				obj.GetKind(),
				path,
				err,
			)
		}

		return ctx
	}
}

// applyClusterScoped creates a cluster-scoped instance (no namespace).
func applyClusterScoped(path string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		obj := decodeManifest(t, path)

		if err := cfg.Client().Resources().Create(ctx, obj); err != nil {
			t.Fatalf(
				"creating %s from %s: %v",
				obj.GetKind(),
				path,
				err,
			)
		}

		return ctx
	}
}

func decodeManifest(t *testing.T, path string) *unstructured.Unstructured {
	t.Helper()

	data, err := os.ReadFile(path)

	if err != nil {
		t.Fatalf(
			"reading %s: %v",
			path,
			err,
		)
	}

	obj := &unstructured.Unstructured{Object: map[string]interface{}{}}

	if err := yaml.Unmarshal(data, &obj.Object); err != nil {
		t.Fatalf(
			"unmarshalling %s: %v",
			path,
			err,
		)
	}

	return obj
}

func assertDeploymentAvailable(namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		// ResourceMatch re-polls on NotFound (KRO creates the Deployment
		// asynchronously), unlike DeploymentConditionMatch which aborts on it.
		err := wait.For(
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
		)

		if err != nil {
			t.Fatalf(
				"deployment %s/%s never became available: %v",
				namespace,
				name,
				err,
			)
		}

		return ctx
	}
}

func assertStatefulSetAvailable(namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()

		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		err := wait.For(
			conditions.New(r).ResourceMatch(sts, func(o k8s.Object) bool {
				s := o.(*appsv1.StatefulSet)
				return s.Status.ReadyReplicas == *s.Spec.Replicas
			}),
			wait.WithTimeout(3*time.Minute),
			wait.WithInterval(3*time.Second),
		)

		if err != nil {
			t.Fatalf(
				"statefulset %s/%s never became available: %v",
				namespace,
				name,
				err,
			)
		}

		return ctx
	}
}

func assertDaemonSetAvailable(namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r := cfg.Client().Resources()

		ds := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		err := wait.For(
			conditions.New(r).ResourceMatch(ds, func(o k8s.Object) bool {
				d := o.(*appsv1.DaemonSet)
				return d.Status.NumberReady == d.Status.DesiredNumberScheduled
			}),
			wait.WithTimeout(3*time.Minute),
			wait.WithInterval(3*time.Second),
		)

		if err != nil {
			t.Fatalf(
				"daemonset %s/%s never became available: %v",
				namespace,
				name,
				err,
			)
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

		err := wait.For(
			conditions.New(r).ResourceMatch(obj, func(k8s.Object) bool { return true }),
			wait.WithTimeout(2*time.Minute),
			wait.WithInterval(2*time.Second),
		)

		if err != nil {
			t.Fatalf(
				"expected %s %s/%s to exist: %v",
				gvk.Kind,
				namespace,
				name,
				err,
			)
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

		err := cfg.Client().Resources().Get(
			ctx,
			name,
			namespace,
			obj,
		)

		if err == nil {
			t.Fatalf(
				"expected %s %s/%s to be absent, but it exists",
				gvk.Kind,
				namespace,
				name,
			)
		}

		if !apierrors.IsNotFound(err) {
			t.Fatalf(
				"unexpected error checking %s %s/%s: %v",
				gvk.Kind,
				namespace,
				name,
				err,
			)
		}

		return ctx
	}
}

// deleteInstance removes an instance on teardown — but only when tearing down.
// By default instances (and their KRO-managed children) linger for inspection.
func deleteInstance(gvk schema.GroupVersionKind, namespace, name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		if !shouldDestroy() {
			t.Logf(
				"leaving %s %q in place for troubleshooting (set %s=true to clean up)",
				gvk.Kind,
				name,
				destroyEnvVar,
			)

			return ctx
		}

		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(name)
		obj.SetNamespace(namespace)

		if err := cfg.Client().Resources().Delete(ctx, obj); err != nil {
			t.Logf(
				"deleting %s %q: %v",
				gvk.Kind,
				name,
				err,
			)
		}

		return ctx
	}
}
