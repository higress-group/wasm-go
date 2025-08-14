package test

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
)

// TestHost is the interface for the test host.
// unit test can call onHttpRequestHeaders etc. to mock the host calls.
// TestHost mock the behavior of the envoy host proxy with the wasm plugin.
type TestHost interface {
	// HostEmulator is the interface for the host emulator.
	proxytest.HostEmulator
	// CallOnHttpRequestHeaders call the onHttpRequestHeaders method in the wasm plugin.
	CallOnHttpRequestHeaders(headers [][2]string) types.Action
	// CallOnHttpRequestBody call the onHttpRequestBody method in the wasm plugin.
	CallOnHttpRequestBody(body []byte) types.Action
	// CallOnHttpStreamingRequestBody call the onHttpRequestBody method in the wasm plugin.
	CallOnHttpStreamingRequestBody(body []byte, endOfStream bool) types.Action
	// CallOnHttpResponseHeaders call the onHttpResponseHeaders method in the wasm plugin.
	CallOnHttpResponseHeaders(headers [][2]string) types.Action
	// CallOnHttpStreamingResponseBody call the onHttpResponseBody method in the wasm plugin.
	CallOnHttpStreamingResponseBody(body []byte, endOfStream bool) types.Action
	// CallOnHttpResponseBody call the onHttpResponseBody method in the wasm plugin.
	CallOnHttpResponseBody(body []byte) types.Action
	// CallOnHttpCall call the proxy_on_http_call_response method in the wasm plugin.
	CallOnHttpCall(headers [][2]string, body []byte)
	// CallOnRedisCall call the proxy_on_redis_call_response method in the wasm plugin.
	CallOnRedisCall(status int32, response []byte)
	// InitHttp init the http context which executes types.PluginContext.NewHttpContext in the plugin.
	InitHttp()
	// CompleteHttpRequest complete the http context which executes types.HttpContext.OnHttpStreamDone in the plugin.
	CompleteHttp()
	// SetRouteName set the property route_name with the route name.
	SetRouteName(routeName string) error
	// SetClusterName set the property cluster_name with the cluster name.
	SetClusterName(clusterName string) error
	// SetRequestId set the property x_request_id with the request id.
	SetRequestId(requestId string) error
	// GetMatchConfig get the match config with default host name.
	GetMatchConfig() (any, error)
	// GetMatchConfigWithHost get the match config with the host name.
	GetMatchConfigWithDomain(domain string) (any, error)
	// GetHttpStreamAction get the http stream action.
	GetHttpStreamAction() types.Action
	// GetRequestHeaders get the request headers.
	GetRequestHeaders() [][2]string
	// GetResponseHeaders get the response headers.
	GetResponseHeaders() [][2]string
	// GetRequestBody get the request body.
	GetRequestBody() []byte
	// GetResponseBody get the response body.
	GetResponseBody() []byte
	// GetLocalResponse get the local response.
	GetLocalResponse() *proxytest.LocalHttpResponse
	// Reset the test host.
	Reset()
}

// testHost is the implementation of the TestHost interface.
// proxytest.HostEmulator is the interface for the host emulator.
// currentContextID is the context id for the current http request.
// currentContextValid is the valid flag for the current http request.
// reset is the function to reset the test host.
type testHost struct {
	proxytest.HostEmulator
	currentContextID    uint32
	currentContextValid bool
	reset               func()
}

// Reset call the reset function to call internal.VMStateReset() and release mutex for currentHost.
func (h *testHost) Reset() {
	h.reset()
}

// NewTestHost create a new test host with config in json format.
func NewTestHost(config json.RawMessage) (TestHost, types.OnPluginStartStatus) {
	// if wasmInitVMContext is not set, set it to the commonVMContext.
	if getWasmInitVMContext() == nil {
		setWasmInitVMContext(proxywasm.GetVMContext())
	}
	// if testVMContext is not set, set it to the wasmInitVMContext.
	if testVMContext == nil {
		testVMContext = getWasmInitVMContext()
	}
	// create a new host emulator with the config and the testVMContext.
	opt := proxytest.NewEmulatorOption().
		WithPluginConfiguration(config).
		WithVMContext(testVMContext)

	host, reset := proxytest.NewHostEmulator(opt)
	// start the plugin.
	status := host.StartPlugin()
	// create a new test host with the host emulator and the reset function.
	testHost := &testHost{
		HostEmulator: host,
		reset:        reset,
	}
	// set the default properties.
	testHost.setDefaultProperties()
	return testHost, status
}

