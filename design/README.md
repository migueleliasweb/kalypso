# Kalypso - A series of high-level resources for Kubernetes-based Platforms

The rationale behind this project is to provide high-level CRDs to ease the adoption/management of any Kubernetes Cluster.

## High-level design

- Written in Golang
- This project will use the default `Kubebuilder` project style
    - Each controller (reconciler) must only manage a single resource at a time to ensure simplicity and ease of debugging
- Capability CRDs
    - Each CRD must expose one and only one capability
    - The CRD must be namespace scoped
    - Each CRD capability is aggregated in the Workload CRD
    - Each capability must be usable in either indirectly via the Workload CRD or in a standalone way by targetting a specific resource e.g `Deployment`, `Daemonset`, `Statefulset` etc...
        - The design needs to ensure new capabilties can be exposed/added in the future by changing the underlycing capability spec
    - Each capability must expose a "escape hatch" where users of the CRD can provide patches for specific resource types it manages to allow maximum flexibility
        - This capability will be provided by a shared Go struct (refer below)
        - The "escape hatch" will be composed of a list of patches that will be applied to the resources managed by the capability CRD
- The Workload CRD:
    - Must be cluster scoped
    - Must only be used to aggregate capabilities and must not provide unique capabilities by itself
    - The Workload CRD is the only CRD that will not be usable in a standalone way
    - References between CRDs will be maintained via `ownerReferences` and will use a shared type called `ResourceRef`
    - Will provide safe defaults to each available referenced capability to ensure minimal configuration is needed on common scenarios

## ResourceRef

```go
// Generic reference struct
type ResourceRef struct {
	Resource string `json:"resource"`
	Kind string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
    Namespace string `json:"namespace"`
}
```

## Escape Hatch (for max flexibility when extending CRD capabilities)

```go
// It allows users to patch any field of any resource type the capability CRD manages
type EscapeHatch struct {
    // The kind of resource to patch (e.g. Pod, Service, Deployment, VirtualService, PVC, etc...)
    Kind string `json:"kind"`
    // The type of patch to apply (JSONPatch or StrategicMergePatch + JSONPatch)
    PatchType string `json:"patchType"`
    // The patch to apply (strategicMergePatch or JSON Patch)
    Patch string `json:"patch"`
}
```

## CRDs

# Workload: Any workload that can be deployed to a Kubernetes cluster

```yaml
apiVersion: calypso.lmoet.io/v1alpha1
kind: Workload
metadata:
  name: <workload-name>
  namespace: <namespace>
spec:
    # Each of these should be treated as different "capabilities" 
    # and will be delivered via different CRDs e.g. Compute, Storage, Networking etc..
    compute: {}
    storage: {}
    networking: {}
    observability: {}
    security: {}
```

# Compute: This CRD will be responsible for defining the compute-related resources

Spec:

- Autoscaling (HPA / KEDA)
    - Min
    - Max
    - Target CPU % (HPA)
    - Target Memory % (HPA)
    - Metric-based scaling (KEDA)
- Scheduling
    - Node selectors
    - Node Affinity
    - Node Anti-affinity
    - Topology Spread Constraints
- Pod Disruption Budgets
    - Min unavailable
    - Max unavailable
    - Max surge

# Storage: This CRD will be responsible for defining the storage-related resources

Spec:

- Storage Class
- Persistent Volume Claim
- Volume Snapshot

# Networking: This CRD will be responsible for defining the networking-related resources

Spec:

- Service
- Ingress (Gateway API)
    - Private Routes
    - Public Routes
    - TLS
- Network Policy: TBD

# Observability: This CRD will be responsible for defining the observability-related resources

Spec:

- Service Monitor
- Pod Monitor
- Probe
- Logging

# Security: This CRD will be responsible for defining the security-related resources

Spec:

- Network Policy: TBD
- RBAC (Role / RoleBinding / ClusterRole / ClusterRoleBinding)
- Resource Quota: TBD
- Limit Range: TBD