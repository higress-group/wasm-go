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

package wrapper

import (
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/stretchr/testify/require"
)

type bodyDeliveryHarness struct {
	host                  proxytest.HostEmulator
	contextID             uint32
	requestBody           []byte
	requestBodyCalls      int
	requestHeaderAction   types.Action
	requestBodyAction     types.Action
	responseBodyCallCount int
}

func newBodyDeliveryHarness(t *testing.T, requireBody bool, headerOptions ...func(HttpContext)) *bodyDeliveryHarness {
	t.Helper()
	h := &bodyDeliveryHarness{
		requestHeaderAction: types.ActionContinue,
		requestBodyAction:   types.ActionContinue,
	}
	vm := NewCommonVmCtx[struct{}](
		"body-delivery-test",
		ProcessRequestHeaders(func(ctx HttpContext, _ struct{}) types.Action {
			if requireBody {
				ctx.RequireRequestBody()
			}
			for _, option := range headerOptions {
				option(ctx)
			}
			return h.requestHeaderAction
		}),
		ProcessRequestBody(func(_ HttpContext, _ struct{}, body []byte) types.Action {
			h.requestBodyCalls++
			h.requestBody = append([]byte(nil), body...)
			return h.requestBodyAction
		}),
		ProcessResponseBody(func(_ HttpContext, _ struct{}, _ []byte) types.Action {
			h.responseBodyCallCount++
			return types.ActionContinue
		}),
	)
	host, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm))
	t.Cleanup(reset)
	host.RegisterForeignFunction("get_log_level", func([]byte) []byte { return make([]byte, 4) })
	require.Equal(t, types.OnPluginStartStatusOK, host.StartPlugin())
	h.host = host
	h.contextID = host.InitializeHttpContext()
	return h
}

func (h *bodyDeliveryHarness) callRequestHeaders(headers [][2]string, endOfStream bool) types.Action {
	headers = append(headers, [2]string{":authority", "default.test.com"})
	return h.host.CallOnRequestHeaders(h.contextID, headers, endOfStream)
}

func TestRequireRequestBodyRawDelivery(t *testing.T) {
	payload := []byte{0, 1, 2, 0xff, 'a', 'b', 'c'}
	tests := []struct {
		name    string
		headers [][2]string
	}{
		{name: "binary content type", headers: [][2]string{{"content-type", "application/octet-stream"}}},
		{name: "identity encoding", headers: [][2]string{{"content-encoding", "identity"}}},
		{name: "gzip encoding", headers: [][2]string{{"content-encoding", "gzip"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBodyDeliveryHarness(t, true)
			require.Equal(t, types.ActionContinue, h.callRequestHeaders(tt.headers, false))
			require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, payload[:3], false))
			require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, payload[3:], true))
			require.Equal(t, 1, h.requestBodyCalls)
			require.Equal(t, payload, h.requestBody)
			require.Nil(t, h.host.GetSentLocalResponse(h.contextID))
		})
	}
}

func TestRequireRequestBodyWebSocketRequestOnly(t *testing.T) {
	h := newBodyDeliveryHarness(t, true)
	headers := [][2]string{{"connection", "upgrade"}, {"upgrade", "websocket"}}
	require.Equal(t, types.ActionContinue, h.callRequestHeaders(headers, false))
	payload := []byte{0x81, 0x02, 'h', 'i'}
	require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, payload, true))
	require.Equal(t, payload, h.requestBody)
	require.Equal(t, 1, h.requestBodyCalls)

	require.Equal(t, types.ActionContinue, h.host.CallOnResponseHeaders(h.contextID, nil, false))
	require.Equal(t, types.ActionContinue, h.host.CallOnResponseBody(h.contextID, []byte("response"), true))
	require.Zero(t, h.responseBodyCallCount)
}

func TestRequireRequestBodyHeaderEndOfStream(t *testing.T) {
	h := newBodyDeliveryHarness(t, true)
	require.Equal(t, types.ActionContinue, h.callRequestHeaders(nil, true))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Empty(t, h.requestBody)
	require.Nil(t, h.host.GetSentLocalResponse(h.contextID))

	require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, []byte("unexpected"), true))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Empty(t, h.requestBody)
}

func TestRequireRequestBodyHeaderPauseDeliversBody(t *testing.T) {
	h := newBodyDeliveryHarness(t, true)
	h.requestHeaderAction = types.ActionPause
	require.Equal(t, types.ActionPause, h.callRequestHeaders(nil, true))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Empty(t, h.requestBody)
	require.Nil(t, h.host.GetSentLocalResponse(h.contextID))

	require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, []byte("unexpected"), true))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Empty(t, h.requestBody)
}

func TestRequireRequestBodyWithoutBufferedHandlerHeaderCallback(t *testing.T) {
	vm := NewCommonVmCtx[struct{}](
		"required-body-without-buffered-handler",
		ProcessRequestHeaders(func(ctx HttpContext, _ struct{}) types.Action {
			ctx.RequireRequestBody()
			return types.ActionContinue
		}),
	)
	host, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm))
	t.Cleanup(reset)
	host.RegisterForeignFunction("get_log_level", func([]byte) []byte { return make([]byte, 4) })
	require.Equal(t, types.OnPluginStartStatusOK, host.StartPlugin())
	contextID := host.InitializeHttpContext()

	headers := [][2]string{{":authority", "default.test.com"}}
	require.Equal(t, types.ActionPause, host.CallOnRequestHeaders(contextID, headers, false))
	response := host.GetSentLocalResponse(contextID)
	require.NotNil(t, response)
	require.Equal(t, uint32(RequiredRequestBodyFailureStatusCode), response.StatusCode)
	require.Equal(t, types.ActionPause, host.CallOnRequestBody(contextID, []byte("body"), true))
	require.Equal(t, types.ActionPause, host.CallOnRequestTrailers(contextID, nil))
}

