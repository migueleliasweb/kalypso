# Implementation Plan - Kalypso High-Level Kubernetes CRDs

This document outlines the design and implementation plan for the **Kalypso** project. Kalypso simplifies Kubernetes cluster and workload management by introducing high-level, opinionated Custom Resource Definitions (CRDs) for common capabilities (Compute, Storage, Networking, Observability, and Security), which are aggregated under a cluster-scoped `Workload` CRD.

## KRO - Kubernets Resource Operator

KRO is a generic operator that can be used to manage Kubernetes resources. KRO is used to provide the substrate for Kalypso capabilities. Each capability and building block is defined via a CRD and managed by KRO. High-level capabilities use RGD chaining to create more complex and comprehensive solutions.

## Capabilities

- Each capability must be self-contained in its own CRD (also managed by KRO)
- Each capability must be able to be used in standalone mode: create a capability CRD directly, without a Workload CRD
- When a capability is used in standalone mode, it must be able to target any resource in the cluster. This is why it must include a `TargetRef`
- Capabilities are namespaced
- Capabilities are aggregatede by their technology stack

## Building Blocks (TBD)

- Each capability is composed of `building blocks`
- Each building block is a reusable component that can be used in multiple capabilities
- Example: A `Deployment` is a building block that can be used in multiple capabilities
- The set of `building blocks` is not exhaustive and can be extended in the future

### Planned building blocks 

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
    - Scheduling
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

The Kalypso CLI is planned to provide higher-level capabilities to manage validate and explore different capabilities and building blocks provided by `Kalypso`.