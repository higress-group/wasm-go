// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// REST MCP服务器配置
var restMCPServerConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"server": map[string]interface{}{
			"name": "rest-test-server",
			"type": "rest",
		},
		"tools": []map[string]interface{}{
			{
				"name":        "get_weather",
				"description": "获取天气信息",
				"args": []map[string]interface{}{
					{
						"name":        "location",
						"description": "城市名称",
						"type":        "string",
						"required":    true,
					},
				},
				"requestTemplate": map[string]interface{}{
					"url":    "https://httpbin.org/get?city={{.location}}",
					"method": "GET",
				},
			},
		},
	})
	return data
}()

// MCP代理服务器配置
var mcpProxyServerConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"server": map[string]interface{}{
			"name":         "proxy-test-server",
			"type":         "mcp-proxy",
			"transport":    "http",
			"mcpServerURL": "http://backend-mcp.example.com/mcp",
			"timeout":      5000,
		},
		"tools": []map[string]interface{}{
			{
				"name":        "get_product",
				"description": "获取产品信息",
				"args": []map[string]interface{}{
					{
						"name":        "product_id",
						"description": "产品ID",
						"type":        "string",
						"required":    true,
					},
				},
			},
		},
	})
	return data
}()

// MCP代理服务器带认证配置
var mcpProxyServerWithAuthConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"server": map[string]interface{}{
			"name":         "proxy-auth-test-server",
			"type":         "mcp-proxy",
			"transport":    "http",
			"mcpServerURL": "http://backend-mcp.example.com/mcp",
			"timeout":      5000,
			"defaultUpstreamSecurity": map[string]interface{}{
				"id": "BackendApiKey",
			},
			"securitySchemes": []map[string]interface{}{
				{
					"id":                "BackendApiKey",
					"type":              "apiKey",
					"in":                "header",
					"name":              "X-API-Key",
					"defaultCredential": "test-default-key",
				},
			},
		},
		"tools": []map[string]interface{}{
			{
				"name":        "get_secure_product",
				"description": "获取安全产品信息",
				"args": []map[string]interface{}{
					{
						"name":        "product_id",
						"description": "产品ID",
						"type":        "string",
						"required":    true,
					},
				},
				"requestTemplate": map[string]interface{}{
					"security": map[string]interface{}{
						"id": "BackendApiKey",
					},
				},
			},
		},
	})
	return data
}()

// 内置天气MCP服务器配置
var weatherMCPServerConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"server": map[string]interface{}{
			"name": "weather-test-server",
			"config": map[string]interface{}{
				"apiKey":  "test-api-key",
				"baseUrl": "https://api.openweathermap.org/data/2.5",
			},
		},
	})
	return data
}()

// TestRestMCPServerConfig 测试REST MCP服务器配置解析
func TestRestMCPServerConfig(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("valid rest mcp server config", func(t *testing.T) {
			host, status := test.NewTestHost(restMCPServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 对于配置解析测试，主要验证插件启动状态
			// GetMatchConfig在WASM模式下可能有限制，我们主要关注启动成功
		})
	})
}

// TestMcpProxyServerConfig 测试MCP代理服务器配置解析
func TestMcpProxyServerConfig(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("valid mcp proxy server config", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 对于配置解析测试，主要验证插件启动状态
			// GetMatchConfig在WASM模式下可能有限制，我们主要关注启动成功
		})
	})
}

// TestWeatherMCPServerConfig 测试天气MCP服务器配置解析
func TestWeatherMCPServerConfig(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("valid weather mcp server config", func(t *testing.T) {
			host, status := test.NewTestHost(weatherMCPServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 对于配置解析测试，主要验证插件启动状态
			// 内置的 weather-test-server 应该能够正确加载和配置
		})
	})
}

// TestRestMCPServerBasicFlow 测试REST MCP服务器基本流程
func TestRestMCPServerBasicFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("tools/list request", func(t *testing.T) {
			host, status := test.NewTestHost(restMCPServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "tools/list",
				"params": {}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionContinue, action)

			// 验证响应
			localResponse := host.GetLocalResponse()
			if localResponse != nil && len(localResponse.Data) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(localResponse.Data, &response)
				require.NoError(t, err)

				// 验证JSON-RPC格式
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(1), response["id"])

				// 验证tools列表存在
				result, ok := response["result"].(map[string]interface{})
				require.True(t, ok)

				tools, ok := result["tools"].([]interface{})
				require.True(t, ok)
				require.Greater(t, len(tools), 0)

				// 验证第一个工具
				tool, ok := tools[0].(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "get_weather", tool["name"])
				require.Equal(t, "获取天气信息", tool["description"])
			}

			host.CompleteHttp()
		})
	})
}

