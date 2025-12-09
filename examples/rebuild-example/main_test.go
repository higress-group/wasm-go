package main

import (
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRebuildExample(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Create test configuration (empty config for this example)
		config := []byte(`{}`)

		// Create test host
		host, status := test.NewTestHost(config)
		require.Equal(t, types.OnPluginStartStatusOK, status)
		defer host.Reset()

		// Set request headers
		headers := [][2]string{
			{":method", "GET"},
			{":path", "/test"},
			{":authority", "example.com"},
		}

		// Call plugin method
		action := host.CallOnHttpRequestHeaders(headers)
		require.Equal(t, types.ActionContinue, action)

		// Verify the custom header was added
		requestHeaders := host.GetRequestHeaders()
		assert.True(t, test.HasHeaderWithValue(requestHeaders, "X-Plugin-Processed", "true"),
			"X-Plugin-Processed header should be added")
	})
}

func TestRebuildMultipleRequests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// Create test configuration
		config := []byte(`{}`)

		// Create test host
		host, status := test.NewTestHost(config)
		require.Equal(t, types.OnPluginStartStatusOK, status)
		defer host.Reset()

		headers := [][2]string{
			{":method", "GET"},
			{":path", "/test"},
			{":authority", "example.com"},
		}

		// Send multiple requests to verify plugin continues to work
		for i := 0; i < 10; i++ {
			action := host.CallOnHttpRequestHeaders(headers)
			assert.Equal(t, types.ActionContinue, action)
			host.CompleteHttp()

			// Create new host for next request
			if i < 9 {
				host.Reset()
				host, status = test.NewTestHost(config)
				require.Equal(t, types.OnPluginStartStatusOK, status)
			}
		}
	})
}
