# Safe Log HTTP Call Example

This example demonstrates how to use `EnableSafeLog` to prevent sensitive information (headers and body) from being logged when making HTTP calls to external services.

## Features

- Makes HTTP POST request to external service using FQDN client
- **Enables safe log mode** to prevent logging sensitive data like API keys, tokens, or personal information
- Configurable FQDN, port, and path

## Why Use Safe Log?

By default, the wasm-go framework logs complete HTTP request/response headers and bodies for debugging purposes. This can be a security risk in production environments where:

- API keys or tokens are passed in headers (e.g., `Authorization: Bearer sk-xxx`)
- Sensitive data is included in request/response bodies
- Personal information may be present in HTTP traffic

When `EnableSafeLog` is enabled, these sensitive logs are suppressed.

## Configuration

```json
{
  "fqdn": "api.example.com",
  "port": 443,
  "path": "/v1/chat"
}
```

## Usage

```go
func init() {
    wrapper.SetCtx(
        "my-plugin",
        wrapper.ParseConfig(parseConfig),
        wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
        // Enable safe log mode
        wrapper.EnableSafeLog[MyPluginConfig](),
    )
}
```

## What Gets Suppressed

With safe log enabled, the following logs will NOT be printed:

- `http call start` - request headers, body, URL, cluster info
- `http call end` - response headers, body, status code
- `route call start` - request headers, body
- `route call end` - response headers, body

## Build

```bash
cd examples/safe-log-http-call
tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer' main.go
```
