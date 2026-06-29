# Kalypso

A series of high-level, opinionated Custom Resources for Kubernetes-based Platforms, powered by [KRO](https://kro.run) (Kube Resource Operator).

<p align="center">
    <img src="./kalypso.png" width="300">
</p>

Kalypso provides platform engineers and developers with ready-to-use building blocks—called **Capabilities**—which aggregate standard, low-level Kubernetes resources into clean, simplified high-level CRDs.

---

## How Kalypso Works

Kalypso abstracts multi-resource deployments by providing a simple, high-level developer spec that automatically generates, binds, and manages a complete set of underlying Kubernetes resources.

![Kalypso Concept Diagram](docs/images/kalypso-concept-d2.svg)

---

## List of Resource Graph Definitions (RGDs)

| RGD Name | Kind | API Version | Scope | Description |
| --- | --- | --- | --- | --- |
| **Core** | `Core` | `kalypso.lmoet.io/v1alpha2` | Namespaced | User-facing API for defining and running application workloads and associated operational addons. |
| **PodSpec** | `PodSpec` | `kalypso.lmoet.io/v1alpha2` | Namespaced | *Internal* type chained by `Core` to resolve container environments, probes, resource bounds, and scheduling settings. |

---

## Quick Example: Using `Core`

Here is a minimal deployment example using the `Core` RGD:

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: hello-kalypso
  namespace: default
spec:
  image: gcr.io/google-samples/hello-app:1.0
  port: 8080
  replicas: 2
```

---

## Kubernetes Compatibility Matrix

Kalypso requires Kubernetes versions with stable Common Expression Language (CEL) validation in Custom Resource Definitions.

| Kubernetes Version | Support Status | Notes |
| --- | --- | --- |
| **v1.32.x** | **Fully Compatible** | Tested in our E2E framework suite (`kindest/node:v1.32.2`). |
| **v1.31.x** | **Compatible** | Recommended, matches KRO v0.9.x features. |
| **v1.30.x** | **Compatible** | Recommended, matches KRO v0.9.x features. |
| **v1.29.x** | **Compatible** | Minimum recommended version for stable CEL support. |
| **&lt;= v1.28.x** | **Not Recommended** | Older CEL feature gates might result in schema compilation errors. |
