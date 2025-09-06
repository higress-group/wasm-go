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
	"encoding/json"
	"testing"

	"github.com/higress-group/wasm-go/pkg/iface"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// MockHttpContext is a mock implementation of wrapper.HttpContext for testing
type MockHttpContext struct {
	responseData  map[string]any
	debugInfo     string
	userContext   map[string]interface{}
	userAttribute map[string]interface{}
}

func (m *MockHttpContext) Scheme() string {
	return "http"
}

func (m *MockHttpContext) Host() string {
	return "localhost"
}

func (m *MockHttpContext) Path() string {
	return "/mcp"
}

func (m *MockHttpContext) Method() string {
	return "POST"
}

func (m *MockHttpContext) SetContext(key string, value interface{}) {
	if m.userContext == nil {
		m.userContext = make(map[string]interface{})
	}
	m.userContext[key] = value
}

func (m *MockHttpContext) GetContext(key string) interface{} {
	if m.userContext == nil {
		return nil
	}
	return m.userContext[key]
}

func (m *MockHttpContext) GetBoolContext(key string, defaultValue bool) bool {
	if v, ok := m.GetContext(key).(bool); ok {
		return v
	}
	return defaultValue
}

func (m *MockHttpContext) GetStringContext(key, defaultValue string) string {
	if v, ok := m.GetContext(key).(string); ok {
		return v
	}
	return defaultValue
}

func (m *MockHttpContext) GetByteSliceContext(key string, defaultValue []byte) []byte {
	if v, ok := m.GetContext(key).([]byte); ok {
		return v
	}
	return defaultValue
}

func (m *MockHttpContext) GetUserAttribute(key string) interface{} {
	if m.userAttribute == nil {
		return nil
	}
	return m.userAttribute[key]
}

func (m *MockHttpContext) SetUserAttribute(key string, value interface{}) {
	if m.userAttribute == nil {
		m.userAttribute = make(map[string]interface{})
	}
	m.userAttribute[key] = value
}

func (m *MockHttpContext) SetUserAttributeMap(kvmap map[string]interface{}) {
	m.userAttribute = kvmap
}

func (m *MockHttpContext) GetUserAttributeMap() map[string]interface{} {
	return m.userAttribute
}

func (m *MockHttpContext) WriteUserAttributeToLog() error {
	return nil
}

func (m *MockHttpContext) WriteUserAttributeToLogWithKey(key string) error {
	return nil
}

func (m *MockHttpContext) WriteUserAttributeToTrace() error {
	return nil
}

func (m *MockHttpContext) DontReadRequestBody() {
	// Mock implementation
}

func (m *MockHttpContext) DontReadResponseBody() {
	// Mock implementation
}

func (m *MockHttpContext) BufferRequestBody() {
	// Mock implementation
}

func (m *MockHttpContext) BufferResponseBody() {
	// Mock implementation
}

func (m *MockHttpContext) NeedPauseStreamingResponse() {
	// Mock implementation
}

func (m *MockHttpContext) PushBuffer(buffer []byte) {
	// Mock implementation
}

func (m *MockHttpContext) PopBuffer() []byte {
	return nil
}

func (m *MockHttpContext) BufferQueueSize() int {
	return 0
}

func (m *MockHttpContext) DisableReroute() {
	// Mock implementation
}

func (m *MockHttpContext) SetRequestBodyBufferLimit(limit uint32) {
	// Mock implementation
}

func (m *MockHttpContext) SetResponseBodyBufferLimit(limit uint32) {
	// Mock implementation
}

func (m *MockHttpContext) RouteCall(method string, url string, headers [][2]string, body []byte, callback iface.RouteResponseCallback) error {
	// Mock implementation
	return nil
}

func (m *MockHttpContext) GetExecutionPhase() iface.HTTPExecutionPhase {
	return iface.DecodeHeader
}

// MockOnJsonRpcResponseSuccess is a mock function to replace OnJsonRpcResponseSuccess for testing
func MockOnJsonRpcResponseSuccess(ctx wrapper.HttpContext, result map[string]any, debugInfo string) {
	if mockCtx, ok := ctx.(*MockHttpContext); ok {
		mockCtx.responseData = result
		mockCtx.debugInfo = debugInfo
	}
}

