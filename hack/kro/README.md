# KRO Workload POC

A proof-of-concept exploring KRO (`Kube Resource Orchestrator`) `ResourceGraphDefinition`s
as high-level, capability-based workload abstractions.

- [`workload-rgd.yaml`](workload-rgd.yaml) — the namespaced `Workload` CRD (Compute,
  Networking, Observability, Security capabilities → Deployment/HPA/Service/Istio/ServiceMonitor).
- [`clusterworkload-rgd.yaml`](clusterworkload-rgd.yaml) — the cluster-scoped
  `ClusterWorkload` CRD that creates/owns a namespace and chains a `Workload` into it.
- [`AGENTS.md`](AGENTS.md) — the spec. [`AGENTS-KRO.md`](AGENTS-KRO.md) — a KRO reference.

## Running the e2e tests

The tests are fully Go-native — a single command spins up a real cluster, installs
everything, applies the RGDs, and asserts the resulting resources.

```bash
cd e2e
go test -v -timeout 20m
```

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (running)
- [kind](https://kind.sigs.k8s.io/) (`v0.26+`)
- [Go](https://go.dev/dl/) (`1.24+`)

No `kubectl`/`helm` needed — the suite talks to the cluster directly via the Go client.

### What the run does

1. Deletes any leftover `kro-e2e` cluster, then creates a fresh KinD cluster.
2. Downloads the Istio + Prometheus CRDs and the KRO install bundle (cached under
   `crd-manifests/`, gitignored — fetched once, reused on later runs) and applies them.
3. Grants the KRO controller RBAC, applies both RGDs, and waits for them to become `Active`.
4. Runs the feature tests:
   - `workload/minimal` — compute-only Workload → only a Deployment + Service.
   - `workload/full` — every capability on → Deployment + HPA + Service + Istio
     VirtualService/DestinationRule + ServiceMonitor.
   - `clusterworkload` — a cluster-scoped ClusterWorkload that owns a `tenant-a` namespace
     containing a chained Workload + its Deployment/Service.

First run is slower (image pulls); later runs reuse the cached manifests.

### Inspecting after a run

By default the cluster **and** the created resources are left running for troubleshooting:

```bash
kind export kubeconfig --name kro-e2e

kubectl get rgd
kubectl get workload,deploy,svc,hpa,virtualservice,destinationrule,servicemonitor -n workloads
kubectl get clusterworkload
kubectl get all -n tenant-a
```

### Cleaning up

Set `KRO_E2E_DESTROY_CLUSTER` to tear down the resources and the cluster at the end:

```bash
cd e2e
KRO_E2E_DESTROY_CLUSTER=true go test -v -timeout 20m
```

Or just delete the cluster directly:

```bash
kind delete cluster --name kro-e2e
```

### Upgrading pinned versions

KRO / Istio / Prometheus-operator / KinD node versions live as constants in
[`e2e/versions.go`](e2e/versions.go); bump them there and delete `crd-manifests/` to
re-fetch.
