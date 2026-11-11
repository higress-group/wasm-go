// Copyright (c) 2026 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrapper

import (
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/stretchr/testify/require"
)

// TestBufferedRequestBodyHeaderAction checks which requests must hold headers for
// a complete body, while preserving explicit actions, opt-outs and streaming.
func TestBufferedRequestBodyHeaderAction(t *testing.T) {
	tests := []struct {
		name        string
		headers     [][2]string
		endOfStream bool
		noHeaders   bool
		noBody      bool
		streaming   bool
		skip        bool
		required    bool
		forceBuffer bool
		action      types.Action
		want        types.Action
	}{
		{name: "buffered body", want: types.HeaderStopIteration},
		{name: "body handler without header handler", noHeaders: true, want: types.HeaderStopIteration},
		{name: "header-only plugin", noBody: true, want: types.ActionContinue},
		{name: "header-only request", endOfStream: true, want: types.ActionContinue},
		{name: "request opt-out", skip: true, want: types.ActionContinue},
		{name: "streaming takes precedence", streaming: true, want: types.ActionContinue},
		{name: "streaming-only plugin", noBody: true, streaming: true, want: types.ActionContinue},
		{name: "switch streaming to buffered", streaming: true, forceBuffer: true, want: types.HeaderStopIteration},
		{name: "binary opt-out", headers: [][2]string{{"content-type", "application/octet-stream"}}, want: types.ActionContinue},
		{name: "grpc opt-out", headers: [][2]string{{"content-type", "application/grpc"}}, want: types.ActionContinue},
		{name: "encoded opt-out", headers: [][2]string{{"content-encoding", "gzip"}}, want: types.ActionContinue},
		{name: "websocket opt-out", headers: [][2]string{{"connection", "upgrade"}, {"upgrade", "websocket"}}, want: types.ActionContinue},
		{name: "required overrides opt-out", required: true, skip: true, want: types.HeaderStopIteration},
		{name: "required overrides streaming", required: true, streaming: true, want: types.HeaderStopIteration},
		{name: "required binary", required: true, headers: [][2]string{{"content-type", "application/octet-stream"}}, want: types.HeaderStopIteration},
		{name: "explicit stop", action: types.HeaderStopIteration, want: types.HeaderStopIteration},
		{name: "explicit stop all and buffer", action: types.HeaderStopAllIterationAndBuffer, want: types.HeaderStopAllIterationAndBuffer},
		{name: "explicit stop all and watermark", action: types.HeaderStopAllIterationAndWatermark, want: types.HeaderStopAllIterationAndWatermark},
		{name: "async header with body opt-out", skip: true, action: types.HeaderStopAllIterationAndWatermark, want: types.HeaderStopAllIterationAndWatermark},
		{name: "required empty body preserves stop all", endOfStream: true, required: true, action: types.HeaderStopAllIterationAndWatermark, want: types.HeaderStopAllIterationAndWatermark},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyCalls := 0
			var options []CtxOption[struct{}]
			if !tt.noHeaders {
				options = append(options, ProcessRequestHeaders(func(ctx HttpContext, _ struct{}) types.Action {
					if tt.skip {
						ctx.DontReadRequestBody()
					}
					if tt.required {
						ctx.RequireRequestBody()
					}
					if tt.forceBuffer {
						ctx.BufferRequestBody()
					}
					return tt.action
				}))
			}
			if !tt.noBody {
				options = append(options, ProcessRequestBody(func(HttpContext, struct{}, []byte) types.Action {
					bodyCalls++
					return types.ActionContinue
				}))
			}
			if tt.streaming {
				options = append(options, ProcessStreamingRequestBody(func(_ HttpContext, _ struct{}, chunk []byte, _ bool) []byte {
					return chunk
				}))
			}
			vm := NewCommonVmCtx[struct{}]("body-iteration-test", options...)
			host, reset := proxytest.NewHostEmulator(proxytest.NewEmulatorOption().WithVMContext(vm))
			defer reset()
			host.RegisterForeignFunction("get_log_level", func([]byte) []byte { return make([]byte, 4) })
			require.Equal(t, types.OnPluginStartStatusOK, host.StartPlugin())
			id := host.InitializeHttpContext()
			headers := append([][2]string{{":authority", "default.test.com"}}, tt.headers...)
			require.Equal(t, tt.want, host.CallOnRequestHeaders(id, headers, tt.endOfStream))
			if tt.want == types.ActionContinue && !tt.endOfStream {
				require.Equal(t, types.ActionContinue, host.CallOnRequestBody(id, []byte("chunk"), false))
				require.Equal(t, types.ActionContinue, host.CallOnRequestTrailers(id, nil))
				require.Zero(t, bodyCalls, "skipped and streaming requests must not call the buffered handler")
			}
		})
	}
}

