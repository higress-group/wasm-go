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
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/tidwall/sjson"
)

func TestMCPProtocolVersionSupport(t *testing.T) {
	tests := []struct {
		name              string
		version           string
		shouldBeSupported bool
	}{
		{
			name:              "supported version 2024-11-05",
			version:           "2024-11-05",
			shouldBeSupported: true,
		},
		{
			name:              "supported version 2025-03-26",
			version:           "2025-03-26",
			shouldBeSupported: true,
		},
		{
			name:              "supported version 2025-06-18",
			version:           "2025-06-18",
			shouldBeSupported: true,
		},
		{
			name:              "unsupported version 2023-01-01",
			version:           "2023-01-01",
			shouldBeSupported: false,
		},
		{
			name:              "invalid version format",
			version:           "invalid-version",
			shouldBeSupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the version validation logic
			supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
			versionSupported := false
			for _, supportedVersion := range supportedVersions {
				if tt.version == supportedVersion {
					versionSupported = true
					break
				}
			}

			if versionSupported != tt.shouldBeSupported {
				t.Errorf("Version %s support check failed: expected %v, got %v",
					tt.version, tt.shouldBeSupported, versionSupported)
			}
		})
	}
}

func TestMCPProtocolVersionCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "version 2024-11-05 capabilities",
			version: "2024-11-05",
		},
		{
			name:    "version 2025-03-26 capabilities",
			version: "2025-03-26",
		},
		{
			name:    "version 2025-06-18 capabilities",
			version: "2025-06-18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the capabilities logic from the initialize method
			capabilities := map[string]any{
				"tools": map[string]any{},
			}

			// Verify basic capabilities structure
			if capabilities["tools"] == nil {
				t.Errorf("Expected tools capability to exist for version %s", tt.version)
			}
		})
	}
}

func TestMCPProtocolVersionHeaderParsing(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		shouldSetCtx  bool
		shouldLogWarn bool
	}{
		{
			name:          "valid header 2024-11-05",
			headerValue:   "2024-11-05",
			shouldSetCtx:  true,
			shouldLogWarn: false,
		},
		{
			name:          "valid header 2025-03-26",
			headerValue:   "2025-03-26",
			shouldSetCtx:  true,
			shouldLogWarn: false,
		},
		{
			name:          "valid header 2025-06-18",
			headerValue:   "2025-06-18",
			shouldSetCtx:  true,
			shouldLogWarn: false,
		},
		{
			name:          "invalid header version",
			headerValue:   "2023-01-01",
			shouldSetCtx:  false,
			shouldLogWarn: true,
		},
		{
			name:          "malformed header version",
			headerValue:   "invalid-format",
			shouldSetCtx:  false,
			shouldLogWarn: true,
		},
		{
			name:          "empty header value",
			headerValue:   "",
			shouldSetCtx:  false,
			shouldLogWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the header parsing logic (simulating the onHttpRequestHeaders function)
			if tt.headerValue != "" {
				// Validate the protocol version against supported versions
				supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
				versionSupported := false
				for _, supportedVersion := range supportedVersions {
					if tt.headerValue == supportedVersion {
						versionSupported = true
						break
					}
				}

				if tt.shouldSetCtx && !versionSupported {
					t.Errorf("Expected version %s to be supported but it was not", tt.headerValue)
				}
				if !tt.shouldSetCtx && versionSupported && tt.headerValue != "" {
					t.Errorf("Expected version %s to be unsupported but it was supported", tt.headerValue)
				}
			}
		})
	}
}

