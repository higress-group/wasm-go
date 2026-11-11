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
	"fmt"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/stretchr/testify/require"
)

// TestRequestHandshakeCompatibility preserves the original header action for
// protocols that can require an upstream response before request data arrives.
// Exemption from automatic header holding must not disable body delivery.
func TestRequestHandshakeCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		headers    [][2]string
		required   bool
		bufferBody bool
	}{
		{name: "expect continue", headers: [][2]string{{"expect", "100-continue"}}, bufferBody: true},
		{name: "expect token case and whitespace", headers: [][2]string{{"expect", " 100-CoNtInUe "}}, bufferBody: true},
		{name: "expect token list", headers: [][2]string{{"expect", "extension, 100-continue"}}, bufferBody: true},
		{name: "required expect body", headers: [][2]string{{"expect", "100-continue"}}, required: true, bufferBody: true},
		{name: "connect", headers: [][2]string{{":method", "CONNECT"}}, bufferBody: true},
		{name: "required connect body", headers: [][2]string{{":method", "CONNECT"}}, required: true, bufferBody: true},
		{name: "websocket automatic opt-out", headers: [][2]string{{"connection", "upgrade"}, {"upgrade", "websocket"}}},
		{name: "required websocket body", headers: [][2]string{{"connection", "upgrade"}, {"upgrade", "websocket"}}, required: true, bufferBody: true},
		{name: "connection token list", headers: [][2]string{{"connection", "keep-alive, UpGrAdE"}, {"upgrade", "websocket"}}, bufferBody: true},
		{name: "required connection token list", headers: [][2]string{{"connection", "keep-alive, Upgrade"}, {"upgrade", "websocket"}}, required: true, bufferBody: true},
		{name: "custom upgrade", headers: [][2]string{{"connection", "upgrade"}, {"upgrade", "custom-protocol"}}, bufferBody: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, action := range []types.Action{
				types.ActionContinue,
				types.HeaderStopIteration,
				types.HeaderStopAllIterationAndBuffer,
				types.HeaderStopAllIterationAndWatermark,
			} {
				t.Run(fmt.Sprintf("action-%d", action), func(t *testing.T) {
					h := newBodyDeliveryHarness(t, tt.required)
					h.requestHeaderAction = action
					require.Equal(t, action, h.callRequestHeaders(tt.headers, false))
					if action != types.ActionContinue {
						return
					}
					wantData := types.ActionContinue
					if tt.bufferBody {
						wantData = types.ActionPause
					}
					require.Equal(t, wantData, h.host.CallOnRequestBody(h.contextID, []byte("abc"), false))
					require.Zero(t, h.requestBodyCalls)
					require.Equal(t, types.ActionContinue, h.host.CallOnRequestBody(h.contextID, []byte("def"), true))
					if tt.bufferBody {
						require.Equal(t, 1, h.requestBodyCalls)
						require.Equal(t, []byte("abcdef"), h.requestBody)
					} else {
						require.Zero(t, h.requestBodyCalls)
					}
				})
			}
		})
	}
}
