// Copyright (c) 2024 Alibaba Group Holding Ltd.
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

//go:build tinygo

package abi

import (
	"unsafe"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// isTinyGo is set to true when compiled with TinyGo.
const isTinyGo = true

// TinyGoABI implements the WasmABI interface for TinyGo compilation target.
// It uses the proxy-wasm-go-sdk directly, which is designed for TinyGo's
// Wasm ABI conventions (export directives, direct memory access, etc.)
type TinyGoABI struct{}

// newABI creates a new TinyGoABI instance.
// This function is only compiled when building with TinyGo.
func newABI() WasmABI {
	return &TinyGoABI{}
}

func (a *TinyGoABI) Name() string {
	return "tinygo"
}

func (a *TinyGoABI) IsTinyGo() bool {
	return true
}

func (a *TinyGoABI) IsGoNative() bool {
	return false
}

// AllocMemory allocates memory using TinyGo's allocator.
// In TinyGo, memory is managed by the runtime's built-in allocator,
// and the Wasm linear memory is directly accessible.
func (a *TinyGoABI) AllocMemory(size int) int {
	buf := make([]byte, size)
	return int(uintptr(unsafe.Pointer(&buf[0])))
}

// FreeMemory is a no-op in TinyGo since memory is managed by the GC.
func (a *TinyGoABI) FreeMemory(offset int) {
	// TinyGo uses garbage collection, so we don't need to manually free.
	// The memory will be reclaimed when no references remain.
}

// ReadMemory reads bytes from the Wasm linear memory at the given offset.
// In TinyGo, the linear memory is directly accessible via unsafe pointers.
func (a *TinyGoABI) ReadMemory(offset int, size int) []byte {
	ptr := unsafe.Pointer(uintptr(offset))
	buf := make([]byte, size)
	for i := 0; i < size; i++ {
		buf[i] = *(*byte)(unsafe.Pointer(uintptr(offset) + uintptr(i)))
	}
	_ = ptr // suppress unused variable warning
	return buf
}

// WriteMemory writes bytes to the Wasm linear memory at the given offset.
// In TinyGo, the linear memory is directly writable via unsafe pointers.
func (a *TinyGoABI) WriteMemory(offset int, data []byte) error {
	for i, b := range data {
		*(*byte)(unsafe.Pointer(uintptr(offset) + uintptr(i))) = b
	}
	return nil
}

// CallForeignFunction calls a proxy-wasm foreign function using the TinyGo ABI convention.
// In TinyGo, foreign function calls go through the proxy-wasm-go-sdk which uses
// the proxy_call_foreign_function FFI under the hood.
// The SDK handles the ABI differences between TinyGo and Go native Wasm automatically
// via build tags (//go:build wasm vs //go:build !wasm).
func (a *TinyGoABI) CallForeignFunction(funcName string, param []byte) ([]byte, error) {
	return proxywasm.CallForeignFunction(funcName, param)
}

// GetExportedMemory returns the exported Wasm memory buffer.
// In TinyGo, memory is directly exported from the Wasm module.
func (a *TinyGoABI) GetExportedMemory() []byte {
	// In TinyGo, the memory buffer is directly accessible.
	// The proxy-wasm-go-sdk manages memory access internally.
	return nil
}