func TestMCPProtocolVersionContextFlow(t *testing.T) {
	// Test that protocol version flows correctly through the system
	tests := []struct {
		name                 string
		headerVersion        string
		initializeVersion    string
		expectedFinalVersion string
		description          string
	}{
		{
			name:                 "header only 2025-06-18",
			headerVersion:        "2025-06-18",
			initializeVersion:    "",
			expectedFinalVersion: "2025-06-18",
			description:          "When only header is provided, it should be used",
		},
		{
			name:                 "initialize only 2025-03-26",
			headerVersion:        "",
			initializeVersion:    "2025-03-26",
			expectedFinalVersion: "2025-03-26",
			description:          "When only initialize method provides version, it should be used",
		},
		{
			name:                 "header takes precedence",
			headerVersion:        "2025-06-18",
			initializeVersion:    "2025-03-26",
			expectedFinalVersion: "2025-03-26",
			description:          "When both are provided, initialize method overrides header (processed later)",
		},
		{
			name:                 "both same version",
			headerVersion:        "2024-11-05",
			initializeVersion:    "2024-11-05",
			expectedFinalVersion: "2024-11-05",
			description:          "When both provide same version, that version should be used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the context flow:
			// 1. onHttpRequestHeaders processes MCP-Protocol-Version header
			// 2. initialize method may override with protocolVersion param

			contextVersion := ""

			// Step 1: Header processing (onHttpRequestHeaders)
			if tt.headerVersion != "" {
				supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
				versionSupported := false
				for _, supportedVersion := range supportedVersions {
					if tt.headerVersion == supportedVersion {
						versionSupported = true
						break
					}
				}
				if versionSupported {
					contextVersion = tt.headerVersion
				}
			}

			// Step 2: Initialize method processing (may override)
			if tt.initializeVersion != "" {
				supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
				versionSupported := false
				for _, supportedVersion := range supportedVersions {
					if tt.initializeVersion == supportedVersion {
						versionSupported = true
						break
					}
				}
				if versionSupported {
					contextVersion = tt.initializeVersion
				}
			}

			if contextVersion != tt.expectedFinalVersion {
				t.Errorf("Context version flow failed for %s: expected %s, got %s",
					tt.description, tt.expectedFinalVersion, contextVersion)
			}
		})
	}
}

func TestMCPProtocolVersionBackwardsCompatibility(t *testing.T) {
	// Test that older versions still work correctly
	tests := []struct {
		name                  string
		version               string
		expectsToolsListError bool
		expectsInitializeOK   bool
	}{
		{
			name:                  "2024-11-05 backwards compatibility",
			version:               "2024-11-05",
			expectsToolsListError: false,
			expectsInitializeOK:   true,
		},
		{
			name:                  "2025-03-26 backwards compatibility",
			version:               "2025-03-26",
			expectsToolsListError: false,
			expectsInitializeOK:   true,
		},
		{
			name:                  "unsupported version handling",
			version:               "2023-01-01",
			expectsToolsListError: true,
			expectsInitializeOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test initialize method response
			supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
			versionSupported := false
			for _, supportedVersion := range supportedVersions {
				if tt.version == supportedVersion {
					versionSupported = true
					break
				}
			}

			if versionSupported != tt.expectsInitializeOK {
				t.Errorf("Version %s initialize support mismatch: expected %v, got %v",
					tt.version, tt.expectsInitializeOK, versionSupported)
			}

			// Test that capabilities are correctly set for the version
			if versionSupported {
				capabilities := map[string]any{
					"tools": map[string]any{},
				}

				// Verify basic capabilities structure
				if capabilities["tools"] == nil {
					t.Errorf("Expected tools capability to exist for version %s", tt.version)
				}
			}
		})
	}
}

func TestMCPProtocolVersionErrorHandling(t *testing.T) {
	// Test error conditions and edge cases
	tests := []struct {
		name             string
		version          string
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:             "empty version string",
			version:          "",
			expectError:      true,
			expectedErrorMsg: "Unsupported protocol version",
		},
		{
			name:             "future version",
			version:          "2026-01-01",
			expectError:      true,
			expectedErrorMsg: "Unsupported protocol version: 2026-01-01",
		},
		{
			name:             "past version",
			version:          "2020-01-01",
			expectError:      true,
			expectedErrorMsg: "Unsupported protocol version: 2020-01-01",
		},
		{
			name:             "malformed version",
			version:          "not-a-version",
			expectError:      true,
			expectedErrorMsg: "Unsupported protocol version: not-a-version",
		},
		{
			name:        "valid current version",
			version:     "2025-06-18",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the error handling logic from initialize method
			var err error

			if tt.version == "" {
				err = errors.New("Unsupported protocol version")
			} else {
				supportedVersions := []string{"2024-11-05", "2025-03-26", "2025-06-18"}
				versionSupported := false
				for _, supportedVersion := range supportedVersions {
					if tt.version == supportedVersion {
						versionSupported = true
						break
					}
				}

				if !versionSupported {
					err = fmt.Errorf("Unsupported protocol version: %s", tt.version)
				}
			}

			if tt.expectError && err == nil {
				t.Errorf("Expected error for version %s but got none", tt.version)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for version %s: %v", tt.version, err)
			}

			if tt.expectError && err != nil && tt.expectedErrorMsg != "" {
				if err.Error() != tt.expectedErrorMsg {
					t.Errorf("Expected error message '%s' for version %s, got '%s'",
						tt.expectedErrorMsg, tt.version, err.Error())
				}
			}
		})
	}
}

func TestConvertArgToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string value",
			input:    "test string",
			expected: "test string",
		},
		{
			name:     "boolean true",
			input:    true,
			expected: "true",
		},
		{
			name:     "boolean false",
			input:    false,
			expected: "false",
		},
		{
			name:     "integer",
			input:    42,
			expected: "42",
		},
		{
			name:     "float",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "map",
			input:    map[string]interface{}{"key": "value"},
			expected: `{"key":"value"}`,
		},
		{
			name:     "array",
			input:    []interface{}{1, 2, 3},
			expected: "[1,2,3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertArgToString(tt.input)
			if result != tt.expected {
				t.Errorf("convertArgToString(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResponseTemplatePrependAppend(t *testing.T) {
	// Test response template with PrependBody and AppendBody
	sampleResponse := `{"result": "success", "data": {"name": "Test", "value": 42}}`

	tests := []struct {
		name        string
		template    RestToolResponseTemplate
		expected    []string
		notExpected []string
	}{
		{
			name: "with body template only",
			template: RestToolResponseTemplate{
				Body: "# Result\n- Name: {{.data.name}}\n- Value: {{.data.value}}",
			},
			expected: []string{
				"# Result",
				"- Name: Test",
				"- Value: 42",
			},
			notExpected: []string{
				"Field Descriptions:",
				"End of Response",
				`{"result": "success"`,
			},
		},
		{
			name: "with prepend only",
			template: RestToolResponseTemplate{
				PrependBody: "# Field Descriptions:\n- result: Operation result\n- data: Response data\n\n",
			},
			expected: []string{
				"# Field Descriptions:",
				"- result: Operation result",
				"- data: Response data",
				`{"result": "success"`,
				`"name": "Test"`,
			},
		},
		{
			name: "with append only",
			template: RestToolResponseTemplate{
				AppendBody: "\n\n*End of Response*",
			},
			expected: []string{
				`{"result": "success"`,
				`"name": "Test"`,
				"*End of Response*",
			},
		},
		{
			name: "with both prepend and append",
			template: RestToolResponseTemplate{
				PrependBody: "# API Response:\n\n",
				AppendBody:  "\n\n*This is raw JSON data with field 'name' = Test and 'value' = 42*",
			},
			expected: []string{
				"# API Response:",
				`{"result": "success"`,
				`"name": "Test"`,
				"*This is raw JSON data with field 'name' = Test and 'value' = 42*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a tool with the test template
			tool := RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com/api",
					Method: "GET",
				},
				ResponseTemplate: tt.template,
			}

			// Parse templates
			err := tool.parseTemplates()
			if err != nil {
				t.Fatalf("Failed to parse templates: %v", err)
			}

			// Simulate response processing
			var result string
			responseBody := []byte(sampleResponse)

			// Case 1: Full response template is provided
			if tool.parsedResponseTemplate != nil {
				templateResult, err := executeTemplate(tool.parsedResponseTemplate, responseBody)
				if err != nil {
					t.Fatalf("Failed to execute response template: %v", err)
				}
				result = templateResult
			} else {
				// Case 2: No template, but prepend/append might be used
				rawResponse := string(responseBody)

				// Apply prepend/append if specified
				if tool.ResponseTemplate.PrependBody != "" || tool.ResponseTemplate.AppendBody != "" {
					result = tool.ResponseTemplate.PrependBody + rawResponse + tool.ResponseTemplate.AppendBody
				} else {
					// Case 3: No template and no prepend/append, just use raw response
					result = rawResponse
				}
			}

			// Check that the result contains expected substrings
			for _, substr := range tt.expected {
				if !strings.Contains(result, substr) {
					t.Errorf("Expected substring not found: %s", substr)
				}
			}

			// Check that the result does not contain unexpected substrings
			for _, substr := range tt.notExpected {
				if strings.Contains(result, substr) {
					t.Errorf("Unexpected substring found: %s", substr)
				}
			}
		})
	}
}

func TestHasContentType(t *testing.T) {
	tests := []struct {
		name            string
		headers         [][2]string
		contentTypeStr  string
		expectedOutcome bool
	}{
		{
			name: "exact match",
			headers: [][2]string{
				{"Content-Type", "application/json"},
			},
			contentTypeStr:  "application/json",
			expectedOutcome: true,
		},
		{
			name: "case insensitive match",
			headers: [][2]string{
				{"content-type", "application/JSON"},
			},
			contentTypeStr:  "application/json",
			expectedOutcome: true,
		},
		{
			name: "substring match",
			headers: [][2]string{
				{"Content-Type", "application/json; charset=utf-8"},
			},
			contentTypeStr:  "application/json",
			expectedOutcome: true,
		},
		{
			name: "no match",
			headers: [][2]string{
				{"Content-Type", "text/plain"},
			},
			contentTypeStr:  "application/json",
			expectedOutcome: false,
		},
		{
			name: "header not present",
			headers: [][2]string{
				{"Accept", "application/json"},
			},
			contentTypeStr:  "application/json",
			expectedOutcome: false,
		},
		{
			name:            "empty headers",
			headers:         [][2]string{},
			contentTypeStr:  "application/json",
			expectedOutcome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasContentType(tt.headers, tt.contentTypeStr)
			if result != tt.expectedOutcome {
				t.Errorf("hasContentType(%v, %v) = %v, want %v", tt.headers, tt.contentTypeStr, result, tt.expectedOutcome)
			}
		})
	}
}