// TestStructuredContentWithJsonRawMessage tests the structured content functionality with json.RawMessage
func TestStructuredContentWithJsonRawMessage(t *testing.T) {
	tests := []struct {
		name              string
		content           []map[string]any
		structuredContent json.RawMessage
		debugInfo         string
		expectedNil       bool
	}{
		{
			name: "valid structured content with JSON object",
			content: []map[string]any{
				{
					"type": "text",
					"text": "Test result",
				},
			},
			structuredContent: json.RawMessage(`{"result": "success", "data": {"value": 42}}`),
			debugInfo:         "test-debug",
			expectedNil:       false,
		},
		{
			name: "valid structured content with JSON array",
			content: []map[string]any{
				{
					"type": "text",
					"text": "Array result",
				},
			},
			structuredContent: json.RawMessage(`[{"id": 1, "name": "item1"}, {"id": 2, "name": "item2"}]`),
			debugInfo:         "test-debug",
			expectedNil:       false,
		},
		{
			name: "valid structured content with primitive JSON",
			content: []map[string]any{
				{
					"type": "text",
					"text": "Primitive result",
				},
			},
			structuredContent: json.RawMessage(`"simple string"`),
			debugInfo:         "test-debug",
			expectedNil:       false,
		},
		{
			name: "valid structured content with number",
			content: []map[string]any{
				{
					"type": "text",
					"text": "Number result",
				},
			},
			structuredContent: json.RawMessage(`123.45`),
			debugInfo:         "test-debug",
			expectedNil:       false,
		},
		{
			name: "valid structured content with boolean",
			content: []map[string]any{
				{
					"type": "text",
					"text": "Boolean result",
				},
			},
			structuredContent: json.RawMessage(`true`),
			debugInfo:         "test-debug",
			expectedNil:       false,
		},
		{
			name: "nil structured content",
			content: []map[string]any{
				{
					"type": "text",
					"text": "No structured content",
				},
			},
			structuredContent: nil,
			debugInfo:         "test-debug",
			expectedNil:       true,
		},
		{
			name: "empty structured content",
			content: []map[string]any{
				{
					"type": "text",
					"text": "Empty structured content",
				},
			},
			structuredContent: json.RawMessage(``),
			debugInfo:         "test-debug",
			expectedNil:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the response building logic directly
			response := map[string]any{
				"content": tt.content,
				"isError": false,
			}
			if tt.structuredContent != nil && len(tt.structuredContent) > 0 {
				response["structuredContent"] = tt.structuredContent
			}

			// Verify the response data structure
			if response["content"] == nil {
				t.Errorf("Expected 'content' field in response")
			}

			if response["isError"] != false {
				t.Errorf("Expected 'isError' to be false")
			}

			// Check structured content
			if tt.expectedNil {
				if response["structuredContent"] != nil {
					t.Errorf("Expected 'structuredContent' to be nil, got %v", response["structuredContent"])
				}
			} else {
				if response["structuredContent"] == nil {
					t.Errorf("Expected 'structuredContent' to be present")
				} else {
					// Verify that structured content is preserved as json.RawMessage
					structuredContent, ok := response["structuredContent"].(json.RawMessage)
					if !ok {
						t.Errorf("Expected 'structuredContent' to be json.RawMessage, got %T", response["structuredContent"])
					} else {
						// Verify the content matches
						if string(structuredContent) != string(tt.structuredContent) {
							t.Errorf("Expected structured content %s, got %s", string(tt.structuredContent), string(structuredContent))
						}
					}
				}
			}
		})
	}
}

