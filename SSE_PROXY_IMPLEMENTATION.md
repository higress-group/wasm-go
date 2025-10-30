# SSE Protocol Proxy Implementation

This document describes the implementation of SSE (Server-Sent Events) protocol support for MCP (Model Context Protocol) proxy servers.

## Overview

The implementation adds support for SSE transport protocol alongside the existing StreamableHTTP protocol for MCP proxy servers. This allows the proxy to handle MCP servers that use SSE for communication.

## Architecture

### Key Components

1. **Transport Protocol Configuration** (`proxy_server.go`)
   - Added `TransportProtocol` type with two values: `TransportHTTP` and `TransportSSE`
   - Added `transport` field to `McpProxyServer` structure
   - Transport field is required for `mcp-proxy` server type and must be either "http" or "sse"

2. **SSE Protocol Handler** (`sse_proxy.go`)
   - `ParseSSEMessage()`: Parses SSE format data and extracts complete messages
   - `ExtractEndpointURL()`: Extracts endpoint URL from SSE endpoint messages
   - `sendSSEInitialize()`: Sends initialize request for SSE protocol
   - `sendSSENotification()`: Sends notifications/initialized message
   - `sendSSEToolRequest()`: Sends tools/list or tools/call requests
   - `handleSSEStreamingResponse()`: Main handler for streaming SSE responses
   - State machine handlers for different SSE states:
     - `handleWaitingEndpoint()`: Waits for endpoint message
     - `handleWaitingInitResp()`: Waits for initialize response
     - `handleWaitingNotifyResp()`: Waits for notification response
     - `handleWaitingToolResp()`: Waits for tool response

3. **Streaming Response Hooks** (`plugin.go`)
   - `onHttpResponseHeaders()`: Validates SSE response and pauses streaming (only for tools/list and tools/call)
   - `onHttpStreamingResponseBody()`: Processes streaming SSE data chunks (only for tools/list and tools/call)
   - Both hooks check `CtxSSEProxyState` to determine if SSE streaming processing is needed

4. **Request Handlers** (`proxy_tool.go`)
   - `handleSSEToolsList()`: Handles tools/list requests for SSE transport
   - `handleSSEToolsCall()`: Handles tools/call requests for SSE transport
   - `initiateSSEChannelInRequestPhase()`: Modifies current request to GET for establishing SSE channel

## Implementation Flow

### SSE Protocol Request Flow

**Important Note**: Only `tools/list` and `tools/call` requests use the SSE channel. Other JSON-RPC methods (like `initialize`, `notifications/initialized`, etc.) follow the standard JSON-RPC handling flow and return appropriate responses (method not found or ACK) without establishing an SSE channel.

1. **Client Request Arrives**
   - Client sends JSON-RPC request
   - Request is parsed in `onHttpRequestBody()`

2. **Method Routing**
   - If method is `tools/list` or `tools/call` with SSE transport:
     - Routes to SSE-specific handlers (`handleSSEToolsList` or `handleSSEToolsCall`)
   - If method is anything else:
     - Follows standard JSON-RPC handling (method not found or json_rpc_ack)

3. **Establish SSE Channel** (tools/list and tools/call only)
   - In `onHttpRequestBody` phase, **modifies current request to GET** in-place
   - Modifies pseudo-headers: `:method` → GET, `:path`, `:authority` (`:scheme` is managed by Envoy)
   - Removes body-related headers: `content-type`, `content-length`, `transfer-encoding`
   - Sets `Accept: text/event-stream`
   - Applies authentication headers (if configured)
   - Request continues through filter chain (does NOT pause)
   - State: `SSEStateWaitingEndpoint`

4. **Process Endpoint Message** (only for tools/list and tools/call)
   - `onHttpResponseHeaders()` checks `CtxSSEProxyState`, validates content-type and pauses streaming
   - `onHttpStreamingResponseBody()` checks `CtxSSEProxyState`, receives SSE chunks
   - For non-tools/list|call methods: both hooks bypass and continue normally
   - Parses SSE messages from streaming response
   - Extracts endpoint URL from "endpoint" event
   - Combines with base URL if needed
   - State: `SSEStateInitializing`

5. **Initialize Protocol**
   - Sends initialize JSON-RPC request (id: 1) to endpoint URL via `RouteCluster` client
   - State: `SSEStateWaitingInitResp`
   - Waits for initialize response through SSE channel

6. **Send Notification**
   - After receiving initialize response, sends notifications/initialized
   - State: `SSEStateWaitingNotifyResp`

7. **Execute Tool Request**
   - Sends actual tools/list or tools/call request (id: 2) to endpoint URL
   - Note: Both tools/list and tools/call use id: 2, as only one is sent per SSE channel
   - State: `SSEStateWaitingToolResp`
   - Waits for response through SSE channel

8. **Return Response**
   - Extracts JSON-RPC response from SSE message event
   - Validates request ID matches
   - Injects response via `proxywasm.InjectEncodedDataToFilterChain()`
   - Returns response to client

### State Machine