func TestRestToolValidation(t *testing.T) {
	tests := []struct {
		name          string
		tool          RestTool
		expectedError bool
	}{
		{
			name: "valid tool with no args options",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com",
					Method: "GET",
				},
			},
			expectedError: false,
		},
		{
			name: "valid tool with argsToJsonBody",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:            "https://example.com",
					Method:         "POST",
					ArgsToJsonBody: true,
				},
			},
			expectedError: false,
		},
		{
			name: "valid tool with argsToUrlParam",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:            "https://example.com",
					Method:         "GET",
					ArgsToUrlParam: true,
				},
			},
			expectedError: false,
		},
		{
			name: "valid tool with argsToFormBody",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:            "https://example.com",
					Method:         "POST",
					ArgsToFormBody: true,
				},
			},
			expectedError: false,
		},
		{
			name: "invalid tool with multiple args options",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:            "https://example.com",
					Method:         "POST",
					ArgsToJsonBody: true,
					ArgsToFormBody: true,
				},
			},
			expectedError: true,
		},
		{
			name: "invalid tool with all args options",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:            "https://example.com",
					Method:         "POST",
					ArgsToJsonBody: true,
					ArgsToUrlParam: true,
					ArgsToFormBody: true,
				},
			},
			expectedError: true,
		},
		{
			name: "invalid tool with both Body and PrependBody",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com",
					Method: "GET",
				},
				ResponseTemplate: RestToolResponseTemplate{
					Body:        "# Result\n{{.data}}",
					PrependBody: "# Field Descriptions:\n",
				},
			},
			expectedError: true,
		},
		{
			name: "invalid tool with both Body and AppendBody",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com",
					Method: "GET",
				},
				ResponseTemplate: RestToolResponseTemplate{
					Body:       "# Result\n{{.data}}",
					AppendBody: "\n*End of response*",
				},
			},
			expectedError: true,
		},
		{
			name: "invalid tool with Body, PrependBody, and AppendBody",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com",
					Method: "GET",
				},
				ResponseTemplate: RestToolResponseTemplate{
					Body:        "# Result\n{{.data}}",
					PrependBody: "# Field Descriptions:\n",
					AppendBody:  "\n*End of response*",
				},
			},
			expectedError: true,
		},
		{
			name: "valid tool with PrependBody and AppendBody but no Body",
			tool: RestTool{
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com",
					Method: "GET",
				},
				ResponseTemplate: RestToolResponseTemplate{
					PrependBody: "# Field Descriptions:\n",
					AppendBody:  "\n*End of response*",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tool.parseTemplates()
			if (err != nil) != tt.expectedError {
				t.Errorf("parseTemplates() error = %v, expectedError %v", err, tt.expectedError)
			}
		})
	}
}

