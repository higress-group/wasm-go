# HTTP Call Example

This example demonstrates how to make HTTP calls to external services from a WASM plugin and how to test them using the test framework.

## Features

- Makes HTTP POST request to external service using FQDN client
- Configurable FQDN, port, and path
- Adds external response to request headers

## Configuration

```json
{
  "fqdn": "httpbin.org",
  "port": 80,
  "path": "/post"
}
```

## Testing

The test demonstrates:

- Plugin configuration parsing
- HTTP callout verification using `GetHttpCalloutAttributes()`
- External service response simulation
- Final result verification

Run tests:

```bash
cd examples/http-call
go test -v
```
