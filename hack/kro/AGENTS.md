# POC

I want to see how far a KRO's ResourceGraphDefinition and if it's suitable for my needs.

## Goal

I want to create a high-level ResourceGraphDefinition called Workload and I want this CRD to wrap various other Kubernetes resources.

The Workload CRD must only expose very high-level directives in its spec. Each spec key must be treated as a "capability".

## Example

```yaml
kind: Workload
spec:
    Compute:
      # Configures Deployment & HPA 
    Networking:
      # Configures Service, VS, DR, SE, etc
    Observability:
      # Configures ServiceMonitor
    Security:
      # Configures securityContext and other security-related directives/resources
```