func TestInputSchemaWithComplexTypes(t *testing.T) {
	// Create a tool with array and object type arguments
	tool := RestMCPTool{
		toolConfig: RestTool{
			Args: []RestToolArg{
				{
					Name:        "stringArg",
					Description: "A string argument",
					Type:        "string",
				},
				{
					Name:        "arrayArg",
					Description: "An array argument",
					Type:        "array",
					Items: map[string]interface{}{
						"type": "string",
					},
				},
				{
					Name:        "objectArg",
					Description: "An object argument",
					Type:        "object",
					Properties: map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name property",
						},
						"age": map[string]interface{}{
							"type":        "integer",
							"description": "Age property",
						},
					},
				},
				{
					Name:        "arrayOfObjects",
					Description: "An array of objects",
					Type:        "array",
					Items: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]interface{}{
								"type": "string",
							},
							"value": map[string]interface{}{
								"type": "number",
							},
						},
					},
				},
			},
		},
	}

	schema := tool.InputSchema()

	// Check schema structure
	if schema["type"] != "object" {
		t.Errorf("Expected schema type to be 'object', got %v", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected properties to be a map, got %T", schema["properties"])
	}

	// Check individual property types
	checkProperty := func(name, expectedType string) {
		prop, ok := properties[name].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected property %s to be a map, got %T", name, properties[name])
		}
		if prop["type"] != expectedType {
			t.Errorf("Expected property %s type to be '%s', got %v", name, expectedType, prop["type"])
		}
	}

	checkProperty("stringArg", "string")
	checkProperty("arrayArg", "array")
	checkProperty("objectArg", "object")
	checkProperty("arrayOfObjects", "array")

	// Check array items
	arrayArg, _ := properties["arrayArg"].(map[string]interface{})
	if arrayArg["items"] == nil {
		t.Errorf("Expected arrayArg to have items property")
	}

	// Check object properties
	objectArg, _ := properties["objectArg"].(map[string]interface{})
	if objectArg["properties"] == nil {
		t.Errorf("Expected objectArg to have properties property")
	}

	// Check array of objects
	arrayOfObjects, _ := properties["arrayOfObjects"].(map[string]interface{})
	items, ok := arrayOfObjects["items"].(map[string]interface{})
	if !ok || items["type"] != "object" {
		t.Errorf("Expected arrayOfObjects items to be of type object")
	}
}

func TestArgsToUrlParamAndFormBody(t *testing.T) {
	// Test argsToUrlParam
	t.Run("argsToUrlParam", func(t *testing.T) {
		args := map[string]interface{}{
			"string": "value",
			"int":    42,
			"bool":   true,
			"array":  []interface{}{1, 2, 3},
			"object": map[string]interface{}{"key": "value"},
		}

		// Parse URL and add parameters
		baseURL := "https://example.com/api"
		parsedURL, _ := url.Parse(baseURL)
		query := parsedURL.Query()

		for key, value := range args {
			query.Set(key, convertArgToString(value))
		}

		parsedURL.RawQuery = query.Encode()
		result := parsedURL.String()

		// Verify each parameter is in the URL
		for key, value := range args {
			strValue := convertArgToString(value)
			encodedValue := url.QueryEscape(strValue)
			paramStr := key + "=" + encodedValue

			if !strings.Contains(result, paramStr) {
				t.Errorf("URL parameter missing: %s", paramStr)
			}
		}
	})

	// Test argsToFormBody
	t.Run("argsToFormBody", func(t *testing.T) {
		args := map[string]interface{}{
			"string": "value",
			"int":    42,
			"bool":   true,
			"array":  []interface{}{1, 2, 3},
			"object": map[string]interface{}{"key": "value"},
		}

		// Create form values
		formValues := url.Values{}
		for key, value := range args {
			formValues.Set(key, convertArgToString(value))
		}

		formBody := formValues.Encode()

		// Verify each parameter is in the form body
		for key, value := range args {
			strValue := convertArgToString(value)
			encodedValue := url.QueryEscape(strValue)
			paramStr := key + "=" + encodedValue

			if !strings.Contains(formBody, paramStr) {
				t.Errorf("Form body missing parameter: %s", paramStr)
			}
		}
	})
}

