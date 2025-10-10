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

package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"mcp-server/config"

	"github.com/higress-group/wasm-go/pkg/mcp/server"
	"github.com/higress-group/wasm-go/pkg/mcp/utils"
	"github.com/tidwall/gjson"
)

var _ server.Tool = WeatherTool{}

type WeatherTool struct {
	Location string `json:"location" jsonschema_description:"城市名称" jsonschema:"example=北京"`
}

// Description returns the description field for the MCP tool definition.
// This corresponds to the "description" field in the MCP tool JSON response,
// which provides a human-readable explanation of the tool's purpose and usage.
func (t WeatherTool) Description() string {
	return `获取指定城市的天气信息。支持全球主要城市的实时天气数据查询，包括温度、湿度、风速等信息。`
}

// InputSchema returns the inputSchema field for the MCP tool definition.
// This corresponds to the "inputSchema" field in the MCP tool JSON response,
// which defines the JSON Schema for the tool's input parameters, including
// property types, descriptions, and required fields.
func (t WeatherTool) InputSchema() map[string]any {
	return server.ToInputSchema(&WeatherTool{})
}

// Create instantiates a new WeatherTool tool instance based on the input parameters
// from an MCP tool call.
func (t WeatherTool) Create(params []byte) server.Tool {
	weatherTool := &WeatherTool{}
	json.Unmarshal(params, &weatherTool)
	return weatherTool
}

// Call implements the core logic for handling an MCP tool call. This method is executed
// when the tool is invoked through the MCP framework. It processes the configured parameters,
// makes the actual API request to the service, parses the response,
// and formats the results to be returned to the caller.
func (t WeatherTool) Call(ctx server.HttpContext, s server.Server) error {
	serverConfig := &config.WeatherServerConfig{}
	s.GetConfig(serverConfig)
	if serverConfig.ApiKey == "" {
		return errors.New("Weather API key not configured")
	}

	baseURL := serverConfig.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openweathermap.org/data/2.5"
	}

	// 构建请求URL
	requestURL := fmt.Sprintf("%s/weather?q=%s&appid=%s&units=metric&lang=zh_cn",
		baseURL, url.QueryEscape(t.Location), serverConfig.ApiKey)

	return ctx.RouteCall(http.MethodGet, requestURL,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
			if statusCode != http.StatusOK {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("weather API call failed, status: %d", statusCode))
				return
			}

			jsonObj := gjson.ParseBytes(responseBody)

			// 解析天气数据
			cityName := jsonObj.Get("name").String()
			country := jsonObj.Get("sys.country").String()
			description := jsonObj.Get("weather.0.description").String()
			temp := jsonObj.Get("main.temp").Float()
			feelsLike := jsonObj.Get("main.feels_like").Float()
			humidity := jsonObj.Get("main.humidity").Int()
			windSpeed := jsonObj.Get("wind.speed").Float()

			// 格式化结果
			result := fmt.Sprintf(`# %s, %s 天气信息

## 当前天气
- **天气状况**: %s
- **温度**: %.1f°C
- **体感温度**: %.1f°C
- **湿度**: %d%%
- **风速**: %.1f m/s

数据来源: OpenWeatherMap`, cityName, country, description, temp, feelsLike, humidity, windSpeed)

			utils.SendMCPToolTextResult(ctx, result)
		})
}
