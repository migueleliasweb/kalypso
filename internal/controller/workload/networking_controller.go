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

// NetworkingReconciler reconciles Networking settings onto a shared WorkloadGraph KRO instance
type NetworkingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NetworkingReconciler) Reconcile(
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

	// Determine if networking capability settings are configured
	networkingActive := len(workload.Spec.Networking.Service.Ports) > 0 ||
		len(workload.Spec.Networking.Ingress.PrivateRoutes) > 0 ||
		len(workload.Spec.Networking.Ingress.PublicRoutes) > 0 ||
		workload.Spec.Networking.TargetRef.Resource != ""

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

			if networkingActive {

				workload.Spec.Networking.TargetRef = workload.Spec.TargetRef

				unstructuredNetworking, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&workload.Spec.Networking)

				if err != nil {
					return fmt.Errorf("failed to convert networking to unstructured: %w", err)
				}

				if err := unstructured.SetNestedField(
					kroInstance.Object,
					unstructuredNetworking,
					"spec",
					"networking",
				); err != nil {
					return fmt.Errorf("failed to set networking spec: %w", err)
				}

			} else {

				unstructured.RemoveNestedField(
					kroInstance.Object,
					"spec",
					"networking",
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

func (r *NetworkingReconciler) SetupWithManager(
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
		Named("workload-networking").
		Complete(r)
}
