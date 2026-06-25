# Implementation Plan - Kalypso High-Level Kubernetes CRDs

This document outlines the design and implementation plan for the **Kalypso** project. Kalypso simplifies Kubernetes cluster and workload management by introducing high-level, opinionated Custom Resource Definitions (CRDs) for common capabilities (Compute, Storage, Networking, Observability, and Security), which are aggregated under a cluster-scoped `Workload` CRD.

---

## User Review & Clarifications Incorporated

> [!NOTE]
> **Kubebuilder Tooling**
> We will use the local kubebuilder binary at `/Users/miguel.santos/Bin/kubebuilder`.

> [!NOTE]
> **Workload CRD Scope**
> The `Workload` CRD is cluster-scoped. It will manage namespace-scoped capability resources in the namespace specified by the target resource reference (`spec.targetRef.namespace`), which allows it to orchestrate resources across namespaces and potentially manage the lifecycle of namespaces themselves.

> [!NOTE]
> **Target Resource Types**
> We will implement the target resource orchestration specifically for `Deployment` resources initially, but we will structure the code/interfaces (e.g., using a general resource adapter pattern) so that support for `StatefulSet`, `DaemonSet`, and custom resources like ArgoCD `Rollout` can be easily added in the future.

> [!NOTE]
> **Autoscaling Implementation**
> For this initial implementation, we will only implement standard `HorizontalPodAutoscaler` (HPA) autoscaling. The spec and controller design will remain open for KEDA `ScaledObject` integration later.

> [!NOTE]
> **Controller Simplicity & Separation**
> In accordance with the updated guidelines, **each controller (reconciler) must only manage a single resource type at a time**. This maps directly to the Kubebuilder controller pattern (one reconciler per CRD: `Workload` reconciler, `Compute` reconciler, etc.) to ensure simplicity and ease of debugging.

---

## Proposed Changes

### Component 1: Project Initialization & Tooling
Initialize the project structure using Kubebuilder.

1. Initialize the project:
   ```bash
   /Users/miguel.santos/Bin/kubebuilder init --domain lmoet.io --repo github.com/migueleliasweb/kalypso
   ```
2. Scaffold API resources and controllers (one primary controller per CRD):
   - `Workload` (Cluster-scoped)
   - `Compute` (Namespace-scoped)
   - `Storage` (Namespace-scoped)
   - `Networking` (Namespace-scoped)
   - `Observability` (Namespace-scoped)
   - `Security` (Namespace-scoped)

---

### Component 2: API Definitions
Define the Go structs for the CRD schemas under `api/v1alpha1/`.

#### [NEW] [types_shared.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/types_shared.go)
Contains the shared structures:
- `ResourceRef` (defines target resources, e.g., GroupVersionKind, name, and namespace).
- `EscapeHatch` (defines the patch type, target resource kind, and the patch itself).

#### [NEW] [workload_types.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/workload_types.go)
Defines the cluster-scoped `Workload` CRD:
- The spec aggregates referencing fields or embedded specs for each capability: `compute`, `storage`, `networking`, `observability`, `security`.
- Annotate with `//+kubebuilder:resource:scope=Cluster` to ensure it is cluster-scoped.

#### [NEW] [compute_types.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/compute_types.go)
Defines the namespace-scoped `Compute` CRD spec:
- Autoscaling (minReplicas, maxReplicas, targetCPUUtilizationPercentage, targetMemoryUtilizationPercentage).
- Scheduling (NodeSelector, Affinity, TopologySpreadConstraints).
- PodDisruptionBudget (MinAvailable, MaxUnavailable).
- EscapeHatches.

#### [NEW] [storage_types.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/storage_types.go)
Defines the namespace-scoped `Storage` CRD spec:
- StorageClassName
- PVC specs (size, access modes) to generate.
- EscapeHatches.

#### [NEW] [networking_types.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/networking_types.go)
Defines the namespace-scoped `Networking` CRD spec:
- Service configuration (Ports, Type).
- Ingress/Gateway routes.
- EscapeHatches.

#### [NEW] [observability_types.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/observability_types.go)
Defines the namespace-scoped `Observability` CRD spec.

#### [NEW] [security_types.go](file:///Users/miguel.santos/Projects/personal/kalypso/api/v1alpha1/security_types.go)
Defines the namespace-scoped `Security` CRD spec.

---

### Component 3: Escape Hatch Patching Engine
A utility to apply dynamic patches onto Kubernetes manifests during reconciliation.

#### [NEW] [patch_engine.go](file:///Users/miguel.santos/Projects/personal/kalypso/pkg/patch/patch_engine.go)
Implements:
- `ApplyEscapeHatches(obj runtime.Object, hatches []EscapeHatch) (runtime.Object, error)`
- Uses `github.com/evanphx/json-patch/v5` for `JSONPatch` and `JSONMergePatch`.

---

### Component 4: Reconcilers & Controllers
Implement the reconciliation loop for the Workload and each Capability. Each controller only manages its own resource.

#### [NEW] [workload_controller.go](file:///Users/miguel.santos/Projects/personal/kalypso/internal/controller/workload_controller.go)
Reconciles `Workload` CRD:
- Spawns/updates capability CRDs (`Compute`, `Storage`, etc.) in the target namespace.
- Sets the `ownerReference` of these capability CRDs pointing to the cluster-scoped `Workload`.

#### [NEW] [compute_controller.go](file:///Users/miguel.santos/Projects/personal/kalypso/internal/controller/compute_controller.go)
Reconciles `Compute` CRD:
- Creates/updates `HorizontalPodAutoscaler` and `PodDisruptionBudget` for the target `Deployment`.
- Patches Scheduling parameters (NodeSelector, Affinity, TopologySpreadConstraints) directly on the target `Deployment`.
- Uses an abstraction layer (e.g. WorkloadAdapter interface) to retrieve/update the target workload so that support for other types (StatefulSet, DaemonSet, Rollout) can be added easily without changing the core reconcile logic.

#### [NEW] [networking_controller.go](file:///Users/miguel.santos/Projects/personal/kalypso/internal/controller/networking_controller.go)
Reconciles `Networking` CRD:
- Creates/updates a `Service` targeting the deployment.
- Applies EscapeHatches.

#### [NEW] [storage_controller.go](file:///Users/miguel.santos/Projects/personal/kalypso/internal/controller/storage_controller.go)
Reconciles `Storage` CRD:
- Creates/updates `PersistentVolumeClaim`.
- Patches the target `Deployment` to mount the volumes.

#### [NEW] [observability_controller.go](file:///Users/miguel.santos/Projects/personal/kalypso/internal/controller/observability_controller.go)
Reconciles `Observability` CRD.

#### [NEW] [security_controller.go](file:///Users/miguel.santos/Projects/personal/kalypso/internal/controller/security_controller.go)
Reconciles `Security` CRD.

---

## Verification Plan

### Automated Tests
- Unit tests to verify:
  1. The Patch Engine applies `JSONPatch` and `JSONMergePatch` correctly.
  2. The `Workload` controller properly maps capability fields and configures target references.
  3. The `Compute` controller creates HPAs and updates target Deployments with scheduling criteria.
- Run tests via `go test ./...`.

### Manual Verification
- Replaced by user-side verification.
