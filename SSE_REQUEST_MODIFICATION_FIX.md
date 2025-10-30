# SSE Request Modification Architecture Fix

## Problem with Original Implementation

The original implementation used `ctx.RouteCall()` to send a separate GET request for establishing the SSE channel:

```go
// WRONG APPROACH
return ctx.RouteCall("GET", finalURL, finalHeaders, nil, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
    // This callback interferes with streaming response processing
})
```

**Issues**:
1. `ctx.RouteCall()` creates a separate HTTP request with its own callback
2. The callback conflicts with the normal HTTP filter chain flow
3. Streaming response hooks (`onHttpResponseHeaders`, `onHttpStreamingResponseBody`) may not be triggered correctly
4. Creates unnecessary complexity and potential race conditions

## Correct Implementation

The correct approach is to **modify the current request in-place** during the `onHttpRequestBody` phase:

```go
// CORRECT APPROACH
func initiateSSEChannelInRequestPhase(ctx wrapper.HttpContext, server *McpProxyServer, authInfo *ProxyAuthInfo) error {
    // Parse target URL
    parsedURL, err := url.Parse(finalURL)
    
    // Modify the current request to be a GET request
    proxywasm.ReplaceHttpRequestHeader(":method", "GET")
    proxywasm.ReplaceHttpRequestHeader(":path", path)
    proxywasm.ReplaceHttpRequestHeader(":authority", authority)
    // Note: :scheme is managed by Envoy and should not be modified
    
    // Remove headers not appropriate for GET
    proxywasm.RemoveHttpRequestHeader("content-type")
    proxywasm.RemoveHttpRequestHeader("content-length")
    proxywasm.RemoveHttpRequestHeader("transfer-encoding")
    
    // Set Accept header for SSE
    proxywasm.ReplaceHttpRequestHeader("accept", "text/event-stream")
    
    // Apply authentication headers
    for _, header := range finalHeaders {
        // Apply additional headers...
    }
    
    // Let the request continue through the filter chain
    return nil
}
```

## Why This Works Better

### 1. **Natural Filter Chain Flow**
- The modified request continues through the normal HTTP filter chain
- `onHttpResponseHeaders` is triggered naturally when response headers arrive
- `onHttpStreamingResponseBody` is triggered for each chunk of SSE data
- No need for special callback handling

### 2. **No Pausing Required**
```go
// Original (WRONG):
ctx.SetContext(utils.CtxNeedPause, true)  // Pause request processing
return nil

// Fixed (CORRECT):
// Explicitly set to NOT pause - let the request continue to establish SSE channel
ctx.SetContext(utils.CtxNeedPause, false)
return nil
```

### 3. **Simplified State Management**
- State is stored in context before request modification
- Response handlers naturally access this state
- No need to coordinate between separate request callbacks

## Request Modification Details

### Pseudo-Headers (HTTP/2)
These headers must be modified to transform the request:

1. **`:method`**: Changed from `POST` to `GET`
2. **`:path`**: Set to target MCP server path (including query string)
3. **`:authority`**: Set to target host:port
4. **`:scheme`**: ~~Set to `http` or `https`~~ **Not modified** - managed by Envoy automatically

### Headers to Remove
- `content-type`: GET requests don't have a body
- `content-length`: GET requests don't have a body
- `transfer-encoding`: Not needed for GET
- Original `accept`: Will be replaced with SSE-specific value

### Headers to Add/Modify
- `accept`: Set to `text/event-stream` (required for SSE)
- Authentication headers: Applied from security schemes
- Other original headers: Preserved (except those in skip list)

## Implementation Flow

### 1. Request Phase (`onHttpRequestBody`)
```
Client POST /mcp with JSON-RPC request
    ↓
handleSSEToolsList/handleSSEToolsCall
    ↓
Store: server, allowTools, JSON-RPC ID, request body, headers, auth info
    ↓
initiateSSEChannelInRequestPhase()
    ↓
Modify request headers: :method=GET, :path=/mcp, accept=text/event-stream
    ↓
Return without pausing → request continues to backend
```

### 2. Response Phase (`onHttpResponseHeaders`)
```
Backend returns 200 OK with content-type: text/event-stream
    ↓
onHttpResponseHeaders()
    ↓
Validate content-type is text/event-stream
    ↓
Call ctx.NeedPauseStreamingResponse()
    ↓
Return HeaderStopIteration → enter streaming mode
```

### 3. Streaming Phase (`onHttpStreamingResponseBody`)
```
Backend sends SSE chunks
    ↓
onHttpStreamingResponseBody()
    ↓
Buffer and parse SSE messages
    ↓
State machine: WaitingEndpoint → Initializing → WaitingInitResp → WaitingNotifyResp → WaitingToolResp
    ↓
Inject JSON-RPC response via proxywasm.InjectEncodedDataToFilterChain()
```

## Key Differences from Original Design

| Aspect | Original (Wrong) | Fixed (Correct) |
|--------|------------------|-----------------|
| Request method | `ctx.RouteCall()` | Modify current request headers |
| Callback | Separate callback function | Natural filter chain hooks |
| Pausing | `ctx.SetContext(utils.CtxNeedPause, true)` | `ctx.SetContext(utils.CtxNeedPause, false)` (explicit) |
| Flow | Asynchronous with callback | Synchronous through filter chain |
| Complexity | High (two request paths) | Low (single request path) |

## Authentication Handling

Authentication is still applied before request modification:

1. **Extract downstream credentials** (if passthrough enabled)
2. **Prepare auth info** structure
3. **Apply authentication** to headers and URL using `applyProxyAuthenticationForSSE()`
4. **Modify request headers** with authenticated headers
5. **Continue request** to backend with authentication

## Benefits of This Approach

1. ✅ **Simpler**: Single request path through filter chain
2. ✅ **More reliable**: Natural hook triggering
3. ✅ **Better debugging**: Easier to trace request flow
4. ✅ **Consistent**: Same pattern as other proxy operations
5. ✅ **No race conditions**: Synchronous header modification
6. ✅ **Explicit control**: `CtxNeedPause` is explicitly set to `false`, no reliance on default values

## Code Structure

### Files Modified
- `pkg/mcp/server/proxy_tool.go`:
  - `handleSSEToolsList()`: Removed pause logic
  - `handleSSEToolsCall()`: Removed pause logic
  - `initiateSSEChannelInRequestPhase()`: Completely rewritten to modify headers instead of RouteCall

### No Changes Needed
- `pkg/mcp/server/plugin.go`: Response handlers unchanged
- `pkg/mcp/server/sse_proxy.go`: State machine unchanged

## Testing

All existing tests pass without modification:
- `TestParseSSEMessage`
- `TestExtractEndpointURL`
- `TestTransportProtocolValidation`
- `TestMcpProxyServerTransport`
- `TestSSEMessageParsing_MultipleMessages`

The fix is purely architectural and doesn't affect the SSE protocol implementation or message parsing logic.

## Conclusion

This fix addresses a fundamental architectural issue in the SSE proxy implementation. By modifying the current request instead of creating a separate one, we achieve a cleaner, more reliable, and easier-to-maintain solution that follows standard proxy patterns.

