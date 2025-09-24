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
	"net/url"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// McpProxyConfig represents the configuration for MCP proxy server
type McpProxyConfig struct {
	McpServerURL    string           `json:"mcpServerURL"`
	Timeout         int              `json:"timeout,omitempty"`
	SecuritySchemes []SecurityScheme `json:"securitySchemes,omitempty"`
}

// ToolArg represents an argument for a proxy tool
type ToolArg struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        string        `json:"type"`
	Required    bool          `json:"required"`
	Default     interface{}   `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
}

// McpProxyToolConfig represents a tool configuration for MCP proxy
type McpProxyToolConfig struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Args            []ToolArg       `json:"args"`
	RequestTemplate RequestTemplate `json:"requestTemplate,omitempty"`
}

// RequestTemplate defines request template configuration for proxy tools
type RequestTemplate struct {
	Security SecurityConfig `json:"security,omitempty"`
}

// SecurityConfig represents security configuration reference
type SecurityConfig struct {
	ID string `json:"id"`
}

// McpProxyServer implements Server interface for MCP-to-MCP proxy
type McpProxyServer struct {
	Name            string
	base            BaseMCPServer
	toolsConfig     map[string]McpProxyToolConfig
	securitySchemes map[string]SecurityScheme
}

// NewMcpProxyServer creates a new MCP proxy server
func NewMcpProxyServer(name string) *McpProxyServer {
	return &McpProxyServer{
		Name:            name,
		base:            NewBaseMCPServer(),
		toolsConfig:     make(map[string]McpProxyToolConfig),
		securitySchemes: make(map[string]SecurityScheme),
	}
}

// AddSecurityScheme adds a security scheme to the server's map
func (s *McpProxyServer) AddSecurityScheme(scheme SecurityScheme) {
	if s.securitySchemes == nil {
		s.securitySchemes = make(map[string]SecurityScheme)
	}
	s.securitySchemes[scheme.ID] = scheme
}

// GetSecurityScheme retrieves a security scheme by its ID from the map
func (s *McpProxyServer) GetSecurityScheme(id string) (SecurityScheme, bool) {
	scheme, ok := s.securitySchemes[id]
	return scheme, ok
}

// AddMCPTool implements Server interface
func (s *McpProxyServer) AddMCPTool(name string, tool Tool) Server {
	s.base.AddMCPTool(name, tool)
	return s
}

// AddProxyTool adds a proxy tool configuration
func (s *McpProxyServer) AddProxyTool(toolConfig McpProxyToolConfig) error {
	s.toolsConfig[toolConfig.Name] = toolConfig
	s.base.AddMCPTool(toolConfig.Name, &McpProxyTool{
		serverName: s.Name,
		name:       toolConfig.Name,
		toolConfig: toolConfig,
	})
	return nil
}

// GetMCPTools implements Server interface
func (s *McpProxyServer) GetMCPTools() map[string]Tool {
	return s.base.GetMCPTools()
}

// SetConfig implements Server interface
func (s *McpProxyServer) SetConfig(config []byte) {
	s.base.SetConfig(config)
}

// GetConfig implements Server interface
func (s *McpProxyServer) GetConfig(v any) {
	s.base.GetConfig(v)
}

// Clone implements Server interface
func (s *McpProxyServer) Clone() Server {
	newServer := &McpProxyServer{
		Name:            s.Name,
		base:            s.base.CloneBase(),
		toolsConfig:     make(map[string]McpProxyToolConfig),
		securitySchemes: make(map[string]SecurityScheme),
	}
	for k, v := range s.toolsConfig {
		newServer.toolsConfig[k] = v
	}
	// Deep copy securitySchemes
	if s.securitySchemes != nil {
		for k, v := range s.securitySchemes {
			newServer.securitySchemes[k] = v
		}
	}
	return newServer
}

// GetToolConfig returns the proxy tool configuration for a given tool name
func (s *McpProxyServer) GetToolConfig(name string) (McpProxyToolConfig, bool) {
	config, ok := s.toolsConfig[name]
	return config, ok
}

// ForwardToolsList forwards tools/list request to backend MCP server
func (s *McpProxyServer) ForwardToolsList(ctx HttpContext, cursor *string) error {
	wrapperCtx := ctx.(wrapper.HttpContext)

	// Get configuration
	var config McpProxyConfig
	s.GetConfig(&config)

	// Create protocol handler
	handler := NewMcpProtocolHandler(config.McpServerURL, config.Timeout)

	// This will handle initialization asynchronously if needed and use ActionPause/Resume
	return handler.ForwardToolsList(wrapperCtx, cursor)
}

// ExtractCredentials extracts credentials from the HTTP context
func (s *McpProxyServer) ExtractCredentials(ctx *ProxyAuthContext, schemeID string) error {
	scheme, exists := s.GetSecurityScheme(schemeID)
	if !exists {
		return fmt.Errorf("security scheme not found: %s", schemeID)
	}

	// Extract credentials based on scheme configuration
	switch scheme.Type {
	case "apiKey":
		for _, header := range ctx.Headers {
			if header[0] == scheme.Name {
				ctx.PassthroughCredential = header[1]
				return nil
			}
		}
	case "http":
		for _, header := range ctx.Headers {
			if header[0] == "Authorization" {
				ctx.PassthroughCredential = header[1]
				return nil
			}
		}
	}

	return nil
}

// ApplyAuthentication applies authentication to the proxy request
func (s *McpProxyServer) ApplyAuthentication(ctx *ProxyAuthContext, schemeID string) error {
	scheme, exists := s.GetSecurityScheme(schemeID)
	if !exists {
		return fmt.Errorf("security scheme not found: %s", schemeID)
	}

	credential := ctx.PassthroughCredential
	if credential == "" && scheme.DefaultCredential != "" {
		credential = scheme.DefaultCredential
	}

	if credential == "" {
		return fmt.Errorf("no credential available for scheme %s", schemeID)
	}

	// Apply authentication based on scheme type
	switch scheme.Type {
	case "apiKey":
		if scheme.In == "header" {
			// Add or update the header
			found := false
			for i, header := range ctx.Headers {
				if header[0] == scheme.Name {
					ctx.Headers[i] = [2]string{scheme.Name, credential}
					found = true
					break
				}
			}
			if !found {
				ctx.Headers = append(ctx.Headers, [2]string{scheme.Name, credential})
			}
		} else if scheme.In == "query" {
			// Add to query parameters (would require URL parsing)
			// For now, implement basic functionality
		}
	case "http":
		// Apply HTTP authentication
		found := false
		for i, header := range ctx.Headers {
			if header[0] == "Authorization" {
				ctx.Headers[i] = [2]string{"Authorization", credential}
				found = true
				break
			}
		}
		if !found {
			ctx.Headers = append(ctx.Headers, [2]string{"Authorization", credential})
		}
	}

	return nil
}

// ProxyAuthContext represents authentication context for proxy requests
type ProxyAuthContext struct {
	Headers               [][2]string
	ParsedURL             *url.URL
	RequestBody           []byte
	PassthroughCredential string
}

// McpProxyTool implements Tool interface for MCP-to-MCP proxy
type McpProxyTool struct {
	serverName string
	name       string
	toolConfig McpProxyToolConfig
	arguments  map[string]interface{}
}

// Create implements Tool interface
func (t *McpProxyTool) Create(params []byte) Tool {
	newTool := &McpProxyTool{
		serverName: t.serverName,
		name:       t.name,
		toolConfig: t.toolConfig,
		arguments:  make(map[string]interface{}),
	}

	if len(params) > 0 {
		json.Unmarshal(params, &newTool.arguments)
	}

	return newTool
}

// Call implements Tool interface - this is where the MCP protocol handling happens
func (t *McpProxyTool) Call(httpCtx HttpContext, server Server) error {
	ctx := httpCtx.(wrapper.HttpContext)

	// Get proxy server instance to access configuration
	proxyServer, ok := server.(*McpProxyServer)
	if !ok {
		return fmt.Errorf("server is not a McpProxyServer")
	}

	// Get configuration
	var config McpProxyConfig
	proxyServer.GetConfig(&config)

	// Create protocol handler
	handler := NewMcpProtocolHandler(config.McpServerURL, config.Timeout)

	// This will handle initialization asynchronously if needed and use ActionPause/Resume
	return handler.ForwardToolsCall(ctx, t.name, t.arguments)
}

// Description implements Tool interface
func (t *McpProxyTool) Description() string {
	return t.toolConfig.Description
}

// InputSchema implements Tool interface
func (t *McpProxyTool) InputSchema() map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]any)
	var required []string

	for _, arg := range t.toolConfig.Args {
		argSchema := map[string]any{
			"type":        arg.Type,
			"description": arg.Description,
		}

		if arg.Default != nil {
			argSchema["default"] = arg.Default
		}

		if len(arg.Enum) > 0 {
			argSchema["enum"] = arg.Enum
		}

		properties[arg.Name] = argSchema

		if arg.Required {
			required = append(required, arg.Name)
		}
	}

	schema["required"] = required
	return schema
}

// ValidateSecurityScheme validates a security scheme configuration
func ValidateSecurityScheme(scheme SecurityScheme) error {
	if scheme.ID == "" {
		return fmt.Errorf("security scheme ID is required")
	}

	if scheme.Type != "apiKey" && scheme.Type != "http" {
		return fmt.Errorf("invalid security scheme type: %s", scheme.Type)
	}

	if scheme.Type == "apiKey" {
		if scheme.Name == "" {
			return fmt.Errorf("security scheme name is required for apiKey type")
		}
		if scheme.In != "header" && scheme.In != "query" && scheme.In != "cookie" {
			return fmt.Errorf("invalid security scheme location: %s", scheme.In)
		}
	}

	if scheme.Type == "http" {
		if scheme.Scheme == "" {
			return fmt.Errorf("security scheme scheme is required for http type")
		}
	}

	return nil
}

// ValidateToolConfig validates a tool configuration
func ValidateToolConfig(config McpProxyToolConfig) error {
	if config.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	if config.Description == "" {
		return fmt.Errorf("tool description is required")
	}

	// Validate arguments
	argNames := make(map[string]bool)
	for _, arg := range config.Args {
		if arg.Name == "" {
			return fmt.Errorf("argument name is required")
		}

		if argNames[arg.Name] {
			return fmt.Errorf("duplicate argument name: %s", arg.Name)
		}
		argNames[arg.Name] = true

		if arg.Description == "" {
			return fmt.Errorf("argument description is required for %s", arg.Name)
		}

		validTypes := []string{"string", "number", "integer", "boolean", "array", "object"}
		validType := false
		for _, t := range validTypes {
			if arg.Type == t {
				validType = true
				break
			}
		}
		if !validType {
			return fmt.Errorf("invalid argument type %s for %s", arg.Type, arg.Name)
		}
	}

	return nil
}
