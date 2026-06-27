# Implementation Plan - Kalypso High-Level Kubernetes CRDs

Implement the Compute capability using only Kubernetes core resources.

## Kubernetes Resource support list

- Deployment/StatefulSet/DaemonSet (only one may be used at a time)
- Horizontal Pod Autoscaler
- PodDisruptionBudget
- Pod Affinity/Anti-Affinity/Node Affinity/Topology Spread Constraints
- ConfigMap / Secret for container environments
- Service Account
