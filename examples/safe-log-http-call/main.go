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

// This example demonstrates how to use EnableSafeLog to prevent sensitive
// information (headers and body) from being logged in HTTP external calls.
package main

import (
	"net/http"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"safe-log-http-call",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		// Enable safe log mode to prevent logging sensitive information
		// such as request/response headers and bodies in HTTP calls.
		// This is recommended for production environments where API keys,
		// tokens, or other sensitive data may be present in HTTP traffic.
		wrapper.EnableSafeLog[SafeLogHttpCallConfig](),
	)
}

type SafeLogHttpCallConfig struct {
	client      wrapper.HttpClient
	requestPath string
}

func parseConfig(json gjson.Result, config *SafeLogHttpCallConfig) error {
	fqdn := json.Get("fqdn").String()
	port := json.Get("port").Int()
	path := json.Get("path").String()

	// Create FQDN cluster
	cluster := wrapper.FQDNCluster{
		FQDN: fqdn,
		Port: port,
	}
	// Create HTTP client
	config.client = wrapper.NewClusterClient(cluster)
	config.requestPath = path

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config SafeLogHttpCallConfig) types.Action {
	// Make HTTP call to external service
	// Note: With EnableSafeLog enabled, the request headers, body, and
	// response headers, body will NOT be logged, protecting sensitive
	// information like API keys or tokens.
	headers := [][2]string{
		{"Authorization", "Bearer sk-xxx-secret-token"},
		{"Content-Type", "application/json"},
	}

	body := []byte(`{"api_key": "sensitive-api-key", "message": "hello"}`)

	err := config.client.Post(config.requestPath, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		// This log is safe - it only logs the status code, not the sensitive body
		log.Infof("HTTP call completed with status: %d", statusCode)
		proxywasm.AddHttpRequestHeader("X-External-Status", string(rune(statusCode)))
		proxywasm.ResumeHttpRequest()
	}, 5000)

	if err != nil {
		log.Errorf("HTTP call failed: %v", err)
		return types.ActionContinue
	}

	return types.ActionPause
}
