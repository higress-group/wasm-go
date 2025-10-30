# SSE Header Handling

## Overview

This document describes how request headers are handled when establishing SSE channels for MCP proxy requests.

## Problem

When establishing an SSE channel via GET request, we need to:
1. Preserve authentication headers and other metadata from the original request
2. Remove headers that are inappropriate or invalid for GET requests
3. Add SSE-specific headers
4. Apply security authentication processing

## Solution

### Header Copying and Cleaning

The `copyAndCleanHeadersForSSE()` function implements proper header handling:

```go
func copyAndCleanHeadersForSSE(ctx wrapper.HttpContext) [][2]string {
    headers := make([][2]string, 0)

    // Headers to skip for GET request
    skipHeaders := map[string]bool{
        "content-type":      true,  // GET has no body
        "content-length":    true,  // GET has no body
        "transfer-encoding": true,  // GET has no body
        "accept":            true,  // Will be set explicitly for SSE
        ":path":             true,  // Pseudo-header, will be set by RouteCall
        ":method":           true,  // Pseudo-header, will be set by RouteCall
        ":scheme":           true,  // Pseudo-header, handled by URL
        ":authority":        true,  // Pseudo-header, handled by URL
    }

    // Get all request headers
    headerMap, err := proxywasm.GetHttpRequestHeaders()
    if err != nil {
        log.Warnf("Failed to get request headers: %v", err)
        return [][2]string{{"Accept", "text/event-stream"}}
    }

    // Copy headers, skipping unwanted ones
    for _, header := range headerMap {
        headerName := strings.ToLower(header[0])
        if skipHeaders[headerName] {
            continue
        }
        headers = append(headers, header)
    }

    // Set/override Accept header for SSE
    headers = append(headers, [2]string{"Accept", "text/event-stream"})

    log.Debugf("Prepared %d headers for SSE GET request", len(headers))
    return headers
}
```

### Headers That Are Preserved

The following headers are **preserved** from the original request:

1. **Authentication Headers**:
   - `Authorization`
   - `Cookie`
   - Custom authentication headers (e.g., `X-API-Key`)

2. **Standard HTTP Headers**:
   - `User-Agent`
   - `Referer`
   - `Accept-Language`
   - `Accept-Encoding`
   - `Cache-Control`
   - etc.

3. **Custom Application Headers**:
   - Any custom headers starting with `X-`
   - Application-specific headers

### Headers That Are Removed

The following headers are **removed** as they are inappropriate for GET requests or will be set explicitly:

1. **Body-Related Headers** (GET requests have no body):
   - `Content-Type`
   - `Content-Length`
   - `Transfer-Encoding`

2. **Accept Header** (will be set explicitly):
   - `Accept` - Will be set to `text/event-stream` for SSE

3. **Pseudo-Headers** (managed by HTTP/2 or the routing layer):
   - `:path` - Will be set based on the target URL
   - `:method` - Will be set to GET by RouteCall
   - `:scheme` - Part of the URL
   - `:authority` - Part of the URL

### Headers That Are Added/Modified

1. **Accept Header**: Set to `text/event-stream` to indicate SSE support

### Authentication Processing

After cleaning the headers, authentication is applied:

```go
if authInfo != nil && authInfo.SecuritySchemeID != "" {
    modifiedURL, err := applyProxyAuthenticationForSSE(
        server, 
        authInfo.SecuritySchemeID, 
        authInfo.PassthroughCredential, 
        &finalHeaders,  // Headers may be modified
        finalURL
    )
    if err == nil {
        finalURL = modifiedURL  // URL may be modified (e.g., query params)
    }
}
```

The authentication processing may:
- Add authentication headers (e.g., `Authorization: Bearer token`)
- Modify existing headers
- Add query parameters to the URL (for query-based auth)

## Flow Diagram

```
Original Request Headers
    |
    v
copyAndCleanHeadersForSSE()
    |
    +-- Get all headers from proxywasm.GetHttpRequestHeaders()
    |
    +-- Filter out:
    |   - content-type, content-length, transfer-encoding
    |   - accept (will be set explicitly)
    |   - :path, :method, :scheme, :authority
    |
    +-- Add: Accept: text/event-stream
    |
    v
Cleaned Headers
    |
    v
applyProxyAuthenticationForSSE()
    |
    +-- Apply upstream security scheme
    |   - Add/modify auth headers
    |   - Modify URL if needed
    |
    v
Final Headers + Final URL
    |
    v
ctx.RouteCall("GET", finalURL, finalHeaders, nil, ...)
```

## Examples

### Example 1: Request with Bearer Token

**Original Request Headers**:
```
POST /mcp
Content-Type: application/json
Content-Length: 123
Authorization: Bearer client-token-123
User-Agent: MCP-Client/1.0
X-Request-ID: abc-123
```

**Cleaned Headers for SSE GET**:
```
Authorization: Bearer client-token-123
User-Agent: MCP-Client/1.0
X-Request-ID: abc-123
Accept: text/event-stream
```

### Example 2: Request with Cookie

**Original Request Headers**:
```
POST /mcp
Content-Type: application/json
Content-Length: 456
Cookie: session=xyz789
Accept-Language: en-US
```

**Cleaned Headers for SSE GET**:
```
Cookie: session=xyz789
Accept-Language: en-US
Accept: text/event-stream
```

### Example 3: With Upstream Authentication

**Original Request Headers**:
```
POST /mcp
Content-Type: application/json
X-Client-ID: client-123
```

**After Cleaning and Auth Processing**:
```
X-Client-ID: client-123
X-Backend-API-Key: backend-secret-key   <- Added by auth
Accept: text/event-stream
```

## Benefits

1. **Preserves Authentication**: All authentication tokens and cookies are maintained
2. **Protocol Compliance**: Removes headers invalid for GET requests
3. **Metadata Preservation**: Custom headers and application metadata are kept
4. **Security**: Proper authentication processing is applied
5. **Debugging**: Request IDs and tracing headers are preserved

## Testing

The header handling is tested implicitly through the SSE proxy tests, which verify that:
- SSE channels can be established successfully
- Authentication works correctly
- Requests are properly processed

## Conclusion

The header handling implementation ensures that SSE GET requests maintain all necessary authentication and metadata while being compliant with HTTP GET request standards. This allows the SSE proxy to work correctly with various authentication schemes and custom application headers.

