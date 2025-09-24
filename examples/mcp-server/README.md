# MCP Server Example

这个示例展示了如何使用最新提交的MCP代理功能，测试REST MCP服务器和MCP代理服务器的功能。

## 功能说明

### 支持的服务器类型

1. **REST MCP Server** (`type: "rest"`)
   - 将REST API转换为MCP工具
   - 支持HTTP请求模板和参数替换
   - 支持认证方案配置

2. **MCP Proxy Server** (`type: "mcp-proxy"`)
   - 代理到后端MCP服务器
   - 支持MCP协议初始化流程
   - 支持会话管理和工具调用透传
   - 支持认证转发

### 测试内容

这个示例测试了最近git commit中新增的MCP代理功能，包括：

- ✅ 配置解析和验证
- ✅ REST MCP服务器的tools/list功能
- ✅ MCP代理服务器的配置解析
- ⚠️ 工具调用功能（需要mock更多HTTP调用上下文）

## 文件结构

```
examples/mcp-server/
├── go.mod          # Go模块配置，使用相对路径依赖本地wasm-go代码
├── main.go         # 主程序，注册REST和代理服务器
├── main_test.go    # 测试文件，包含各种功能测试
└── README.md       # 本说明文件
```

## 运行测试

```bash
cd examples/mcp-server
go test -v .
```

## 编译

```bash
cd examples/mcp-server
go build -o mcp-server .
```

## 配置示例

### REST MCP服务器配置

```yaml
server:
  name: rest-test-server
  type: rest
  securitySchemes:
  - id: ApiKeyAuth
    type: apiKey
    in: header
    name: X-API-Key
    defaultCredential: test-key
tools:
- name: get_weather
  description: 获取天气信息
  args:
  - name: location
    description: 城市名称
    type: string
    required: true
  requestTemplate:
    url: https://api.openweathermap.org/data/2.5/weather?q={{.location}}
    method: GET
    security:
      id: ApiKeyAuth
```

### MCP代理服务器配置

```yaml
server:
  name: proxy-test-server
  type: mcp-proxy
  mcpServerURL: http://backend-mcp.example.com/mcp
  timeout: 5000
  securitySchemes:
  - id: BackendApiKey
    type: apiKey
    in: header
    name: X-API-Key
    defaultCredential: backend-key
tools:
- name: get_product
  description: 获取产品信息
  args:
  - name: product_id
    description: 产品ID
    type: string
    required: true
  requestTemplate:
    security:
      id: BackendApiKey
```

## 核心功能验证

### 1. MCP协议初始化

对于`mcp-proxy`类型的服务器，系统会自动执行MCP协议初始化：

1. 发送`initialize`请求到后端MCP服务器
2. 接收初始化响应并提取会话ID
3. 发送`notifications/initialized`通知
4. 后续请求携带会话ID

### 2. 工具调用代理

- `tools/list`请求会被透传到后端MCP服务器
- `tools/call`请求会包含正确的参数和认证信息
- 响应会被适当包装并返回给客户端

### 3. 错误处理

- 协议版本不匹配的错误处理
- 后端服务器连接失败的错误处理
- 超时配置的支持

## 技术特点

1. **使用相对路径依赖**：通过`replace`指令使用本地的wasm-go代码，确保测试最新功能
2. **完整的MCP协议支持**：实现了MCP 2025-03-26协议版本
3. **异步处理**：支持HTTP异步调用和响应处理
4. **认证透传**：支持多种认证方案的配置和透传

## 性能基准

包含了基准测试来验证性能表现：

```bash
go test -bench=. -benchmem
```

## 注意事项

1. 这是一个示例程序，主要用于测试新增的MCP代理功能
2. 实际部署时需要根据具体需求调整配置
3. 测试中的mock数据仅用于验证功能流程
4. 生产环境使用时需要配置真实的后端MCP服务器地址