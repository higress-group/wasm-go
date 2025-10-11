// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/mcp/utils"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	// Context keys for MCP proxy state management
	CtxMcpProxyInitialized = "mcp_proxy_initialized"
	CtxMcpProxySessionID   = "mcp_proxy_session_id"
	CtxMcpProxyToolName    = "mcp_proxy_tool_name"
	CtxMcpProxyToolArgs    = "mcp_proxy_tool_args"
	CtxMcpProxyOperation   = "mcp_proxy_operation"
)

// ProxyAuthInfo holds authentication information for proxy tool calls
type ProxyAuthInfo struct {
	SecuritySchemeID      string          // RequestTemplate.Security.ID for gateway-to-backend auth
	PassthroughCredential string          // Credential extracted from client request (if passthrough enabled)
	Server                *McpProxyServer // Server instance for accessing security schemes
}

// McpProxyOperation represents the current operation type
type McpProxyOperation string

const (
	OpToolsList McpProxyOperation = "tools/list"
	OpToolsCall McpProxyOperation = "tools/call"
)

// McpProtocolHandler handles MCP protocol initialization and communication
type McpProtocolHandler struct {
	backendURL string
	timeout    int
	sessionID  string
}

// NewMcpProtocolHandler creates a new MCP protocol handler
func NewMcpProtocolHandler(backendURL string, timeout int) *McpProtocolHandler {
	return &McpProtocolHandler{
		backendURL: backendURL,
		timeout:    timeout,
	}
}

// Initialize performs the MCP protocol initialization sequence asynchronously
func (h *McpProtocolHandler) Initialize(ctx wrapper.HttpContext, authInfo *ProxyAuthInfo) error {
	log.Infof("Starting MCP protocol initialization for %s", h.backendURL)

	// Check if already initialized for this context
	if initialized := ctx.GetContext(CtxMcpProxyInitialized); initialized != nil {
		if sessionID := ctx.GetContext(CtxMcpProxySessionID); sessionID != nil {
			h.sessionID = sessionID.(string)
			log.Debugf("MCP proxy already initialized with session ID: %s", h.sessionID)
			return nil
		}
	}

	// Step 1: Send initialize request
	initRequest := h.createInitializeRequest()
	requestBody, err := json.Marshal(initRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize request: %v", err)
	}

	// Send initialize request to backend asynchronously
	err = h.sendMcpRequest(ctx, requestBody, authInfo, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
		// Don't resume here - either OnMCPResponseError will send response directly,
		// or sendInitializedNotification will continue the async flow
		if statusCode != 200 {
			log.Errorf("Initialize request failed with status %d: %s", statusCode, string(responseBody))
			utils.OnMCPResponseError(ctx, fmt.Errorf("backend initialization failed"), utils.ErrInternalError, "mcp-proxy:initialize:backend_error")
			return
		}

		// Parse initialize response
		var response map[string]interface{}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			log.Errorf("Failed to parse initialize response: %v", err)
			utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:initialize:parse_error")
			return
		}

		// Check for protocol version compatibility
		if errorObj, exists := response["error"]; exists {
			log.Errorf("Backend initialization error: %v", errorObj)

			// Check if it's a version compatibility error
			if errorMap, ok := errorObj.(map[string]interface{}); ok {
				if code, codeOk := errorMap["code"]; codeOk && code == -32602 {
					// Protocol version not supported
					utils.OnMCPResponseError(ctx, fmt.Errorf("protocol version not supported by backend"), utils.ErrInvalidParams, "mcp-proxy:initialize:version_incompatible")
					return
				}
			}

			utils.OnMCPResponseError(ctx, fmt.Errorf("backend initialization failed"), utils.ErrInternalError, "mcp-proxy:initialize:backend_error")
			return
		}

		// Extract session ID from response headers if present
		for _, header := range responseHeaders {
			if header[0] == "Mcp-Session-Id" {
				h.sessionID = header[1]
				ctx.SetContext(CtxMcpProxySessionID, h.sessionID)
				log.Infof("Received MCP session ID: %s", h.sessionID)
				break
			}
		}

		// Step 2: Send notifications/initialized
		h.sendInitializedNotification(ctx, authInfo)
	})

	return err
}

