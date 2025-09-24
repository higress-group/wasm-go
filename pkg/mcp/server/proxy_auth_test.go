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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApiKeyAuthentication tests API key authentication forwarding
func TestApiKeyAuthentication(t *testing.T) {
	server := NewMcpProxyServer("auth-test")

	// Configure security scheme
	scheme := SecurityScheme{
		ID:                "ApiKeyAuth",
		Type:              "apiKey",
		In:                "header",
		Name:              "X-API-Key",
		DefaultCredential: "default-api-key",
	}

	server.AddSecurityScheme(scheme)

	config := McpProxyConfig{
		McpServerURL:    "http://secure-backend.example.com/mcp",
		Timeout:         5000,
		SecuritySchemes: []SecurityScheme{scheme},
	}

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)
	server.SetConfig(configBytes)

	// Create tool with security requirement
	toolConfig := McpProxyToolConfig{
		Name:        "secure_tool",
		Description: "Tool requiring authentication",
		Args: []ToolArg{
			{
				Name:        "data",
				Description: "Data parameter",
				Type:        "string",
				Required:    true,
			},
		},
		RequestTemplate: RequestTemplate{
			Security: SecurityConfig{
				ID: "ApiKeyAuth",
			},
		},
	}

	err = server.AddProxyTool(toolConfig)
	require.NoError(t, err)

	tool, exists := server.GetMCPTools()["secure_tool"]
	require.True(t, exists)

	params := map[string]interface{}{
		"data": "test data",
	}
	paramsBytes, err := json.Marshal(params)
	require.NoError(t, err)

	toolInstance := tool.Create(paramsBytes)
	require.NotNil(t, toolInstance)

	// Skip HttpContext-dependent test for now - will be tested in integration
	// Test authentication context preparation
	authCtx := &ProxyAuthContext{
		Headers: [][2]string{
			{"X-API-Key", "user-provided-key"},
		},
		RequestBody: []byte(`{"data": "test data"}`),
	}

	err = server.ExtractCredentials(authCtx, "ApiKeyAuth")
	assert.NoError(t, err)
	assert.Equal(t, "user-provided-key", authCtx.PassthroughCredential)
}

// TestBearerAuthentication tests Bearer token authentication
func TestBearerAuthentication(t *testing.T) {
	server := NewMcpProxyServer("bearer-auth-test")

	// Configure Bearer security scheme
	scheme := SecurityScheme{
		ID:     "BearerAuth",
		Type:   "http",
		Scheme: "bearer",
	}

	server.AddSecurityScheme(scheme)

	config := McpProxyConfig{
		McpServerURL:    "https://secure-backend.example.com/mcp",
		Timeout:         8000,
		SecuritySchemes: []SecurityScheme{scheme},
	}

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)
	server.SetConfig(configBytes)

	// Create tool with Bearer authentication
	toolConfig := McpProxyToolConfig{
		Name:        "bearer_tool",
		Description: "Tool with Bearer authentication",
		Args: []ToolArg{
			{
				Name:        "query",
				Description: "Query parameter",
				Type:        "string",
				Required:    true,
			},
		},
		RequestTemplate: RequestTemplate{
			Security: SecurityConfig{
				ID: "BearerAuth",
			},
		},
	}

	err = server.AddProxyTool(toolConfig)
	require.NoError(t, err)

	tool, exists := server.GetMCPTools()["bearer_tool"]
	require.True(t, exists)

	params := map[string]interface{}{
		"query": "test query",
	}
	paramsBytes, err := json.Marshal(params)
	require.NoError(t, err)

	toolInstance := tool.Create(paramsBytes)
	require.NotNil(t, toolInstance)

	// Skip HttpContext-dependent test for now - will be tested in integration
	// Test Bearer authentication context
	authCtx := &ProxyAuthContext{
		Headers: [][2]string{
			{"Authorization", "Bearer user-token-123"},
		},
		RequestBody: []byte(`{"query": "test query"}`),
	}

	err = server.ExtractCredentials(authCtx, "BearerAuth")
	assert.NoError(t, err)
	assert.Equal(t, "Bearer user-token-123", authCtx.PassthroughCredential)
}