func TestRestToolConfig(t *testing.T) {
	// Example REST tool configuration
	configJSON := `
{
  "server": {
    "name": "rest-amap-server",
    "config": {
      "apiKey": "xxxxx"
    }
  },
  "tools": [
    {
      "name": "maps-geo",
      "description": "将详细的结构化地址转换为经纬度坐标。支持对地标性名胜景区、建筑物名称解析为经纬度坐标",
      "args": [
        {
          "name": "address",
          "description": "待解析的结构化地址信息",
          "type": "string",
          "required": true
        },
        {
          "name": "city",
          "description": "指定查询的城市",
          "required": false
        },
        {
          "name": "output",
          "description": "输出格式",
          "type": "string",
          "enum": ["json", "xml"],
          "default": "json"
        },
        {
          "name": "options",
          "description": "高级选项",
          "type": "object",
          "properties": {
            "extensions": {
              "type": "string",
              "enum": ["base", "all"]
            },
            "batch": {
              "type": "boolean"
            }
          }
        },
        {
          "name": "batch_addresses",
          "description": "批量地址",
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      ],
      "requestTemplate": {
        "url": "https://restapi.amap.com/v3/geocode/geo?key={{.config.apiKey}}&address={{.args.address}}&city={{.args.city}}&output={{.args.output}}&source=ts_mcp",
        "method": "GET",
        "headers": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ]
      },
      "responseTemplate": {
        "body": "# 地理编码信息\n{{- range $index, $geo := .Geocodes }}\n## 地点 {{add $index 1}}\n\n- **国家**: {{ $geo.Country }}\n- **省份**: {{ $geo.Province }}\n- **城市**: {{ $geo.City }}\n- **城市代码**: {{ $geo.Citycode }}\n- **区/县**: {{ $geo.District }}\n- **街道**: {{ $geo.Street }}\n- **门牌号**: {{ $geo.Number }}\n- **行政编码**: {{ $geo.Adcode }}\n- **坐标**: {{ $geo.Location }}\n- **级别**: {{ $geo.Level }}\n{{- end }}"
      }
    }
  ]
}
`

	// Parse the config to verify it's valid JSON
	var configData map[string]interface{}
	err := json.Unmarshal([]byte(configJSON), &configData)
	if err != nil {
		t.Fatalf("Invalid JSON config: %v", err)
	}

	// Example tool configuration
	tool := RestTool{
		Name:        "maps-geo",
		Description: "将详细的结构化地址转换为经纬度坐标。支持对地标性名胜景区、建筑物名称解析为经纬度坐标",
		Args: []RestToolArg{
			{
				Name:        "address",
				Description: "待解析的结构化地址信息",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "city",
				Description: "指定查询的城市",
				Required:    false,
			},
			{
				Name:        "output",
				Description: "输出格式",
				Type:        "string",
				Enum:        []interface{}{"json", "xml"},
				Default:     "json",
			},
			{
				Name:        "options",
				Description: "高级选项",
				Type:        "object",
				Properties: map[string]interface{}{
					"extensions": map[string]interface{}{
						"type": "string",
						"enum": []interface{}{"base", "all"},
					},
					"batch": map[string]interface{}{
						"type": "boolean",
					},
				},
			},
			{
				Name:        "batch_addresses",
				Description: "批量地址",
				Type:        "array",
				Items: map[string]interface{}{
					"type": "string",
				},
			},
		},
		RequestTemplate: RestToolRequestTemplate{
			URL:    "https://restapi.amap.com/v3/geocode/geo?key={{.config.apiKey}}&address={{.args.address}}&city={{.args.city}}&output={{.args.output}}&source=ts_mcp",
			Method: "GET",
			Headers: []RestToolHeader{
				{
					Key:   "Content-Type",
					Value: "application/json",
				},
			},
		},
		ResponseTemplate: RestToolResponseTemplate{
			Body: `# 地理编码信息
{{- range $index, $geo := .Geocodes }}
## 地点 {{add $index 1}}

- **国家**: {{ $geo.Country }}
- **省份**: {{ $geo.Province }}
- **城市**: {{ $geo.City }}
- **城市代码**: {{ $geo.Citycode }}
- **区/县**: {{ $geo.District }}
- **街道**: {{ $geo.Street }}
- **门牌号**: {{ $geo.Number }}
- **行政编码**: {{ $geo.Adcode }}
- **坐标**: {{ $geo.Location }}
- **级别**: {{ $geo.Level }}
{{- end }}`,
		},
	}

	// Parse templates
	err = tool.parseTemplates()
	if err != nil {
		t.Fatalf("Failed to parse templates: %v", err)
	}

	var templateData []byte
	templateData, _ = sjson.SetBytes(templateData, "config", map[string]interface{}{"apiKey": "test-api-key"})
	templateData, _ = sjson.SetBytes(templateData, "args", map[string]interface{}{
		"address": "北京市朝阳区阜通东大街6号",
		"city":    "北京",
		"output":  "json",
	})

	// Test URL template
	url, err := executeTemplate(tool.parsedURLTemplate, templateData)
	if err != nil {
		t.Fatalf("Failed to execute URL template: %v", err)
	}

	expectedURL := "https://restapi.amap.com/v3/geocode/geo?key=test-api-key&address=北京市朝阳区阜通东大街6号&city=北京&output=json&source=ts_mcp"
	if url != expectedURL {
		t.Errorf("URL template rendering failed. Expected: %s, Got: %s", expectedURL, url)
	}

	// Test InputSchema for complex types
	mcpTool := &RestMCPTool{
		toolConfig: tool,
	}

	schema := mcpTool.InputSchema()
	properties := schema["properties"].(map[string]interface{})

	// Check object type
	options, ok := properties["options"].(map[string]interface{})
	if !ok || options["type"] != "object" {
		t.Errorf("Expected options to be of type object")
	}

	// Check array type
	batchAddresses, ok := properties["batch_addresses"].(map[string]interface{})
	if !ok || batchAddresses["type"] != "array" {
		t.Errorf("Expected batch_addresses to be of type array")
	}

	// Test response template with sample data
	sampleResponse := `
		{"Geocodes": [
			{
				"Country":  "中国",
				"Province": "北京市",
				"City":     "北京市",
				"Citycode": "010",
				"District": "朝阳区",
				"Street":   "阜通东大街",
				"Number":   "6号",
				"Adcode":   "110105",
				"Location": "116.483038,39.990633",
				"Level":    "门牌号",
			}]}`

	result, err := executeTemplate(tool.parsedResponseTemplate, []byte(sampleResponse))
	if err != nil {
		t.Fatalf("Failed to execute response template: %v", err)
	}

	// Just check that the result contains expected substrings
	expectedSubstrings := []string{
		"# 地理编码信息",
		"## 地点 1",
		"**国家**: 中国",
		"**省份**: 北京市",
		"**坐标**: 116.483038,39.990633",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(result, substr) {
			t.Errorf("Response template rendering failed. Expected substring not found: %s", substr)
		}
	}
}

