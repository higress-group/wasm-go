package test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/stretchr/testify/require"
)

var (
	// defaultTestDomain is the default host name for the test host.
	defaultTestDomain = "default.test.com"
	// CommonVmCtx is init in wasm plugin by wrapper.SetCtx() once
	// wasmInitVMContext store the init CommonVmCtx for each go mode unit test
	wasmInitVMContext types.VMContext
	// testVMContext is the VM context for the each unit test.
	// testVMContext is wasmInitVMContext for go mode unit test
	// testVMContext is WasmVMContext wrap the wasm plugin for wasm mode unit test
	testVMContext types.VMContext
	// wasmInitMutex is the mutex for set the wasm init VM context.
	wasmInitMutex = &sync.Mutex{}
	// testMutex is the mutex for set and clear the test VM context.
	testMutex = &sync.Mutex{}
)

// getDefaultWasmPath returns the default wasm file path, supporting both environment variable
// and intelligent path detection
func getDefaultWasmPath() string {
	// Priority 1: Environment variable
	if path := os.Getenv("WASM_FILE_PATH"); path != "" {
		fmt.Printf("[WASM_PATH] Using environment variable WASM_FILE_PATH: %s\n", path)
		return path
	}

	// Priority 2: Intelligent path detection
	possiblePaths := []string{
		"main.wasm",
		"plugin.wasm",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("[WASM_PATH] Found wasm file at: %s\n", path)
			return path
		}
	}

	// Priority 3: Default fallback
	fmt.Printf("[WASM_PATH] No wasm file found, using default path: main.wasm\n")
	return "main.wasm"
}

// RunGoTest run the test in go mode, and the testVMContext will be set to the wasmInitVMContext.
// Run unit test in go mode using interface in abi_hostcalls_mock.go in proxy-wasm-go-sdk
func RunGoTest(t *testing.T, f func(*testing.T)) {
	t.Helper()
	t.Run("go", func(t *testing.T) {
		setTestVMContext(getWasmInitVMContext())
		defer clearTestVMContext()
		f(t)
	})
}

// RunWasmTestWithPath run the test in wasm mode with a specified wasm file path.
// This function allows callers to specify custom wasm file paths for testing.
func RunWasmTestWithPath(t *testing.T, wasmPath string, f func(*testing.T)) {
	t.Helper()
	t.Run("wasm", func(t *testing.T) {
		wasm, err := os.ReadFile(wasmPath)
		if err != nil {
			t.Skipf("wasm file not found at path: %s", wasmPath)
		}
		vm, err := proxytest.NewWasmVMContext(wasm)
		require.NoError(t, err)
		defer vm.Close()
		setTestVMContext(vm)
		defer clearTestVMContext()
		f(t)
	})
}

// RunWasmTest run the test in wasm mode, and the testVMContext will be set to the WasmVMContext.
// Run unit test with the compiled wasm binary helps to ensure that the plugin will run when actually compiled to wasm.
// This function maintains backward compatibility and uses intelligent path detection.
func RunWasmTest(t *testing.T, f func(*testing.T)) {
	RunWasmTestWithPath(t, getDefaultWasmPath(), f)
}

// RunTestWithPath run the test both in go and wasm mode with a specified wasm file path.
func RunTestWithPath(t *testing.T, wasmPath string, f func(*testing.T)) {
	t.Helper()

	t.Run("go", func(t *testing.T) {
		t.Log("go mode test start")
		setTestVMContext(getWasmInitVMContext())
		defer clearTestVMContext()
		f(t)
		t.Log("go mode test end")
	})

	t.Run("wasm", func(t *testing.T) {
		t.Log("wasm mode test start")
		wasm, err := os.ReadFile(wasmPath)
		if err != nil {
			t.Skipf("wasm file not found at path: %s", wasmPath)
		}
		vm, err := proxytest.NewWasmVMContext(wasm)
		require.NoError(t, err)
		defer vm.Close()
		setTestVMContext(vm)
		defer clearTestVMContext()
		f(t)
		t.Log("wasm mode test end")
	})
}

// Run unit test both in go and wasm mode.
func RunTest(t *testing.T, f func(*testing.T)) {
	RunTestWithPath(t, getDefaultWasmPath(), f)
}

// setWasmInitVMContext set the wasm init VM context.
func setWasmInitVMContext(vm types.VMContext) {
	wasmInitMutex.Lock()
	if wasmInitVMContext == nil {
		wasmInitVMContext = vm
	}
	wasmInitMutex.Unlock()
}

// getWasmInitVMContext get the wasm init VM context.
func getWasmInitVMContext() types.VMContext {
	return wasmInitVMContext
}

// setTestVMContext set the test VM context.
func setTestVMContext(vm types.VMContext) {
	testMutex.Lock()
	testVMContext = vm
}

// clearTestVMContext clear the test VM context.
func clearTestVMContext() {
	testVMContext = nil
	testMutex.Unlock()
}
