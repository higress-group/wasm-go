## 背景

当前这个 proxy_server.go 中的逻辑已经支持针对 stremableHTTP 协议的 MCP Server 的直接代理。
例如配置如下：

```yaml
server:
  name: my-mcpserver-proxy
  type: mcp-proxy
  mcpServerURL: "http://backend-mcp.example.com/mcp"
  timeout: 5000
  defaultDownstreamSecurity: # 客户端到网关的默认认证
    id: ClientApiKey
  defaultUpstreamSecurity: # 网关到后端的默认认证
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
  description: "获取安全产品信息"
  args:
  - name: product_id
    description: "产品ID"
    type: string
    required: true
  requestTemplate:
    security: # 工具级别的网关到后端认证，覆盖默认配置
      id: BackendApiKey
      credential: "special-key-for-this-tool"
```
现在要支持对于 SSE 协议的 MCP Server 直接代理。
在配置上，增加 server.transport 字段，这个字段如果配置为 http 则是按当前 stremableHTTP 协议处理，如果配置为 sse 则是按照 SSE 协议处理，且这个字段在当 server.type 配置 mcp-proxy 时是必填字段。

例如：
```yaml
  server:
	name: my-mcpserver-proxy
	type: mcp-proxy
	# 设置传输协议，streamableHTTP 填 http，SSE 填 sse
	transport: http
	# transport: sse
	mcpServerURL: "http://backend-mcp.example.com/mcp"
	timeout: 60000
    defaultDownstreamSecurity: # 客户端到网关的默认认证
	  id: ClientApiKey
    defaultUpstreamSecurity: # 网关到后端的默认认证
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
	description: "获取安全产品信息"
	args:
	- name: product_id
	  description: "产品ID"
	  type: string
	  required: true
    requestTemplate:
      security: # 工具级别的网关到后端认证，覆盖默认配置
        id: BackendApiKey
        credential: "special-key-for-this-tool"
```

## 协议代理实现原理

当前对于 streamableHTTP 协议的 MCP Server 直接代理的实现，以及如何进一步扩展，支持 SSE 协议的实现原理说明如下：

### 传输协议说明
下面通过 curl 命令的方式来分别介绍请求 streamableHTTP 协议后端和 SSE 协议后端时，MCP 的 tools/list 和 tools/call 两类调用的流程

#### streamableHTTP 协议
**step-1: 发送 initialize 请求**

```bash
curl localhost:8012/mcp -X POST -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{},"elicitation":{}},"clientInfo":{"name":"ExampleClient","title":"Example Client Display Name","version":"1.0.0"}}}'  -H 'content-type: application/json' -H 'Accept: application/json,text/event-stream' -i
```

返回内容：

```text
HTTP/1.1 200 OK
date: Thu, 23 Oct 2025 09:16:32 GMT
server: uvicorn
cache-control: no-cache, no-transform
connection: keep-alive
content-type: text/event-stream
mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2
x-accel-buffering: no
Transfer-Encoding: chunked

event: message
data: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"experimental":{},"prompts":{"listChanged":true},"resources":{"subscribe":false,"listChanged":true},"tools":{"listChanged":true}},"serverInfo":{"name":"Echo Server","version":"1.17.0"}}}

```

**step-2: 发送 notifications/initialized 请求**

注意带上了上一步返回的 mcp-seesion-id 响应头：

```bash
curl localhost:8012/mcp -X POST -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'  -H 'content-type: application/json' -H 'Accept: application/json,text/event-stream' -H 'mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2' -i
```

返回内容：

```text
HTTP/1.1 202 Accepted
date: Thu, 23 Oct 2025 09:17:09 GMT
server: uvicorn
content-type: application/json
mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2
content-length: 0

```

**step-3: 发送 tools/list 或者 tools/call 请求**

1. tools/list

注意始终带上 mcp-seesion-id：

```bash
curl localhost:8012/mcp -X POST -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'  -H 'content-type: application/json' -H 'Accept: application/json,text/event-stream' -H 'mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2' -i
```

返回内容：

```text
HTTP/1.1 200 OK
date: Thu, 23 Oct 2025 09:19:34 GMT
server: uvicorn
cache-control: no-cache, no-transform
connection: keep-alive
content-type: text/event-stream
mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2
x-accel-buffering: no
Transfer-Encoding: chunked

event: message
data: {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","description":"Echo tool that returns the input message","inputSchema":{"properties":{"message":{"type":"string"}},"required":["message"],"type":"object"},"outputSchema":{"properties":{"result":{"type":"string"}},"required":["result"],"type":"object","x-fastmcp-wrap-result":true},"_meta":{"_fastmcp":{"tags":[]}}}]}}

```

