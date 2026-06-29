# Kalypso Documentation - v1alpha2

Welcome to the Kalypso documentation. Kalypso simplifies the deployment and management of Kubernetes workloads by providing high-level, opinionated Custom Resource Definitions (CRDs) built on top of [KRO](https://kro.run) (Kube Resource Operator).

This folder contains design documents, guides, and specifications for the capabilities provided by Kalypso.

---

## Capabilities Overview

The core capability implemented in `v1alpha2` is the **Core** capability, which is defined across two Resource Graph Definitions (RGDs):
1. **Core RGD** ([core-rgd.yaml](../capabilities/core/v1alpha2/core-rgd.yaml)): The main user-facing Namespaced API that represents a logical application workload.
2. **PodSpec RGD** ([podspec-rgd.yaml](../capabilities/core/v1alpha2/podspec-rgd.yaml)): An internal helper RGD chained by `Core` to resolve container specs, environment variables, mounts, and scheduling policies.

---

## Architectural Flow

# Core RGD

The following diagram illustrates how a single `Core` Custom Resource is processed by KRO to automatically generate various standard Kubernetes resources based on your specification.

![Core Flow Diagram](images/core-flow2.svg)

---

## API Schema Reference

Below is a detailed reference of the schema fields defined in the `Core` spec.

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `computeType` | `string` | `"Deployment"` | The type of workload to run. Allowed values: `Deployment`, `StatefulSet`, `DaemonSet`. |
| `replicas` | `integer` | `1` | Number of replicas to run (applicable to `Deployment` and `StatefulSet` only). |
| `image` | `string` | *Required* | Container image to run. |
| `port` | `integer` | `8080` | Container port that the application listens on. |
| `command` | `[]string` | `[]` | Entrypoint array. |
| `args` | `[]string` | `[]` | Arguments to the entrypoint. |
| `env` | `[]object` | `[]` | Environment variables list (standard EnvVar schema). |
| `resources` | `object` | Requests: `cpu: 200m`, `memory: 128Mi` | Resource CPU/Memory requests and limits. |
| `probes` | `object` | Liveness/Readiness enabled | Health checking probes configurations (Liveness, Readiness, Startup). |
| `serviceAccount.create` | `boolean` | `true` | Whether to create a dedicated ServiceAccount. |
| `serviceAccount.name` | `string` | `""` | Dedicated ServiceAccount override name. |
| `configMap.enabled` | `boolean` | `false` | When true, creates a ConfigMap populated with `configMap.data` and mounts it via `envFrom`. |
| `secret.enabled` | `boolean` | `false` | When true, creates a Secret populated with `secret.data` and mounts it via `envFrom`. |
| `pdb.enabled` | `boolean` | `true` | Creates a PodDisruptionBudget. |
| `pdb.maxUnavailable` | `string` | `"1"` | Max unavailable pods. Can be integer or percentage string. |
| `autoscaling.enabled` | `boolean` | `false` | Enforce HPA for autoscaling. |
| `volumeClaimTemplates` | `[]object` | `[]` | PVC templates (for StatefulSet workloads). |
| `rbac.enabled` | `boolean` | `false` | Whether to enable RBAC resources creation. |
| `rbac` | `object` | `{}` | Namespaced rules and cluster rules list. |
| `networkPolicy.enabled` | `boolean` | `false` | Whether to enable NetworkPolicy creation for this workload. |
| `networkPolicy` | `object` | `{}` | Ingress/Egress traffic network policies. |

---

## Detailed Examples

### 1. Minimal Deployment
A simple HTTP hello application with default liveness and readiness probes.

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: hello-minimal
  namespace: default
spec:
  image: gcr.io/google-samples/hello-app:1.0
  port: 8080
```

### 2. Complete Deployment with HPA & PDB
Features customized resources, environment variables, environment variable binding, custom readiness checks, HPA, and a percentage-based PDB.

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: app-production
  namespace: default
spec:
  computeType: Deployment
  image: gcr.io/google-samples/hello-app:2.0
  port: 8080
  replicas: 3
  env:
    - name: ENV_MODE
      value: "production"
  resources:
    requests:
      cpu: "500m"
      memory: "256Mi"
    limits:
      cpu: "1000m"
      memory: "512Mi"
  probes:
    readiness:
      path: "/healthz"
      port: 8080
      initialDelaySeconds: 5
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilization: 75
  pdb:
    enabled: true
    maxUnavailable: "33%"
  configMap:
    enabled: true
    data:
      APP_COLOR: "blue"
      LOG_LEVEL: "info"
```

### 3. StatefulSet with Persistent Volumes
Demonstrates defining a stateful workload running 2 replicas with standard volume claims templates.

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: database-stateful
  namespace: default
spec:
  computeType: StatefulSet
  image: redis:7.0-alpine
  port: 6379
  replicas: 2
  volumeClaimTemplates:
    - metadata:
        name: redis-data
      spec:
        accessModes: [ "ReadWriteOnce" ]
        resources:
          requests:
            storage: 5Gi
```

### 4. DaemonSet Configuration
Runs a lightweight agent on all nodes.

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: logging-agent
  namespace: kube-system
spec:
  computeType: DaemonSet
  image: fluent/fluent-bit:2.1
  port: 2020
  serviceAccount:
    create: true
```

### 5. Workload with Custom RBAC permissions
Allows the application pod to inspect namespaces and read nodes.

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: Cluster-watcher
  namespace: monitoring
spec:
  image: alpine:latest
  command: ["/bin/sh", "-c", "sleep 3600"]
  serviceAccount:
    create: true
  rbac:
    enabled: true
    rules:
      - apiGroups: [""]
        resources: ["namespaces"]
        verbs: ["get", "list", "watch"]
    clusterRules:
      - apiGroups: [""]
        resources: ["nodes"]
        verbs: ["get", "list"]
```

### 6. Workload with Ingress/Egress Network Policies
Restricts ingress traffic to only allow connections from pods with label `role: frontend`, and egress traffic to only target the cluster DNS in `kube-system`.

```yaml
apiVersion: kalypso.lmoet.io/v1alpha2
kind: Core
metadata:
  name: backend-secure
  namespace: default
spec:
  image: gcr.io/google-samples/hello-app:1.0
  networkPolicy:
    enabled: true
    ingress:
      allowFrom:
        - from:
            - podSelector:
                matchLabels:
                  role: frontend
          ports:
            - protocol: TCP
              port: 8080
    egress:
      allowTo:
        - to:
            - namespaceSelector:
                matchLabels:
                  kubernetes.io/metadata.name: kube-system
          ports:
            - protocol: UDP
              port: 53
```