// setDefaultProperties set the default properties.
// set the default properties include route_name, cluster_name, x_request_id.
// unitTest can override the default properties.
func (h *testHost) setDefaultProperties() {
	h.SetRouteName("test-route-default")
	h.SetClusterName("test-cluster-default")
	h.SetRequestId("test-request-id-default")
}

// InitHttpRequest initialize the http request and set the currentContextID and currentContextValid.
func (h *testHost) InitHttp() {
	contextID := h.HostEmulator.InitializeHttpContext()
	h.currentContextID = contextID
	h.currentContextValid = true
}

// CompleteHttpRequest complete the http request and set the currentContextValid to false.
func (h *testHost) CompleteHttp() {
	h.HostEmulator.CompleteHttpContext(h.currentContextID)
	h.currentContextValid = false
}

// CallOnHttpRequestHeaders call the onHttpRequestHeaders method in the wasm plugin.
func (h *testHost) CallOnHttpRequestHeaders(headers [][2]string) types.Action {
	if !h.currentContextValid {
		h.InitHttp()
	}
	action := h.HostEmulator.CallOnRequestHeaders(h.currentContextID, headers, false)
	return action
}

// CallOnHttpRequestBody call the onHttpRequestBody method in the wasm plugin.
func (h *testHost) CallOnHttpRequestBody(body []byte) types.Action {
	if !h.currentContextValid {
		h.InitHttp()
		action := h.HostEmulator.CallOnRequestHeaders(h.currentContextID, [][2]string{{":authority", defaultTestDomain}}, false)
		if action != types.ActionContinue {
			panic("wasm plugin unit test should CallOnHttpRequestHeaderss first")
		}
	}
	action := h.HostEmulator.CallOnRequestBody(h.currentContextID, body, true)
	return action
}

// CallOnHttpStreamingRequestBody call the onHttpRequestBody method in the wasm plugin.
// endOfStream is true if the body is the last chunk of the request body.
func (h *testHost) CallOnHttpStreamingRequestBody(body []byte, endOfStream bool) types.Action {
	if !h.currentContextValid {
		h.InitHttp()
		action := h.HostEmulator.CallOnRequestHeaders(h.currentContextID, [][2]string{{":authority", defaultTestDomain}}, false)
		if action != types.ActionContinue {
			panic("wasm plugin unit test should CallOnHttpRequestHeaderss first")
		}
	}
	action := h.HostEmulator.CallOnRequestBody(h.currentContextID, body, endOfStream)
	return action
}

// CallOnHttpStreamingResponseBody call the onHttpResponseBody method in the wasm plugin.
// endOfStream is true if the body is the last chunk of the response body.
func (h *testHost) CallOnHttpStreamingResponseBody(body []byte, endOfStream bool) types.Action {
	if !h.currentContextValid {
		h.InitHttp()
		action := h.HostEmulator.CallOnRequestHeaders(h.currentContextID, [][2]string{{":authority", defaultTestDomain}}, false)
		if action != types.ActionContinue {
			panic("wasm plugin unit test should CallOnHttpRequestHeaderss first")
		}
	}
	action := h.HostEmulator.CallOnResponseBody(h.currentContextID, body, endOfStream)
	return action
}

// CallOnHttpResponseHeaders call the onHttpResponseHeaders method in the wasm plugin.
func (h *testHost) CallOnHttpResponseHeaders(headers [][2]string) types.Action {
	if !h.currentContextValid {
		h.InitHttp()
		action := h.HostEmulator.CallOnRequestHeaders(h.currentContextID, [][2]string{{":authority", defaultTestDomain}}, false)
		if action != types.ActionContinue {
			panic("wasm plugin unit test should CallOnHttpRequestHeaderss first")
		}
	}
	action := h.HostEmulator.CallOnResponseHeaders(h.currentContextID, headers, false)
	return action
}

