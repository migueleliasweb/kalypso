# CODING STANDARDS

- Use multiline arguments when dealing with > 2 arguments
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