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
	"github.com/higress-group/wasm-go/pkg/mcp"
	"github.com/higress-group/wasm-go/pkg/mcp/server"
)

func main() {}

func init() {
	// 加载RestMCPServer用于测试REST API转MCP的功能
	restServer := server.NewRestMCPServer("rest-test-server")

	// 添加一个测试工具：获取天气信息
	weatherTool := server.RestTool{
		Name:        "get_weather",
		Description: "获取指定城市的天气信息",
		Args: []server.RestToolArg{
			{
				Name:        "location",
				Description: "城市名称",
				Type:        "string",
				Required:    true,
			},
		},
		RequestTemplate: server.RestToolRequestTemplate{
			URL:    "https://api.openweathermap.org/data/2.5/weather?q={{.location}}&appid=test-api-key&units=metric",
			Method: "GET",
		},
	}

	if err := restServer.AddRestTool(weatherTool); err != nil {
		panic(err)
	}

	// 加载McpProxyServer用于测试MCP代理功能
	proxyServer := server.NewMcpProxyServer("proxy-test-server")

	// 添加一个代理工具配置
	proxyTool := server.McpProxyToolConfig{
		Name:        "get_product",
		Description: "获取产品信息",
		Args: []server.ToolArg{
			{
				Name:        "product_id",
				Description: "产品ID",
				Type:        "string",
				Required:    true,
			},
		},
	}

	if err := proxyServer.AddProxyTool(proxyTool); err != nil {
		panic(err)
	}

	// 注册服务器
	mcp.LoadMCPServer(
		mcp.AddMCPServer("rest-test-server", restServer),
		mcp.AddMCPServer("proxy-test-server", proxyServer),
	)

	mcp.InitMCPServer()
}