```
SSEStateWaitingEndpoint
    |
    v (endpoint message received)
SSEStateInitializing
    |
    v (initialize sent)
SSEStateWaitingInitResp
    |
    v (initialize response received)
SSEStateWaitingNotifyResp
    |
    v (notification sent)
SSEStateWaitingToolResp
    |
    v (tool response received)
[Response sent to client]
```

## Configuration

### Example Configuration

```yaml
server:
  name: my-mcpserver-proxy
  type: mcp-proxy
  transport: sse  # Required: "http" or "sse"
  mcpServerURL: "http://backend-mcp.example.com/sse"
  timeout: 60000
  defaultDownstreamSecurity:
    id: ClientApiKey
  defaultUpstreamSecurity:
    id: BackendApiKey
  securitySchemes:
  - id: ClientApiKey
    type: apiKey
    in: header
    name: X-Client-API-Key
  - id: BackendApiKey
    type: apiKey
    in: header
    name: X-Backend-API-Key
    defaultCredential: "backend-secret-key"

tools:
- name: get-secure-product
  description: "Get secure product information"
  args:
  - name: product_id
    description: "Product ID"
    type: string
    required: true
  requestTemplate:
    security:
      id: BackendApiKey
      credential: "special-key-for-this-tool"
```

## Security

- Supports both downstream (client-to-gateway) and upstream (gateway-to-backend) authentication
- Can use tool-level security that overrides server-level defaults
- Supports credential passthrough from client to backend
- Authentication is applied consistently across all SSE requests:
  - GET request for establishing SSE channel (uses original request headers with security processing)
  - POST requests for initialize, notification, and tool calls

### Header Handling for SSE Channel

When establishing the SSE channel via GET request:

1. **Copies all original request headers** - Preserves authentication tokens, cookies, custom headers
2. **Removes headers inappropriate for GET requests**:
   - Body-related: `content-type`, `content-length`, `transfer-encoding`
   - `accept` - Will be set explicitly for SSE
   - Pseudo-headers: `:path`, `:method`, `:scheme`, `:authority`
3. **Sets required headers**:
   - `Accept: text/event-stream` - Required for SSE
4. **Applies authentication processing** - Upstream security schemes are applied to modify headers/URL as needed

This ensures the SSE GET request maintains all necessary authentication and metadata while being compliant with GET request standards.

## Buffer Management

- SSE responses are buffered with a maximum size limit of 100MB
- Buffer is cleared after successfully processing each complete response
- Prevents memory exhaustion from malicious or malformed responses

## Error Handling

- **Content-type validation**: 
  - In `onHttpResponseHeaders`: Checks if response content-type is `text/event-stream`
  - In `onHttpStreamingResponseBody`: Validates backend returned `text/event-stream` on first chunk
  - If validation fails: Injects JSON-RPC error via `proxywasm.InjectEncodedDataToFilterChain()`
- **Request ID matching**: Validates response IDs match request IDs
- **State validation**: Ensures responses arrive in correct order
- **Timeout handling**: Respects configured timeout for all requests
- **Backend error propagation**: Properly forwards backend errors to client
- **Streaming response phase**: Uses `proxywasm.InjectEncodedDataToFilterChain()` to inject JSON-RPC responses during streaming response body phase, as `utils.OnMCPResponseError` and `utils.OnMCPResponseSuccess` cannot be used in this phase

## Response Injection in Streaming Phase

In the streaming response body phase (`onHttpStreamingResponseBody`), normal response methods like `utils.OnMCPResponseSuccess` and `utils.OnMCPResponseError` cannot be used. Instead, the implementation uses:

- `injectSSEResponseSuccess()`: Injects successful JSON-RPC responses via `proxywasm.InjectEncodedDataToFilterChain()`
- `injectSSEResponseError()`: Injects error JSON-RPC responses via `proxywasm.InjectEncodedDataToFilterChain()`

Both functions:
1. Retrieve the JSON-RPC ID from context (stored during request parsing)
2. Construct proper JSON-RPC response with correct ID (string or integer)
3. Marshal the response to JSON
4. Inject the response using `proxywasm.InjectEncodedDataToFilterChain(body, true)`

This approach ensures responses are correctly delivered to the client during the streaming response phase.

## Testing

Test coverage includes:
- SSE message parsing (complete and incomplete messages)
- Endpoint URL extraction (full URLs and path-only)
- Transport protocol validation
- Server transport getter/setter
- Multiple message parsing in sequence

All tests are in `sse_proxy_test.go` and can be run with:
```bash
cd pkg/mcp/server
go test -run "TestParseSSEMessage|TestExtractEndpointURL|TestTransportProtocolValidation|TestMcpProxyServerTransport|TestSSEMessageParsing_MultipleMessages" -v
```

## Compatibility

- Compatible with existing StreamableHTTP protocol implementation
- No breaking changes to existing configurations
- SSE and HTTP transports can coexist in the same deployment
- Transport type is explicitly configured per server

## Future Enhancements

Potential areas for future improvement:
1. Connection pooling for SSE channels
2. SSE channel keep-alive and reconnection logic
3. Support for multiple concurrent requests over single SSE channel
4. Metrics and monitoring for SSE connections
5. Configuration validation at startup time

