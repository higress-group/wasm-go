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

package tokenusage

import (
	"testing"

	"github.com/higress-group/wasm-go/pkg/iface"
	"github.com/stretchr/testify/require"
)

type tokenUsageTestContext struct {
	iface.HttpContext
	context        map[string]interface{}
	userAttributes map[string]interface{}
}

func newTokenUsageTestContext() *tokenUsageTestContext {
	return &tokenUsageTestContext{
		context:        make(map[string]interface{}),
		userAttributes: make(map[string]interface{}),
	}
}

func (c *tokenUsageTestContext) SetContext(key string, value interface{}) {
	c.context[key] = value
}

func (c *tokenUsageTestContext) GetByteSliceContext(key string, defaultValue []byte) []byte {
	value, ok := c.context[key].([]byte)
	if !ok {
		return defaultValue
	}
	return value
}

func (c *tokenUsageTestContext) GetUserAttribute(key string) interface{} {
	return c.userAttributes[key]
}

func (c *tokenUsageTestContext) SetUserAttribute(key string, value interface{}) {
	c.userAttributes[key] = value
}

func TestGetTokenUsageMergesFragmentedResponsesCompletedEvent(t *testing.T) {
	ctx := newTokenUsageTestContext()
	fragments := [][]byte{
		[]byte(`{"response":{"id":"resp_123","model":"gpt-5.6-sol","tools":[{"parameters":{"properties":{"model":{"description":"not the response model"}}}}],`),
		[]byte(`"usage":{"input_tokens":10,"output_tokens":2,"total_tokens":12}},"sequence_number":1,`),
		[]byte("\"type\":\"response.completed\"}\n\n"),
	}

	var usage TokenUsage
	for _, fragment := range fragments {
		usage = GetTokenUsage(ctx, fragment)
	}

	require.Equal(t, "gpt-5.6-sol", usage.Model)
	require.EqualValues(t, 10, usage.InputToken)
	require.EqualValues(t, 2, usage.OutputToken)
	require.EqualValues(t, 12, usage.TotalToken)
}
