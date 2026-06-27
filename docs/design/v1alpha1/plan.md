# Implementation Plan - Kalypso High-Level Kubernetes CRDs

Implement the Compute capability using only Kubernetes core resources.

KRO resources must be placed under `capabilities/compute/v1alpha1`.

## Kubernetes Resource support list

- Deployment/StatefulSet/DaemonSet (only one may be used at a time)
- Horizontal Pod Autoscaler
- PodDisruptionBudget
- Pod Affinity/Anti-Affinity/Node Affinity/Topology Spread Constraints
- ConfigMap / Secret for container environments
- Service Account

## Deliverables

- A `Compute` CRD that provides high level access to the resources listed above
- Sane defaults (with override options)
    - Default resources
    - Default probes (Liveness/Readiness/Startup)
    - Default restart policy
    - Default service account
    - Default pod disruption budget
    - Default topology spread constraints

## RGD

- A RGD that creates a Deployment, Horizontal Pod Autoscaler, PodDisruptionBudget, Pod Affinity/Anti-Affinity/Node Affinity/Topology Spread Constraints, ConfigMap, Secret, Service Account.

The name of the RGD must be the name of the capability: `Compute`.