// TestBasicAuthentication tests Basic authentication
func TestBasicAuthentication(t *testing.T) {
	server := NewMcpProxyServer("basic-auth-test")

	// Configure Basic security scheme
	scheme := SecurityScheme{
		ID:     "BasicAuth",
		Type:   "http",
		Scheme: "basic",
	}

	server.AddSecurityScheme(scheme)

	// Test tool call with Basic authentication
	toolConfig := McpProxyToolConfig{
		Name:        "basic_tool",
		Description: "Tool with Basic authentication",
		Args: []ToolArg{
			{
				Name:        "resource",
				Description: "Resource identifier",
				Type:        "string",
				Required:    true,
			},
		},
		RequestTemplate: RequestTemplate{
			Security: SecurityConfig{
				ID: "BasicAuth",
			},
		},
	}

	err := server.AddProxyTool(toolConfig)
	require.NoError(t, err)

	tool, exists := server.GetMCPTools()["basic_tool"]
	require.True(t, exists)

	params := map[string]interface{}{
		"resource": "test-resource",
	}
	paramsBytes, err := json.Marshal(params)
	require.NoError(t, err)

	toolInstance := tool.Create(paramsBytes)
	require.NotNil(t, toolInstance)

	// Skip HttpContext-dependent test for now - will be tested in integration
	// Test Basic authentication context
	authCtx := &ProxyAuthContext{
		Headers: [][2]string{
			{"Authorization", "Basic dXNlcjpwYXNzd29yZA=="}, // user:password
		},
		RequestBody: []byte(`{"resource": "test-resource"}`),
	}

	err = server.ExtractCredentials(authCtx, "BasicAuth")
	assert.NoError(t, err)
	assert.Equal(t, "Basic dXNlcjpwYXNzd29yZA==", authCtx.PassthroughCredential)
}

// TestCredentialPassthrough tests credential passthrough mechanism
func TestCredentialPassthrough(t *testing.T) {
	server := NewMcpProxyServer("passthrough-test")

	// Configure scheme with passthrough enabled
	scheme := SecurityScheme{
		ID:   "PassthroughAuth",
		Type: "apiKey",
		In:   "header",
		Name: "X-Custom-Auth",
	}

	server.AddSecurityScheme(scheme)

	// Test credential extraction and passthrough
	authContext := &ProxyAuthContext{
		Headers: [][2]string{
			{"X-Custom-Auth", "client-provided-token"},
			{"Content-Type", "application/json"},
		},
		RequestBody:           []byte(`{"test": "data"}`),
		PassthroughCredential: "",
	}

	err := server.ExtractCredentials(authContext, "PassthroughAuth")

	// This test will fail until credential extraction is implemented
	assert.NoError(t, err)
	assert.Equal(t, "client-provided-token", authContext.PassthroughCredential)
}

// TestDefaultCredentials tests default credential usage
func TestDefaultCredentials(t *testing.T) {
	server := NewMcpProxyServer("default-creds-test")

	// Configure scheme with default credential
	scheme := SecurityScheme{
		ID:                "DefaultAuth",
		Type:              "apiKey",
		In:                "header",
		Name:              "X-Service-Key",
		DefaultCredential: "default-service-key",
	}

	server.AddSecurityScheme(scheme)

	// Test default credential application when no client credential provided
	authContext := &ProxyAuthContext{
		Headers: [][2]string{
			{"Content-Type", "application/json"},
		},
		RequestBody:           []byte(`{"test": "data"}`),
		PassthroughCredential: "",
	}

	err := server.ApplyAuthentication(authContext, "DefaultAuth")

	// This test will fail until authentication application is implemented
	assert.NoError(t, err)

	// Verify default credential was applied
	foundHeader := false
	for _, header := range authContext.Headers {
		if header[0] == "X-Service-Key" && header[1] == "default-service-key" {
			foundHeader = true
			break
		}
	}
	assert.True(t, foundHeader)
}

// TestSecuritySchemeNotFound tests handling of missing security schemes
func TestSecuritySchemeNotFound(t *testing.T) {
	server := NewMcpProxyServer("missing-scheme-test")

	authContext := &ProxyAuthContext{
		Headers:               [][2]string{},
		RequestBody:           []byte(`{}`),
		PassthroughCredential: "",
	}

	err := server.ApplyAuthentication(authContext, "NonExistentScheme")

	// Should return error for missing security scheme
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security scheme not found")
}

// TestMultipleSecuritySchemes tests multiple security schemes in one server
func TestMultipleSecuritySchemes(t *testing.T) {
	server := NewMcpProxyServer("multi-auth-test")

	// Add multiple security schemes
	schemes := []SecurityScheme{
		{
			ID:   "ApiKeyAuth",
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
		{
			ID:     "BearerAuth",
			Type:   "http",
			Scheme: "bearer",
		},
	}

	for _, scheme := range schemes {
		server.AddSecurityScheme(scheme)
	}

	// Test that both schemes are available
	for _, scheme := range schemes {
		retrievedScheme, exists := server.GetSecurityScheme(scheme.ID)
		assert.True(t, exists)
		assert.Equal(t, scheme.ID, retrievedScheme.ID)
		assert.Equal(t, scheme.Type, retrievedScheme.Type)
	}
}

// ProxyAuthContext, RequestTemplate, SecurityConfig and authentication methods
// are now implemented in proxy_server.go
