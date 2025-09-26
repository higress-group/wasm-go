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
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpCall(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 1. Create test configuration
		config := []byte(`{
			"fqdn": "httpbin.org",
			"port": 80,
			"path": "/post"
		}`)

		// 2. Create test host
		host, status := test.NewTestHost(config)
		require.Equal(t, types.OnPluginStartStatusOK, status)
		defer host.Reset()

		// 3. Set request headers
		headers := [][2]string{
			{":method", "GET"},
			{":path", "/test"},
			{":authority", "example.com"},
		}

		// 4. Call plugin method
		action := host.CallOnHttpRequestHeaders(headers)
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
