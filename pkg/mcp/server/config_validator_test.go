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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// TestMcpProxyConfigValidation tests configuration validation for mcp-proxy servers
func TestMcpProxyConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid basic proxy config",
			config: `{
				"server": {
					"name": "test-proxy",
					"type": "mcp-proxy",
					"mcpServerURL": "http://backend.example.com/mcp",
					"timeout": 5000
				},
				"tools": [
					{
						"name": "test-tool",
						"description": "Test tool",
						"args": [
							{
								"name": "input",
								"description": "Input parameter",
								"type": "string",
								"required": true
							}
						]
					}
				]
			}`,
			shouldErr: false,
		},
		{
			name: "proxy config with security schemes",
			config: `{
				"server": {
					"name": "secure-proxy",
					"type": "mcp-proxy",
					"mcpServerURL": "https://secure.example.com/mcp",
					"timeout": 8000,
					"securitySchemes": [
						{
							"id": "ApiKeyAuth",
							"type": "apiKey",
							"in": "header",
							"name": "X-API-Key",
							"defaultCredential": "test-key"
						}
					]
				},
				"tools": [
					{
						"name": "secure-tool",
						"description": "Secure tool",
						"args": [
							{
								"name": "data",
								"description": "Data parameter",
								"type": "object",
								"required": true
							}
						],
						"requestTemplate": {
							"security": {
								"id": "ApiKeyAuth"
							}
						}
					}
				]
			}`,
			shouldErr: false,
		},
		{
			name: "missing mcpServerURL should fail",
			config: `{
				"server": {
					"name": "invalid-proxy",
					"type": "mcp-proxy",
					"timeout": 5000
				},
				"tools": [
					{
						"name": "test-tool",
						"description": "Test tool",
						"args": []
					}
				]
			}`,
			shouldErr: true,
			errMsg:    "mcpServerURL is required",
		},
		{
			name: "invalid server type should use default REST handling",
			config: `{
				"server": {
					"name": "rest-server",
					"type": "rest-api"
				},
				"tools": [
					{
						"name": "rest-tool",
						"description": "REST tool",
						"args": []
					}
				]
			}`,
			shouldErr: false, // Should fall back to REST server logic
		},
		{
			name: "missing tools should fail",
			config: `{
				"server": {
					"name": "proxy-no-tools",
					"type": "mcp-proxy",
					"mcpServerURL": "http://backend.example.com/mcp"
				}
			}`,
			shouldErr: true,
			errMsg:    "tools array is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configJson := gjson.Parse(tt.config)
			config := &McpServerConfig{}

			// Create validation options (similar to validator package)
			toolRegistry := &GlobalToolRegistry{}
			toolRegistry.Initialize()

			opts := &ConfigOptions{
				Servers:                  make(map[string]Server),
				ToolRegistry:             toolRegistry,
				SkipPreRegisteredServers: true,
			}

			err := ParseConfigCore(configJson, config, opts)

			if tt.shouldErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
			}
		})
	}
}

// TestSecuritySchemeValidation tests security scheme configuration validation
func TestSecuritySchemeValidation(t *testing.T) {
	tests := []struct {
		name      string
		scheme    SecurityScheme
		shouldErr bool
	}{
		{
			name: "valid API key scheme",
			scheme: SecurityScheme{
				ID:   "ApiKeyAuth",
				Type: "apiKey",
				In:   "header",
				Name: "X-API-Key",
			},
			shouldErr: false,
		},
		{
			name: "valid HTTP bearer scheme",
			scheme: SecurityScheme{
				ID:     "BearerAuth",
				Type:   "http",
				Scheme: "bearer",
			},
			shouldErr: false,
		},
		{
			name: "invalid scheme - missing ID",
			scheme: SecurityScheme{
				Type: "apiKey",
				In:   "header",
				Name: "X-API-Key",
			},
			shouldErr: true,
		},
		{
			name: "invalid scheme - missing Name for apiKey",
			scheme: SecurityScheme{
				ID:   "ApiKeyAuth",
				Type: "apiKey",
				In:   "header",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will test the validation logic once SecurityScheme validation is implemented
			err := ValidateSecurityScheme(tt.scheme)

			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestToolConfigValidation tests tool configuration validation
func TestToolConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		toolCfg   McpProxyToolConfig
		shouldErr bool
	}{
		{
			name: "valid tool config",
			toolCfg: McpProxyToolConfig{
				Name:        "valid-tool",
				Description: "A valid tool",
				Args: []ToolArg{
					{
						Name:        "param1",
						Description: "Parameter 1",
						Type:        "string",
						Required:    true,
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "invalid tool - missing name",
			toolCfg: McpProxyToolConfig{
				Description: "Tool without name",
				Args:        []ToolArg{},
			},
			shouldErr: true,
		},
		{
			name: "invalid tool - empty description",
			toolCfg: McpProxyToolConfig{
				Name:        "tool-no-desc",
				Description: "",
				Args:        []ToolArg{},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolConfig(tt.toolCfg)

			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// These validation functions are now implemented in proxy_server.go
