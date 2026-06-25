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

package compute

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	calypsov1alpha1 "github.com/migueleliasweb/kalypso/api/v1alpha1"
)

var _ = Describe("Compute HPA Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		compute := &calypsov1alpha1.Compute{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Compute")

			if err := k8sClient.Get(
				ctx,
				typeNamespacedName,
				compute,
			); err != nil && errors.IsNotFound(err) {
				resource := &calypsov1alpha1.Compute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: calypsov1alpha1.ComputeSpec{
						Autoscaling: calypsov1alpha1.AutoscalingSpec{
							MaxReplicas: 1,
						},
					},
				}

				Expect(k8sClient.Create(
					ctx,
					resource,
				)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &calypsov1alpha1.Compute{}

			if err := k8sClient.Get(
				ctx,
				typeNamespacedName,
				resource,
			); err != nil {
				Expect(err).NotTo(HaveOccurred())
			}

			By("Cleanup the specific resource instance Compute")

			Expect(k8sClient.Delete(
				ctx,
				resource,
			)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			controllerReconciler := &HPAReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(
				ctx,
				reconcile.Request{
					NamespacedName: typeNamespacedName,
				},
			)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