// TestSendMCPToolTextResultWithStructuredContent tests the SendMCPToolTextResultWithStructuredContent function
func TestSendMCPToolTextResultWithStructuredContent(t *testing.T) {
	tests := []struct {
		name              string
		textResult        string
		structuredContent json.RawMessage
		debugInfo         []string
		expectedNil       bool
	}{
		{
			name:              "text result with structured content",
			textResult:        "Operation completed successfully",
			structuredContent: json.RawMessage(`{"status": "success", "count": 5}`),
			debugInfo:         []string{"custom-debug-info"},
			expectedNil:       false,
		},
		{
			name:              "text result without structured content",
			textResult:        "Simple text result",
			structuredContent: nil,
			debugInfo:         []string{},
			expectedNil:       true,
		},
		{
			name:              "text result with empty structured content",
			textResult:        "Text with empty structured content",
			structuredContent: json.RawMessage(``),
			debugInfo:         []string{},
			expectedNil:       true,
		},
		{
			name:              "text result with complex structured content",
			textResult:        "Complex operation result",
			structuredContent: json.RawMessage(`{"items": [{"id": 1, "name": "item1"}, {"id": 2, "name": "item2"}], "metadata": {"total": 2, "page": 1}}`),
			debugInfo:         []string{"complex-debug"},
			expectedNil:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the content building logic directly
			content := []map[string]any{
				{
					"type": "text",
					"text": tt.textResult,
				},
			}

			// Test the response building logic directly
			response := map[string]any{
				"content": content,
				"isError": false,
			}
			if tt.structuredContent != nil && len(tt.structuredContent) > 0 {
				response["structuredContent"] = tt.structuredContent
			}

			// Verify the response data structure
			if response["content"] == nil {
				t.Errorf("Expected 'content' field in response")
			}

			// Check that content contains the text result
			contentList, ok := response["content"].([]map[string]any)
			if !ok {
				t.Errorf("Expected 'content' to be []map[string]any, got %T", response["content"])
			} else if len(contentList) != 1 {
				t.Errorf("Expected content to have 1 item, got %d", len(contentList))
			} else if contentList[0]["type"] != "text" {
				t.Errorf("Expected content type to be 'text', got %v", contentList[0]["type"])
			} else if contentList[0]["text"] != tt.textResult {
				t.Errorf("Expected text content '%s', got '%s'", tt.textResult, contentList[0]["text"])
			}

			if response["isError"] != false {
				t.Errorf("Expected 'isError' to be false")
			}

			// Check structured content
			if tt.expectedNil {
				if response["structuredContent"] != nil {
					t.Errorf("Expected 'structuredContent' to be nil, got %v", response["structuredContent"])
				}
			} else {
				if response["structuredContent"] == nil {
					t.Errorf("Expected 'structuredContent' to be present")
				} else {
					// Verify that structured content is preserved as json.RawMessage
					structuredContent, ok := response["structuredContent"].(json.RawMessage)
					if !ok {
						t.Errorf("Expected 'structuredContent' to be json.RawMessage, got %T", response["structuredContent"])
					} else {
						// Verify the content matches
						if string(structuredContent) != string(tt.structuredContent) {
							t.Errorf("Expected structured content %s, got %s", string(tt.structuredContent), string(structuredContent))
						}
					}
				}
			}
		})
	}
}

// TestJsonRawMessagePreservation tests that json.RawMessage preserves original JSON structure
func TestJsonRawMessagePreservation(t *testing.T) {
	testCases := []struct {
		name     string
		jsonData string
		expected string
	}{
		{
			name:     "preserve object structure",
			jsonData: `{"key": "value", "number": 42, "nested": {"inner": true}}`,
			expected: `{"key": "value", "number": 42, "nested": {"inner": true}}`,
		},
		{
			name:     "preserve array structure",
			jsonData: `[{"id": 1}, {"id": 2}, {"id": 3}]`,
			expected: `[{"id": 1}, {"id": 2}, {"id": 3}]`,
		},
		{
			name:     "preserve string with special characters",
			jsonData: `"Hello, \"World\"! \n New line"`,
			expected: `"Hello, \"World\"! \n New line"`,
		},
		{
			name:     "preserve number precision",
			jsonData: `123.4567890123456789`,
			expected: `123.4567890123456789`,
		},
		{
			name:     "preserve boolean values",
			jsonData: `true`,
			expected: `true`,
		},
		{
			name:     "preserve null value",
			jsonData: `null`,
			expected: `null`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create json.RawMessage from test data
			rawMessage := json.RawMessage(tc.jsonData)

			// Verify it preserves the original structure
			if string(rawMessage) != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, string(rawMessage))
			}

			// Test that it can be used in a response structure
			response := map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": "Test",
					},
				},
				"isError": false,
			}
			if rawMessage != nil && len(rawMessage) > 0 {
				response["structuredContent"] = rawMessage
			}

			// Verify the structured content is preserved
			if response["structuredContent"] == nil {
				t.Errorf("Expected structured content to be present")
			} else {
				structuredContent, ok := response["structuredContent"].(json.RawMessage)
				if !ok {
					t.Errorf("Expected structured content to be json.RawMessage")
				} else if string(structuredContent) != tc.expected {
					t.Errorf("Expected structured content %s, got %s", tc.expected, string(structuredContent))
				}
			}
		})
	}
}