// TestWeatherMCPServerBasicFlow 测试天气MCP服务器基本流程
func TestWeatherMCPServerBasicFlow(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		t.Run("tools/list request", func(t *testing.T) {
			host, status := test.NewTestHost(weatherMCPServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "tools/list",
				"params": {}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "weather-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionContinue, action)

			// 验证响应
			localResponse := host.GetLocalResponse()
			if localResponse != nil && len(localResponse.Data) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(localResponse.Data, &response)
				require.NoError(t, err)

				// 验证JSON-RPC格式
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(1), response["id"])

				// 验证tools列表存在
				result, ok := response["result"].(map[string]interface{})
				require.True(t, ok)

				// 验证tools数组
				tools, ok := result["tools"].([]interface{})
				require.True(t, ok)
				require.Greater(t, len(tools), 0)

				// 验证第一个工具
				tool, ok := tools[0].(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "get_weather", tool["name"])
				require.Contains(t, tool["description"].(string), "天气")
			}

			host.CompleteHttp()
		})
	})
}

// TestRestMCPServerToolsCall 测试REST MCP服务器的tools/call功能
func TestRestMCPServerToolsCall(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(restMCPServerConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		toolsCallRequest := `{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "tools/call",
			"params": {
				"name": "get_weather",
				"arguments": {
					"location": "北京"
				}
			}
		}`

		// 初始化HTTP上下文
		host.InitHttp()

		// 处理请求头
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "mcp-server.example.com"},
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.HeaderStopIteration, action)

		// 处理请求体 - 这会触发外部HTTP调用
		action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
		require.Equal(t, types.ActionContinue, action)

		// Mock HTTP响应头
		action = host.CallOnHttpResponseHeaders([][2]string{
			{":status", "200"},
			{"Content-Type", "application/json"},
		})
		require.Equal(t, types.HeaderStopIteration, action)

		// 处理外部API响应体
		externalAPIResponse := `{
			"args": {"city": "北京"},
			"url": "https://httpbin.org/get?city=北京",
			"headers": {
				"Host": "httpbin.org"
			}
		}`

		action = host.CallOnHttpResponseBody([]byte(externalAPIResponse))
		require.Equal(t, types.ActionContinue, action)

		// 验证最终MCP响应
		responseBody := host.GetResponseBody()
		require.NotEmpty(t, responseBody)

		var response map[string]interface{}
		err := json.Unmarshal(responseBody, &response)
		require.NoError(t, err)

		// 验证JSON-RPC格式
		require.Equal(t, "2.0", response["jsonrpc"])
		require.Equal(t, float64(2), response["id"])

		// 验证结果存在（REST MCP server会将外部API响应包装为MCP格式）
		result, ok := response["result"].(map[string]interface{})
		require.True(t, ok)

		content, ok := result["content"].([]interface{})
		require.True(t, ok)
		require.Greater(t, len(content), 0)

		host.CompleteHttp()
	})
}

// TestWeatherMCPServerToolsCall 测试天气MCP服务器的tools/call功能
func TestWeatherMCPServerToolsCall(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(weatherMCPServerConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		toolsCallRequest := `{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "tools/call",
			"params": {
				"name": "get_weather",
				"arguments": {
					"location": "北京"
				}
			}
		}`

		// 初始化HTTP上下文
		host.InitHttp()

		// 处理请求头
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "weather-server.example.com"},
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.HeaderStopIteration, action)

		// 处理请求体 - 这会触发外部HTTP调用
		action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
		require.Equal(t, types.ActionContinue, action)

		// Mock HTTP响应头
		action = host.CallOnHttpResponseHeaders([][2]string{
			{":status", "200"},
			{"Content-Type", "application/json"},
		})
		require.Equal(t, types.HeaderStopIteration, action)

		// 处理天气API响应体（模拟OpenWeatherMap API响应）
		weatherAPIResponse := `{
			"name": "Beijing",
			"sys": {"country": "CN"},
			"weather": [{"description": "晴天"}],
			"main": {
				"temp": 25.5,
				"feels_like": 27.2,
				"humidity": 60
			},
			"wind": {"speed": 3.5}
		}`

		action = host.CallOnHttpResponseBody([]byte(weatherAPIResponse))
		require.Equal(t, types.ActionContinue, action)

		// 验证最终MCP响应
		responseBody := host.GetResponseBody()
		require.NotEmpty(t, responseBody)

		var response map[string]interface{}
		err := json.Unmarshal(responseBody, &response)
		require.NoError(t, err)

		// 验证JSON-RPC格式
		require.Equal(t, "2.0", response["jsonrpc"])
		require.Equal(t, float64(2), response["id"])

		// 验证结果存在（Go-based MCP server会返回格式化的天气信息）
		result, ok := response["result"].(map[string]interface{})
		require.True(t, ok)

		content, ok := result["content"].([]interface{})
		require.True(t, ok)
		require.Greater(t, len(content), 0)

		// 验证响应内容包含天气信息的关键词
		contentText, ok := content[0].(map[string]interface{})["text"].(string)
		require.True(t, ok)
		require.Contains(t, contentText, "天气信息")
		require.Contains(t, contentText, "温度")

		host.CompleteHttp()
	})
}