// ForwardToolsList forwards tools/list request to backend MCP server
func (h *McpProtocolHandler) ForwardToolsList(ctx wrapper.HttpContext, cursor *string, authInfo *ProxyAuthInfo) error {
	log.Debugf("Forwarding tools/list request to %s", h.backendURL)

	// Store the cursor for later execution
	ctx.SetContext(CtxMcpProxyOperation, OpToolsList)
	if cursor != nil {
		ctx.SetContext("mcp_proxy_cursor", *cursor)
	}
	if authInfo != nil {
		ctx.SetContext("mcp_proxy_auth_info", authInfo)
	}

	// Check if MCP is already initialized
	if initialized := ctx.GetContext(CtxMcpProxyInitialized); initialized != nil {
		// Already initialized, execute directly
		return h.executeToolsList(ctx)
	}

	// Need to initialize first, which will execute tools/list in its callback
	return h.Initialize(ctx, authInfo)
}

// executeToolsList executes the actual tools/list request
func (h *McpProtocolHandler) executeToolsList(ctx wrapper.HttpContext) error {
	var cursor *string
	if cursorVal := ctx.GetContext("mcp_proxy_cursor"); cursorVal != nil {
		cursorStr := cursorVal.(string)
		cursor = &cursorStr
	}

	listRequest := h.createToolsListRequest(cursor)
	requestBody, err := json.Marshal(listRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal tools/list request: %v", err)
	}

	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	// Add session ID if we have one
	if h.sessionID != "" {
		headers = append(headers, [2]string{"Mcp-Session-Id", h.sessionID})
	}

	// Start with the original backend URL
	finalURL := h.backendURL

	// Apply authentication if auth info was provided
	if authInfoCtx := ctx.GetContext("mcp_proxy_auth_info"); authInfoCtx != nil {
		if authInfo, ok := authInfoCtx.(*ProxyAuthInfo); ok && authInfo.SecuritySchemeID != "" {
			// Apply authentication using shared utilities
			modifiedURL, err := h.applyProxyAuthentication(authInfo.Server, authInfo.SecuritySchemeID, authInfo.PassthroughCredential, &headers)
			if err != nil {
				log.Errorf("Failed to apply authentication for tools/list request: %v", err)
			} else {
				// Use the modified URL if authentication was applied successfully
				finalURL = modifiedURL
				log.Debugf("Using modified URL for tools/list request: %s", finalURL)
			}
		}
	}

	// Use RouteCall for the final tools/list request with potentially modified URL
	return ctx.RouteCall("POST", finalURL, headers, requestBody, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
		if statusCode != 200 {
			log.Errorf("Tools/list request failed with status %d: %s", statusCode, string(responseBody))
			utils.OnMCPResponseError(ctx, fmt.Errorf("backend tools/list failed"), utils.ErrInternalError, "mcp-proxy:tools/list:backend_error")
			return
		}

		// Parse response and forward to client
		var response map[string]interface{}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			log.Errorf("Failed to parse tools/list response: %v", err)
			utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:tools/list:parse_error")
			return
		}

		// Forward the tools/list result with allowTools filtering
		if result, hasResult := response["result"]; hasResult {
			if resultMap, ok := result.(map[string]interface{}); ok {
				// Apply allowTools filtering if needed
				filteredResult := h.applyAllowToolsFilter(ctx, resultMap)
				utils.OnMCPResponseSuccess(ctx, filteredResult, "mcp-proxy:tools/list:success")
			} else {
				utils.OnMCPResponseError(ctx, fmt.Errorf("invalid tools/list result type"), utils.ErrInternalError, "mcp-proxy:tools/list:invalid_type")
			}
		} else {
			utils.OnMCPResponseError(ctx, fmt.Errorf("invalid tools/list response"), utils.ErrInternalError, "mcp-proxy:tools/list:invalid_response")
		}
	})
}