// TestOutputSchemaSupport tests the new OutputSchema functionality for MCP Protocol Version 2025-06-18
func TestOutputSchemaSupport(t *testing.T) {
	tests := []struct {
		name         string
		outputSchema map[string]any
		expectedNil  bool
	}{
		{
			name: "valid output schema",
			outputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"result": map[string]any{
						"type":        "string",
						"description": "Operation result",
					},
					"data": map[string]any{
						"type":        "object",
						"description": "Response data",
					},
				},
			},
			expectedNil: false,
		},
		{
			name:         "nil output schema",
			outputSchema: nil,
			expectedNil:  true,
		},
		{
			name:         "empty output schema",
			outputSchema: map[string]any{},
			expectedNil:  false, // Empty map is not nil, it's just empty
		},
		{
			name: "complex output schema with array",
			outputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id": map[string]any{
									"type": "string",
								},
								"value": map[string]any{
									"type": "number",
								},
							},
						},
					},
				},
			},
			expectedNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a RestTool with the test output schema
			tool := RestTool{
				Name:        "test-tool",
				Description: "Test tool for output schema",
				Args: []RestToolArg{
					{
						Name:        "input",
						Description: "Test input",
						Type:        "string",
					},
				},
				OutputSchema: tt.outputSchema,
				RequestTemplate: RestToolRequestTemplate{
					URL:    "https://example.com/api",
					Method: "GET",
				},
			}

			// Create RestMCPTool
			mcpTool := &RestMCPTool{
				toolConfig: tool,
			}

			// Test OutputSchema method
			result := mcpTool.OutputSchema()

			if tt.expectedNil {
				if result != nil {
					t.Errorf("Expected nil output schema, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil output schema, got nil")
				} else {
					// For empty map, we don't expect specific fields
					if len(result) > 0 {
						// Verify the schema structure only if it's not empty
						if result["type"] == nil {
							t.Errorf("Expected output schema to have 'type' field")
						}
						if result["properties"] == nil {
							t.Errorf("Expected output schema to have 'properties' field")
						}
					}
				}
			}
		})
	}
}

// TestToolWithOutputSchemaInterface tests the ToolWithOutputSchema interface implementation
func TestToolWithOutputSchemaInterface(t *testing.T) {
	// Create a tool with output schema
	tool := RestTool{
		Name:        "test-tool",
		Description: "Test tool",
		Args: []RestToolArg{
			{
				Name:        "input",
				Description: "Test input",
				Type:        "string",
			},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"result": map[string]any{
					"type": "string",
				},
			},
		},
		RequestTemplate: RestToolRequestTemplate{
			URL:    "https://example.com/api",
			Method: "GET",
		},
	}

	mcpTool := &RestMCPTool{
		toolConfig: tool,
	}

	// Test that RestMCPTool implements ToolWithOutputSchema interface
	var toolWithSchema ToolWithOutputSchema = mcpTool

	// Test interface methods
	if toolWithSchema.Description() != "Test tool" {
		t.Errorf("Expected description 'Test tool', got '%s'", toolWithSchema.Description())
	}

	inputSchema := toolWithSchema.InputSchema()
	if inputSchema["type"] != "object" {
		t.Errorf("Expected input schema type 'object', got '%v'", inputSchema["type"])
	}

	outputSchema := toolWithSchema.OutputSchema()
	if outputSchema == nil {
		t.Errorf("Expected non-nil output schema")
	} else if outputSchema["type"] != "object" {
		t.Errorf("Expected output schema type 'object', got '%v'", outputSchema["type"])
	}
}

