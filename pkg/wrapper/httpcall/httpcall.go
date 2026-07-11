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

// Package httpcall provides the lightweight HTTP callout API for Proxy-Wasm
// plugins. Build plugins that use this package with the wasm_lite_http tag so
// the legacy net/http-based API is excluded from the parent wrapper package.
package httpcall

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"

	"github.com/higress-group/wasm-go/pkg/litehttp"
	"github.com/higress-group/wasm-go/pkg/liteuuid"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// Header is the lightweight HTTP header type used by callout callbacks.
type Header = litehttp.Header

// ResponseCallback handles an HTTP callout response.
type ResponseCallback func(statusCode int, responseHeaders Header, responseBody []byte)

// HttpClient dispatches HTTP callouts through a configured Proxy-Wasm cluster.
type HttpClient interface {
	Get(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Head(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Options(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Post(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Put(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Patch(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Delete(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Connect(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Trace(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Call(method, rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	ClusterName() string
}

// ClusterClient dispatches HTTP callouts through cluster.
type ClusterClient[C wrapper.Cluster] struct {
	cluster C
}

// NewClusterClient returns a lightweight HTTP client for cluster.
func NewClusterClient[C wrapper.Cluster](cluster C) *ClusterClient[C] {
	return &ClusterClient[C]{cluster: cluster}
}

// Get dispatches a GET callout.
func (c ClusterClient[C]) Get(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodGet, rawURL, headers, nil, cb, timeoutMillisecond...)
}

// Head dispatches a HEAD callout.
func (c ClusterClient[C]) Head(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodHead, rawURL, headers, nil, cb, timeoutMillisecond...)
}

// Options dispatches an OPTIONS callout.
func (c ClusterClient[C]) Options(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodOptions, rawURL, headers, nil, cb, timeoutMillisecond...)
}

// Post dispatches a POST callout.
func (c ClusterClient[C]) Post(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodPost, rawURL, headers, body, cb, timeoutMillisecond...)
}

// Put dispatches a PUT callout.
func (c ClusterClient[C]) Put(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodPut, rawURL, headers, body, cb, timeoutMillisecond...)
}

// Patch dispatches a PATCH callout.
func (c ClusterClient[C]) Patch(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodPatch, rawURL, headers, body, cb, timeoutMillisecond...)
}

// Delete dispatches a DELETE callout.
func (c ClusterClient[C]) Delete(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodDelete, rawURL, headers, body, cb, timeoutMillisecond...)
}

// Connect dispatches a CONNECT callout.
func (c ClusterClient[C]) Connect(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodConnect, rawURL, headers, body, cb, timeoutMillisecond...)
}

// Trace dispatches a TRACE callout.
func (c ClusterClient[C]) Trace(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, litehttp.MethodTrace, rawURL, headers, body, cb, timeoutMillisecond...)
}

// Call dispatches a callout with an arbitrary HTTP method.
func (c ClusterClient[C]) Call(method, rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, method, rawURL, headers, body, cb, timeoutMillisecond...)
}

// ClusterName returns the Proxy-Wasm cluster name.
func (c ClusterClient[C]) ClusterName() string {
	return c.cluster.ClusterName()
}

// HttpCall dispatches an HTTP callout through cluster.
func HttpCall(cluster wrapper.Cluster, method, rawURL string, headers [][2]string, body []byte,
	callback ResponseCallback, timeoutMillisecond ...uint32) error {
	for i := len(headers) - 1; i >= 0; i-- {
		key := headers[i][0]
		if key == ":method" || key == ":path" || key == ":authority" {
			headers = append(headers[:i], headers[i+1:]...)
		}
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		proxywasm.LogCriticalf("invalid rawURL:%s", rawURL)
		return err
	}
	authority := cluster.HostName()
	if parsedURL.Host != "" {
		authority = parsedURL.Host
	}
	if authority == "" {
		authority = "unknownhost"
	}
	path := "/" + strings.TrimPrefix(parsedURL.Path, "/")
	if parsedURL.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, parsedURL.RawQuery)
	}
	// Keep the legacy wrapper timeout behavior: the default is 500ms.
	var timeout uint32 = 500
	if len(timeoutMillisecond) > 0 {
		timeout = timeoutMillisecond[0]
	}
	headers = append(headers, [2]string{":method", method}, [2]string{":path", path}, [2]string{":authority", authority})
	requestID := liteuuid.New().String()
	_, err = proxywasm.DispatchHttpCall(cluster.ClusterName(), headers, body, nil, timeout, func(numHeaders, bodySize, numTrailers int) {
		respBody, err := proxywasm.GetHttpCallResponseBody(0, bodySize)
		if err != nil {
			proxywasm.LogDebugf("body is empty")
		}
		respHeaders, err := proxywasm.GetHttpCallResponseHeaders()
		if err != nil {
			proxywasm.LogCriticalf("failed to get response headers: %v", err)
		}
		code := litehttp.StatusBadGateway
		var normalResponse bool
		responseHeaders := make(Header)
		for _, header := range respHeaders {
			if header[0] == ":status" {
				code, err = strconv.Atoi(header[1])
				if err != nil {
					proxywasm.LogErrorf("failed to parse status: %v", err)
					code = litehttp.StatusInternalServerError
				} else {
					normalResponse = true
				}
			}
			responseHeaders.Add(header[0], header[1])
		}
		log.UnsafeInfof("http call end, id: %s, code: %d, normal: %t, body: %s",
			requestID, code, normalResponse, strings.ReplaceAll(string(respBody), "\n", `\n`))
		callback(code, responseHeaders, respBody)
	})
	if err == nil {
		log.UnsafeInfof("http call start, id: %s, cluster: %s, method: %s, url: %s, headers: %#v, body: %s, timeout: %d",
			requestID, cluster.ClusterName(), method, rawURL, headers, strings.ReplaceAll(string(body), "\n", `\n`), timeout)
	}
	return err
}
