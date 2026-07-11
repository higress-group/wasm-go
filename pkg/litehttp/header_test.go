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

package litehttp

import "testing"

func TestHeaderSemantics(t *testing.T) {
	header := make(Header)
	header.Add("content-type", "application/json")
	header.Add("Content-Type", "text/plain")

	if got := header.Get("CONTENT-TYPE"); got != "application/json" {
		t.Fatalf("Get() = %q, want %q", got, "application/json")
	}
	values := header.Values("content-type")
	if len(values) != 2 || values[0] != "application/json" || values[1] != "text/plain" {
		t.Fatalf("Values() = %#v", values)
	}

	header.Set("content-type", "application/protobuf")
	if got := header.Get("Content-Type"); got != "application/protobuf" {
		t.Fatalf("Set()/Get() = %q", got)
	}

	clone := header.Clone()
	clone.Add("Content-Type", "application/xml")
	if got := len(header.Values("Content-Type")); got != 1 {
		t.Fatalf("Clone() shares values with source: source has %d values", got)
	}

	header.Del("CONTENT-TYPE")
	if got := header.Get("Content-Type"); got != "" {
		t.Fatalf("Del() left value %q", got)
	}
}

func TestNilHeaderClone(t *testing.T) {
	var header Header
	if clone := header.Clone(); clone != nil {
		t.Fatalf("nil Header.Clone() = %#v, want nil", clone)
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "empty", want: "text/plain; charset=utf-8"},
		{name: "binary", data: []byte{1, 2, 3}, want: "application/octet-stream"},
		{name: "html", data: []byte("  <!DOCTYPE HTML>"), want: "text/html; charset=utf-8"},
		{name: "png", data: []byte("\x89PNG\r\n\x1a\n"), want: "image/png"},
		{name: "jpeg", data: []byte("\xff\xd8\xff"), want: "image/jpeg"},
		{name: "pdf", data: []byte("%PDF-"), want: "application/pdf"},
		{name: "wasm", data: []byte("\x00asm\x01\x00"), want: "application/wasm"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DetectContentType(test.data); got != test.want {
				t.Fatalf("DetectContentType() = %q, want %q", got, test.want)
			}
		})
	}
}
