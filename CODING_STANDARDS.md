# CODING STANDARDS

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