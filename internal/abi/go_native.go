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

//go:build !tinygo

package abi

import (
	"fmt"
	"sync"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// isTinyGo is set to false when not compiled with TinyGo (i.e., Go native Wasm).
const isTinyGo = false

// GoNativeABI implements the WasmABI interface for Go 1.24+ native Wasm (wasip1)
// compilation target. It uses Go's standard memory management and the go:wasmimport
// convention for host function calls.
type GoNativeABI struct {
	mu       sync.Mutex
	memory   []byte
	allocs   map[int]int // offset -> size mapping for tracking allocations
	nextOff  int
	memSize  int
}

// newABI creates a new GoNativeABI instance.
// This function is only compiled when building with Go native Wasm (GOOS=wasip1).
func newABI() WasmABI {
	initialSize := 1024 * 1024 // 1MB initial memory
	return &GoNativeABI{
		memory:  make([]byte, initialSize),
		allocs:  make(map[int]int),
		nextOff: 0,
		memSize: initialSize,
	}
}

func (a *GoNativeABI) Name() string {
	return "go_native"
}

func (a *GoNativeABI) IsTinyGo() bool {
	return false
}

func (a *GoNativeABI) IsGoNative() bool {
	return true
}

// AllocMemory allocates a block of memory in the managed memory pool.
// In Go native Wasm, memory is managed by the Go runtime's garbage collector.
// This implementation uses a simple bump allocator with a free list for
// compatibility with the proxy-wasm host interface.
func (a *GoNativeABI) AllocMemory(size int) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Align to 8-byte boundary for proper memory alignment
	alignedSize := (size + 7) &^ 7

	// Check if we need to grow the memory
	if a.nextOff+alignedSize > a.memSize {
		newSize := a.memSize * 2
		for a.nextOff+alignedSize > newSize {
			newSize *= 2
		}
		newMem := make([]byte, newSize)
		copy(newMem, a.memory)
		a.memory = newMem
		a.memSize = newSize
	}

	offset := a.nextOff
	a.nextOff += alignedSize
	a.allocs[offset] = alignedSize
	return offset
}

// FreeMemory marks a previously allocated block as freed.
// In Go native Wasm, the actual memory is managed by the Go runtime,
// so this just removes the allocation tracking entry.
func (a *GoNativeABI) FreeMemory(offset int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.allocs, offset)
}

// ReadMemory reads bytes from the managed memory pool at the given offset.
// In Go native Wasm, memory is accessed through the Go slice directly,
// avoiding the need for unsafe pointer operations.
func (a *GoNativeABI) ReadMemory(offset int, size int) []byte {
	a.mu.Lock()
	defer a.mu.Unlock()

	if offset < 0 || offset+size > len(a.memory) {
		return nil
	}

	buf := make([]byte, size)
	copy(buf, a.memory[offset:offset+size])
	return buf
}

// WriteMemory writes bytes to the managed memory pool at the given offset.
// In Go native Wasm, memory is written through the Go slice directly,
// avoiding the need for unsafe pointer operations.
func (a *GoNativeABI) WriteMemory(offset int, data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if offset < 0 || offset+len(data) > len(a.memory) {
		return fmt.Errorf("memory write out of bounds: offset=%d, size=%d, memSize=%d",
			offset, len(data), len(a.memory))
	}

	copy(a.memory[offset:], data)
	return nil
}

// CallForeignFunction calls a proxy-wasm foreign function using the Go native ABI convention.
// In Go native Wasm (wasip1), the proxy-wasm-go-sdk handles foreign function calls through
// the go:wasmimport directive (when compiled with //go:build wasm) or through the mock host
// (when compiled with //go:build !wasm for testing).
//
// The SDK's internal.ProxyCallForeignFunction is wrapped by proxywasm.CallForeignFunction,
// which handles the ABI differences automatically via build tags:
//   - Wasm build: uses //go:wasmimport env proxy_call_foreign_function
//   - Non-Wasm build: delegates to the registered mock host
func (a *GoNativeABI) CallForeignFunction(funcName string, param []byte) ([]byte, error) {
	return proxywasm.CallForeignFunction(funcName, param)
}

// GetExportedMemory returns the managed memory buffer.
// In Go native Wasm, memory is managed by the Go runtime and accessed
// through the internal memory pool rather than as a directly exported Wasm memory.
func (a *GoNativeABI) GetExportedMemory() []byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.memory
}