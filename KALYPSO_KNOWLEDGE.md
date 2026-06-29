# Kalypso & KRO Knowledge Base

This document contains compiled architectural and syntax knowledge for developing **Kalypso** capabilities using **KRO** (Kubernetes Resource Operator).

---

## 1. Kalypso Architecture

Kalypso provides high-level, opinionated Kubernetes Custom Resource Definitions (CRDs) called **Capabilities** (Core, Networking, Observability, Security). These capabilities are aggregated under a cluster-scoped `Workload` CRD using **RGD Chaining**. Users can also create their own aggregation layers by defining other high-level CRDs that reference the lower-level Kalypso capabilities. By abstracting the underlying Kubernetes resources, Kalypso aims to provide a simplified and more user-friendly interface for managing Kubernetes resources.

---

## 2. KRO Schema Syntax Reference (SimpleSchema)

KRO schemas define the API of the generated CRDs.

### Types and Formats

- **Scalars**: `string`, `integer`, `boolean`, `number` / `float`.
- **Arrays**: `"[]string"`, `"[]integer"`, `"[]object"`, `"[]MyType"` (must be quoted in YAML).
- **Maps**: `"map[string]string"`, `"map[string]MyType"`.
- **Free-form**: `object` (bypasses KRO validation and forwards unstructured data).
- **Custom Types**: Reusable structures declared under `schema.types`.

### Markers (using the `|` syntax)

- `required=true`: The field must be provided.
- `default=<val>`: Sane fallback when the field is omitted.
- `minimum=<num>` / `maximum=<num>`: Numeric validation bounds.
- `enum="a,b,c"`: Restricts allowed string values.
- `pattern="regex"`: Regular expression match.
- `description="..."`: Documentation for fields.
- `immutable=true`: Disallows changes after creation.

---

## 3. CEL Expressions & Graph Resolution

KRO maps instance fields into Kubernetes resources using Common Expression Language (CEL) inside `${...}`.

- **Dynamic Refs**: `${schema.spec.fieldName}`.
- **Cross-Resource Refs**: `${resourceId.metadata.name}` or `${resourceId.status.readyReplicas}`. This establishes implicit execution ordering.
- **List Concatenation**: Combining arrays using `+`, e.g. `listA + listB`.
- **Ternary Operator**: `condition ? valueIfTrue : valueIfFalse`.

---

## 4. Conditional Resource Graph Patterns

### `includeWhen`

- List of CEL boolean conditions.
- If any condition is false, KRO does not create the resource (and deletes it if it already exists).
- Cascading: if resource `A` is skipped, all resources referencing `A` are also skipped.

### Exclusive Workloads Pattern

To support only one of Deployment, StatefulSet, or DaemonSet at a time, each resource has an `includeWhen` checking the `workloadType`:
- **Deployment**: `${schema.spec.workloadType == "Deployment"}`
- **StatefulSet**: `${schema.spec.workloadType == "StatefulSet"}`
- **DaemonSet**: `${schema.spec.workloadType == "DaemonSet"}`

### Status reporting

Each `Kalypso` CRDs defined via `KRO`, will provide a `status` block to report its status and the status of its dependencies. We must leverage this information to provide a unified status for the `Workload` CRD.

Each status will be aggregated by the child's type. E.g:

```yaml
status:
  core:
    ready: true
  networking:
    ready: true
  observability: 
    ready: true
  # etc...
```

### Conditional Status Mapping

To safely reference resource statuses when some resources might be excluded:
```yaml
readyReplicas: ${schema.spec.workloadType == "Deployment" ? deployment.status.readyReplicas : (schema.spec.workloadType == "StatefulSet" ? statefulSet.status.readyReplicas : daemonSet.status.numberReady)}
```
*Note: The short-circuiting in CEL prevents evaluating properties of uncreated resources.*

---

## 5. Key KRO Gotchas & Best Practices

### A. CEL Ternary Operator Type Matching

In CEL, the ternary operator `cond ? branch1 : branch2` requires both branches to evaluate to the **exact same static type** at compilation time. Mixing custom objects, maps, or lists with primitive types or `null` will throw type-checking errors.
* **Solution**: Wrap both branches in the `dyn()` function to cast them to type `any`.
  ```yaml
  nodeSelector: '${has(schema.spec.scheduling.nodeSelector) ? dyn(schema.spec.scheduling.nodeSelector) : dyn(null)}'
  ```

### B. Guarding Optional Fields

Accessing optional fields that are omitted in a Custom Resource instance throws a `no such key` error during graph reconciliation.
* **Solution**: Always guard optional fields using the `has()` function:
  ```yaml
  livenessProbe: '${schema.spec.probes.liveness.enabled ? (has(schema.spec.probes.liveness.custom) ? schema.spec.probes.liveness.custom : dyn({...})) : null}'
  ```

### C. Handling Kubernetes `IntOrString` Types

Fields like `maxUnavailable` in a `PodDisruptionBudget` accept either raw integers (`1`) or percentage strings (`"50%"`).
* Declaring the schema field as an `object` creates an OpenAPI validation mismatch (since Kubernetes expects a JSON object map for `object` schemas, not a scalar integer or string).
* Declaring it as a `string` causes Kubernetes validation to reject numeric string representations (like `"1"`) because it expects strings to end with a `%` symbol.
* **Solution**: Keep the schema type as `string` (e.g. `default="1"`), and handle the type casting dynamically at runtime in the CEL template using `endsWith()` and `int()` wrapped in `dyn()`:
  ```yaml
  maxUnavailable: '${schema.spec.pdb.maxUnavailable.endsWith("%") ? dyn(schema.spec.pdb.maxUnavailable) : dyn(int(schema.spec.pdb.maxUnavailable))}'
  ```

### D. RGD Dependency Validation

When testing chained RGDs (e.g., `Core` referencing `PodSpec` custom types), the KRO CLI `kro validate` command cannot resolve the referenced schema unless it is already applied to the Kubernetes cluster.
* **Solution**: In E2E tests, apply the dependee RGD (`PodSpec`) first, wait for its CRD to become established, and then validate and apply the dependent RGD (`Core`).

