/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package networking

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
	"github.com/migueleliasweb/kalypso/pkg/patch"
)

// NetworkingReconciler reconciles a Networking object
type NetworkingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NetworkingReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {

	log := logf.FromContext(ctx)

	// 1. Fetch the Networking resource
	var net calypsov1alpha1.Networking

	if err := r.Get(
		ctx,
		req.NamespacedName,
		&net,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if net.Spec.TargetRef.Resource == "" {

		log.Info("Networking targetRef is nil, skipping reconciliation", "name", net.Name)

		return ctrl.Result{}, nil
	}

	// 2. Reconcile Service
	svcName := net.Spec.TargetRef.Resource

	targetSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: net.Namespace,
		},
	}

	if len(net.Spec.Service.Ports) == 0 {

		var svc corev1.Service

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: net.Namespace, Name: svcName},
			&svc,
		); err == nil {

			if err := r.Delete(
				ctx,
				&svc,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetSvc,
			func() error {
				targetSvc.Spec.Selector = map[string]string{
					"app": net.Spec.TargetRef.Resource,
				}
				targetSvc.Spec.Ports = net.Spec.Service.Ports
				targetSvc.Spec.Type = net.Spec.Service.Type

				if err := ctrl.SetControllerReference(
					&net,
					targetSvc,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedSvcObj, err := patch.ApplyEscapeHatches(
					targetSvc,
					net.Spec.EscapeHatches,
					"Service",
				)

				if err != nil {
					return fmt.Errorf("failed to apply escape hatch to Service: %w", err)
				}

				*targetSvc = *(patchedSvcObj.(*corev1.Service))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	// 3. Reconcile Ingress
	ingName := fmt.Sprintf("%s-ingress", net.Name)

	targetIng := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingName,
			Namespace: net.Namespace,
		},
	}

	hasIngress := len(net.Spec.Ingress.PublicRoutes) > 0 ||
		len(net.Spec.Ingress.PrivateRoutes) > 0 ||
		net.Spec.Ingress.TLS.SecretName != ""

	if !hasIngress {

		var ing networkingv1.Ingress

		if err := r.Get(
			ctx,
			client.ObjectKey{Namespace: net.Namespace, Name: ingName},
			&ing,
		); err == nil {

			if err := r.Delete(
				ctx,
				&ing,
			); err != nil {
				return ctrl.Result{}, err
			}

		}

	} else {

		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			targetIng,
			func() error {
				pathType := networkingv1.PathTypePrefix

				var rules []networkingv1.IngressRule

				// Aggregate Public and Private Routes
				var allRoutes []calypsov1alpha1.GatewayRoute

				allRoutes = append(allRoutes, net.Spec.Ingress.PublicRoutes...)
				allRoutes = append(allRoutes, net.Spec.Ingress.PrivateRoutes...)

				if len(allRoutes) == 0 {

					rules = append(rules, networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: net.Spec.TargetRef.Resource,
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					})

				} else {

					for _, route := range allRoutes {

						var httpPaths []networkingv1.HTTPIngressPath

						paths := route.Paths

						if len(paths) == 0 {
							paths = []string{"/"}
						}

						var portNum int32 = 80

						if len(net.Spec.Service.Ports) > 0 {
							portNum = net.Spec.Service.Ports[0].Port
						}

						for _, pathStr := range paths {

							httpPaths = append(httpPaths, networkingv1.HTTPIngressPath{
								Path:     pathStr,
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: net.Spec.TargetRef.Resource,
										Port: networkingv1.ServiceBackendPort{
											Number: portNum,
										},
									},
								},
							})

						}

						hostnames := route.Hostnames

						if len(hostnames) == 0 {
							hostnames = []string{""}
						}

						for _, hostname := range hostnames {

							rules = append(rules, networkingv1.IngressRule{
								Host: hostname,
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: httpPaths,
									},
								},
							})

						}

					}

				}

				targetIng.Spec.Rules = rules

				if net.Spec.Ingress.TLS.SecretName != "" {

					var hosts []string

					for _, rule := range rules {

						if rule.Host != "" {
							hosts = append(hosts, rule.Host)
						}

					}

					targetIng.Spec.TLS = []networkingv1.IngressTLS{
						{
							Hosts:      hosts,
							SecretName: net.Spec.Ingress.TLS.SecretName,
						},
					}

				} else {
					targetIng.Spec.TLS = nil
				}

				if err := ctrl.SetControllerReference(
					&net,
					targetIng,
					r.Scheme,
				); err != nil {
					return err
				}

				patchedIngObj, err := patch.ApplyEscapeHatches(
					targetIng,
					net.Spec.EscapeHatches,
					"Ingress",
				)

				if err != nil {
					return fmt.Errorf("failed to apply escape hatch to Ingress: %w", err)
				}

				*targetIng = *(patchedIngObj.(*networkingv1.Ingress))

				return nil
			},
		)

		if err != nil {
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkingReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&calypsov1alpha1.Networking{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Named("networking").
		Complete(r)
}