2. tools/call

注意始终带上 mcp-seesion-id：

```bash
curl localhost:8012/mcp -X POST -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{ "name":"echo", "arguments":{"message":"123"}}}'  -H 'content-type: application/json' -H 'Accept: application/json,text/event-stream' -H 'mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2' -i
```

返回内容：

```text
HTTP/1.1 200 OK
date: Thu, 23 Oct 2025 09:20:48 GMT
server: uvicorn
cache-control: no-cache, no-transform
connection: keep-alive
content-type: text/event-stream
mcp-session-id: 9ca6e9e43cd8422eb1377267763dd0b2
x-accel-buffering: no
Transfer-Encoding: chunked

event: message
data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"123"}],"structuredContent":{"result":"123"},"isError":false}}

```

#### SSE 协议

**step-1: 发送 GET 请求，建立基于 SSE 的响应通道**

```bash
 curl localhost:8012/sse -i
 ```

返回内容，注意这是一个长连接，后端可能不断返回 ping message 来维持连接

```text
HTTP/1.1 200 OK
date: Thu, 23 Oct 2025 09:22:38 GMT
server: uvicorn
cache-control: no-store
connection: keep-alive
x-accel-buffering: no
content-type: text/event-stream; charset=utf-8
Transfer-Encoding: chunked

event: endpoint
data: /messages/?session_id=b3a6f73b634942a08a11e7bee26b21c0

: ping - 2025-10-23 09:22:53.146891+00:00

: ping - 2025-10-23 09:23:08.148424+00:00

: ping - 2025-10-23 09:23:23.149092+00:00

: ping - 2025-10-23 09:23:38.150461+00:00

: ping - 2025-10-23 09:23:53.151750+00:00

: ping - 2025-10-23 09:24:08.153213+00:00

: ping - 2025-10-23 09:24:23.154680+00:00

: ping - 2025-10-23 09:24:38.156173+00:00

: ping - 2025-10-23 09:24:53.157225+00:00

: ping - 2025-10-23 09:25:08.157946+00:00

: ping - 2025-10-23 09:25:23.159021+00:00

```
需要解析收到的 SSE messages 中 event 为 endpoint 的 message，取出 data 字段，用于后续所有请求的 url，即后续都要请求这个 url： 

localhost:8012/messages/?session_id=b3a6f73b634942a08a11e7bee26b21c0


**step-2: 发送 initialize 请求**


```bash
curl localhost:8012/messages/?session_id=b3a6f73b634942a08a11e7bee26b21c0 -X POST -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{},"elicitation":{}},"clientInfo":{"name":"ExampleClient","title":"Example Client Display Name","version":"1.0.0"}}}'  -H 'content-type: application/json'  -i
```

返回内容：

```text
HTTP/1.1 202 Accepted
date: Thu, 23 Oct 2025 09:29:17 GMT
server: uvicorn
content-length: 8

Accepted
```

可以看到 SSE 的响应通道上新增返回：（这里核心的就是 event type 为 message 的消息，其 ping message 都是定时发送的，跟当前请求无关 ）

```text
: ping - 2025-10-23 09:28:23.173935+00:00

: ping - 2025-10-23 09:29:00.175458+00:00

event: message
data: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"experimental":{},"prompts":{"listChanged":true},"resources":{"subscribe":false,"listChanged":true},"tools":{"listChanged":true}},"serverInfo":{"name":"Echo Server","version":"1.17.0"}}}

```

**step-3: 发送 notifications/initialized 请求**

```bash
curl localhost:8012/messages/?session_id=b3a6f73b634942a08a11e7bee26b21c0 -X POST -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'  -H 'content-type: application/json'  -i
```

返回内容：

```text
HTTP/1.1 202 Accepted
date: Thu, 23 Oct 2025 09:29:34 GMT
server: uvicorn
content-length: 8

Accepted
```

对于这个请求， SSE 的响应通道上不会有返回，但是服务端已经知道初始化完成了，后续可以发送 tools/list 和tools/call 请求了

**step-4: 发送 tools/list 或者 tools/call 请求**

1. tools/list

```bash
curl localhost:8012/messages/?session_id=b3a6f73b634942a08a11e7bee26b21c0 -X POST -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'  -H 'content-type: application/json'
```

返回内容：

