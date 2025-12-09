package main

import (
	"encoding/binary"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"rebuild-example",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		// Rebuild after 1000 requests
		wrapper.WithRebuildAfterRequests[RebuildConfig](1000),
		// Rebuild when memory reaches 100MB
		wrapper.WithRebuildMaxMemBytes[RebuildConfig](100*1024*1024),
	)
}

type RebuildConfig struct {
	// Configuration fields can be added here
}

func parseConfig(json gjson.Result, config *RebuildConfig) error {
	// Parse configuration if needed
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config RebuildConfig) types.Action {
	// Get VM memory size for monitoring
	data, err := proxywasm.GetProperty([]string{"plugin_vm_memory"})
	if err != nil {
		proxywasm.LogDebugf("Failed to get VM memory: %v", err)
	} else if len(data) == 8 {
		memorySize := binary.LittleEndian.Uint64([]byte(data))
		proxywasm.LogDebugf("Current VM memory usage: %d bytes (%.2f MB)",
			memorySize,
			float64(memorySize)/(1024*1024))
	}

	// Add custom header with memory info for testing
	proxywasm.AddHttpRequestHeader("X-Plugin-Processed", "true")

	return types.ActionContinue
}
