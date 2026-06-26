# POC — KRO `Workload` capability abstraction

## Intent

See how far KRO's `ResourceGraphDefinition` (RGD) can be taken and whether it's suitable
for our needs. Concretely: can a single high-level `Workload` CRD wrap many lower-level
Kubernetes resources, exposing only **capabilities** in its spec while hiding the
underlying Deployment/HPA/Service/Istio/ServiceMonitor machinery?

KRO reference material lives in [AGENTS-KRO.md](AGENTS-KRO.md).

## Goal

Create a high-level RGD named **`workload-rgd.yaml`** that generates a `Workload` CRD.
Each top-level `spec` key is a **capability**; the user sets high-level directives and the
RGD expands them into the right resources.

Capabilities (camelCase, idiomatic K8s/KRO):

| Capability      | Wraps                                              |
|-----------------|----------------------------------------------------|
| `compute`       | Deployment, HorizontalPodAutoscaler                |
| `networking`    | Service, (Istio) VirtualService + DestinationRule  |
| `observability` | ServiceMonitor                                     |
| `security`      | Pod/container `securityContext`                    |

## Capability schemas

### `compute`
| Field                              | Type    | Default | Notes                          |
|------------------------------------|---------|---------|--------------------------------|
| `image`                            | string  | —       | required                       |
| `replicas`                         | integer | `1`     | min 1; ignored if autoscaling  |
| `port`                             | integer | `8080`  | container port                 |
| `resources.requests.cpu`           | string  | `100m`  |                                |
| `resources.requests.memory`        | string  | `128Mi` |                                |
| `resources.limits.cpu`             | string  | `500m`  |                                |
| `resources.limits.memory`          | string  | `256Mi` |                                |
| `autoscaling.enabled`              | boolean | `false` | gates the HPA                  |
| `autoscaling.minReplicas`          | integer | `1`     |                                |
| `autoscaling.maxReplicas`          | integer | `10`    |                                |
| `autoscaling.targetCPUUtilization` | integer | `80`    | percent                        |

### `networking`
| Field            | Type    | Default | Notes                                   |
|------------------|---------|---------|-----------------------------------------|
| `expose`         | boolean | `true`  | gates the Service                       |
| `port`           | integer | `80`    | Service port (targets `compute.port`)   |
| `mesh.enabled`   | boolean | `false` | gates Istio VirtualService + DestinationRule |
| `mesh.host`      | string  | —       | VirtualService host                     |
| `mesh.gateway`   | string  | `mesh`  | gateway to attach the VirtualService to |

### `observability`
| Field          | Type    | Default     | Notes                  |
|----------------|---------|-------------|------------------------|
| `enabled`      | boolean | `false`     | gates the ServiceMonitor |
| `path`         | string  | `/metrics`  | scrape path            |
| `port`         | string  | `http`      | named Service port to scrape |
| `interval`     | string  | `30s`       | scrape interval        |

### `security`
| Field                    | Type       | Default   | Notes                       |
|--------------------------|------------|-----------|-----------------------------|
| `runAsNonRoot`           | boolean    | `true`    |                             |
| `runAsUser`              | integer    | `1000`    |                             |
| `readOnlyRootFilesystem` | boolean    | `true`    |                             |
| `dropCapabilities`       | `[]string` | `["ALL"]` | container capabilities drop |

## Resource mapping & creation conditions

| Resource id       | Kind (apiVersion)                              | Created when                                  |
|-------------------|------------------------------------------------|-----------------------------------------------|
| `deployment`      | Deployment (`apps/v1`)                          | always                                        |
| `hpa`             | HorizontalPodAutoscaler (`autoscaling/v2`)      | `compute.autoscaling.enabled`                 |
| `service`         | Service (`v1`)                                  | `networking.expose`                           |
| `virtualService`  | VirtualService (`networking.istio.io/v1`)       | `networking.mesh.enabled`                     |
| `destinationRule` | DestinationRule (`networking.istio.io/v1`)      | `networking.mesh.enabled`                     |
| `serviceMonitor`  | ServiceMonitor (`monitoring.coreos.com/v1`)     | `observability.enabled`                       |

Conditional creation uses KRO's `includeWhen`. **Optional-capability behavior is explicit:**
omitting (or disabling) a capability means the corresponding resource is *not* created —
disable `observability` ⇒ no ServiceMonitor; leave `autoscaling.enabled` false ⇒ no HPA;
leave `mesh.enabled` false ⇒ no Istio resources. This is the core property the spike must
demonstrate.

## Example instances

Minimal — just compute, exposed via a plain Service:

```yaml
apiVersion: kalypso.io/v1alpha1
kind: Workload
metadata:
  name: hello
spec:
  compute:
    image: ghcr.io/acme/hello:1.0.0
```

Full — every capability on:

```yaml
apiVersion: kalypso.io/v1alpha1
kind: Workload
metadata:
  name: payments
spec:
  compute:
    image: ghcr.io/acme/payments:2.3.1
    port: 8080
    resources:
      requests: { cpu: 250m, memory: 256Mi }
      limits:   { cpu: "1",  memory: 512Mi }
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 20
      targetCPUUtilization: 70
  networking:
    expose: true
    port: 80
    mesh:
      enabled: true
      host: payments.acme.internal
      gateway: mesh
  observability:
    enabled: true
    path: /metrics
    port: http
    interval: 15s
  security:
    runAsNonRoot: true
    runAsUser: 1000
    readOnlyRootFilesystem: true
    dropCapabilities: ["ALL"]
```

## Dependencies

- **KRO** controller installed in the target cluster.
- **Istio** CRDs (`networking.istio.io`) — only needed when `mesh.enabled`.
- **Prometheus Operator** CRDs (`monitoring.coreos.com`) — only needed when
  `observability.enabled`.

## Success criteria for the spike

1. RGD is accepted by KRO and generates a `Workload` CRD that reaches `Active`.
2. A minimal `Workload` (compute only) produces exactly a Deployment + Service.
3. **Optional-capability test:** toggling `autoscaling.enabled`, `mesh.enabled`, and
   `observability.enabled` adds/removes the HPA, Istio VS+DR, and ServiceMonitor
   respectively — validating `includeWhen`.
4. Status surfaces useful runtime info (e.g. available replicas) back onto the instance.
5. **Abstraction-leak assessment (the real question):** when a user needs a knob the
   capability schema doesn't expose, what's the escape hatch, and is the
   golden-path-vs-expressivity trade-off acceptable? Record findings here rather than
   solving it in this first cut.

> Verification/testing is intentionally out of scope for this first cut — a dedicated
> testing harness will be built afterward.
