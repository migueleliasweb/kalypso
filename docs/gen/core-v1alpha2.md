# Core API Schema Reference

- **Group:** `kalypso.lmoet.io`
- **Version:** `v1alpha2`
- **Scope:** `Namespaced`

## Spec Schema

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `computeType` | `string` | No | `Deployment` | The type of compute to run (Enum: `Deployment`, `StatefulSet`, `DaemonSet`) |
| `replicas` | `integer` | No | `1` | Number of desired replicas (only for Deployment/StatefulSet) |
| `image` | `string` | Yes | - | Container image to run |
| `port` | `integer` | No | `8080` | Container port the application listens on |
| `command` | `[]string` | No | `[]` | - |
| `args` | `[]string` | No | `[]` | - |
| `env` | `[]object` | No | `[]` | - |
| `resources.requests.cpu` | `string` | No | `200m` | - |
| `resources.requests.memory` | `string` | No | `128Mi` | - |
| `resources.limits.cpu` | `string` | No | - | - |
| `resources.limits.memory` | `string` | No | - | - |
| `probes.liveness.enabled` | `boolean` | No | `true` | - |
| `probes.liveness.path` | `string` | No | `/healthz` | - |
| `probes.liveness.port` | `integer` | No | `8080` | - |
| `probes.liveness.initialDelaySeconds` | `integer` | No | `0` | - |
| `probes.liveness.periodSeconds` | `integer` | No | `10` | - |
| `probes.liveness.timeoutSeconds` | `integer` | No | `1` | - |
| `probes.liveness.successThreshold` | `integer` | No | `1` | - |
| `probes.liveness.failureThreshold` | `integer` | No | `3` | - |
| `probes.liveness.custom` | `object` | No | - | - |
| `probes.readiness.enabled` | `boolean` | No | `true` | - |
| `probes.readiness.path` | `string` | No | `/readyz` | - |
| `probes.readiness.port` | `integer` | No | `8080` | - |
| `probes.readiness.initialDelaySeconds` | `integer` | No | `0` | - |
| `probes.readiness.periodSeconds` | `integer` | No | `10` | - |
| `probes.readiness.timeoutSeconds` | `integer` | No | `1` | - |
| `probes.readiness.successThreshold` | `integer` | No | `1` | - |
| `probes.readiness.failureThreshold` | `integer` | No | `3` | - |
| `probes.readiness.custom` | `object` | No | - | - |
| `probes.startup.enabled` | `boolean` | No | `false` | - |
| `probes.startup.path` | `string` | No | `/healthz` | - |
| `probes.startup.port` | `integer` | No | `8080` | - |
| `probes.startup.initialDelaySeconds` | `integer` | No | `0` | - |
| `probes.startup.periodSeconds` | `integer` | No | `10` | - |
| `probes.startup.timeoutSeconds` | `integer` | No | `1` | - |
| `probes.startup.successThreshold` | `integer` | No | `1` | - |
| `probes.startup.failureThreshold` | `integer` | No | `3` | - |
| `probes.startup.custom` | `object` | No | - | - |
| `restartPolicy` | `string` | No | `Always` | Allowed values (Enum: `Always`) |
| `serviceAccount.create` | `boolean` | No | `true` | - |
| `serviceAccount.name` | `string` | No | - | - |
| `scheduling.nodeSelector` | `map[string]string` | No | - | - |
| `scheduling.affinity` | `object` | No | - | - |
| `scheduling.tolerations` | `[]object` | No | `[]` | - |
| `scheduling.topologySpread.enabled` | `boolean` | No | `true` | - |
| `scheduling.topologySpread.maxSkew` | `integer` | No | `1` | - |
| `scheduling.topologySpread.topologyKey` | `string` | No | `kubernetes.io/hostname` | - |
| `scheduling.topologySpread.whenUnsatisfiable` | `string` | No | `ScheduleAnyway` | - |
| `scheduling.topologySpread.customConstraints` | `[]object` | No | `[]` | - |
| `configMap.enabled` | `boolean` | No | `false` | - |
| `configMap.data` | `map[string]string` | No | - | - |
| `secret.enabled` | `boolean` | No | `false` | - |
| `secret.data` | `map[string]string` | No | - | - |
| `volumes` | `[]object` | No | `[]` | - |
| `volumeMounts` | `[]object` | No | `[]` | - |
| `pdb.enabled` | `boolean` | No | `true` | Whether to create a PodDisruptionBudget |
| `pdb.maxUnavailable` | `string` | No | `1` | Number/percentage of pods that can be unavailable |
| `autoscaling.enabled` | `boolean` | No | `false` | - |
| `autoscaling.minReplicas` | `integer` | No | `1` | - |
| `autoscaling.maxReplicas` | `integer` | No | `10` | - |
| `autoscaling.targetCPUUtilization` | `integer` | No | `80` | - |
| `volumeClaimTemplates` | `[]object` | No | `[]` | - |
| `rbac.enabled` | `boolean` | No | `false` | Whether to enable RBAC resources creation |
| `rbac.rules` | `[]object` | No | `[]` | Optional namespaced RBAC rules (creates Role & RoleBinding) |
| `rbac.clusterRules` | `[]object` | No | `[]` | Optional cluster-wide RBAC rules (creates ClusterRole & ClusterRoleBinding) |
| `networkPolicy.enabled` | `boolean` | No | `false` | Whether to enable NetworkPolicy creation for this workload |
| `networkPolicy.ingress.allowFrom` | `[]object` | No | `[]` | List of ingress rules (podSelector, namespaceSelector, ipBlock, ports) |
| `networkPolicy.egress.allowTo` | `[]object` | No | `[]` | List of egress rules (podSelector, namespaceSelector, ipBlock, ports) |
