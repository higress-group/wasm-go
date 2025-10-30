# SSE Streaming Response Fix

## Issue

In the original implementation, the SSE proxy used `utils.OnMCPResponseError()` and `utils.OnMCPResponseSuccess()` to send responses during the streaming response body phase (`onHttpStreamingResponseBody`). However, these functions cannot work correctly in the streaming response phase because they attempt to send HTTP responses in a phase where the response headers have already been sent and the response body is being streamed.

## Solution

Replace all `utils.OnMCPResponseError()` and `utils.OnMCPResponseSuccess()` calls in the streaming response body phase with `proxywasm.InjectEncodedDataToFilterChain()`.

Additionally, validate that the backend returns `text/event-stream` content-type in the first chunk of the streaming response.

### Implementation Details

1. **Added Helper Functions** (`sse_proxy.go`):
   - `injectSSEResponseSuccess(ctx, result)`: Injects successful JSON-RPC responses
   - `injectSSEResponseError(ctx, err, errorCode)`: Injects error JSON-RPC responses

2. **Context Storage**:
   - Added `CtxSSEProxyJsonRpcID` constant to store the JSON-RPC request ID
   - Store the JSON-RPC ID in context during request parsing in `proxy_tool.go`:
     - `handleSSEToolsList()`: Stores ID before initiating SSE channel
     - `handleSSEToolsCall()`: Stores ID before initiating SSE channel

3. **Content-Type Validation**:
   - In `handleSSEStreamingResponse()`, on the first chunk:
     - Validate that backend returned `content-type: text/event-stream`
     - If not, inject JSON-RPC error via `injectSSEResponseError()`
     - Prevents processing non-SSE responses as SSE

4. **Response Injection Logic**:
   Both helper functions:
   - Retrieve JSON-RPC ID from context
   - Handle both string and integer ID types correctly
   - Construct proper JSON-RPC 2.0 response format
   - Marshal response to JSON
   - Inject using `proxywasm.InjectEncodedDataToFilterChain(body, true)`

5. **Replaced Calls** (all in `sse_proxy.go`):
   - `handleSSEStreamingResponse()`: Buffer overflow error
   - `handleWaitingEndpoint()`: Parse error, server not found, endpoint extraction error, initialization error
   - `handleWaitingInitResp()`: Parse error, backend initialize error, notification send error, tool send error
   - `handleWaitingToolResp()`: Request ID not found, parse error, backend tool error, invalid format error, success response

## Changes Made

### Files Modified

1. **`pkg/mcp/server/sse_proxy.go`**:
   - Added `CtxSSEProxyJsonRpcID` constant
   - Added `injectSSEResponseSuccess()` function
   - Added `injectSSEResponseError()` function
   - Replaced 16 occurrences of `utils.OnMCPResponseError()` with `injectSSEResponseError()`
   - Replaced 1 occurrence of `utils.OnMCPResponseSuccess()` with `injectSSEResponseSuccess()`

2. **`pkg/mcp/server/proxy_tool.go`**:
   - Modified `handleSSEToolsList()`: Store JSON-RPC ID in context
   - Modified `handleSSEToolsCall()`: Store JSON-RPC ID in context

3. **`SSE_PROXY_IMPLEMENTATION.md`**:
   - Updated documentation to explain response injection in streaming phase

## Benefits

1. **Correctness**: Responses are now properly injected during streaming response phase
2. **Protocol Compliance**: JSON-RPC responses maintain correct format with proper ID handling
3. **Type Safety**: Handles both string and integer JSON-RPC IDs correctly
4. **Error Handling**: All error cases properly inject error responses to client
5. **No Breaking Changes**: Implementation remains compatible with existing code

## Testing

All existing tests pass:
```bash
cd pkg/mcp/server
go test -run "TestParseSSEMessage|TestExtractEndpointURL|TestTransportProtocolValidation|TestMcpProxyServerTransport|TestSSEMessageParsing_MultipleMessages" -v
```

No linter errors or warnings remain.

## Example Response Format

### Success Response
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [...]
  }
}
```

### Error Response
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "error": {
    "code": -32603,
    "message": "backend tool call failed"
  }
}
```

## Conclusion

This fix ensures that SSE proxy responses are correctly delivered to clients during the streaming response body phase, maintaining proper JSON-RPC protocol compliance and handling both success and error cases appropriately.

