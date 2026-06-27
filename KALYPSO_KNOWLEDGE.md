# Kalypso & KRO Knowledge Base

This document contains compiled architectural and syntax knowledge for developing **Kalypso** capabilities using **KRO** (Kube Resource Operator).

---

## 1. Kalypso Architecture

Kalypso provides high-level, opinionated Kubernetes Custom Resource Definitions (CRDs) called **Capabilities** (Compute, Storage, Networking, Observability, Security). These capabilities are aggregated under a cluster-scoped `Workload` CRD using **RGD Chaining**.

### Release V1Alpha1 Goals
- Implement the **Compute** capability as a standalone, namespace-scoped KRO RGD.
- Provide a `Compute` CRD that encapsulates Deployment, StatefulSet, DaemonSet (exclusive, only one at a time), HPA, PDB, ConfigMap, Secret, and Service Account.
- Provide sensible defaults with user override capabilities.

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

### Conditional Status Mapping
To safely reference resource statuses when some resources might be excluded:
```yaml
readyReplicas: ${schema.spec.workloadType == "Deployment" ? deployment.status.readyReplicas : (schema.spec.workloadType == "StatefulSet" ? statefulSet.status.readyReplicas : daemonSet.status.numberReady)}
```
*Note: The short-circuiting in CEL prevents evaluating properties of uncreated resources.*
