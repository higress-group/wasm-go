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

// Package litehttp provides the small HTTP data types and constants needed by
// proxy-wasm host calls without linking Go's HTTP client, server, HTTP/2, and
// TLS implementations.
package litehttp

import "net/textproto"

// Header stores HTTP header values. Its Add, Set, Get, Values, and Del methods
// have the same canonicalization behavior as net/http.Header.
type Header map[string][]string

func (h Header) Add(key, value string) {
	textproto.MIMEHeader(h).Add(key, value)
}

func (h Header) Set(key, value string) {
	textproto.MIMEHeader(h).Set(key, value)
}

func (h Header) Get(key string) string {
	return textproto.MIMEHeader(h).Get(key)
}

func (h Header) Values(key string) []string {
	return textproto.MIMEHeader(h).Values(key)
}

func (h Header) Del(key string) {
	textproto.MIMEHeader(h).Del(key)
}

// Clone returns a copy of h or nil if h is nil.
func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	clone := make(Header, len(h))
	for key, values := range h {
		if values == nil {
			clone[key] = nil
			continue
		}
		clone[key] = make([]string, len(values))
		copy(clone[key], values)
	}
	return clone
}

const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"

	StatusOK                  = 200
	StatusInternalServerError = 500
	StatusBadGateway          = 502
)