// ForwardToolsCall forwards tools/call request to backend MCP server
func (h *McpProtocolHandler) ForwardToolsCall(ctx wrapper.HttpContext, toolName string, arguments map[string]interface{}, authInfo *ProxyAuthInfo) error {
	log.Debugf("Forwarding tools/call request for tool %s to %s", toolName, h.backendURL)

	// Store the tool call parameters for later execution
	ctx.SetContext(CtxMcpProxyOperation, OpToolsCall)
	ctx.SetContext(CtxMcpProxyToolName, toolName)
	ctx.SetContext(CtxMcpProxyToolArgs, arguments)
	if authInfo != nil {
		ctx.SetContext("mcp_proxy_auth_info", authInfo)
	}

	// Check if MCP is already initialized
	if initialized := ctx.GetContext(CtxMcpProxyInitialized); initialized != nil {
		// Already initialized, execute directly
		return h.executeToolsCall(ctx)
	}

	// Need to initialize first, which will execute tools/call in its callback
	return h.Initialize(ctx, authInfo)
}

// executeToolsCall executes the actual tools/call request
func (h *McpProtocolHandler) executeToolsCall(ctx wrapper.HttpContext) error {
	toolName := ctx.GetContext(CtxMcpProxyToolName).(string)
	arguments := ctx.GetContext(CtxMcpProxyToolArgs).(map[string]interface{})

	callRequest := h.createToolsCallRequest(toolName, arguments)
	requestBody, err := json.Marshal(callRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal tools/call request: %v", err)
	}

	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	// Add session ID if we have one
	if h.sessionID != "" {
		headers = append(headers, [2]string{"Mcp-Session-Id", h.sessionID})
	}

	// Start with the original backend URL
	finalURL := h.backendURL

	// Apply authentication if auth info was provided
	if authInfoCtx := ctx.GetContext("mcp_proxy_auth_info"); authInfoCtx != nil {
		if authInfo, ok := authInfoCtx.(*ProxyAuthInfo); ok && authInfo.SecuritySchemeID != "" {
			// Apply authentication using shared utilities
			modifiedURL, err := h.applyProxyAuthentication(authInfo.Server, authInfo.SecuritySchemeID, authInfo.PassthroughCredential, &headers)
			if err != nil {
				log.Errorf("Failed to apply authentication for proxy tool call: %v", err)
			} else {
				// Use the modified URL if authentication was applied successfully
				finalURL = modifiedURL
				log.Debugf("Using modified URL for tools/call request: %s", finalURL)
			}
		}
	}

	// Use RouteCall for the final tools/call request with potentially modified URL
	return ctx.RouteCall("POST", finalURL, headers, requestBody, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
		if statusCode != 200 {
			log.Errorf("Tools/call request failed with status %d: %s", statusCode, string(responseBody))
			utils.OnMCPResponseError(ctx, fmt.Errorf("backend tools/call failed"), utils.ErrInternalError, "mcp-proxy:tools/call:backend_error")
			return
		}

		// Parse response to check for backend errors
		var callResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &callResponse); err == nil {
			if result, hasResult := callResponse["result"]; hasResult {
				if resultMap, ok := result.(map[string]interface{}); ok {
					if isError, hasIsError := resultMap["isError"]; hasIsError && isError == true {
						// Backend reported an error through isError flag
						log.Warnf("Backend reported tool call error for %s", toolName)
						// Still forward the response but with source attribution
						h.wrapBackendError(responseBody, ctx)
						return
					}
				}
			}
		}

		// Parse response and forward to client
		var finalResponse map[string]interface{}
		if err := json.Unmarshal(responseBody, &finalResponse); err != nil {
			log.Errorf("Failed to parse tools/call response: %v", err)
			utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:tools/call:parse_error")
			return
		}

		// Forward the tools/call result
		if result, hasResult := finalResponse["result"]; hasResult {
			if resultMap, ok := result.(map[string]interface{}); ok {
				utils.OnMCPResponseSuccess(ctx, resultMap, "mcp-proxy:tools/call:success")
			} else {
				utils.OnMCPResponseError(ctx, fmt.Errorf("invalid tools/call result type"), utils.ErrInternalError, "mcp-proxy:tools/call:invalid_type")
			}
		} else {
			utils.OnMCPResponseError(ctx, fmt.Errorf("invalid tools/call response"), utils.ErrInternalError, "mcp-proxy:tools/call:invalid_response")
		}
	})
}

