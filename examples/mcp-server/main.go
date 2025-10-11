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
	"mcp-server/tools"

	"github.com/higress-group/wasm-go/pkg/mcp"
)

func main() {}

func init() {
	// 使用 pre-registered Go-based server 而不是 REST MCP Server
	// 这种方式允许我们实现自定义的 Go 工具，而不是依赖于配置驱动的 REST 工具
	mcp.LoadMCPServer(mcp.AddMCPServer("weather-test-server",
		tools.LoadTools(mcp.NewMCPServer())))
	mcp.InitMCPServer()
}