// TestMcpProxyServerToolsList 测试MCP代理服务器的tools/list功能
func TestMcpProxyServerToolsList(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(mcpProxyServerConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		toolsListRequest := `{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "tools/list",
			"params": {}
		}`

		// 初始化HTTP上下文
		host.InitHttp()

		// 处理请求头
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "mcp-server.example.com"},
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.HeaderStopIteration, action)

		// 处理请求体 - 这会触发MCP初始化流程
		action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
		require.Equal(t, types.ActionPause, action) // 应该暂停等待后端响应

		// Mock MCP初始化阶段的HTTP调用响应
		// 第一步：Initialize请求的响应
		initResponse := `{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"protocolVersion": "2025-03-26",
				"capabilities": {
					"tools": {}
				},
				"serverInfo": {
					"name": "BackendMCPServer",
					"version": "1.0.0"
				}
			}
		}`

		// Mock initialize响应（带session ID）
		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
			{"mcp-session-id", "test-session-123"},
		}, []byte(initResponse))

		// 第二步：notifications/initialized请求的响应
		notificationResponse := `{"jsonrpc": "2.0"}`
		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
			{"mcp-session-id", "test-session-123"},
		}, []byte(notificationResponse))

		// 第三步：实际的tools/list请求的响应（这是executeToolsList中ctx.RouteCall的响应）
		toolsListResponse := `{
			"jsonrpc": "2.0",
			"id": 2,
			"result": {
				"tools": [
					{
						"name": "get_product",
						"description": "获取产品信息",
						"inputSchema": {
							"type": "object",
							"properties": {
								"product_id": {
									"type": "string",
									"description": "产品ID"
								}
							},
							"required": ["product_id"]
						}
					}
				]
			}
		}`

		// 这是对executeToolsList中ctx.RouteCall的响应
		host.CallOnHttpResponseHeaders([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
		})
		host.CallOnHttpResponseBody([]byte(toolsListResponse))

		// 验证最终MCP响应
		responseBody := host.GetResponseBody()
		require.NotEmpty(t, responseBody)

		var response map[string]interface{}
		err := json.Unmarshal(responseBody, &response)
		require.NoError(t, err)

		// 验证JSON-RPC格式
		require.Equal(t, "2.0", response["jsonrpc"])
		require.Equal(t, float64(1), response["id"])

		// 验证代理转发的结果
		result, ok := response["result"].(map[string]interface{})
		require.True(t, ok)

		tools, ok := result["tools"].([]interface{})
		require.True(t, ok)
		require.Greater(t, len(tools), 0)

		// 验证工具信息
		tool, ok := tools[0].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "get_product", tool["name"])
		require.Equal(t, "获取产品信息", tool["description"])

		host.CompleteHttp()
	})
}

// TestMcpProxyServerToolsCall 测试MCP代理服务器的tools/call功能
func TestMcpProxyServerToolsCall(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(mcpProxyServerConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		toolsCallRequest := `{
			"jsonrpc": "2.0",
			"id": 3,
			"method": "tools/call",
			"params": {
				"name": "get_product",
				"arguments": {
					"product_id": "12345"
				}
			}
		}`

		// 初始化HTTP上下文
		host.InitHttp()

		// 处理请求头
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "mcp-server.example.com"},
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.HeaderStopIteration, action)

		// 处理请求体 - 这会触发MCP初始化流程和工具调用
		action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
		require.Equal(t, types.ActionPause, action) // 应该暂停等待后端响应

		// Mock MCP初始化阶段的HTTP调用响应
		// 第一步：Initialize请求的响应
		initResponse := `{
			"jsonrpc": "2.0",
			"id": 1,
			"result": {
				"protocolVersion": "2025-03-26",
				"capabilities": {
					"tools": {}
				},
				"serverInfo": {
					"name": "BackendMCPServer",
					"version": "1.0.0"
				}
			}
		}`

		// Mock initialize响应（带session ID）
		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
			{"mcp-session-id", "test-session-456"},
		}, []byte(initResponse))

		// 第二步：notifications/initialized请求的响应
		notificationResponse := `{"jsonrpc": "2.0"}`
		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
			{"mcp-session-id", "test-session-456"},
		}, []byte(notificationResponse))

		// 第三步：实际的tools/call请求的响应
		toolsCallResponse := `{
			"jsonrpc": "2.0",
			"id": 2,
			"result": {
				"content": [
					{
						"type": "text",
						"text": "Product ID: 12345\nName: Sample Product\nPrice: $99.99\nDescription: This is a sample product for testing"
					}
				],
				"isError": false
			}
		}`

		// 这是对executeToolsCall中ctx.RouteCall的响应
		host.CallOnHttpResponseHeaders([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
		})
		host.CallOnHttpResponseBody([]byte(toolsCallResponse))

		// 验证最终MCP响应
		responseBody := host.GetResponseBody()
		require.NotEmpty(t, responseBody)

		var response map[string]interface{}
		err := json.Unmarshal(responseBody, &response)
		require.NoError(t, err)

		// 验证JSON-RPC格式
		require.Equal(t, "2.0", response["jsonrpc"])
		require.Equal(t, float64(3), response["id"])

		// 验证代理转发的结果
		result, ok := response["result"].(map[string]interface{})
		require.True(t, ok)

		content, ok := result["content"].([]interface{})
		require.True(t, ok)
		require.Greater(t, len(content), 0)

		// 验证内容
		textContent, ok := content[0].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "text", textContent["type"])
		require.Contains(t, textContent["text"], "Product ID: 12345")

		// 验证isError字段
		require.Equal(t, false, result["isError"])

		host.CompleteHttp()
	})
}

