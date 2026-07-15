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

// Package abi provides the Wasm ABI (Application Binary Interface) adaptation layer
// that abstracts differences between TinyGo and Go 1.24 native Wasm (wasip1) compilation.
//
// The ABI layer handles:
//   - Memory access patterns (direct export vs wasm.Memory)
//   - Host function calling conventions (_wasm_call_host vs go:wasmimport)
//   - Export function registration (export directive vs go:wasmexport)
//   - Memory allocation strategies (custom allocator vs Go runtime)
package abi

// WasmABI defines the Wasm ABI adaptation interface that abstracts
// differences between TinyGo and Go native Wasm compilation targets.
type WasmABI interface {
	// Name returns the name of the ABI implementation ("tinygo" or "go_native").
	Name() string

	// IsTinyGo returns true if the current compilation target is TinyGo.
	IsTinyGo() bool

	// IsGoNative returns true if the current compilation target is Go native Wasm (wasip1).
	IsGoNative() bool

	// AllocMemory allocates a block of memory of the given size in the Wasm linear memory.
	// Returns the offset of the allocated block.
	AllocMemory(size int) int

	// FreeMemory frees a previously allocated block of memory at the given offset.
	FreeMemory(offset int)

	// ReadMemory reads bytes from the Wasm linear memory starting at the given offset.
	ReadMemory(offset int, size int) []byte

	// WriteMemory writes bytes to the Wasm linear memory starting at the given offset.
	WriteMemory(offset int, data []byte) error

	// CallForeignFunction calls a proxy-wasm foreign function by name with the given parameter.
	// Foreign functions are host-specific extensions registered by the proxy-wasm host (Envoy).
	// This wraps proxywasm.CallForeignFunction which uses proxy_call_foreign_function FFI.
	// Returns the result bytes from the foreign function call.
	CallForeignFunction(funcName string, param []byte) ([]byte, error)

	// GetExportedMemory returns the exported Wasm memory buffer.
	// In TinyGo, this is the directly exported memory.
	// In Go native, this accesses memory through the wasm.Memory API.
	GetExportedMemory() []byte
}

// abiInstance is the singleton ABI instance, initialized at startup.
var abiInstance WasmABI

// Init initializes the ABI layer based on the compilation target.
// This must be called once at program startup before any ABI operations.
func Init() {
	abiInstance = newABI()
}

// GetABI returns the current ABI instance.
// Panics if Init() has not been called.
func GetABI() WasmABI {
	if abiInstance == nil {
		Init()
	}
	return abiInstance
}

// IsTinyGoBuild returns true if the code is being compiled with TinyGo.
// This is determined at compile time using build tags.
func IsTinyGoBuild() bool {
	return isTinyGo
}

// IsGoNativeBuild returns true if the code is being compiled with Go native Wasm (GOOS=wasip1).
// This is determined at compile time using build tags.
func IsGoNativeBuild() bool {
	return !isTinyGo
}