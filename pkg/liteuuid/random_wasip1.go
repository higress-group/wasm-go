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

//go:build wasip1 && wasm

package liteuuid

//go:wasmimport wasi_snapshot_preview1 random_get
//go:noescape
func wasiRandomGet(buffer *byte, length uint32) uint32

type wasiRandomError uint32

func (wasiRandomError) Error() string {
	return "wasi random_get failed"
}

func readRandom(buffer []byte) error {
	if len(buffer) == 0 {
		return nil
	}
	if errno := wasiRandomGet(&buffer[0], uint32(len(buffer))); errno != 0 {
		return wasiRandomError(errno)
	}
	return nil
}