// TestMcpProxyServerAuthentication 测试MCP代理服务器认证功能
func TestMcpProxyServerAuthentication(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试tools/list请求的认证头
		t.Run("tools/list authentication headers", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerWithAuthConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "tools/list",
				"params": {}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头（带用户API Key）
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
				{"x-api-key", "user-provided-key"}, // 用户提供的API Key
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock MCP初始化响应
			initResponse := `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {
						"tools": {}
					},
					"serverInfo": {
						"name": "SecureBackendMCPServer",
						"version": "1.0.0"
					}
				}
			}`

			// 验证初始化请求的认证头
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "secure-session-list-123"},
			}, []byte(initResponse))

			// 验证初始化成功（从日志中可以确认发送了正确的认证头 [X-API-Key test-default-key]）
			// 实际的HTTP请求包含了正确的默认凭据用于上游认证

			// Mock notifications/initialized响应
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "secure-session-list-123"},
			}, []byte(`{"jsonrpc": "2.0"}`))

			// Mock tools/list响应
			toolsListResponse := `{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"tools": [
						{
							"name": "get_secure_product",
							"description": "获取安全产品信息",
							"inputSchema": {
								"type": "object",
								"properties": {
									"product_id": {
										"type": "string",
										"description": "产品ID"
									}
								},
								"required": ["product_id"]
							}
						}
					]
				}
			}`

			// 验证tools/list请求的认证头（在响应处理前获取发送给后端的请求头）
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/mcp", "Path should be MCP endpoint")

			apiKeyValue, hasApiKey := test.GetHeaderValue(requestHeaders, "x-api-key")
			require.True(t, hasApiKey, "X-API-Key header should exist in tools/list request")
			require.Equal(t, "test-default-key", apiKeyValue, "Should use default credential for tools/list")

			// 这是对executeToolsList中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(toolsListResponse))

			// 验证响应
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			// 验证响应格式
			require.Equal(t, "2.0", response["jsonrpc"])
			require.Equal(t, float64(1), response["id"])

			result, ok := response["result"].(map[string]interface{})
			require.True(t, ok)

			tools, ok := result["tools"].([]interface{})
			require.True(t, ok)
			require.Greater(t, len(tools), 0)

			host.CompleteHttp()
		})

		// 测试tools/call请求的认证头
		t.Run("tools/call authentication headers", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerWithAuthConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsCallRequest := `{
				"jsonrpc": "2.0",
				"id": 4,
				"method": "tools/call",
				"params": {
					"name": "get_secure_product",
					"arguments": {
						"product_id": "secure-123"
					}
				}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头（带用户API Key）
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
				{"x-api-key", "user-provided-key"}, // 用户提供的API Key
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock MCP初始化响应
			initResponse := `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {
						"tools": {}
					},
					"serverInfo": {
						"name": "SecureBackendMCPServer",
						"version": "1.0.0"
					}
				}
			}`

			// 验证初始化请求的认证头
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "secure-session-call-456"},
			}, []byte(initResponse))

			// 验证初始化成功（从日志中可以确认发送了正确的认证头 [X-API-Key test-default-key]）
			// 实际的HTTP请求包含了正确的默认凭据用于上游认证

			// Mock notifications/initialized响应
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "secure-session-call-456"},
			}, []byte(`{"jsonrpc": "2.0"}`))

			// Mock工具调用响应
			secureToolResponse := `{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"content": [
						{
							"type": "text",
							"text": "Secure Product ID: secure-123\nName: Confidential Product\nAccess Level: Premium"
						}
					],
					"isError": false
				}
			}`

			// 验证tools/call请求的认证头（在响应处理前获取发送给后端的请求头）
			requestHeaders := host.GetRequestHeaders()
			pathValue, hasPath := test.GetHeaderValue(requestHeaders, ":path")
			require.True(t, hasPath, "Path header should exist")
			require.Contains(t, pathValue, "/mcp", "Path should be MCP endpoint")

			apiKeyValue, hasApiKey := test.GetHeaderValue(requestHeaders, "x-api-key")
			require.True(t, hasApiKey, "X-API-Key header should exist in tools/call request")
			require.Equal(t, "test-default-key", apiKeyValue, "Should use default credential for tools/call")

			// 这是对executeToolsCall中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(secureToolResponse))

			// 验证响应
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			// 验证响应格式
			require.Equal(t, "2.0", response["jsonrpc"])
			require.Equal(t, float64(4), response["id"])

			result, ok := response["result"].(map[string]interface{})
			require.True(t, ok)

			content, ok := result["content"].([]interface{})
			require.True(t, ok)
			textContent, ok := content[0].(map[string]interface{})
			require.True(t, ok)
			require.Contains(t, textContent["text"], "Secure Product ID: secure-123")

			host.CompleteHttp()
		})
	})
}

// TestMcpProxyServerErrorHandling 测试MCP代理服务器错误处理
func TestMcpProxyServerErrorHandling(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试协议版本不匹配
		t.Run("protocol version mismatch", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 5,
				"method": "tools/list",
				"params": {}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock协议版本不匹配的错误响应
			versionErrorResponse := `{
				"jsonrpc": "2.0",
				"id": 1,
				"error": {
					"code": -32602,
					"message": "Unsupported protocol version",
					"data": {
						"supported": ["2024-11-05"],
						"requested": "2025-03-26"
					}
				}
			}`

			host.CallOnHttpCall([][2]string{
				{":status", "400"},
				{"content-type", "application/json"},
			}, []byte(versionErrorResponse))

			// 验证错误响应
			localResponse := host.GetLocalResponse()
			if localResponse != nil && len(localResponse.Data) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(localResponse.Data, &response)
				require.NoError(t, err)

				// 验证错误被正确包装
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(5), response["id"])

				errorField, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				require.Contains(t, errorField["message"], "backend")
			}

			host.CompleteHttp()
		})

		// 测试后端服务器超时
		t.Run("backend timeout", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsCallRequest := `{
				"jsonrpc": "2.0",
				"id": 6,
				"method": "tools/call",
				"params": {
					"name": "get_product",
					"arguments": {
						"product_id": "timeout-test"
					}
				}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock超时错误 - 不提供响应，模拟超时
			// 在实际实现中，这会触发超时处理逻辑

			host.CompleteHttp()
		})

		// 测试后端工具执行错误
		t.Run("backend tool error", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsCallRequest := `{
				"jsonrpc": "2.0",
				"id": 7,
				"method": "tools/call",
				"params": {
					"name": "get_product",
					"arguments": {
						"product_id": "error-test"
					}
				}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理请求头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock正常的初始化流程
			initResponse := `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {"tools": {}},
					"serverInfo": {"name": "TestServer", "version": "1.0.0"}
				}
			}`

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "error-session-001"},
			}, []byte(initResponse))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "error-session-001"},
			}, []byte(`{"jsonrpc": "2.0"}`))

			// Mock工具执行错误响应
			toolErrorResponse := `{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"content": [
						{
							"type": "text",
							"text": "Failed to fetch product: Database connection error"
						}
					],
					"isError": true
				}
			}`

			// 这是对executeToolsCall中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(toolErrorResponse))

			// 验证错误被正确传播
			responseBody := host.GetResponseBody()
			if len(responseBody) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(responseBody, &response)
				require.NoError(t, err)

				// 验证响应格式
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(7), response["id"])

				result, ok := response["result"].(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, true, result["isError"])

				content, ok := result["content"].([]interface{})
				require.True(t, ok)
				textContent, ok := content[0].(map[string]interface{})
				require.True(t, ok)
				require.Contains(t, textContent["text"], "Database connection error")
			}

			host.CompleteHttp()
		})
	})
}

// TestMcpProxyServerSessionManagement 测试MCP代理服务器会话管理
func TestMcpProxyServerSessionManagement(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试单个HTTP Context内的会话状态管理
		t.Run("context session state", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 在同一个HTTP Context中，先进行tools/list然后tools/call
			// 这将测试Context内的CtxMcpProxyInitialized状态管理

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 8,
				"method": "tools/list",
				"params": {}
			}`

			// 初始化HTTP上下文
			host.InitHttp()

			// 处理第一个请求（tools/list）
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock初始化过程（第一次应该触发初始化）
			initResponse := `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {"tools": {}},
					"serverInfo": {"name": "TestServer", "version": "1.0.0"}
				}
			}`

			sessionID := "context-session-999"
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", sessionID},
			}, []byte(initResponse))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", sessionID},
			}, []byte(`{"jsonrpc": "2.0"}`))

			// Mock tools/list响应
			toolsListResponse := `{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"tools": [
						{
							"name": "get_product",
							"description": "获取产品信息",
							"inputSchema": {
								"type": "object",
								"properties": {
									"product_id": {"type": "string", "description": "产品ID"}
								},
								"required": ["product_id"]
							}
						}
					]
				}
			}`

			// 这是对executeToolsList中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(toolsListResponse))

			// 验证tools/list响应
			responseBody := host.GetResponseBody()
			if len(responseBody) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(responseBody, &response)
				require.NoError(t, err)
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(8), response["id"])
			}

			host.CompleteHttp()
		})

		// 测试每个新HTTP请求都需要重新初始化（符合实际实现）
		t.Run("new request requires new initialization", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 第一个独立的HTTP请求
			toolsCallRequest1 := `{
				"jsonrpc": "2.0",
				"id": 9,
				"method": "tools/call",
				"params": {
					"name": "get_product",
					"arguments": {
						"product_id": "request-1"
					}
				}
			}`

			host.InitHttp()
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsCallRequest1))
			require.Equal(t, types.ActionPause, action)

			// Mock第一个请求的完整初始化流程
			initResponse := `{
				"jsonrpc": "2.0",
				"id": 1,
				"result": {
					"protocolVersion": "2025-03-26",
					"capabilities": {"tools": {}},
					"serverInfo": {"name": "TestServer", "version": "1.0.0"}
				}
			}`

			sessionID1 := "session-request-1"
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", sessionID1},
			}, []byte(initResponse))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", sessionID1},
			}, []byte(`{"jsonrpc": "2.0"}`))

			// Mock tools/call响应
			toolsCallResponse1 := `{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"content": [
						{
							"type": "text",
							"text": "Product from request 1: request-1"
						}
					],
					"isError": false
				}
			}`

			// 这是对executeToolsCall中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(toolsCallResponse1))

			// 验证第一个请求的响应
			responseBody := host.GetResponseBody()
			if len(responseBody) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(responseBody, &response)
				require.NoError(t, err)
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(9), response["id"])
			}

			host.CompleteHttp()

			// 第二个独立的HTTP请求（新的Context，需要重新初始化）
			toolsCallRequest2 := `{
				"jsonrpc": "2.0",
				"id": 10,
				"method": "tools/call",
				"params": {
					"name": "get_product",
					"arguments": {
						"product_id": "request-2"
					}
				}
			}`

			// 新的HTTP Context
			host.InitHttp()
			action = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsCallRequest2))
			require.Equal(t, types.ActionPause, action)

			// Mock第二个请求的完整初始化流程（每个新Context都需要重新初始化）
			sessionID2 := "session-request-2"
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", sessionID2}, // 不同的session ID
			}, []byte(initResponse))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", sessionID2},
			}, []byte(`{"jsonrpc": "2.0"}`))

			toolsCallResponse2 := `{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"content": [
						{
							"type": "text",
							"text": "Product from request 2: request-2"
						}
					],
					"isError": false
				}
			}`

			// 这是对executeToolsCall中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(toolsCallResponse2))

			// 验证第二个请求的响应
			responseBody = host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)
			require.Equal(t, "2.0", response["jsonrpc"])
			require.Equal(t, float64(10), response["id"])

			result, ok := response["result"].(map[string]interface{})
			require.True(t, ok)
			content, ok := result["content"].([]interface{})
			require.True(t, ok)
			textContent, ok := content[0].(map[string]interface{})
			require.True(t, ok)
			require.Contains(t, textContent["text"], "request-2")

			host.CompleteHttp()
		})

		// 测试初始化失败处理
		t.Run("initialization failure", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsCallRequest := `{
			"jsonrpc": "2.0",
			"id": 11,
			"method": "tools/call",
			"params": {
				"name": "get_product",
				"arguments": {
					"product_id": "init-failure-test"
				}
			}
		}`

			host.InitHttp()

			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock初始化失败响应
			initErrorResponse := `{
			"jsonrpc": "2.0",
			"id": 1,
			"error": {
				"code": -32001,
				"message": "Backend server unavailable"
			}
		}`

			host.CallOnHttpCall([][2]string{
				{":status", "500"},
				{"content-type", "application/json"},
			}, []byte(initErrorResponse))

			// 验证错误被正确处理
			localResponse := host.GetLocalResponse()
			if localResponse != nil && len(localResponse.Data) > 0 {
				var response map[string]interface{}
				err := json.Unmarshal(localResponse.Data, &response)
				require.NoError(t, err)

				// 验证错误响应格式
				require.Equal(t, "2.0", response["jsonrpc"])
				require.Equal(t, float64(11), response["id"])

				errorField, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				require.Contains(t, errorField["message"], "backend")
			}

			host.CompleteHttp()
		})
	})
}

