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
	"fmt"
	"net/http"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"complex-http-call",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.WithMaxRequestsPerIoCycle[HttpCallConfig](20),
	)
}

type HttpCallConfig struct {
	client       wrapper.HttpClient
	requestPath  string
	computeLoops int64 // Number of computation loops (linear time complexity)
	timeout      int64 // HTTP call timeout in milliseconds
}

func parseConfig(json gjson.Result, config *HttpCallConfig) error {
	fqdn := json.Get("fqdn").String()
	port := json.Get("port").Int()
	path := json.Get("path").String()
	computeLoops := json.Get("computeLoops").Int()
	timeout := json.Get("timeout").Int()

	// Default to 1000000 loops if not specified (~10ms on typical hardware)
	if computeLoops == 0 {
		computeLoops = 1000000
	}

	// Default timeout 5000ms
	if timeout == 0 {
		timeout = 5000
	}

	// Create FQDN cluster
	cluster := wrapper.FQDNCluster{
		FQDN: fqdn,
		Port: port,
	}
	// Create HTTP client
	config.client = wrapper.NewClusterClient(cluster)
	config.requestPath = path
	config.computeLoops = computeLoops
	config.timeout = timeout

	return nil
}

// busyLoop performs a fixed number of simple computations
// Time complexity is O(n) - linear with the loop count
// This allows precise control over computation time for experiments
//
//go:noinline
func busyLoop(loops int64) int64 {
	var result int64 = 0
	for i := int64(0); i < loops; i++ {
		// Simple arithmetic to prevent compiler optimization
		result += i * 3
		result ^= i
	}
	return result
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config HttpCallConfig) types.Action {
	// ===== SIMULATE COMPLEX PLUGIN LOGIC =====
	// This demonstrates that heavy computation before HTTP call
	// may affect the perceived timeout behavior
	// Using linear time complexity for predictable experiment results
	startTime := time.Now()

	log.Infof("Starting computation: loops=%d", config.computeLoops)
	result := busyLoop(config.computeLoops)
	computeElapsed := time.Since(startTime)

	log.Infof("Computation completed: loops=%d, result=%d, elapsed=%v",
		config.computeLoops, result, computeElapsed)
	// ==========================================

	// Get x-request-id from original request headers
	requestID, _ := proxywasm.GetHttpRequestHeader("x-request-id")

	// Make HTTP call to external service
	headers := [][2]string{
		{"User-Agent", "wasm-plugin"},
		{"Content-Type", "application/json"},
		{"x-request-id", requestID},
	}

	body := []byte(`{"message": "hello from wasm after computation"}`)

	// Record HTTP call start time to measure actual HTTP latency
	httpCallStartTime := time.Now()

	// Use configured HTTP client to make the call
	err := config.client.Post(config.requestPath, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		// Calculate actual HTTP call duration
		httpCallElapsed := time.Since(httpCallStartTime)

		log.Infof("HTTP call response: status=%d, /=%v", statusCode, httpCallElapsed)
		log.Infof("Timing summary: computation_time=%v, http_call_time=%v, total=%v",
			computeElapsed, httpCallElapsed, computeElapsed+httpCallElapsed)

		// Add response to request headers for downstream
		proxywasm.AddHttpRequestHeader("X-External-Response", string(responseBody))
		proxywasm.AddHttpRequestHeader("X-Computation-Time-Ms", fmt.Sprintf("%d", computeElapsed.Milliseconds()))
		proxywasm.AddHttpRequestHeader("X-Http-Call-Time-Ms", fmt.Sprintf("%d", httpCallElapsed.Milliseconds()))

		// Resume the paused request
		proxywasm.ResumeHttpRequest()
	}, uint32(config.timeout))

	if err != nil {
		log.Errorf("HTTP call failed: %v", err)
		return types.ActionContinue
	}

	// Pause request processing until HTTP call completes
	return types.ActionPause
}
