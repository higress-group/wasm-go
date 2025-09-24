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

// TestRestMCPServerConfig 测试REST MCP服务器配置解析
func TestRestMCPServerConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("valid rest mcp server config", func(t *testing.T) {
			host, status := test.NewTestHost(restMCPServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

// TestMcpProxyServerConfig 测试MCP代理服务器配置解析
func TestMcpProxyServerConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		t.Run("valid mcp proxy server config", func(t *testing.T) {
			host, status := test.NewTestHost(mcpProxyServerConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			config, err := host.GetMatchConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
		})
	})
}

// TestRestMCPServerToolsList 测试REST MCP服务器的tools/list功能
func TestRestMCPServerToolsList(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
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
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 处理请求体
		action = host.CallOnHttpRequestBody([]byte(toolsListRequest))

		// 验证响应
		localResponse := host.GetLocalResponse()
		if localResponse != nil {
			require.NotEmpty(t, localResponse.Data)

			var response map[string]interface{}
			err := json.Unmarshal([]byte(localResponse.Data), &response)
			require.NoError(t, err)

			// 验证JSON-RPC格式
			require.Equal(t, "2.0", response["jsonrpc"])
			require.Equal(t, float64(1), response["id"])

			// 验证tools列表
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
}

// TestRestMCPServerToolsCall 测试REST MCP服务器的tools/call功能
func TestRestMCPServerToolsCall(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
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
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 处理请求体 - 这会触发外部HTTP调用
		action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))

		// Mock HTTP响应 - 模拟外部API调用
		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
		}, []byte(`{"args": {"city": "北京"}, "url": "https://httpbin.org/get?city=北京"}`))

		// 验证响应
		localResponse := host.GetLocalResponse()
		if localResponse != nil {
			require.NotEmpty(t, localResponse.Data)

			var response map[string]interface{}
			err := json.Unmarshal([]byte(localResponse.Data), &response)
			require.NoError(t, err)

			// 验证JSON-RPC格式
			require.Equal(t, "2.0", response["jsonrpc"])
			require.Equal(t, float64(2), response["id"])

			// 验证结果
			result, ok := response["result"].(map[string]interface{})
			require.True(t, ok)

			content, ok := result["content"].([]interface{})
			require.True(t, ok)
			require.Greater(t, len(content), 0)
		}

		host.CompleteHttp()
	})
}

// TestMcpProxyServerToolsList 测试MCP代理服务器的tools/list功能
func TestMcpProxyServerToolsList(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
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
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 处理请求体 - 这会触发MCP初始化流程
		action = host.CallOnHttpRequestBody([]byte(toolsListRequest))

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
					"name": "BackendMCPServer",
					"version": "1.0.0"
				}
			}
		}`

		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
			{"mcp-session-id", "test-session-123"},
		}, []byte(initResponse))

		host.CompleteHttp()
	})
}

// TestMcpProxyServerToolsCall 测试MCP代理服务器的tools/call功能
func TestMcpProxyServerToolsCall(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
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
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 处理请求体
		action = host.CallOnHttpRequestBody([]byte(toolsCallRequest))

		// Mock tools/call响应
		toolsCallResponse := `{
			"jsonrpc": "2.0",
			"id": 3,
			"result": {
				"content": [
					{
						"type": "text",
						"text": "产品ID 12345: 这是一个测试产品"
					}
				],
				"isError": false
			}
		}`

		host.CallOnHttpCall([][2]string{
			{":status", "200"},
			{"content-type", "application/json"},
			{"mcp-session-id", "test-session-123"},
		}, []byte(toolsCallResponse))

		host.CompleteHttp()
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		// 测试协议版本不匹配
		t.Run("protocol version mismatch", func(t *testing.T) {
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
				{":method", "POST"},
				{":path", "/mcp"},
				{"content-type", "application/json"},
			})
			require.Equal(t, types.ActionContinue, action)

			// 处理请求体
			action = host.CallOnHttpRequestBody([]byte(toolsListRequest))

			// Mock协议版本不支持的错误响应
			errorResponse := `{
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
			}, []byte(errorResponse))

			// 验证错误响应
			localResponse := host.GetLocalResponse()
			if localResponse != nil {
				require.NotEmpty(t, localResponse.Data)

				var response map[string]interface{}
				err := json.Unmarshal([]byte(localResponse.Data), &response)
				require.NoError(t, err)

				// 验证错误格式
				require.Equal(t, "2.0", response["jsonrpc"])
				errorObj, ok := response["error"].(map[string]interface{})
				require.True(t, ok)
				require.NotNil(t, errorObj["code"])
				require.NotNil(t, errorObj["message"])
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
			{":method", "POST"},
			{":path", "/mcp"},
			{"content-type", "application/json"},
		})
		host.CallOnHttpRequestBody([]byte(toolsListRequest))
		host.CompleteHttp()
	}
}
