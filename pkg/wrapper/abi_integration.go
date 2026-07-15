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

package wrapper

import (
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/internal/abi"
)

// callForeignFunction routes a foreign function call through the ABI layer.
// Foreign functions are host-specific extensions registered by the proxy-wasm host (Envoy).
//
// On TinyGo, this delegates to the ABI's TinyGoABI implementation which uses
// proxy-wasm-go-sdk's export directive conventions.
// On Go native Wasm (GOOS=wasip1), this uses the GoNativeABI implementation which
// wraps proxywasm.CallForeignFunction through go:wasmimport conventions.
//
// This function provides a unified interface that works correctly on both
// compilation targets, abstracting away the underlying ABI differences.
//
// In non-Wasm test environments (GOOS != wasip1 && !tinygo), the proxywasm
// mock may panic. This function recovers from panics to ensure graceful
// degradation during testing.
func callForeignFunction(funcName string, param []byte) (result []byte, err error) {
	// Recover from panics that may occur in non-Wasm test environments
	// where the proxywasm mock host is not properly initialized.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("callForeignFunction(%q) panicked: %v", funcName, r)
			result = nil
		}
	}()

	abiInst := abi.GetABI()
	if abiInst != nil {
		return abiInst.CallForeignFunction(funcName, param)
	}
	// Fallback: use proxywasm directly (should not normally happen after abi.Init())
	return proxywasm.CallForeignFunction(funcName, param)
}

// CallForeignFunction is the exported version for use by plugins that need
// to call host-specific foreign functions registered by Envoy.
//
// Example usage:
//
//	result, err := wrapper.CallForeignFunction("my_custom_function", paramBytes)
func CallForeignFunction(funcName string, param []byte) ([]byte, error) {
	return callForeignFunction(funcName, param)
}

// GetMemoryBuffer returns a byte slice directly mapped to the Wasm exported memory.
// This is useful for zero-copy access to data already in Wasm linear memory.
//
// On TinyGo, this returns the directly exported memory buffer.
// On Go native, this accesses memory through the wasm.Memory API.
//
// The returned slice should not be modified; use it for read-only access.
func GetMemoryBuffer() []byte {
	abiInst := abi.GetABI()
	if abiInst != nil {
		return abiInst.GetExportedMemory()
	}
	return nil
}

// ReadMemory reads bytes from Wasm linear memory at the given offset.
// Returns nil if the ABI is not initialized.
func ReadMemory(offset int, size int) []byte {
	abiInst := abi.GetABI()
	if abiInst != nil {
		return abiInst.ReadMemory(offset, size)
	}
	return nil
}

// WriteMemory writes bytes to Wasm linear memory at the given offset.
// Returns an error if the ABI is not initialized or the write fails.
func WriteMemory(offset int, data []byte) error {
	abiInst := abi.GetABI()
	if abiInst != nil {
		return abiInst.WriteMemory(offset, data)
	}
	return nil
}

// AllocMemory allocates a block of memory of the given size in the Wasm linear memory.
// Returns the offset of the allocated block, or -1 if the ABI is not initialized.
func AllocMemory(size int) int {
	abiInst := abi.GetABI()
	if abiInst != nil {
		return abiInst.AllocMemory(size)
	}
	return -1
}

// FreeMemory frees a previously allocated block of memory at the given offset.
func FreeMemory(offset int) {
	abiInst := abi.GetABI()
	if abiInst != nil {
		abiInst.FreeMemory(offset)
	}
}

// IsTinyGoBuild returns true if the code is compiled with TinyGo.
// This can be used by plugins for conditional logic based on the compilation target.
func IsTinyGoBuild() bool {
	return abi.IsTinyGoBuild()
}

// IsGoNativeBuild returns true if the code is compiled with Go native Wasm (GOOS=wasip1).
// This can be used by plugins for conditional logic based on the compilation target.
func IsGoNativeBuild() bool {
	return abi.IsGoNativeBuild()
}