```text
HTTP/1.1 202 Accepted
date: Thu, 23 Oct 2025 09:36:02 GMT
server: uvicorn
content-length: 8

Accepted
```

可以看到 SSE 的响应通道上新增返回：（这里核心的就是 event type 为 message 的消息，其 ping message 都是定时发送的，跟当前请求无关 ）

```text
: ping - 2025-10-23 09:35:53.211859+00:00

event: message
data: {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","description":"Echo tool that returns the input message","inputSchema":{"properties":{"message":{"type":"string"}},"required":["message"],"type":"object"},"outputSchema":{"properties":{"result":{"type":"string"}},"required":["result"],"type":"object","x-fastmcp-wrap-result":true},"_meta":{"_fastmcp":{"tags":[]}}}]}}

```

2. tools/call

```bash
curl localhost:8012/messages/?session_id=b3a6f73b634942a08a11e7bee26b21c0 -X POST -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{ "name":"echo", "arguments":{"message":"123"}}}'   -H 'content-type: application/json'  -i
```

返回内容：

```text
HTTP/1.1 202 Accepted
date: Thu, 23 Oct 2025 09:38:07 GMT
server: uvicorn
content-length: 8

Accepted
```

可以看到 SSE 的响应通道上新增返回：（这里核心的就是 event type 为 message 的消息，其 ping message 都是定时发送的，跟当前请求无关 ）

```text
: ping - 2025-10-23 09:37:53.222677+00:00

event: message
data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"123"}],"structuredContent":{"result":"123"},"isError":false}}

```


### streamableHTTP 协议代理实现机制说明

在 plugin.go 的 Initialize 函数中定义了插件 hook 的 http 位点，当前只 hook 了请求头和请求 body 两个处理点：

```go
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
```

onHttpRequestHeaders 的实现中返回了 types.HeaderStopIteration，从而暂停了 header 往后端发送，可以在 onHttpRequestBody 中对 header 进行修改，并跟修改后的 body 一起发给后端。

对于 streamableHTTP 协议来说，onHttpRequestBody 的实现，是将对于 JSON-RPC 协议的 body 解析交给一个工具函数处理：

```go
func onHttpRequestBody(ctx wrapper.HttpContext, config McpServerConfig, body []byte) types.Action {
	return utils.HandleJsonRpcMethod(ctx, body, config.methodHandlers)
}
```

并在 parseConfigCore 这个配置解析阶段执行的函数里，通过下面的方式，hook 了对于 tools/list 和 tools/call 的回调函数：

```go
	proxyHandlers := CreateMcpProxyMethodHandlers(proxyServer, allowTools)
	config.methodHandlers["tools/list"] = proxyHandlers["tools/list"]
	config.methodHandlers["tools/call"] = proxyHandlers["tools/call"]
```

在 utils.HandleJsonRpcMethod 处理 JSON-RPC 请求的过程中会回调到这个 hook 的函数上。并在对应的函数中会通过下面 RouteCluster Client 这个方式，向当前路由的目标服务发起前置 HTTP 调用：

```go
	client := wrapper.NewClusterClient(wrapper.RouteCluster{})
    ...
	...
	client.Post(finalURL, headers, body, wrappedCallback, timeout)
```
并在最终的 tools/list 或者 tools/call 请求的时候，直接通过当前路由进行请求,例如：

```go
ctx.RouteCall("POST", finalURL, headers, requestBody, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
		...
  		...
	})
```

但对于 SSE 协议来说，这样的机制不足以支持

### SSE 协议代理实现方案
接下来都是对 SSE 协议直接代理的处理逻辑，如果是 streamableHTTP 协议直接代理，或者 RESTMCPServer 等其他场景需要保留之前的实现，不能执行下面描述的逻辑：

对于 tools/list 和 tools/call 之外的 JSON-RPC method， SSE 协议应该参考 HandleJsonRpcMethod 中的逻辑，即走到 method not found 或者 json_rpc_ack 相关逻辑上。

对于 tools/list 和 tools/call，对于每次请求都需要需要建立一个 SSE 输出通道，并在得到 endpoint Message 之后再发起后续 HTTP 调用，所以需要在 onHttpRequestBody 阶段**修改当前请求的 headers**，将其转换为 GET 请求以建立基于 SSE 的响应通道。

