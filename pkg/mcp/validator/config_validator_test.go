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

package validator

import (
	"testing"
)

func TestValidateConfig_RestServer(t *testing.T) {
	// Test REST server configuration
	configJSON := `{
		"server": {
			"name": "test-rest-server",
			"config": {}
		},
		"tools": [
			{
				"name": "test-tool",
				"description": "A test tool",
				"args": [
					{
						"name": "input",
						"description": "Input parameter",
						"type": "string",
						"required": true
					}
				],
				"requestTemplate": {
					"url": "https://api.example.com/test",
					"method": "POST"
				},
				"responseTemplate": {
					"body": "{{.}}"
				}
			}
		]
	}`

	result, err := ValidateConfig(configJSON)
	if err != nil {
		t.Fatalf("ValidateConfig returned error: %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected config to be valid, but got invalid with error: %v", result.Error)
	}

	if result.ServerName != "test-rest-server" {
		t.Errorf("Expected server name 'test-rest-server', got '%s'", result.ServerName)
	}

	if result.IsComposed {
		t.Errorf("Expected single server (not composed), but got composed")
	}
}

func TestValidateConfig_ToolSet(t *testing.T) {
	// Test toolSet configuration
	configJSON := `{
		"toolSet": {
			"name": "test-toolset",
			"serverTools": [
				{
					"serverName": "server1",
					"tools": ["tool1", "tool2"]
				},
				{
					"serverName": "server2",
					"tools": ["tool3"]
				}
			]
		}
	}`

	result, err := ValidateConfig(configJSON)
	if err != nil {
		t.Fatalf("ValidateConfig returned error: %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected config to be valid, but got invalid with error: %v", result.Error)
	}

	if result.ServerName != "test-toolset" {
		t.Errorf("Expected server name 'test-toolset', got '%s'", result.ServerName)
	}

	if !result.IsComposed {
		t.Errorf("Expected composed server, but got single server")
	}
}

func TestValidateConfig_PreRegisteredServer(t *testing.T) {
	// Test pre-registered Go-based server configuration (should be skipped in validation)
	configJSON := `{
		"server": {
			"name": "some-go-server",
			"config": {
				"someParam": "value"
			}
		}
	}`

	result, err := ValidateConfig(configJSON)
	if err != nil {
		t.Fatalf("ValidateConfig returned error: %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected config to be valid (pre-registered servers should be skipped), but got invalid with error: %v", result.Error)
	}

	if result.ServerName != "some-go-server" {
		t.Errorf("Expected server name 'some-go-server', got '%s'", result.ServerName)
	}

	if result.IsComposed {
		t.Errorf("Expected single server (not composed), but got composed")
	}
}

func TestValidateConfig_InvalidConfig(t *testing.T) {
	// Test invalid configuration (missing required fields)
	configJSON := `{
		"server": {
			"config": {}
		}
	}`

	result, err := ValidateConfig(configJSON)
	if err != nil {
		t.Fatalf("ValidateConfig returned error: %v", err)
	}

	if result.IsValid {
		t.Errorf("Expected config to be invalid, but got valid")
	}

	if result.Error == nil {
		t.Errorf("Expected validation error, but got nil")
	}
}

func TestValidateConfig_MissingServerAndToolSet(t *testing.T) {
	// Test configuration missing both server and toolSet
	configJSON := `{
		"allowTools": ["tool1"]
	}`

	result, err := ValidateConfig(configJSON)
	if err != nil {
		t.Fatalf("ValidateConfig returned error: %v", err)
	}

	if result.IsValid {
		t.Errorf("Expected config to be invalid, but got valid")
	}

	if result.Error == nil {
		t.Errorf("Expected validation error, but got nil")
	}
}