func TestRequireRequestBodyWithoutBufferedHandlerStreamingCallback(t *testing.T) {
	streamingCalls := 0
	vm := NewCommonVmCtx[struct{}](
		"required-body-with-streaming-handler",
		ProcessStreamingRequestBody(func(ctx HttpContext, _ struct{}, chunk []byte, _ bool) []byte {
			streamingCalls++
			ctx.RequireRequestBody()
			return chunk
		}),
	)
	host, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm))
	t.Cleanup(reset)
	host.RegisterForeignFunction("get_log_level", func([]byte) []byte { return make([]byte, 4) })
	require.Equal(t, types.OnPluginStartStatusOK, host.StartPlugin())
	contextID := host.InitializeHttpContext()

	headers := [][2]string{{":authority", "default.test.com"}}
	require.Equal(t, types.ActionContinue, host.CallOnRequestHeaders(contextID, headers, false))
	require.Equal(t, types.ActionPause, host.CallOnRequestBody(contextID, []byte("body"), true))
	require.Equal(t, 1, streamingCalls)
	response := host.GetSentLocalResponse(contextID)
	require.NotNil(t, response)
	require.Equal(t, uint32(RequiredRequestBodyFailureStatusCode), response.StatusCode)
}

func TestRequireRequestBodyEmptyTerminalBody(t *testing.T) {
	h := newBodyDeliveryHarness(t, true)
	require.Equal(t, types.ActionContinue, h.callRequestHeaders(nil, false))
	require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, nil, true))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Empty(t, h.requestBody)
	require.Nil(t, h.host.GetSentLocalResponse(h.contextID))
}

func TestRequireRequestBodyTrailersDeliverBufferedBody(t *testing.T) {
	h := newBodyDeliveryHarness(t, true)
	payload := []byte{0, 0, 0, 0, 1, 'x'}
	headers := [][2]string{{"content-type", "application/grpc"}}
	require.Equal(t, types.ActionContinue, h.callRequestHeaders(headers, false))
	require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, payload, false))
	require.Zero(t, h.requestBodyCalls)
	require.Equal(t, types.ActionContinue, h.host.CallOnRequestTrailers(h.contextID, [][2]string{{"x-checksum", "abc"}}))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Equal(t, payload, h.requestBody)
	require.Nil(t, h.host.GetSentLocalResponse(h.contextID))

	require.Equal(t, types.ActionContinue, h.host.CallOnRequestTrailers(h.contextID, nil))
	require.Equal(t, 1, h.requestBodyCalls)
}

func TestRequireRequestBodyOverridesDontReadRequestBody(t *testing.T) {
	h := newBodyDeliveryHarness(t, true, func(ctx HttpContext) {
		ctx.DontReadRequestBody()
	})
	require.Equal(t, types.ActionContinue, h.callRequestHeaders(nil, false))
	payload := []byte("required")
	require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, payload, true))
	require.Equal(t, 1, h.requestBodyCalls)
	require.Equal(t, payload, h.requestBody)
	require.Nil(t, h.host.GetSentLocalResponse(h.contextID))
}

func TestRequireRequestBodyBufferLimitExceeded(t *testing.T) {
	const bufferLimit = 4
	payload := []byte("larger than the buffer limit")
	h := newBodyDeliveryHarness(t, true, func(ctx HttpContext) {
		ctx.SetRequestBodyBufferLimit(bufferLimit)
	})
	require.Equal(t, types.ActionContinue, h.callRequestHeaders(nil, false))
	require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, payload[:2], false))
	require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, payload[2:], true))
	require.Zero(t, h.requestBodyCalls)
	response := h.host.GetSentLocalResponse(h.contextID)
	require.NotNil(t, response)
	require.Equal(t, uint32(RequiredRequestBodyFailureStatusCode), response.StatusCode)
}

func TestRequestBodyLegacySkips(t *testing.T) {
	tests := []struct {
		name    string
		headers [][2]string
	}{
		{name: "binary content type", headers: [][2]string{{"content-type", "application/octet-stream"}}},
		{name: "content encoding", headers: [][2]string{{"content-encoding", "gzip"}}},
		{name: "websocket", headers: [][2]string{{"connection", "upgrade"}, {"upgrade", "websocket"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBodyDeliveryHarness(t, false)
			require.Equal(t, types.ActionContinue, h.callRequestHeaders(tt.headers, false))
			require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, []byte("skipped"), true))
			require.Zero(t, h.requestBodyCalls)
			require.Nil(t, h.host.GetSentLocalResponse(h.contextID))
		})
	}

	t.Run("plain body remains delivered", func(t *testing.T) {
		h := newBodyDeliveryHarness(t, false)
		require.Equal(t, types.ActionContinue, h.callRequestHeaders(nil, false))
		require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, []byte("plain"), true))
		require.Equal(t, []byte("plain"), h.requestBody)
		require.Equal(t, 1, h.requestBodyCalls)
	})
}
