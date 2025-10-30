# SSE Request Phase Improvement

## Overview

This document describes an important architectural improvement to the SSE proxy implementation: initiating the SSE channel during the request body phase (`onHttpRequestBody`) instead of later phases.

## Problem Statement

The original design needed clarification on:
1. When to establish the SSE channel
2. Which JSON-RPC methods should use SSE vs standard handling
3. How to properly integrate SSE with the request/response flow

## Solution

### Method Handling Strategy

**SSE Channel (tools/list and tools/call only)**:
- Only `tools/list` and `tools/call` requests establish an SSE channel
- Other JSON-RPC methods (`initialize`, `notifications/initialized`, `ping`, etc.) follow standard JSON-RPC handling:
  - Return appropriate responses (method not found or json_rpc_ack)
  - Do not establish SSE channels

### Request Phase Initiation

For `tools/list` and `tools/call` requests with SSE transport:

1. **During `onHttpRequestBody` Phase**:
   - Detect SSE transport and route to `handleSSEToolsList` or `handleSSEToolsCall`
   - Store necessary context (JSON-RPC ID, auth info, request body, etc.)
   - Call `initiateSSEChannelInRequestPhase()` which uses `ctx.RouteCall("GET", ...)`
   - Return `ActionPause` to wait for streaming response

2. **During `onHttpResponseHeaders` Phase**:
   - Validate response content-type is `text/event-stream`
   - Call `ctx.NeedPauseStreamingResponse()` to control streaming
   - Return `HeaderStopIteration` to proceed to body processing

3. **During `onHttpStreamingResponseBody` Phase**:
   - Receive SSE chunks
   - Parse SSE messages
   - Process through state machine
   - Inject final JSON-RPC response via `proxywasm.InjectEncodedDataToFilterChain()`

## Key Changes

### 1. Renamed Function

**Before**: `initiateSSEChannel()`
**After**: `initiateSSEChannelInRequestPhase()`

This clearly indicates that the SSE channel is established during the request body phase.

### 2. Explicit ActionPause

Both `handleSSEToolsList` and `handleSSEToolsCall` now explicitly:
```go
// Signal that we need to pause and wait for streaming response
ctx.SetContext(utils.CtxNeedPause, true)
return nil
```

This ensures the request processing pauses and waits for the streaming response.

### 3. Clear Method Routing

Only `tools/list` and `tools/call` methods trigger SSE channel creation:
```go
if server.GetTransport() == TransportSSE {
    return handleSSEToolsList(ctx, id, params, server, allowTools)
}
```

All other methods follow standard JSON-RPC handling through `utils.HandleJsonRpcMethod`.

## Flow Diagram

```
Client Request
    |
    v
onHttpRequestBody
    |
    +-- Non-tools/list/call method --> Standard JSON-RPC handling --> Response
    |
    +-- tools/list or tools/call with SSE transport
        |
        v
    handleSSEToolsList / handleSSEToolsCall
        |
        +-- Store context (ID, auth, request body)
        |
        +-- initiateSSEChannelInRequestPhase()
        |   |
        |   +-- ctx.RouteCall("GET", sseURL, ...)
        |
        +-- ctx.SetContext(CtxNeedPause, true)
        |
        +-- return ActionPause
        |
        v
    Wait for Response...
        |
        v
    onHttpResponseHeaders
        |
        +-- Validate content-type: text/event-stream
        |
        +-- ctx.NeedPauseStreamingResponse()
        |
        +-- return HeaderStopIteration
        |
        v
    onHttpStreamingResponseBody (called multiple times)
        |
        +-- Parse SSE chunks
        |
        +-- State machine processing
        |   |
        |   +-- Wait for endpoint message
        |   +-- Send initialize
        |   +-- Send notification
        |   +-- Send actual tool request
        |   +-- Wait for tool response
        |
        +-- proxywasm.InjectEncodedDataToFilterChain(jsonRpcResponse)
        |
        v
    Response to Client
```

## Benefits

1. **Clear Separation**: SSE logic is only invoked for `tools/list` and `tools/call`
2. **Standard Compliance**: Other JSON-RPC methods follow standard handling
3. **Request Phase Control**: SSE channel is established at the right time (during request processing)
4. **Proper Pause/Resume**: Uses ActionPause correctly to wait for streaming response
5. **Better Logging**: Clear log messages indicate when SSE channel is being established

## Implementation Details

### handleSSEToolsList Example

```go
func handleSSEToolsList(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result, 
                       server *McpProxyServer, allowTools *map[string]struct{}) error {
    // ... extract and store context ...
    
    // Initiate GET request to establish SSE channel in onHttpRequestBody phase
    err = initiateSSEChannelInRequestPhase(ctx, server, authInfo)
    if err != nil {
        log.Errorf("Failed to initiate SSE channel: %v", err)
        return err
    }
    
    // Signal that we need to pause and wait for streaming response
    ctx.SetContext(utils.CtxNeedPause, true)
    return nil
}
```

### initiateSSEChannelInRequestPhase

```go
func initiateSSEChannelInRequestPhase(ctx wrapper.HttpContext, server *McpProxyServer, 
                                      authInfo *ProxyAuthInfo) error {
    // Copy original request headers and clean for SSE GET request
    getHeaders := copyAndCleanHeadersForSSE(ctx)
    
    // Apply authentication to headers and URL
    finalURL := server.GetMcpServerURL()
    finalHeaders := getHeaders
    
    if authInfo != nil && authInfo.SecuritySchemeID != "" {
        modifiedURL, err := applyProxyAuthenticationForSSE(server, authInfo.SecuritySchemeID, 
                                                           authInfo.PassthroughCredential, 
                                                           &finalHeaders, finalURL)
        if err == nil {
            finalURL = modifiedURL
        }
    }
    
    log.Infof("Initiating SSE channel GET request to: %s", finalURL)
    
    // Use RouteCall to send GET request in onHttpRequestBody phase
    return ctx.RouteCall("GET", finalURL, finalHeaders, nil, func(statusCode int, ...) {
        // Response will be processed in onHttpStreamingResponseBody
        if statusCode != 200 {
            log.Errorf("SSE GET request failed with status %d", statusCode)
        }
    })
}
```

### Header Handling

The `copyAndCleanHeadersForSSE()` function properly prepares headers for the SSE GET request:

1. **Copies all original request headers** - Preserves authentication and other metadata
2. **Removes headers inappropriate for GET**:
   - `content-type`, `content-length`, `transfer-encoding` (no body in GET)
   - `accept` (will be set explicitly for SSE)
   - `:path`, `:method`, `:scheme`, `:authority` (pseudo-headers)
3. **Sets `Accept: text/event-stream`** - Required for SSE
4. **Applies security authentication** - Via `applyProxyAuthenticationForSSE()`

This ensures the SSE GET request includes all necessary headers (like authorization, cookies, custom headers) while removing those that are invalid for GET requests.

## Testing

All existing tests continue to pass, confirming backward compatibility:

```bash
cd pkg/mcp/server
go test -run "TestParseSSEMessage|TestExtractEndpointURL|..." -v
```

## Conclusion

This improvement clarifies the SSE proxy architecture by:
1. Establishing SSE channels only for `tools/list` and `tools/call`
2. Initiating SSE channels during the request body phase
3. Using proper pause/resume mechanisms
4. Maintaining clear separation between SSE and standard JSON-RPC handling

The result is a cleaner, more maintainable implementation that follows the design document's intent.

