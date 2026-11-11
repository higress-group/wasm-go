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

// Package liteuuid generates RFC 9562 UUID version 4 identifiers without
// pulling a general-purpose UUID implementation into proxy-wasm plugins.
// On wasip1 it reads entropy directly from wasi_snapshot_preview1.random_get;
// hosts that restrict WASI capabilities must allow random_get.
//
// This package is intended for identifiers, not key material. Its wasip1
// implementation bypasses crypto/rand's optional Go FIPS DRBG; callers that
// require that validated path should continue to use crypto/rand.
package liteuuid

// Size is the number of bytes in a UUID.
const Size = 16

const hexTable = "0123456789abcdef"

// UUID is a 128-bit universally unique identifier.
type UUID [Size]byte

// NewV4 returns a UUID version 4 generated from the platform's cryptographic
// random source.
func NewV4() (UUID, error) {
	var id UUID
	if err := readRandom(id[:]); err != nil {
		return UUID{}, err
	}

	// RFC 9562, section 5.4: version 4 and the RFC variant consume six bits,
	// leaving 122 random bits.
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return id, nil
}

// New returns a UUID version 4. It panics if the platform random source fails.
func New() UUID {
	return Must(NewV4())
}

// Must returns id if err is nil and panics otherwise.
func Must(id UUID, err error) UUID {
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the canonical 8-4-4-4-12 lowercase representation of id.
func (id UUID) String() string {
	var text [36]byte
	textIndex := 0
	for byteIndex, value := range id {
		switch byteIndex {
		case 4, 6, 8, 10:
			text[textIndex] = '-'
			textIndex++
		}
		text[textIndex] = hexTable[value>>4]
		text[textIndex+1] = hexTable[value&0x0f]
		textIndex += 2
	}
	return string(text[:])
}
