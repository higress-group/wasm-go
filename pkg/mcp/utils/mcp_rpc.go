// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

package utils

import (
	"encoding/base64"
	"fmt"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func OnMCPResponseSuccess(ctx wrapper.HttpContext, result map[string]any, debugInfo string) {
	OnJsonRpcResponseSuccess(ctx, result, debugInfo)
	// TODO: support pub to redis when use POST + SSE
}

func OnMCPResponseError(ctx wrapper.HttpContext, err error, code int, debugInfo string) {
	OnJsonRpcResponseError(ctx, err, code, debugInfo)
	// TODO: support pub to redis when use POST + SSE
}

func OnMCPToolCallSuccess(ctx wrapper.HttpContext, content []map[string]any, debugInfo string) {
	OnMCPResponseSuccess(ctx, map[string]any{
		"content": content,
		"isError": false,
	}, debugInfo)
}

// OnMCPToolCallSuccessWithStructuredData sends a successful MCP tool response with structured data
// (MCP Protocol Version 2025-06-18)
func OnMCPToolCallSuccessWithStructuredData(ctx wrapper.HttpContext, content []map[string]any, structuredData map[string]any, debugInfo string) {
	response := map[string]any{
		"content": content,
		"isError": false,
	}
	if structuredData != nil && len(structuredData) > 0 {
		response["structuredData"] = structuredData
	}
	OnMCPResponseSuccess(ctx, response, debugInfo)
}

func OnMCPToolCallError(ctx wrapper.HttpContext, err error, debugInfo ...string) {
	responseDebugInfo := fmt.Sprintf("mcp:tools/call:error(%s)", err)
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	OnMCPResponseSuccess(ctx, map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": err.Error(),
			},
		},
		"isError": true,
	}, responseDebugInfo)
}

func SendMCPToolTextResult(ctx wrapper.HttpContext, result string, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	OnMCPToolCallSuccess(ctx, []map[string]any{
		{
			"type": "text",
			"text": result,
		},
	}, responseDebugInfo)
}

func SendMCPToolImageResult(ctx wrapper.HttpContext, image []byte, contentType string, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}

	content := []map[string]any{
		{
			"type":     "image",
			"data":     base64.StdEncoding.EncodeToString(image),
			"mimeType": contentType,
		},
	}

	// Check protocol version for automatic format selection
	protocolVersion := ctx.GetStringContext("MCP_PROTOCOL_VERSION", "")
	if protocolVersion == "2025-06-18" {
		// For 2025-06-18, we could include structured data if needed
		// For now, just use the enhanced response format (ready for future extensions)
		OnMCPToolCallSuccessWithStructuredData(ctx, content, nil, responseDebugInfo)
	} else {
		// For older versions, use traditional response
		OnMCPToolCallSuccess(ctx, content, responseDebugInfo)
	}
}

// SendMCPToolImageWithStructuredResult sends an image result with structured data
// (MCP Protocol Version 2025-06-18)
func SendMCPToolImageWithStructuredResult(ctx wrapper.HttpContext, image []byte, contentType string, structuredData map[string]any, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}

	content := []map[string]any{
		{
			"type":     "image",
			"data":     base64.StdEncoding.EncodeToString(image),
			"mimeType": contentType,
		},
	}

	// Check protocol version for automatic format selection
	protocolVersion := ctx.GetStringContext("MCP_PROTOCOL_VERSION", "")
	if protocolVersion == "2025-06-18" && structuredData != nil && len(structuredData) > 0 {
		OnMCPToolCallSuccessWithStructuredData(ctx, content, structuredData, responseDebugInfo)
	} else {
		// For older versions or when no structured data, use traditional response
		OnMCPToolCallSuccess(ctx, content, responseDebugInfo)
	}
}

// SendMCPToolStructuredResult sends a tool result with both text content and structured data
// (MCP Protocol Version 2025-06-18)
func SendMCPToolStructuredResult(ctx wrapper.HttpContext, result string, structuredData map[string]any, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	content := []map[string]any{
		{
			"type": "text",
			"text": result,
		},
	}
	OnMCPToolCallSuccessWithStructuredData(ctx, content, structuredData, responseDebugInfo)
}

// SendMCPToolStructuredOnlyResult sends a tool result with only structured data
// (MCP Protocol Version 2025-06-18)
func SendMCPToolStructuredOnlyResult(ctx wrapper.HttpContext, structuredData map[string]any, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}
	OnMCPToolCallSuccessWithStructuredData(ctx, []map[string]any{}, structuredData, responseDebugInfo)
}

// SendMCPToolResult automatically chooses the appropriate response format based on protocol version
// This is the recommended function to use for sending tool results
func SendMCPToolResult(ctx wrapper.HttpContext, textResult string, structuredData map[string]any, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}

	// Check protocol version stored during initialization
	protocolVersion := ctx.GetStringContext("MCP_PROTOCOL_VERSION", "")

	// For protocol version 2025-06-18 and later, include structured data if provided
	if protocolVersion == "2025-06-18" && structuredData != nil && len(structuredData) > 0 {
		content := []map[string]any{
			{
				"type": "text",
				"text": textResult,
			},
		}
		OnMCPToolCallSuccessWithStructuredData(ctx, content, structuredData, responseDebugInfo)
	} else {
		// For older versions or when no structured data, use traditional text response
		SendMCPToolTextResult(ctx, textResult, debugInfo...)
	}
}

// SendMCPToolResultWithContent automatically chooses the appropriate response format
// and allows custom content array (for images, etc.)
func SendMCPToolResultWithContent(ctx wrapper.HttpContext, content []map[string]any, structuredData map[string]any, debugInfo ...string) {
	responseDebugInfo := "mcp:tools/call::result"
	if len(debugInfo) > 0 {
		responseDebugInfo = debugInfo[0]
	}

	// Check protocol version stored during initialization
	protocolVersion := ctx.GetStringContext("MCP_PROTOCOL_VERSION", "")

	// For protocol version 2025-06-18 and later, include structured data if provided
	if protocolVersion == "2025-06-18" && structuredData != nil && len(structuredData) > 0 {
		OnMCPToolCallSuccessWithStructuredData(ctx, content, structuredData, responseDebugInfo)
	} else {
		// For older versions or when no structured data, use traditional response
		OnMCPToolCallSuccess(ctx, content, responseDebugInfo)
	}
}
