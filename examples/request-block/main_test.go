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

var testConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"blocked_code":      403,
		"blocked_message":   "Access denied",
		"case_sensitive":    false,
		"block_urls":        []string{"blocked", "forbidden"},
		"block_exact_urls":  []string{"/exact-block", "/admin"},
		"block_regexp_urls": []string{`/api/v\d+/blocked`},
		"block_headers":     []string{"blocked-header", "malicious"},
		"block_bodies":      []string{"blocked-content", "spam"},
	})
	return data
}()

func TestParseConfig(t *testing.T) {
	test.RunGoTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()
		config, err := host.GetMatchConfig()
		require.NoError(t, err)
		require.NotNil(t, config)

		blockConfig := config.(*RequestBlockConfig)
		require.Equal(t, uint32(403), blockConfig.blockedCode)
		require.Equal(t, "Access denied", blockConfig.blockedMessage)
		require.False(t, blockConfig.caseSensitive)
		require.Contains(t, blockConfig.blockUrls, "blocked")
		require.Contains(t, blockConfig.blockUrls, "forbidden")
		require.Contains(t, blockConfig.blockExactUrls, "/exact-block")
		require.Contains(t, blockConfig.blockExactUrls, "/admin")
		require.Contains(t, blockConfig.blockHeaders, "blocked-header")
		require.Contains(t, blockConfig.blockHeaders, "malicious")
		require.Contains(t, blockConfig.blockBodies, "blocked-content")
		require.Contains(t, blockConfig.blockBodies, "spam")
	})
}

func TestBlockUrlByKeyword(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// Test blocked URL by keyword
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/api/blocked/endpoint"},
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(403), localResponse.StatusCode)
		require.Equal(t, "Access denied", string(localResponse.Data))
		host.CompleteHttpRequest()
	})
}

func TestBlockUrlByExactMatch(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// Test blocked URL by exact match
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/exact-block"},
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(403), localResponse.StatusCode)
		require.Equal(t, "Access denied", string(localResponse.Data))
		host.CompleteHttpRequest()
	})
}

func TestBlockUrlByRegexp(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// Test blocked URL by regexp
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/api/v1/blocked"},
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(403), localResponse.StatusCode)
		require.Equal(t, "Access denied", string(localResponse.Data))
		host.CompleteHttpRequest()
	})
}

func TestBlockByHeaders(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// Test blocked by headers
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/api/valid"},
			{"blocked-header", "some-value"},
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(403), localResponse.StatusCode)
		require.Equal(t, "Access denied", string(localResponse.Data))
		host.CompleteHttpRequest()
	})
}

func TestBlockByBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Use a config that only has body blocking rules
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// First call headers to set up context - use a path that won't be blocked by URL rules
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/api/safe/endpoint"},
		})
		require.Equal(t, types.ActionContinue, action)

		// Test blocked by body content
		action = host.CallOnHttpRequestBody([]byte("This is blocked-content in the body"))
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(403), localResponse.StatusCode)
		require.Equal(t, "Access denied", string(localResponse.Data))
		host.CompleteHttpRequest()
	})
}

func TestAllowValidRequest(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// Test valid request should be allowed
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/api/valid/endpoint"},
			{"valid-header", "valid-value"},
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.Nil(t, localResponse, "Valid request should not be blocked")
		host.CompleteHttpRequest()
	})
}

func TestCaseInsensitiveBlocking(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host := test.NewTestHost(testConfig)
		defer host.Reset()

		// Test case insensitive blocking (config has case_sensitive: false)
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/API/BLOCKED/ENDPOINT"}, // Uppercase should still be blocked
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(403), localResponse.StatusCode)
		host.CompleteHttpRequest()
	})
}

func TestCustomBlockedCode(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		customConfig := func() json.RawMessage {
			data, _ := json.Marshal(map[string]interface{}{
				"blocked_code":    429,
				"blocked_message": "Too many requests",
				"case_sensitive":  false,
				"block_urls":      []string{"rate-limit"},
			})
			return data
		}()

		host := test.NewTestHost(customConfig)
		defer host.Reset()

		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "test.com"},
			{":path", "/api/rate-limit/test"},
		})
		require.Equal(t, types.ActionContinue, action)

		localResponse := host.GetLocalResponse()
		require.NotNil(t, localResponse)
		require.Equal(t, uint32(429), localResponse.StatusCode)
		require.Equal(t, "Too many requests", string(localResponse.Data))
		host.CompleteHttpRequest()
	})
}
