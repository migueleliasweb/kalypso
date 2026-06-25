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

package controller

// WORKLOAD CONTROLLER RBAC
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=workloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=workloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=workloads/finalizers,verbs=update
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes;storages;networkings;observabilities;securities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes/status;storages/status;networkings/status;observabilities/status;securities/status,verbs=get;update;patch

// COMPUTE CONTROLLER RBAC
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=computes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete

// OBSERVABILITY CONTROLLER RBAC
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=observabilities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=observabilities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=observabilities/finalizers,verbs=update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete

// SECURITY CONTROLLER RBAC
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=securities,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=securities/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=securities/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

// NETWORKING CONTROLLER RBAC
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=networkings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=networkings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=networkings/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// STORAGE CONTROLLER RBAC
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=storages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=calypso.lmoet.io,resources=storages/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
