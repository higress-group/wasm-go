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
		"http-call",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
	)
}

type HttpCallConfig struct {
	client      wrapper.HttpClient
	requestPath string
}

func parseConfig(json gjson.Result, config *HttpCallConfig) error {
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

func onHttpRequestHeaders(ctx wrapper.HttpContext, config HttpCallConfig) types.Action {

	// Make HTTP call to external service
	headers := [][2]string{
		{"User-Agent", "wasm-plugin"},
		{"Content-Type", "application/json"},
	}

	body := []byte(`{"message": "hello from wasm"}`)

	// Use configured HTTP client to make the call
	err := config.client.Post(config.requestPath, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Infof("HTTP call response: status=%d, body=%s", statusCode, string(responseBody))
		// Add response to request headers for downstream
		proxywasm.AddHttpRequestHeader("X-External-Response", string(responseBody))
		// Resume the paused request
		proxywasm.ResumeHttpRequest()
	}, 5000) // 5 second timeout

	if err != nil {
		log.Errorf("HTTP call failed: %v", err)
		return types.ActionContinue
	}

	return types.ActionPause
}
