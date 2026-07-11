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

package liteuuid

import (
	"errors"
	"testing"
)

func TestNewV4(t *testing.T) {
	const count = 10_000
	seen := make(map[UUID]struct{}, count)
	for range count {
		id, err := NewV4()
		if err != nil {
			t.Fatalf("NewV4() failed: %v", err)
		}
		if version := id[6] >> 4; version != 4 {
			t.Fatalf("version = %d, want 4", version)
		}
		if variant := id[8] & 0xc0; variant != 0x80 {
			t.Fatalf("variant bits = %#x, want %#x", variant, byte(0x80))
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate UUID: %s", id.String())
		}
		seen[id] = struct{}{}
	}
}

func TestString(t *testing.T) {
	id := UUID{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x46, 0x07, 0x88, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	const want = "00010203-0405-4607-8809-0a0b0c0d0e0f"
	if got := id.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestMust(t *testing.T) {
	want := errors.New("random source failed")
	defer func() {
		if got := recover(); got != want {
			t.Fatalf("panic = %#v, want %#v", got, want)
		}
	}()
	Must(UUID{}, want)
}
