package test

import (
	"os"
	"sync"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/proxytest"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/stretchr/testify/require"
)

var (
	// defaultTestHostName is the default host name for the test host.
	defaultTestHostName = "test.com"
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

// RunWasmTest run the test in wasm mode, and the testVMContext will be set to the WasmVMContext.
// Run unit test with the compiled wasm binary helps to ensure that the plugin will run when actually compiled to wasm.
func RunWasmTest(t *testing.T, f func(*testing.T)) {
	t.Helper()
	t.Run("wasm", func(t *testing.T) {
		wasm, err := os.ReadFile("main.wasm")
		if err != nil {
			t.Skip("wasm not found")
		}
		vm, err := proxytest.NewWasmVMContext(wasm)
		require.NoError(t, err)
		defer vm.Close()
		setTestVMContext(vm)
		defer clearTestVMContext()
		f(t)
	})
}

// Run unit test both in go and wasm mode.
func RunTest(t *testing.T, f func(*testing.T)) {
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
		wasm, err := os.ReadFile("main.wasm")
		if err != nil {
			t.Skip("wasm not found")
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
