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
	"testing"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplexHttpCall(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 1. Create test configuration with loop-based computation
		// Using 10M loops which takes noticeable time
		config := []byte(`{
			"fqdn": "httpbin.org",
			"port": 80,
			"path": "/post",
			"computeLoops": 10000000,
			"timeout": 5000
		}`)

		// 2. Create test host with foreign function registered
		host, status := test.NewTestHostWithForeignFuncs(config, map[string]func([]byte) []byte{
			"set_global_max_requests_per_io_cycle": func(b []byte) []byte { return b },
		})
		require.Equal(t, types.OnPluginStartStatusOK, status)
		defer host.Reset()

		// 3. Set request headers
		headers := [][2]string{
			{":method", "GET"},
			{":path", "/test"},
			{":authority", "example.com"},
		}

		// 4. Measure time taken by plugin logic
		startTime := time.Now()
		action := host.CallOnHttpRequestHeaders(headers)
		elapsed := time.Since(startTime)

		t.Logf("Plugin request headers processing took: %v", elapsed)
		require.Equal(t, types.ActionPause, action)

		// 5. Verify outbound HTTP call was made
		httpCallouts := host.GetHttpCalloutAttributes()
		require.Len(t, httpCallouts, 1, "Expected exactly one HTTP callout")

		callout := httpCallouts[0]
		assert.Equal(t, "outbound|80||httpbin.org", callout.Upstream, "Upstream name should match")
		assert.True(t, test.HasHeaderWithValue(callout.Headers, ":method", "POST"), "Method should be POST")
		assert.True(t, test.HasHeaderWithValue(callout.Headers, ":path", "/post"), "Path should match config")
		assert.True(t, test.HasHeaderWithValue(callout.Headers, ":authority", "httpbin.org"), "Authority should match upstream")
		assert.True(t, test.HasHeaderWithValue(callout.Headers, "User-Agent", "wasm-plugin"), "User-Agent should be set")
		assert.True(t, test.HasHeaderWithValue(callout.Headers, "Content-Type", "application/json"), "Content-Type should be set")
		assert.Contains(t, string(callout.Body), "hello from wasm", "Request body should contain expected message")

		// 6. Simulate external service response
		responseHeaders := [][2]string{
			{":status", "200"},
			{"Content-Type", "application/json"},
		}
		responseBody := []byte(`{"received": "hello from wasm", "status": "success"}`)
		host.CallOnHttpCall(responseHeaders, responseBody)

		// 7. Complete request
		host.CompleteHttp()

		// 8. Verify final result
		requestHeaders := host.GetRequestHeaders()
		assert.True(t, test.HasHeader(requestHeaders, "X-External-Response"), "External response should be added to request headers")
	})
}

func TestBusyLoopLinearity(t *testing.T) {
	// Test that busyLoop computation time is linear with loop count
	testCases := []int64{1000000, 2000000, 4000000, 8000000}

	var prevElapsed time.Duration
	for _, loops := range testCases {
		startTime := time.Now()
		result := busyLoop(loops)
		elapsed := time.Since(startTime)
		t.Logf("busyLoop(%d) = %d, took %v", loops, result, elapsed)

		// After first case, verify approximate linearity (within 3x tolerance)
		if prevElapsed > 0 {
			ratio := float64(elapsed) / float64(prevElapsed)
			t.Logf("  -> ratio to previous: %.2fx (expected ~2x)", ratio)
			// Allow some variance due to system load
			assert.True(t, ratio > 1.0 && ratio < 4.0,
				"Time should roughly double when loops double, got ratio: %.2f", ratio)
		}
		prevElapsed = elapsed
	}
}

func TestComplexHttpCallWithDifferentLoops(t *testing.T) {
	// Test with different loop counts to verify linear scaling
	testCases := []struct {
		name  string
		loops int64
	}{
		{"low_complexity", 100000},
		{"medium_complexity", 1000000},
		{"high_complexity", 10000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test.RunTest(t, func(t *testing.T) {
				config := []byte(fmt.Sprintf(`{
					"fqdn": "httpbin.org",
					"port": 80,
					"path": "/post",
					"computeLoops": %d,
					"timeout": 5000
				}`, tc.loops))

				host, status := test.NewTestHostWithForeignFuncs(config, map[string]func([]byte) []byte{
					"set_global_max_requests_per_io_cycle": func(b []byte) []byte { return b },
				})
				require.Equal(t, types.OnPluginStartStatusOK, status)
				defer host.Reset()

				headers := [][2]string{
					{":method", "GET"},
					{":path", "/test"},
					{":authority", "example.com"},
				}

				startTime := time.Now()
				action := host.CallOnHttpRequestHeaders(headers)
				elapsed := time.Since(startTime)

				t.Logf("Plugin with loops=%d took: %v", tc.loops, elapsed)
				require.Equal(t, types.ActionPause, action)

				// Verify the HTTP call was still made
				httpCallouts := host.GetHttpCalloutAttributes()
				require.Len(t, httpCallouts, 1, "Expected exactly one HTTP callout")
			})
		})
	}
}
