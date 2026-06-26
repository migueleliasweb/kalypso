package kro

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	krov1alpha1 "github.com/kubernetes-sigs/kro/api/v1alpha1"
)

// ResourceGraphDefinitionReconciler syncs the WorkloadGraph ResourceGraphDefinition in the cluster
type ResourceGraphDefinitionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ResourceGraphDefinitionReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	// We only care about our specific RGD singleton
	if req.Name != "workloadgraphs.kalypso.io" {
		return ctrl.Result{}, nil
	}

	desiredRGD := r.getDesiredRGD()

	var existingRGD krov1alpha1.ResourceGraphDefinition

	if err := r.Get(
		ctx,
		client.ObjectKey{Name: req.Name},
		&existingRGD,
	); err != nil {
		if apierrors.IsNotFound(err) {

			log.Info("Creating workloadgraphs.kalypso.io ResourceGraphDefinition")

			if err := r.Create(
				ctx,
				desiredRGD,
			); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to create ResourceGraphDefinition: %w", err)
			}

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Update the existing RGD spec to keep it in sync with the desired state in Go code
	existingRGD.Spec = desiredRGD.Spec

	if err := r.Update(
		ctx,
		&existingRGD,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update ResourceGraphDefinition: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller and bootstraps the RGD on startup using APIReader
func (r *ResourceGraphDefinitionReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	// Bootstrap the RGD on startup directly via APIReader to avoid caching delays
	apiReader := mgr.GetAPIReader()

	var existingRGD krov1alpha1.ResourceGraphDefinition

	err := apiReader.Get(
		context.Background(),
		client.ObjectKey{Name: "workloadgraphs.kalypso.io"},
		&existingRGD,
	)

	if err != nil && apierrors.IsNotFound(err) {

		desiredRGD := r.getDesiredRGD()

		if err := mgr.GetClient().Create(
			context.Background(),
			desiredRGD,
		); err != nil {
			return fmt.Errorf("failed to bootstrap ResourceGraphDefinition: %w", err)
		}

	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&krov1alpha1.ResourceGraphDefinition{}, builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {}))).
		Named("rgd-sync").
		Complete(r)
}

func (r *ResourceGraphDefinitionReconciler) getDesiredRGD() *krov1alpha1.ResourceGraphDefinition {
	return &krov1alpha1.ResourceGraphDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "workloadgraphs.kalypso.io",
		},
		Spec: krov1alpha1.ResourceGraphDefinitionSpec{
			Schema: &krov1alpha1.Schema{
				Kind:       "WorkloadGraph",
				APIVersion: "v1alpha1",
				Group:      "kalypso.io",
				Scope:      krov1alpha1.ResourceScopeNamespaced,
				Spec: runtime.RawExtension{
					Raw: []byte(`{
						"targetRef": {
							"resource": "string | required=true",
							"kind": "string | required=true",
							"apiVersion": "string | required=true",
							"namespace": "string | required=true"
						},
						"compute": {
							"autoscaling": {
								"minReplicas": "integer | default=1",
								"maxReplicas": "integer | required=true",
								"targetCPUUtilizationPercentage": "integer",
								"targetMemoryUtilizationPercentage": "integer"
							}
						},
						"networking": {
							"service": {
								"type": "string | default=ClusterIP",
								"ports": "[]object"
							}
						}
					}`),
				},
			},
			Resources: []*krov1alpha1.Resource{
				{
					ID: "service",
					IncludeWhen: []string{
						"has(instance.spec.networking) && has(instance.spec.networking.service) && size(instance.spec.networking.service.ports) > 0",
					},
					Template: runtime.RawExtension{
						Raw: []byte(`{
							"apiVersion": "v1",
							"kind": "Service",
							"metadata": {
								"name": "${instance.metadata.name}",
								"namespace": "${instance.metadata.namespace}"
							},
							"spec": {
								"selector": {
									"app": "${instance.spec.targetRef.resource}"
								},
								"ports": "${instance.spec.networking.service.ports}",
								"type": "${instance.spec.networking.service.type}"
							}
						}`),
					},
				},
			},
		},
	}
}
