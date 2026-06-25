package workload

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
)

// ObservabilityReconciler reconciles Observability settings onto a shared WorkloadGraph KRO instance
type ObservabilityReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ObservabilityReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	var workload calypsov1alpha1.Workload

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&workload,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	targetNamespace := workload.Spec.TargetRef.Namespace

	if targetNamespace == "" {

		log.Info("Target namespace is empty, skipping reconciliation", "workload", workload.Name)

		return ctrl.Result{}, nil
	}

	// Initialize KRO WorkloadGraph instance
	kroInstance := &unstructured.Unstructured{}

	kroInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kalypso.io",
		Version: "v1alpha1",
		Kind:    "WorkloadGraph",
	})

	kroInstance.SetName(workload.Name)

	kroInstance.SetNamespace(targetNamespace)

	// Determine if observability capability settings are configured
	observabilityActive := workload.Spec.Observability.TargetRef.Name != ""

	_, err := controllerutil.CreateOrPatch(
		ctx,
		r.Client,
		kroInstance,
		func() error {
			if err := ctrl.SetControllerReference(
				&workload,
				kroInstance,
				r.Scheme,
			); err != nil {
				return fmt.Errorf("failed to set controller reference: %w", err)
			}

			unstructuredTargetRef, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&workload.Spec.TargetRef)

			if err != nil {
				return fmt.Errorf("failed to convert targetRef to unstructured: %w", err)
			}

			if err := unstructured.SetNestedField(
				kroInstance.Object,
				unstructuredTargetRef,
				"spec",
				"targetRef",
			); err != nil {
				return fmt.Errorf("failed to set targetRef: %w", err)
			}

			if observabilityActive {

				workload.Spec.Observability.TargetRef = workload.Spec.TargetRef

				unstructuredObservability, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&workload.Spec.Observability)

				if err != nil {
					return fmt.Errorf("failed to convert observability to unstructured: %w", err)
				}

				if err := unstructured.SetNestedField(
					kroInstance.Object,
					unstructuredObservability,
					"spec",
					"observability",
				); err != nil {
					return fmt.Errorf("failed to set observability spec: %w", err)
				}

			} else {

				unstructured.RemoveNestedField(
					kroInstance.Object,
					"spec",
					"observability",
				)

			}

			return nil
		},
	)

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ObservabilityReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	kroInstance := &unstructured.Unstructured{}

	kroInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kalypso.io",
		Version: "v1alpha1",
		Kind:    "WorkloadGraph",
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Workload{}).
		Owns(kroInstance).
		Named("workload-observability").
		Complete(r)
}