// TestBufferedRequestBodyReadFailures preserves the legacy fail-open policy and
// required delivery's fail-closed policy if the host cannot supply its reported body.
func TestBufferedRequestBodyReadFailures(t *testing.T) {
	for _, required := range []bool{false, true} {
		for _, truncated := range []bool{false, true} {
			name := map[bool]string{false: "legacy", true: "required"}[required]
			name += map[bool]string{false: " missing", true: " truncated"}[truncated]
			t.Run(name, func(t *testing.T) {
				var context *CommonHttpCtx[struct{}]
				h := newBodyDeliveryHarness(t, required, func(ctx HttpContext) {
					context = ctx.(*CommonHttpCtx[struct{}])
				})
				require.Equal(t, types.HeaderStopIteration, h.callRequestHeaders(nil, false))
				if truncated {
					require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, []byte("abc"), false))
				}
				// Deliberately report more bytes than the host actually makes available.
				action := context.OnHttpRequestBody(8, true)
				if required {
					require.Equal(t, types.ActionPause, action)
					require.Zero(t, h.requestBodyCalls)
					response := h.host.GetSentLocalResponse(h.contextID)
					require.NotNil(t, response)
					require.Equal(t, uint32(RequiredRequestBodyFailureStatusCode), response.StatusCode)
				} else {
					require.Equal(t, types.ActionContinue, action)
					require.Nil(t, h.host.GetSentLocalResponse(h.contextID))
					if truncated {
						require.Equal(t, []byte("abc"), h.requestBody)
					} else {
						require.Zero(t, h.requestBodyCalls)
					}
				}
			})
		}
	}
}

// TestBufferedRequestBodyDelivery checks complete, exactly-once body delivery for
// both DATA and trailer termination, including fragmented and empty bodies.
func TestBufferedRequestBodyDelivery(t *testing.T) {
	for _, trailers := range []bool{false, true} {
		name := "data end stream"
		if trailers {
			name = "trailers"
		}
		t.Run(name, func(t *testing.T) {
			h := newBodyDeliveryHarness(t, false)
			require.Equal(t, types.HeaderStopIteration, h.callRequestHeaders(nil, false))
			require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, []byte("abc"), false))
			require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, []byte("def"), false))
			require.Zero(t, h.requestBodyCalls)
			if trailers {
				require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, []byte("ghi"), false))
				require.Equal(t, types.ActionContinue, h.host.CallOnRequestTrailers(h.contextID, [][2]string{{"x-checksum", "abc"}}))
			} else {
				require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, []byte("ghi"), true))
			}
			require.Equal(t, []byte("abcdefghi"), h.requestBody)
			require.Equal(t, 1, h.requestBodyCalls)
			require.Equal(t, types.ActionContinue, h.host.CallOnRequestTrailers(h.contextID, nil))
			require.Equal(t, 1, h.requestBodyCalls)
		})
	}
	for _, trailers := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty data", true: "empty trailers"}[trailers], func(t *testing.T) {
			h := newBodyDeliveryHarness(t, false)
			require.Equal(t, types.HeaderStopIteration, h.callRequestHeaders(nil, false))
			if trailers {
				require.Equal(t, types.ActionContinue, h.host.CallOnRequestTrailers(h.contextID, nil))
			} else {
				require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, nil, true))
			}
			require.Equal(t, 1, h.requestBodyCalls)
			require.Empty(t, h.requestBody)
		})
	}
}

// TestBufferedRequestBodyPreservesBodyPause ensures an asynchronous body handler
// still controls its own continuation after the wrapper finishes buffering.
func TestBufferedRequestBodyPreservesBodyPause(t *testing.T) {
	for _, trailers := range []bool{false, true} {
		t.Run(map[bool]string{false: "data", true: "trailers"}[trailers], func(t *testing.T) {
			h := newBodyDeliveryHarness(t, false)
			h.requestBodyAction = types.ActionPause
			require.Equal(t, types.HeaderStopIteration, h.callRequestHeaders(nil, false))
			if trailers {
				require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, []byte("body"), false))
				require.Equal(t, types.ActionPause, h.host.CallOnRequestTrailers(h.contextID, nil))
			} else {
				require.Equal(t, types.ActionPause, h.host.CallOnRequestBody(h.contextID, []byte("body"), true))
			}
			require.Equal(t, 1, h.requestBodyCalls)
			require.Equal(t, []byte("body"), h.requestBody)
			require.NoError(t, proxywasm.ResumeHttpRequest())
			require.Equal(t, types.ActionContinue, h.host.GetCurrentHttpStreamAction(h.contextID))
			require.Equal(t, 1, h.requestBodyCalls)
		})
	}
}