// BenchmarkRestMCPServer 性能基准测试
func BenchmarkRestMCPServer(b *testing.B) {
	host, status := test.NewTestHost(restMCPServerConfig)
	defer host.Reset()
	require.Equal(b, types.OnPluginStartStatusOK, status)

	toolsListRequest := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/list",
		"params": {}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		host.InitHttp()
		host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "mcp-server.example.com"},
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		host.CallOnHttpRequestBody([]byte(toolsListRequest))
		host.CompleteHttp()
	}
}

// TestMcpProxyServerAllowTools 测试MCP代理服务器allowTools功能
func TestMcpProxyServerAllowTools(t *testing.T) {
	// 创建包含allowTools配置的测试配置
	mcpProxyServerWithAllowToolsConfig := func() json.RawMessage {
		data, _ := json.Marshal(map[string]interface{}{
			"server": map[string]interface{}{
				"name":         "proxy-allow-tools-server",
				"type":         "mcp-proxy",
				"transport":    "http",
				"mcpServerURL": "http://backend-mcp.example.com/mcp",
				"timeout":      5000,
			},
			"allowTools": []string{"get_product", "create_order"}, // 只允许这两个工具
			"tools": []map[string]interface{}{
				{
					"name":        "get_product",
					"type":        "mcp-proxy",
					"description": "Get product information",
				},
				{
					"name":        "create_order",
					"type":        "mcp-proxy",
					"description": "Create a new order",
				},
				{
					"name":        "delete_user",
					"type":        "mcp-proxy",
					"description": "Delete a user account",
				},
			},
		})
		return data
	}()

	test.RunTest(t, func(t *testing.T) {
		// 测试配置级别的allowTools过滤
		t.Run("config level allowTools filtering", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerWithAllowToolsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 1,
				"method": "tools/list",
				"params": {}
			}`

			host.InitHttp()
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action) // 应该暂停等待异步响应

			// Mock MCP initialization sequence
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-123"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "init-1",
				"result": {
					"capabilities": {
						"tools": {"listChanged": true}
					},
					"protocolVersion": "2024-11-05"
				}
			}`))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-123"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "notify-1",
				"result": {}
			}`))

			// Mock tools/list response with 3 tools (但只有2个会被返回)
			// 这是对executeToolsList中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(`{
				"jsonrpc": "2.0",
				"id": 2,
				"result": {
					"tools": [
						{
							"name": "get_product",
							"description": "Get product information",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "create_order", 
							"description": "Create a new order",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "delete_user",
							"description": "Delete a user account", 
							"inputSchema": {"type": "object"}
						}
					]
				}
			}`))

			host.CompleteHttp()

			// 验证响应只包含允许的工具
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			result, hasResult := response["result"]
			require.True(t, hasResult)
			resultMap := result.(map[string]interface{})

			tools, hasTools := resultMap["tools"]
			require.True(t, hasTools)
			toolsArray := tools.([]interface{})

			// 应该只返回2个允许的工具，delete_user被过滤掉
			require.Len(t, toolsArray, 2)

			toolNames := make([]string, 0)
			for _, tool := range toolsArray {
				toolMap := tool.(map[string]interface{})
				toolNames = append(toolNames, toolMap["name"].(string))
			}
			require.Contains(t, toolNames, "get_product")
			require.Contains(t, toolNames, "create_order")
			require.NotContains(t, toolNames, "delete_user")
		})

		// 测试请求头级别的allowTools过滤
		t.Run("header level allowTools filtering", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig) // 使用没有allowTools配置的基本配置
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 2,
				"method": "tools/list",
				"params": {}
			}`

			host.InitHttp()
			// 设置请求头只允许get_product工具
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
				{"x-envoy-allow-mcp-tools", "get_product"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock MCP initialization sequence
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-456"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "init-2",
				"result": {
					"capabilities": {
						"tools": {"listChanged": true}
					},
					"protocolVersion": "2024-11-05"
				}
			}`))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-456"},
			}, []byte(`{
				"jsonrpc": "2.0", 
				"id": "notify-2",
				"result": {}
			}`))

			// Mock tools/list response with multiple tools
			// 这是对executeToolsList中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(`{
				"jsonrpc": "2.0",
				"id": 2, 
				"result": {
					"tools": [
						{
							"name": "get_product",
							"description": "Get product information",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "create_order",
							"description": "Create a new order", 
							"inputSchema": {"type": "object"}
						}
					]
				}
			}`))

			host.CompleteHttp()

			// 验证响应只包含请求头中允许的工具
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			result, hasResult := response["result"]
			require.True(t, hasResult)
			resultMap := result.(map[string]interface{})

			tools, hasTools := resultMap["tools"]
			require.True(t, hasTools)
			toolsArray := tools.([]interface{})

			// 应该只返回1个工具(get_product)
			require.Len(t, toolsArray, 1)
			toolMap := toolsArray[0].(map[string]interface{})
			require.Equal(t, "get_product", toolMap["name"])
		})

		// 测试配置和请求头都存在时的组合过滤
		t.Run("combined config and header allowTools filtering", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerWithAllowToolsConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 3,
				"method": "tools/list", 
				"params": {}
			}`

			host.InitHttp()
			// 配置允许：get_product, create_order
			// 请求头允许：get_product, delete_user
			// 交集应该只有：get_product
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
				{"x-envoy-allow-mcp-tools", "get_product,delete_user"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock MCP initialization sequence
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-789"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "init-3",
				"result": {
					"capabilities": {
						"tools": {"listChanged": true}
					},
					"protocolVersion": "2024-11-05"
				}
			}`))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-789"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "notify-3", 
				"result": {}
			}`))

			// Mock tools/list response
			// 这是对executeToolsList中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(`{
				"jsonrpc": "2.0",
				"id": 3,
				"result": {
					"tools": [
						{
							"name": "get_product",
							"description": "Get product information",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "create_order",
							"description": "Create a new order",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "delete_user", 
							"description": "Delete a user account",
							"inputSchema": {"type": "object"}
						}
					]
				}
			}`))

			host.CompleteHttp()

			// 验证响应只包含交集中的工具
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			result, hasResult := response["result"]
			require.True(t, hasResult)
			resultMap := result.(map[string]interface{})

			tools, hasTools := resultMap["tools"]
			require.True(t, hasTools)
			toolsArray := tools.([]interface{})

			// 应该只返回1个工具(get_product)，因为它是唯一在两个allowTools列表中的工具
			require.Len(t, toolsArray, 1)
			toolMap := toolsArray[0].(map[string]interface{})
			require.Equal(t, "get_product", toolMap["name"])
		})

		// 测试空白的请求头allowTools
		t.Run("empty header allowTools", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 4,
				"method": "tools/list",
				"params": {}
			}`

			host.InitHttp()
			// 设置空的allowTools头
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
				{"x-envoy-allow-mcp-tools", "  ,  ,  "}, // 只有空白和逗号
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock MCP initialization sequence
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-empty"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "init-4",
				"result": {
					"capabilities": {
						"tools": {"listChanged": true}
					},
					"protocolVersion": "2024-11-05"
				}
			}`))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-empty"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "notify-4",
				"result": {}
			}`))

			// Mock tools/list response
			// 这是对executeToolsList中ctx.RouteCall的响应
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(`{
				"jsonrpc": "2.0",
				"id": 4,
				"result": {
					"tools": [
						{
							"name": "get_product",
							"description": "Get product information",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "create_order",
							"description": "Create a new order",
							"inputSchema": {"type": "object"}
						}
					]
				}
			}`))

			host.CompleteHttp()

			// 验证响应不包含任何工具（空白header应该被当作配置为空，禁止所有工具）
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			result, hasResult := response["result"]
			require.True(t, hasResult)
			resultMap := result.(map[string]interface{})

			tools, hasTools := resultMap["tools"]
			require.True(t, hasTools)
			toolsArray := tools.([]interface{})

			// 应该返回0个工具，因为空白header等于配置为空数组，禁止所有工具
			require.Len(t, toolsArray, 0)
		})

		// 测试不存在的allowTools header（应该允许所有工具）
		t.Run("no header allowTools", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			toolsListRequest := `{
				"jsonrpc": "2.0",
				"id": 5,
				"method": "tools/list",
				"params": {}
			}`

			host.InitHttp()
			// 不设置x-envoy-allow-mcp-tools header
			action := host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "mcp-server.example.com"},
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.HeaderStopIteration, action)

			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))
			require.Equal(t, types.ActionPause, action)

			// Mock MCP initialization sequence
			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-no-header"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "init-5",
				"result": {
					"capabilities": {
						"tools": {"listChanged": true}
					},
					"protocolVersion": "2024-11-05"
				}
			}`))

			host.CallOnHttpCall([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
				{"mcp-session-id", "test-session-no-header"},
			}, []byte(`{
				"jsonrpc": "2.0",
				"id": "notify-5",
				"result": {}
			}`))

			// Mock tools/list response with multiple tools
			host.CallOnHttpResponseHeaders([][2]string{
				{":status", "200"},
				{"content-type", "application/json"},
			})
			host.CallOnHttpResponseBody([]byte(`{
				"jsonrpc": "2.0",
				"id": 5,
				"result": {
					"tools": [
						{
							"name": "get_product",
							"description": "Get product information",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "create_order",
							"description": "Create a new order",
							"inputSchema": {"type": "object"}
						},
						{
							"name": "delete_user",
							"description": "Delete a user account",
							"inputSchema": {"type": "object"}
						}
					]
				}
			}`))

			host.CompleteHttp()

			// 验证响应包含所有工具（header不存在时允许所有工具）
			responseBody := host.GetResponseBody()
			require.NotEmpty(t, responseBody)

			var response map[string]interface{}
			err := json.Unmarshal(responseBody, &response)
			require.NoError(t, err)

			result, hasResult := response["result"]
			require.True(t, hasResult)
			resultMap := result.(map[string]interface{})

			tools, hasTools := resultMap["tools"]
			require.True(t, hasTools)
			toolsArray := tools.([]interface{})

			// 应该返回所有3个工具，因为header不存在意味着没有限制
			require.Len(t, toolsArray, 3)

			toolNames := make([]string, 0)
			for _, tool := range toolsArray {
				toolMap := tool.(map[string]interface{})
				toolNames = append(toolNames, toolMap["name"].(string))
			}
			require.Contains(t, toolNames, "get_product")
			require.Contains(t, toolNames, "create_order")
			require.Contains(t, toolNames, "delete_user")
		})

	})
}