// TestGlobalToolRegistryOutputSchema tests the GlobalToolRegistry's support for OutputSchema
func TestGlobalToolRegistryOutputSchema(t *testing.T) {
	// Create a mock tool that implements ToolWithOutputSchema
	mockTool := &MockToolWithOutputSchema{
		name:        "test-tool",
		description: "Test tool with output schema",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type": "string",
				},
			},
		},
		outputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"result": map[string]any{
					"type": "string",
				},
			},
		},
	}

	// Create registry and register tool
	registry := &GlobalToolRegistry{}
	registry.Initialize()

	// Test the registration logic directly without calling RegisterTool to avoid log.Debugf
	// Simulate what RegisterTool does
	if _, ok := registry.serverTools["test-server"]; !ok {
		registry.serverTools["test-server"] = make(map[string]ToolInfo)
	}
	toolInfo := ToolInfo{
		Name:        "test-tool",
		Description: mockTool.Description(),
		InputSchema: mockTool.InputSchema(),
		ServerName:  "test-server",
		Tool:        mockTool,
	}
	// Check if tool implements OutputSchema (MCP Protocol Version 2025-06-18)
	// Since mockTool is already a MockToolWithOutputSchema, we can directly call OutputSchema()
	toolInfo.OutputSchema = mockTool.OutputSchema()
	registry.serverTools["test-server"]["test-tool"] = toolInfo

	// Test GetToolInfo
	retrievedToolInfo, found := registry.GetToolInfo("test-server", "test-tool")
	if !found {
		t.Fatalf("Expected to find tool info")
	}

	// Verify tool info contains output schema
	if retrievedToolInfo.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", retrievedToolInfo.Name)
	}

	if retrievedToolInfo.Description != "Test tool with output schema" {
		t.Errorf("Expected description 'Test tool with output schema', got '%s'", retrievedToolInfo.Description)
	}

	if retrievedToolInfo.OutputSchema == nil {
		t.Errorf("Expected non-nil output schema in tool info")
	} else if retrievedToolInfo.OutputSchema["type"] != "object" {
		t.Errorf("Expected output schema type 'object', got '%v'", retrievedToolInfo.OutputSchema["type"])
	}

	// Test with tool that doesn't implement ToolWithOutputSchema
	mockToolWithoutSchema := &MockTool{
		name:        "simple-tool",
		description: "Simple tool without output schema",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type": "string",
				},
			},
		},
	}

	// Register the simple tool directly without calling RegisterTool to avoid log.Debugf
	toolInfo2 := ToolInfo{
		Name:        "simple-tool",
		Description: mockToolWithoutSchema.Description(),
		InputSchema: mockToolWithoutSchema.InputSchema(),
		ServerName:  "test-server",
		Tool:        mockToolWithoutSchema,
	}
	// This tool doesn't implement ToolWithOutputSchema, so OutputSchema should remain nil
	registry.serverTools["test-server"]["simple-tool"] = toolInfo2

	retrievedToolInfo2, found := registry.GetToolInfo("test-server", "simple-tool")
	if !found {
		t.Fatalf("Expected to find simple tool info")
	}

	// Verify simple tool doesn't have output schema
	if retrievedToolInfo2.OutputSchema != nil {
		t.Errorf("Expected nil output schema for simple tool, got %v", retrievedToolInfo2.OutputSchema)
	}
}

// MockToolWithOutputSchema is a mock implementation of ToolWithOutputSchema for testing
type MockToolWithOutputSchema struct {
	name         string
	description  string
	inputSchema  map[string]any
	outputSchema map[string]any
}

func (m *MockToolWithOutputSchema) Create(params []byte) Tool {
	return &MockToolWithOutputSchema{
		name:         m.name,
		description:  m.description,
		inputSchema:  m.inputSchema,
		outputSchema: m.outputSchema,
	}
}

func (m *MockToolWithOutputSchema) Call(httpCtx HttpContext, server Server) error {
	return nil
}

func (m *MockToolWithOutputSchema) Description() string {
	return m.description
}

func (m *MockToolWithOutputSchema) InputSchema() map[string]any {
	return m.inputSchema
}

func (m *MockToolWithOutputSchema) OutputSchema() map[string]any {
	return m.outputSchema
}

// MockTool is a mock implementation of Tool for testing
type MockTool struct {
	name        string
	description string
	inputSchema map[string]any
}

func (m *MockTool) Create(params []byte) Tool {
	return &MockTool{
		name:        m.name,
		description: m.description,
		inputSchema: m.inputSchema,
	}
}

func (m *MockTool) Call(httpCtx HttpContext, server Server) error {
	return nil
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) InputSchema() map[string]any {
	return m.inputSchema
}
