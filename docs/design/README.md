# Kalypso

This document outlines the design and implementation plan for the **Kalypso** project. Kalypso simplifies Kubernetes cluster and workload management by introducing high-level, opinionated Custom Resource Definitions (CRDs) for common capabilities (Compute, Storage, Networking, Observability, and Security), which are aggregated under a cluster-scoped `Workload` CRD.

## KRO - Kubernets Resource Operator

KRO is a generic operator that can be used to manage Kubernetes resources. KRO is used to provide the substrate for Kalypso capabilities. Each capability is defined via a CRD and managed by KRO. High-level capabilities use RGD chaining to create more complex and comprehensive solutions.

## Capabilities

- Each capability must be self-contained in its own CRD (also managed by KRO)
- Each capability must be able to be used in standalone mode: create a capability CRD directly, without a Workload CRD
- When a capability is used in standalone mode, it must be able to target any resource in the cluster. This is why it must include a `TargetRef`
- Capabilities are namespaced

## Future

### Planned Kubernetes Resources support (non-exhaustive list)

- Deployment
- StatefulSet
- DaemonSet
- CronJob
- Job
- Service
- Ingress
- Network Policy: TBD
- RBAC
- Resource Quota: TBD
- Limit Range: TBD
- Storage Class
- Persistent Volume Claim
- Volume Snapshot
- Service Monitor
- Pod Monitor
- Probe
- Logging: TBD

### Planned capabilities 

- Compute: CRD responsible for defining compute-related resources
    - Autoscaling (HPA / KEDA)
    - Scheduling (Node Selector/Topology Spread Contraints/Affinity/Anti-Affinity)
    - Pod Disruption Budgets
    - Deployment/StatefulSet/DaemonSet/CronJob/Job
- Storage: CRD responsible for defining storage-related resources
    - Storage Class
    - Persistent Volume Claim
    - Volume Snapshot: TBD
- Networking: CRD responsible for defining networking-related resources
    - Service
    - Ingress (Gateway API)
    - Network Policy: TBD
- Observability: CRD responsible for defining observability-related resources
    - Service Monitor
    - Pod Monitor
    - Probe
    - Logging: TBD
- Security: CRD responsible for defining security-related resources
    - Network Policy: TBD
    - RBAC (Role / RoleBinding / ClusterRole / ClusterRoleBinding)
    - Resource Quota: TBD
    - Limit Range: TBD

## TargetRef

```yaml
   targetRef:
      apiVersion: "apps/v1" # TBD: Maybe we don't need this? 
      kind: "Deployment" # TBD: Maybe we don't need this?
      name: "nginx"
      namespace: "default" # Optional, defaults to the namespace of the capability CRD
```

## Kalypso CLI (TBD)

The Kalypso CLI is planned to provide higher-level features to manage validate and explore different capabilities provided by `Kalypso`.

## Releases

### Release V1Alpha1

The V1Alpha1 release will provide support for the Compute caopability. The goal of the V1Alpha1 release is to provide a solid foundation for future development and to demonstrate the value of `Kalypso`.

A test harness to allow quick iteration on the CRDs and RGDs provided by `Kalypso` using the Kubernetes e2e framework + KinD.

### Release V1Alpha2

The V1Alpha2 release will provide support for other (core) compute-related Kubernetes components as Kalypso building blocks. E.g. RBAC, Network Policy. 

### Release V1Alpha3

The V1Alpha3 release will provide support for Networking-related Kubernetes components and initial integration with Istio-related capabilities as Kalypso building blocks.

### Release V1Alpha4

The V1Alpha4 release will provide support for Observability-related Kubernetes components and initial integration with Prometheus-related capabilities as Kalypso building blocks.

## #Release V1Alpha5

The V1Alpha5 release will provide support for Security-related Kubernetes components and initial integration with RBAC-related capabilities as Kalypso building blocks.