// CallOnHttpResponseBody call the onHttpResponseBody method in the wasm plugin.
func (h *testHost) CallOnHttpResponseBody(body []byte) types.Action {
	if !h.currentContextValid {
		h.InitHttp()
		action := h.HostEmulator.CallOnRequestHeaders(h.currentContextID, [][2]string{{":authority", defaultTestDomain}}, false)
		if action != types.ActionContinue {
			panic("wasm plugin unit test should CallOnHttpRequestHeaderss first")
		}
	}
	action := h.HostEmulator.CallOnResponseBody(h.currentContextID, body, true)
	return action
}

// CallOnHttpCall call the proxy_on_http_call_response method in the wasm plugin.
func (h *testHost) CallOnHttpCall(headers [][2]string, body []byte) {
	attrs := h.HostEmulator.GetCalloutAttributesFromContext(h.currentContextID)
	calloutID := attrs[0].CalloutID
	h.HostEmulator.CallOnHttpCallResponse(calloutID, headers, nil, body)
}

// CallOnRedisCall call the proxy_on_redis_call_response method in the wasm plugin.
func (h *testHost) CallOnRedisCall(status int32, response []byte) {
	attrs := h.HostEmulator.GetRedisCalloutAttributesFromContext(h.currentContextID)
	calloutID := attrs[0].CalloutID
	h.HostEmulator.CallOnRedisCallResponse(calloutID, status, response)
}

// SetRouteName set the property route_name with the route name.
func (h *testHost) SetRouteName(routeName string) error {
	return h.SetProperty([]string{"route_name"}, []byte(routeName))
}

// SetClusterName set the property cluster_name with the cluster name.
func (h *testHost) SetClusterName(clusterName string) error {
	return h.SetProperty([]string{"cluster_name"}, []byte(clusterName))
}

// SetRequestId set the property x_request_id with the request id.
func (h *testHost) SetRequestId(requestId string) error {
	return h.SetProperty([]string{"x_request_id"}, []byte(requestId))
}

// Set host name to defaultTestHostName if not provided, to make sure match config is not empty
func (h *testHost) GetMatchConfig() (any, error) {
	return h.GetMatchConfigWithDomain(defaultTestDomain)
}

// GetMatchConfigWithDomain get the match config with domain.
// GetMatchConfig depends on reflect feature so it can only be used in go mode.
// return config type is any, so unitTest needs to cast the config to the actual type.
func (h *testHost) GetMatchConfigWithDomain(domain string) (any, error) {
	if domain == "" {
		return nil, errors.New("domain is empty")
	}
	contextID := h.HostEmulator.InitializeHttpContext()

	h.HostEmulator.SetHttpRequestHeaders(contextID, [][2]string{{":authority", domain}})

	httpContext := proxywasm.GetHttpContext(contextID)
	h.HostEmulator.CompleteHttpContext(contextID)

	httpContextValue := reflect.ValueOf(httpContext)
	if httpContextValue.Kind() == reflect.Ptr && !httpContextValue.IsNil() {
		// Try to call GetMatchConfig method using reflection
		method := httpContextValue.MethodByName("GetMatchConfig")
		if method.IsValid() {
			results := method.Call(nil)
			if len(results) == 2 {
				var err error
				if results[1].Interface() != nil {
					err = results[1].Interface().(error)
				}
				return results[0].Interface(), err
			}
		}
	}
	return nil, errors.New("http context is not a common http context")
}

// GetHttpStreamAction get the http stream action.
func (h *testHost) GetHttpStreamAction() types.Action {
	return h.HostEmulator.GetCurrentHttpStreamAction(h.currentContextID)
}

// GetRequestHeaders get the request headers.
func (h *testHost) GetRequestHeaders() [][2]string {
	return h.HostEmulator.GetCurrentRequestHeaders(h.currentContextID)
}

// GetRequestBody get the request body.
func (h *testHost) GetRequestBody() []byte {
	return h.HostEmulator.GetCurrentRequestBody(h.currentContextID)
}

// GetResponseBody get the response body.
func (h *testHost) GetResponseBody() []byte {
	return h.HostEmulator.GetCurrentResponseBody(h.currentContextID)
}

// GetResponseHeaders get the response headers.
func (h *testHost) GetResponseHeaders() [][2]string {
	return h.HostEmulator.GetCurrentResponseHeaders(h.currentContextID)
}

// GetLocalResponse get the local response.
func (h *testHost) GetLocalResponse() *proxytest.LocalHttpResponse {
	return h.HostEmulator.GetSentLocalResponse(h.currentContextID)
}
