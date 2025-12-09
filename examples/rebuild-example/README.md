# Plugin Rebuild Example

This example demonstrates how to use the plugin rebuild features in wasm-go, specifically:

- `WithRebuildAfterRequests`: Rebuild the plugin after a specified number of requests
- `WithRebuildMaxMemBytes`: Rebuild the plugin when memory usage reaches a specified threshold

## Features

### WithRebuildAfterRequests

Automatically rebuilds the plugin after processing a specified number of requests. This is useful for:
- Preventing memory leaks from accumulating over time
- Resetting plugin state periodically
- Managing long-running plugin instances

Usage:
```go
wrapper.SetCtx(
    "my-plugin",
    wrapper.WithRebuildAfterRequests[MyConfig](1000), // Rebuild after 1000 requests
)
```

### WithRebuildMaxMemBytes

Automatically rebuilds the plugin when its memory usage reaches a specified threshold. This is useful for:
- Preventing out-of-memory errors
- Managing plugins with dynamic memory growth
- Ensuring consistent performance

Usage:
```go
wrapper.SetCtx(
    "my-plugin",
    wrapper.WithRebuildMaxMemBytes[MyConfig](100*1024*1024), // Rebuild at 100MB
)
```

## Configuration

Both options can be used together:

```go
wrapper.SetCtx(
    "my-plugin",
    wrapper.WithRebuildAfterRequests[MyConfig](1000),
    wrapper.WithRebuildMaxMemBytes[MyConfig](100*1024*1024),
)
```

When either condition is met, the plugin will be automatically rebuilt.

## Memory Monitoring

The plugin VM memory usage can be monitored using:

```go
data, err := proxywasm.GetProperty([]string{"plugin_vm_memory"})
if err == nil && len(data) == 8 {
    memorySize := binary.LittleEndian.Uint64([]byte(data))
    // memorySize is in bytes
}
```

## Building

```bash
tinygo build -o main.wasm -scheduler=none -target=wasi -gc=custom -tags='custommalloc nottinygc_finalizer' main.go
```

## Testing

Deploy the compiled WASM file and send requests to trigger the rebuild conditions.