**注意**：~~原设计建议使用 `ctx.RouteCall` 发送 GET 请求，但这样会导致 callback 与正常的 HTTP filter chain 流式响应处理机制冲突。~~ 正确的做法是修改当前请求的伪头部（pseudo-headers）:
- 将 `:method` 修改为 `GET`
- 设置 `:authority` 为目标域名
- 设置 `:path` 为目标 HTTP path
- **不需要**设置 `:scheme`（由 Envoy 自动管理）
- 移除 `content-type`、`content-length`、`transfer-encoding` 等不适合 GET 请求的头
- 设置 `Accept: text/event-stream`

这样请求会继续通过正常的 filter chain 流转，自然触发 `onHttpResponseHeaders` 和 `onHttpStreamingResponseBody` hook。

并通过下面这个方式挂载 response header, 以及流式解析 response body 的 hook：

```go
    wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
	wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
```

在 onHttpResponseHeaders 中需要通过 wrapper.HasResponseBody 判断是否存在请求 Body，如果存在，则通过 ctx.NeedPauseStreamingResponse() 暂缓流式响应返回给客户端，并返回 HeaderStopIteration，交由 body 阶段进行后续处理。如果没有 Body，则使用 utils.OnMCPResponseError 返回错误。

在 onHttpStreamingResponseBody 中需要将 :status 这个头，修改为 200 状态码。因为即使出现错误，也是通过 JSON-RPC 协议在 body 中体现。

在 onHttpStreamingResponseBody 明确后端返回的 content-type 类型是 text/event-stream，如果不是的话，要通过 proxywasm.InjectEncodedDataToFilterChain(bytes, true) 的方式，返回JSON-RPC协议的错误。

在 onHttpStreamingResponseBody 要移除 content-length 头，并将 content-type 修改为 "application/json; charset=utf-8"。

onHttpStreamingResponseBody 的函数参数如下：

```go
onHttpStreamingResponseBody(ctx wrapper.HttpContext, config McpServerConfig, data []byte, endOfStream bool) []byte {
}
```

这个函数会跟随流式地收到后端响应，多次触发。每次收到的后端响应会放到 data 参数中，返回值是用于替换当次相应片段，可以返回空切片，这样就不会向客户端发送任何消息，相当于阻塞住响应 body。所以注意，只需要在第一次触发时，进行 header 相关的判断和调整。

每次收到的 data 参数不一定是一个完整的 SSE message，需要放到 ctx 里暂存起来，直到解析得到一个完整的 endpoint SSE message，提取出 data 中的 url。

注意提取出的 url 可能是一个带协议头的完整 url，或者是 / 开头的 path，如果是 url 则直接拿来发起后续调用，如果是 path，需要取出配置的 mcpServerURL 中的协议头和域名部分，再拼接上这个 path 用来发起后续调用。

通过 RouteCluster Client 的方式，使用解析得到的 url 发起后续的调用：
1. 发送 initialize
2. 收到 initialize 的成功响应后 （200 或者 202），发送 notifications/initialized
3. 收到 notifications/initialized 的成功响应后 （200 或者 202），发送 tools/list 或者 tools/call 请求

这过程中如果有错误发生，需要通过 proxywasm.InjectEncodedDataToFilterChain(bytes, true) 的方式，返回JSON-RPC协议的错误。

这些调用都是和 onHttpStreamingResponseBody 函数异步执行的。即 onHttpStreamingResponseBody 在收到 SSE 响应通道回复的数据时会不断被触发，在发送 tools/list 或者 tools/call 请求之后，应立即在 ctx 中设置一个标记，用于提示 onHttpStreamingResponseBody 函数处理后续的响应。onHttpStreamingResponseBody 函数在判断这个标记存在后，将开始对收到的 data 片段进行缓存，直到解析得到一个完整的且 JSON-RPC 协议 的id 字段能对应上的 message event（需要校验这个 message 中的 data 的 JSON-RPC 协议中的 id 字段，需要跟发送 tools/list 或者 tools/call 请求的 id 字段对应）。将完整的 JSON-RPC 消息解析出来后，通过 proxywasm.InjectEncodedDataToFilterChain(bytes, true) 的方式将内容返回给客户端。

注意在 onHttpStreamingResponseBody 里缓存 SSE 响应的过程，缓存的最大上限需要控制在 100MB。

另外需要注意的是，在 onHttpStreamingResponseBody 阶段发起的所有请求，都需要携带原始请求 header，包括处理安全认证相关的 header 修改。这个在 onHttpRequestBody 阶段就需要处理好并将所有 header 在 ctx 中保存下来，才能在 onHttpStreamingResponseBody 阶段被使用。
