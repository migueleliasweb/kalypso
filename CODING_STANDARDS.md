# CODING STANDARDS

# Golang

- Use multiline arguments when dealing with > 2 arguments
    - When dealing with > 4 arguments, create a `<funcName>Params` struct and pass it instead
- Do not use pointers when configuring Go structs within `api/` as they will be converted to OpenAPI schemas and used to generate Kuberetes CRDs
- Use newlines for readability between:
    - Function or method calls (before and after)
    - If statements (before and after)
    - Struct initialisation (before and after)
- Use inline error checking when a single error is being returned from a function call or method call. Do NOT use error inline checking when there are multiple returns.
    -   ```go
        if err := functionCall(); err != nil {
            // Handle error
        }
        ```

# Kubernetes Controller Reconcilers

## Single-purpose reconcilers

Each reconciler must only manage the lifecycle of a single Kubernetes resource type. This way, we can ensure retries and alerts can be more details and provide better understanding of what is going on inside the Controller as a whole.

Consider grouping the reconcilers in folder with the capabiltiy they are related to. Ask the user if you need clarification. 

E.g:

- `internal/controller/<capability>/<kind>_controller.go`
- `internal/controller/networking/virtual_service_controller.go`
- `internal/controller/networking/service_controller.go`
- `internal/controller/networking/destination_rule_controller.go`

## The `Reconcile()` method and mutating state

When mutating state from within `Reconcile()` function, never create a separate method only to wrap a call to `controllerutils.CreateOrPath*()`. That call must always be part of the `Reconcile()` method of the reconciler. 

If you think the mutate func of the `CreateOrPatch()` is getting too big (over 80 lines), consider exporting part of the mutate logic to a separate method. When separating the mutate fn logic, take a boundary of a specific property that you are about to mutate.