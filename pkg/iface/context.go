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

package iface

type RouteResponseCallback func(statusCode int, responseHeaders [][2]string, responseBody []byte)

type HTTPExecutionPhase int

const (
	DecodeHeader HTTPExecutionPhase = iota
	DecodeData
	EncodeHeader
	EncodeData
	Done
)

type PluginContext interface {
	SetContext(key string, value interface{})
	GetContext(key string) interface{}
	// When this switch is enabled, the failure of a single configuration rule to parse will not block other configurations within the plugin from taking effect,
	// and in case of a parsing failure, it will attempt to use the last successfully parsed configuration stored in memory.
	EnableRuleLevelConfigIsolation()
	IsRuleLevelConfigIsolation() bool
	GetFingerPrint() string
	DoLeaderElection()
	IsLeader() bool
}

type HttpContext interface {
	Scheme() string
	Host() string
	Path() string
	Method() string
	SetContext(key string, value interface{})
	GetContext(key string) interface{}
	GetBoolContext(key string, defaultValue bool) bool
	GetStringContext(key, defaultValue string) string
	GetByteSliceContext(key string, defaultValue []byte) []byte
	GetUserAttribute(key string) interface{}
	SetUserAttribute(key string, value interface{})
	SetUserAttributeMap(kvmap map[string]interface{})
	GetUserAttributeMap() map[string]interface{}
	// You can call this function to set custom log
	WriteUserAttributeToLog() error
	// You can call this function to set custom log with your specific key
	WriteUserAttributeToLogWithKey(key string) error
	// You can call this function to set custom trace span attribute
	WriteUserAttributeToTrace() error
	// If the onHttpRequestBody handle is not set, the request body will not be read by default
	DontReadRequestBody()
	// If the onHttpResponseBody handle is not set, the request body will not be read by default
	DontReadResponseBody()
	// If the onHttpStreamingRequestBody handle is not set, and the onHttpRequestBody handle is set, the request body will be buffered by default
	BufferRequestBody()
	// If the onHttpStreamingResponseBody handle is not set, and the onHttpResponseBody handle is set, the response body will be buffered by default
	BufferResponseBody()
	// If any request header is changed in onHttpRequestHeaders, envoy will re-calculate the route. Call this function to disable the re-routing.
	// You need to call this before making any header modification operations.
	DisableReroute()
	// Note that this parameter affects the gateway's memory usage！Support setting a maximum buffer size for each request body individually in request phase.
	SetRequestBodyBufferLimit(byteSize uint32)
	// Note that this parameter affects the gateway's memory usage! Support setting a maximum buffer size for each response body individually in response phase.
	SetResponseBodyBufferLimit(byteSize uint32)
	// Make a request to the target service of the current route using the specified URL and header.
	RouteCall(method, url string, headers [][2]string, body []byte, callback RouteResponseCallback) error
	// Get the execution phase of the current plugin
	GetExecutionPhase() HTTPExecutionPhase
}