// sendMcpRequest sends an MCP request to the backend server using POST method
func (h *McpProtocolHandler) sendMcpRequest(ctx wrapper.HttpContext, body []byte, authInfo *ProxyAuthInfo, callback func(int, [][2]string, []byte)) error {
	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	// Add session ID if we have one
	if h.sessionID != "" {
		headers = append(headers, [2]string{"Mcp-Session-Id", h.sessionID})
	}

	// Start with the original backend URL
	finalURL := h.backendURL

	// Apply authentication if auth info was provided
	if authInfo != nil && authInfo.SecuritySchemeID != "" {
		modifiedURL, err := h.applyProxyAuthentication(authInfo.Server, authInfo.SecuritySchemeID, authInfo.PassthroughCredential, &headers)
		if err != nil {
			log.Errorf("Failed to apply authentication for MCP request: %v", err)
		} else {
			// Use the modified URL if authentication was applied successfully
			finalURL = modifiedURL
			log.Debugf("Using modified URL for MCP request: %s", finalURL)
		}
	}

	// Determine timeout
	timeout := uint32(h.timeout)
	if timeout == 0 {
		timeout = 5000 // Default 5 seconds
	}

	// Create HTTP client using RouteCluster
	client := wrapper.NewClusterClient(wrapper.RouteCluster{})

	// Convert callback to the expected format
	wrappedCallback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		// Convert http.Header to [][2]string format
		headerSlice := make([][2]string, 0, len(responseHeaders))
		for key, values := range responseHeaders {
			if len(values) > 0 {
				headerSlice = append(headerSlice, [2]string{key, values[0]})
			}
		}
		callback(statusCode, headerSlice, responseBody)
	}

	// All MCP requests use POST method with potentially modified URL
	return client.Post(finalURL, headers, body, wrappedCallback, timeout)
}

// createInitializeRequest creates an MCP initialize request
func (h *McpProtocolHandler) createInitializeRequest() map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "Higress-mcp-proxy",
				"version": "1.0.0",
			},
		},
	}
}

// sendInitializedNotification sends the notifications/initialized message
func (h *McpProtocolHandler) sendInitializedNotification(ctx wrapper.HttpContext, authInfo *ProxyAuthInfo) {
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}

	requestBody, err := json.Marshal(notification)
	if err != nil {
		log.Errorf("Failed to marshal initialized notification: %v", err)
		utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:notifications/initialized:marshal_error")
		return
	}

	// Send the notification (no response expected)
	err = h.sendMcpRequest(ctx, requestBody, authInfo, func(statusCode int, responseHeaders [][2]string, responseBody []byte) {
		// Always resume at the end, regardless of success or failure
		defer proxywasm.ResumeHttpRequest()

		if statusCode != 200 {
			log.Warnf("Initialized notification failed with status %d: %s", statusCode, string(responseBody))
			// Even if notification fails, we can still proceed with the operation
			// The backend might still be functional for actual tool calls
		} else {
			log.Debugf("MCP initialization completed successfully")
		}

		// Mark initialization as complete
		ctx.SetContext(CtxMcpProxyInitialized, true)

		// Now execute the originally requested operation
		operation := ctx.GetContext(CtxMcpProxyOperation)
		if operation != nil {
			switch operation.(McpProxyOperation) {
			case OpToolsList:
				if err := h.executeToolsList(ctx); err != nil {
					log.Errorf("Failed to execute tools/list: %v", err)
					utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:tools/list:execution_error")
				}
			case OpToolsCall:
				if err := h.executeToolsCall(ctx); err != nil {
					log.Errorf("Failed to execute tools/call: %v", err)
					utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:tools/call:execution_error")
				}
			default:
				log.Warnf("Unknown MCP proxy operation: %v", operation)
				utils.OnMCPResponseError(ctx, fmt.Errorf("unknown operation"), utils.ErrInternalError, "mcp-proxy:unknown_operation")
			}
		} else {
			// No pending operation, just complete the initialization
			log.Debugf("MCP initialization completed, no pending operation")
		}
	})

	if err != nil {
		log.Errorf("Failed to send initialized notification: %v", err)
		utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:notifications/initialized:send_error")
	}
}

