// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0
package wrapper_test

import (
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"testing"
)

type trailerConfig struct{}

func TestBufferedBodyFinalizedOnce(t *testing.T) {
	for _, response := range []bool{false, true} {
		for _, body := range []string{"", "firstsecond"} {
			t.Run("lifecycle", func(t *testing.T) {
				calls := 0
				got := ""
				handler := func(_ wrapper.HttpContext, _ trailerConfig, b []byte) types.Action {
					calls++
					got = string(b)
					return types.ActionPause
				}
				vm := wrapper.NewCommonVmCtx("trailers", wrapper.ParseConfig(func(gjson.Result, *trailerConfig) error { return nil }), wrapper.ProcessRequestBody(handler), wrapper.ProcessResponseBody(handler))
				h, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm).WithPluginConfiguration([]byte(`{}`)))
				h.RegisterForeignFunction("get_log_level", func([]byte) []byte { return []byte{0, 0, 0, 0} })
				defer reset()
				require.Equal(t, types.OnPluginStartStatusOK, h.StartPlugin())
				id := h.InitializeHttpContext()
				h.CallOnRequestHeaders(id, [][2]string{{":authority", "test"}, {":method", "POST"}, {"content-type", "application/json"}}, false)
				trailers := [][2]string{{"x-end", "done"}}
				if response {
					h.CallOnResponseHeaders(id, [][2]string{{":status", "200"}, {"content-type", "application/json"}}, false)
					if body != "" {
						h.CallOnResponseBody(id, []byte(body[:5]), false)
						h.CallOnResponseBody(id, []byte(body[5:]), false)
					}
					require.Equal(t, types.ActionPause, h.CallOnResponseTrailers(id, trailers))
					h.CallOnResponseTrailers(id, trailers)
				} else {
					if body != "" {
						h.CallOnRequestBody(id, []byte(body[:5]), false)
						h.CallOnRequestBody(id, []byte(body[5:]), false)
					}
					require.Equal(t, types.ActionPause, h.CallOnRequestTrailers(id, trailers))
					h.CallOnRequestTrailers(id, trailers)
				}
				require.Equal(t, 1, calls)
				require.Equal(t, body, got)
				h.CompleteHttpContext(id)
			})
		}
	}
}

func TestStreamingTrailersDoNotInvokeBufferedHandler(t *testing.T) {
	t.Run("lifecycle", func(t *testing.T) {
		buffered, streamed := 0, 0
		vm := wrapper.NewCommonVmCtx("trailers", wrapper.ParseConfig(func(gjson.Result, *trailerConfig) error { return nil }),
			wrapper.ProcessResponseBody(func(wrapper.HttpContext, trailerConfig, []byte) types.Action { buffered++; return types.ActionContinue }),
			wrapper.ProcessStreamingResponseBody(func(_ wrapper.HttpContext, _ trailerConfig, b []byte, _ bool) []byte { streamed++; return b }))
		h, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm).WithPluginConfiguration([]byte(`{}`)))
		h.RegisterForeignFunction("get_log_level", func([]byte) []byte { return []byte{0, 0, 0, 0} })
		defer reset()
		h.StartPlugin()
		id := h.InitializeHttpContext()
		h.CallOnRequestHeaders(id, [][2]string{{":authority", "test"}, {":method", "GET"}}, false)
		h.CallOnResponseHeaders(id, [][2]string{{":status", "200"}, {"content-type", "text/event-stream"}}, false)
		h.CallOnResponseBody(id, []byte("data: first\n\n"), false)
		h.CallOnResponseTrailers(id, [][2]string{{"x-end", "done"}})
		require.Equal(t, 1, streamed)
		require.Zero(t, buffered)
		h.CompleteHttpContext(id)
	})
}

func TestStreamingTrailersCanWaitForAsyncBody(t *testing.T) {
	calls := 0
	vm := wrapper.NewCommonVmCtx("trailers", wrapper.ParseConfig(func(gjson.Result, *trailerConfig) error { return nil }),
		wrapper.ProcessStreamingResponseBody(func(_ wrapper.HttpContext, _ trailerConfig, b []byte, _ bool) []byte { return b }),
		wrapper.ProcessResponseTrailers(func(wrapper.HttpContext, trailerConfig) types.Action { calls++; return types.ActionPause }))
	h, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm).WithPluginConfiguration([]byte(`{}`)))
	h.RegisterForeignFunction("get_log_level", func([]byte) []byte { return []byte{0, 0, 0, 0} })
	defer reset()
	require.Equal(t, types.OnPluginStartStatusOK, h.StartPlugin())
	id := h.InitializeHttpContext()
	h.CallOnRequestHeaders(id, [][2]string{{":authority", "test"}, {":method", "GET"}}, true)
	h.CallOnResponseHeaders(id, [][2]string{{":status", "200"}, {"content-type", "text/event-stream"}}, false)
	h.CallOnResponseBody(id, []byte("data: test\n\n"), false)
	require.Equal(t, types.ActionPause, h.CallOnResponseTrailers(id, [][2]string{{"x-end", "done"}}))
	require.Equal(t, 1, calls)
	h.CompleteHttpContext(id)
}
