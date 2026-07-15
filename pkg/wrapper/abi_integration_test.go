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
	"testing"

	"github.com/higress-group/wasm-go/internal/abi"
)

func init() {
	// Ensure ABI is initialized before tests run
	abi.Init()
}

// TestCallForeignFunction ensures the ABI-routed foreign function call does not panic.
// In non-Wasm test environments, the proxywasm mock may panic; the ABI integration
// should recover from panics and return an error instead.
func TestCallForeignFunction(t *testing.T) {
	result, err := callForeignFunction("get_log_level", nil)
	if err != nil {
		t.Logf("callForeignFunction returned error (expected in non-Wasm test env): %v", err)
	} else {
		t.Logf("callForeignFunction returned: %v (len=%d)", result, len(result))
	}
	// The test passes as long as we don't panic - either result is acceptable
}

// TestCallForeignFunctionWithParam tests ABI-routed foreign function with parameters.
func TestCallForeignFunctionWithParam(t *testing.T) {
	param := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	result, err := callForeignFunction("set_global_max_requests_per_io_cycle", param)
	if err != nil {
		t.Logf("callForeignFunction with param returned error (expected in non-Wasm test env): %v", err)
	} else {
		t.Logf("callForeignFunction with param returned: %v", result)
	}
}

// TestCallForeignFunctionExported ensures the public API does not panic.
func TestCallForeignFunctionExported(t *testing.T) {
	result, err := CallForeignFunction("get_log_level", nil)
	if err != nil {
		t.Logf("CallForeignFunction returned error (expected in non-Wasm test env): %v", err)
	} else {
		t.Logf("CallForeignFunction returned: %v", result)
	}
}

// TestGetMemoryBuffer verifies that GetMemoryBuffer doesn't panic.
func TestGetMemoryBuffer(t *testing.T) {
	buf := GetMemoryBuffer()
	if buf != nil {
		t.Logf("GetMemoryBuffer returned buffer of length: %d", len(buf))
	} else {
		t.Log("GetMemoryBuffer returned nil (expected in mock env)")
	}
}

// TestReadWriteMemoryIntegration tests memory read/write through the ABI layer.
func TestReadWriteMemoryIntegration(t *testing.T) {
	// Allocate a small buffer
	offset := AllocMemory(256)
	if offset < 0 {
		t.Skip("AllocMemory returned -1, skipping memory tests (ABI not initialized or no linear memory)")
	}
	defer FreeMemory(offset)

	// Write data
	testData := []byte("Hello, Wasm!")
	err := WriteMemory(offset, testData)
	if err != nil {
		t.Fatalf("WriteMemory failed: %v", err)
	}

	// Read it back
	readData := ReadMemory(offset, len(testData))
	if readData == nil {
		t.Fatal("ReadMemory returned nil")
	}
	if string(readData) != string(testData) {
		t.Fatalf("ReadMemory mismatch: got %q, want %q", string(readData), string(testData))
	}
	t.Logf("Memory read/write round-trip successful: %q", string(readData))
}

// TestIsTinyGoBuild verifies build detection functions work.
func TestIsTinyGoBuild(t *testing.T) {
	// On Go 1.24+ native builds, IsTinyGoBuild should return false
	// On TinyGo, it should return true
	tinyGo := IsTinyGoBuild()
	goNative := IsGoNativeBuild()
	t.Logf("IsTinyGoBuild=%v, IsGoNativeBuild=%v", tinyGo, goNative)
	// These should be mutually exclusive
	if tinyGo && goNative {
		t.Error("Both IsTinyGoBuild and IsGoNativeBuild returned true - this is a bug")
	}
}