// createToolsListRequest creates a tools/list request
func (h *McpProtocolHandler) createToolsListRequest(cursor *string) map[string]interface{} {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	if cursor != nil && *cursor != "" {
		request["params"].(map[string]interface{})["cursor"] = *cursor
	}

	return request
}

// createToolsCallRequest creates a tools/call request
func (h *McpProtocolHandler) createToolsCallRequest(toolName string, arguments map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}
}

// wrapBackendError wraps backend errors with source attribution
func (h *McpProtocolHandler) wrapBackendError(originalResponse []byte, ctx wrapper.HttpContext) {
	var response map[string]interface{}
	if err := json.Unmarshal(originalResponse, &response); err != nil {
		log.Errorf("Failed to parse backend error response: %v", err)
		utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:error:parse_failure")
		return
	}

	// Add source attribution to error data
	if result, hasResult := response["result"]; hasResult {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if content, hasContent := resultMap["content"]; hasContent {
				if contentArray, ok := content.([]interface{}); ok && len(contentArray) > 0 {
					if textContent, ok := contentArray[0].(map[string]interface{}); ok {
						if text, hasText := textContent["text"]; hasText {
							// Wrap the error text to indicate backend source
							wrappedText := fmt.Sprintf("Backend error: %v", text)
							textContent["text"] = wrappedText
						}
					}
				}
			}
		}
	}

	// Send wrapped response
	wrappedResponse, err := json.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal wrapped error response: %v", err)
		utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:error:marshal_failure")
		return
	}

	// Parse and forward wrapped response
	var wrappedResult map[string]interface{}
	if err := json.Unmarshal(wrappedResponse, &wrappedResult); err != nil {
		log.Errorf("Failed to parse wrapped error response: %v", err)
		utils.OnMCPResponseError(ctx, err, utils.ErrInternalError, "mcp-proxy:error:wrap_failure")
		return
	}

	if result, hasResult := wrappedResult["result"]; hasResult {
		if resultMap, ok := result.(map[string]interface{}); ok {
			utils.OnMCPResponseSuccess(ctx, resultMap, "mcp-proxy:error:backend_wrapped")
		} else {
			utils.OnMCPResponseError(ctx, fmt.Errorf("invalid wrapped result type"), utils.ErrInternalError, "mcp-proxy:error:wrap_failure")
		}
	} else {
		utils.OnMCPResponseError(ctx, fmt.Errorf("wrapped response parse error"), utils.ErrInternalError, "mcp-proxy:error:wrap_failure")
	}
}

// McpSession represents a temporary MCP session
type McpSession struct {
	ID         string
	BackendURL string
	CreatedAt  time.Time
	LastUsed   time.Time
}

// McpSessionManagerImpl manages temporary MCP sessions
type McpSessionManagerImpl struct {
	sessions map[string]*McpSession
}

// NewMcpSessionManagerImpl creates a new session manager
func NewMcpSessionManagerImpl() *McpSessionManagerImpl {
	return &McpSessionManagerImpl{
		sessions: make(map[string]*McpSession),
	}
}

// CreateSession creates a new temporary session
func (m *McpSessionManagerImpl) CreateSession(backendURL string) (string, error) {
	sessionID := fmt.Sprintf("mcp-session-%d", time.Now().UnixNano())
	session := &McpSession{
		ID:         sessionID,
		BackendURL: backendURL,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}

	m.sessions[sessionID] = session
	log.Debugf("Created MCP session %s for %s", sessionID, backendURL)

	return sessionID, nil
}

