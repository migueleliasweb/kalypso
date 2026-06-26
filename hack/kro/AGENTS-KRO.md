# KRO Knowledge Dump

A working reference for **KRO — Kube Resource Orchestrator** (`kro.run`), focused on
authoring `ResourceGraphDefinition`s. Compiled from the kro.run docs (see [Sources](#sources)).

## What KRO is

KRO lets you bundle a set of Kubernetes resources behind a single, custom, high-level API.
You author a **`ResourceGraphDefinition` (RGD)**; KRO reads it and:

1. Generates a new **CRD** from the RGD's `schema` (e.g. a `Workload` kind).
2. Runs a **controller** for that CRD that, on each instance, renders the RGD's
   `resources` (with values substituted from the instance) and applies them.
3. Computes a dependency graph from the CEL references between resources and applies
   them in topological order, wiring outputs of one resource into inputs of another.

The net effect: users create one small `Workload` object; KRO materializes the
Deployment, Service, HPA, etc. behind it and keeps them reconciled.

```
RGD (you author) ──▶ KRO generates CRD ──▶ user applies an instance ──▶
    KRO renders + applies the resource graph ──▶ status flows back to the instance
```

## RGD anatomy

```yaml
apiVersion: kro.run/v1alpha1
kind: ResourceGraphDefinition
metadata:
  name: my-rgd                      # name of the RGD object itself
spec:
  schema:
    apiVersion: v1alpha1            # version of the generated CRD
    kind: MyKind                    # generated CRD kind (PascalCase)
    group: mycompany.io            # optional; defaults to kro.run
    scope: Namespaced               # or Cluster (immutable after creation)

    metadata:                       # optional labels/annotations applied to instances
      labels: { team: platform }

    spec:                           # USER INPUT — written in SimpleSchema (see below)
      name: string | required=true
      replicas: integer | default=3

    status:                         # OUTPUT — CEL expressions reading managed resources
      availableReplicas: ${deployment.status.availableReplicas}
      endpoint: "http://${service.spec.clusterIP}"

    types:                          # optional reusable struct definitions
      ContainerConfig:
        image: string | required=true
        tag:   string | default="latest"

    additionalPrinterColumns:       # optional `kubectl get` columns
      - name: Replicas
        type: integer
        jsonPath: .spec.replicas

  resources:                        # the graph of resources to manage
    - id: deployment                # unique id; used as the CEL reference handle
      template:                     # a normal K8s manifest, with ${...} substitutions
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: ${schema.spec.name}
        spec:
          replicas: ${schema.spec.replicas}
    - id: service
      includeWhen:                  # optional: create only when all conditions are true
        - ${schema.spec.expose}
      template:
        apiVersion: v1
        kind: Service
        metadata:
          name: ${schema.spec.name}
```

## SimpleSchema

The `schema.spec` (and `types`) are written in KRO's **SimpleSchema**: a compact
`field: type | marker marker ...` syntax.

### Types

| Category   | Syntax                                   |
|------------|------------------------------------------|
| Scalars    | `string`, `integer`, `boolean`, `number`/`float` |
| Arrays     | `"[]string"`, `"[]integer"`, `"[]MyType"` (quote in YAML) |
| Maps       | `"map[string]string"`, `"map[string]MyType"` |
| Nested     | inline nested object, or reference a custom `types:` entry |
| Custom     | any key under `schema.types` (recursive definitions allowed) |

Collection/map types must be **quoted** in YAML because of the `[]` / `map[...]` brackets.

### Markers (after the `|`)

| Marker             | Purpose                          | Example                          |
|--------------------|----------------------------------|----------------------------------|
| `required=true`    | field must be provided           | `name: string \| required=true`  |
| `default=`         | default when omitted             | `replicas: integer \| default=3` |
| `minimum=` / `maximum=` | numeric bounds              | `\| minimum=1 maximum=100`       |
| `enum="a,b,c"`     | allowed values                   | `env: string \| enum="prod,dev"` |
| `pattern="regex"`  | regex string validation          | `\| pattern="^[a-z-]+$"`         |
| `description="..."`| field documentation              | `\| description="Pod count"`     |
| `immutable=true`   | cannot change after creation     | `name: string \| immutable=true` |

### Example

```yaml
spec:
  name:     string  | required=true immutable=true description="App name"
  replicas: integer | default=1 minimum=1 maximum=100
  image:    string  | required=true
  ingress:
    enabled: boolean | default=false
    host:    string  | default="example.com"
  ports:    "[]integer"
  labels:   "map[string]string"
  container: ContainerConfig          # references schema.types.ContainerConfig
```

## CEL expressions

KRO uses **CEL (Common Expression Language)** inside `${...}`.

- **Input refs:** `${schema.spec.<path>}` — reads the instance's spec.
- **Cross-resource refs:** `${<resourceId>.<path>}` — reads another resource's
  live fields/status. This is what builds the dependency graph and ordering.
- **String templating:** multiple `${...}` inside one string:
  `"postgres://${secret.data.user}@${service.spec.clusterIP}:5432"`.
- **Functions / comprehensions:**
  - `${deployment.status.availableReplicas >= deployment.spec.replicas}`
  - `${pods.items.map(p, p.metadata.name)}`
  - `${pods.items.filter(p, p.status.phase == "Running").size()}`

KRO infers result types from expressions and type-checks them against the K8s schemas
at RGD-creation time.

## Conditional creation — `includeWhen`

```yaml
resources:
  - id: certificate
    includeWhen:
      - ${schema.spec.ingress.enabled}
      - ${schema.spec.ingress.tls}
    template: { ... }
```

- Each entry is a CEL expression that **must return a boolean**.
- Multiple entries are **AND**ed. For **OR**, combine into a single expression
  (`${a || b}`).
- Re-evaluated every reconcile: if a condition flips to false, KRO **removes** the
  resource.
- **Cascading skip:** if a resource is skipped, **every resource that references it is
  also skipped** — so a skippable resource can't be a hard dependency of an
  always-on one.
- **Avoid volatile status fields** in conditions (e.g. replica counts) — they cause
  flip-flop create/delete churn. Prefer stable, user-controlled toggles.

## Status & reserved fields

- `schema.status` fields are CEL expressions surfaced onto the instance's `.status`.
- KRO injects reserved status fields automatically:
  - `conditions` — array tracking instance state.
  - `state` — high-level summary: `ACTIVE`, `IN_PROGRESS`, `FAILED`, `DELETING`, `ERROR`.

## Instance lifecycle

1. `kubectl apply -f my-rgd.yaml` → KRO validates it and generates the CRD; RGD reaches
   `Active`.
2. `kubectl get crd` shows the new kind (e.g. `mykinds.mycompany.io`).
3. User applies an instance of that kind; KRO renders + applies the resource graph.
4. Managed-resource status flows back into the instance's `.status` per the
   `schema.status` CEL.
5. Deleting the instance garbage-collects the managed resources (owner refs).

## Gotchas worth remembering

- Quote any type containing `[]` or `map[...]` in YAML.
- `scope` is immutable after creation.
- A `default=` on a nested toggle (e.g. `mesh.enabled`) is what makes `includeWhen`
  predictable for omitted blocks.
- Cross-resource refs define ordering — there's no manual `dependsOn`; the graph is
  inferred from `${...}`.

## Sources

- [ResourceGraphDefinitions](https://kro.run/docs/concepts/resource-group-definitions/)
- [Schema Definition](https://kro.run/docs/concepts/rgd/schema/)
- [Simple Schema spec](https://kro.run/api/specifications/simple-schema/)
- [Conditional creation (`includeWhen`)](https://kro.run/docs/concepts/rgd/resource-definitions/conditional-creation/)
- [Quick Start](https://kro.run/docs/getting-started/deploy-a-resource-graph-definition/)
