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

package main

import (
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {}

// init registers a complete-body fixture with optional required delivery. Its
// header callback always continues so the test observes only SDK-added holds.
func init() {
	wrapper.SetCtx[struct{}]("handshake-regression",
		wrapper.ProcessRequestHeaders(func(ctx wrapper.HttpContext, _ struct{}) types.Action {
			required, _ := proxywasm.GetHttpRequestHeader("x-require-body")
			if required == "1" {
				ctx.RequireRequestBody()
			}
			return types.ActionContinue
		}),
		wrapper.ProcessRequestBody(func(wrapper.HttpContext, struct{}, []byte) types.Action {
			return types.ActionContinue
		}),
	)
}