// GetSession retrieves a session by ID
func (m *McpSessionManagerImpl) GetSession(sessionID string) (*McpSession, bool) {
	session, exists := m.sessions[sessionID]
	if exists {
		session.LastUsed = time.Now()
	}
	return session, exists
}

// CleanupSession removes a session
func (m *McpSessionManagerImpl) CleanupSession(sessionID string) {
	if _, exists := m.sessions[sessionID]; exists {
		delete(m.sessions, sessionID)
		log.Debugf("Cleaned up MCP session %s", sessionID)
	}
}

// CleanupExpiredSessions removes sessions older than specified duration
func (m *McpSessionManagerImpl) CleanupExpiredSessions(maxAge time.Duration) {
	now := time.Now()
	for sessionID, session := range m.sessions {
		if now.Sub(session.LastUsed) > maxAge {
			delete(m.sessions, sessionID)
			log.Debugf("Cleaned up expired MCP session %s", sessionID)
		}
	}
}

// CreateMcpProxyMethodHandlers creates JSON-RPC method handlers for MCP proxy operations
func CreateMcpProxyMethodHandlers(server *McpProxyServer, allowTools map[string]struct{}) utils.MethodHandlers {
	return utils.MethodHandlers{
		"tools/list": func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
			// Extract cursor parameter if present
			var cursor *string
			if cursorResult := params.Get("cursor"); cursorResult.Exists() {
				cursorStr := cursorResult.String()
				cursor = &cursorStr
			}

			// Extract allowTools information from headers and store in context for callback use
			allowToolsHeaderStr, _ := proxywasm.GetHttpRequestHeader("x-envoy-allow-mcp-tools")
			proxywasm.RemoveHttpRequestHeader("x-envoy-allow-mcp-tools")
			ctx.SetContext("mcp_proxy_allow_tools_header", allowToolsHeaderStr)

			// Store server reference and allowTools in context for use in callback
			ctx.SetContext("mcp_proxy_server", server)
			ctx.SetContext("mcp_proxy_allow_tools", allowTools)

			// This will trigger async initialization if needed
			err := server.ForwardToolsList(ctx, cursor)
			if err != nil {
				return err
			}

			// Signal that we need to pause and wait for async response
			ctx.SetContext(utils.CtxNeedPause, true)
			return nil
		},
		"tools/call": func(ctx wrapper.HttpContext, id utils.JsonRpcID, params gjson.Result) error {
			// Extract tool name and arguments
			toolName := params.Get("name").String()
			if toolName == "" {
				return fmt.Errorf("missing tool name")
			}

			// Extract arguments (optional)
			arguments := make(map[string]interface{})
			argsResult := params.Get("arguments")
			if argsResult.Exists() {
				if err := json.Unmarshal([]byte(argsResult.Raw), &arguments); err != nil {
					return fmt.Errorf("invalid arguments: %v", err)
				}
			}

			// Set properties for monitoring and debugging (consistent with default handler)
			proxywasm.SetProperty([]string{"mcp_server_name"}, []byte(server.Name))
			proxywasm.SetProperty([]string{"mcp_tool_name"}, []byte(toolName))

			// Create a tool instance and call it
			toolConfig, exists := server.GetToolConfig(toolName)
			if !exists {
				return fmt.Errorf("tool not found: %s", toolName)
			}

			// Debug logging (consistent with default handler)
			log.Debugf("Tool call [%s] on server [%s] with arguments[%s]", toolName, server.Name, argsResult.Raw)

			tool := &McpProxyTool{
				serverName: server.Name,
				name:       toolName,
				toolConfig: toolConfig,
				arguments:  arguments,
			}

			// This will trigger async initialization if needed
			err := tool.Call(ctx, server)
			if err != nil {
				return err
			}

			// Signal that we need to pause and wait for async response
			ctx.SetContext(utils.CtxNeedPause, true)
			return nil
		},
	}
}

// applyAllowToolsFilter applies allowTools filtering to the tools/list response
func (h *McpProtocolHandler) applyAllowToolsFilter(ctx wrapper.HttpContext, resultMap map[string]interface{}) map[string]interface{} {
	// Get allowTools configuration from context
	var allowTools map[string]struct{}
	if allowToolsCtx := ctx.GetContext("mcp_proxy_allow_tools"); allowToolsCtx != nil {
		if allowToolsMap, ok := allowToolsCtx.(map[string]struct{}); ok {
			allowTools = allowToolsMap
		}
	}
	if allowTools == nil {
		allowTools = make(map[string]struct{})
	}

	// Get allowTools from request header (stored earlier in context)
	allowToolsFromHeader := make(map[string]struct{})
	if allowToolsHeaderStr := ctx.GetContext("mcp_proxy_allow_tools_header"); allowToolsHeaderStr != nil {
		headerStr := allowToolsHeaderStr.(string)
		for tool := range strings.SplitSeq(headerStr, ",") {
			trimmedTool := strings.TrimSpace(tool)
			if trimmedTool == "" {
				continue
			}
			allowToolsFromHeader[trimmedTool] = struct{}{}
		}
	}

	// If no filtering is needed, return original result
	if len(allowTools) == 0 && len(allowToolsFromHeader) == 0 {
		return resultMap
	}

	// Apply filtering to tools array
	if tools, hasTools := resultMap["tools"]; hasTools {
		if toolsArray, ok := tools.([]interface{}); ok {
			filteredTools := make([]interface{}, 0)

			for _, tool := range toolsArray {
				if toolMap, ok := tool.(map[string]interface{}); ok {
					if name, hasName := toolMap["name"]; hasName {
						if toolName, ok := name.(string); ok {
							// Check against configuration allowTools
							if len(allowTools) > 0 {
								if _, allow := allowTools[toolName]; !allow {
									continue
								}
							}

							// Check against header allowTools
							if len(allowToolsFromHeader) > 0 {
								if _, allow := allowToolsFromHeader[toolName]; !allow {
									continue
								}
							}

							// Tool is allowed, add to filtered list
							filteredTools = append(filteredTools, tool)
						}
					}
				}
			}

			// Create new result with filtered tools
			filteredResult := make(map[string]interface{})
			for k, v := range resultMap {
				filteredResult[k] = v
			}
			filteredResult["tools"] = filteredTools
			return filteredResult
		}
	}

	// If tools array not found or invalid format, return original
	return resultMap
}

// applyProxyAuthentication applies authentication to the proxy request headers and URL
func (h *McpProtocolHandler) applyProxyAuthentication(server *McpProxyServer, schemeID string, passthroughCredential string, headers *[][2]string) (string, error) {
	// Parse the backend URL to create a proper URL object for the shared function
	parsedURL, err := url.Parse(h.backendURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse backend URL: %v", err)
	}

	// Create authentication context
	authCtx := AuthRequestContext{
		Method:                "POST",
		Headers:               *headers,
		ParsedURL:             parsedURL,
		RequestBody:           []byte{}, // Not used for header/query auth
		PassthroughCredential: passthroughCredential,
	}

	// Create security config for gateway-to-backend authentication
	// The passthrough credential (if any) comes from client-to-gateway authentication
	securityConfig := SecurityRequirement{
		ID:          schemeID,
		Credential:  "",                          // Will use passthrough credential or default credential from scheme
		Passthrough: passthroughCredential != "", // Use passthrough if we have a credential
	}

	// Apply authentication using shared utilities
	err = ApplySecurity(securityConfig, server, &authCtx)
	if err != nil {
		return "", err
	}

	// Update headers with authentication applied
	*headers = authCtx.Headers

	// Reconstruct URL from potentially modified ParsedURL (similar to rest_server.go logic)
	u := authCtx.ParsedURL
	encodedPath := u.EscapedPath()
	var urlStr string
	if u.Scheme != "" && u.Host != "" {
		urlStr = u.Scheme + "://" + u.Host + encodedPath
	} else {
		urlStr = "/" + strings.TrimPrefix(encodedPath, "/")
	}
	if u.RawQuery != "" {
		urlStr += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		urlStr += "#" + u.Fragment
	}

	return urlStr, nil
}
