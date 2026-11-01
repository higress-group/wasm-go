// Copyright (c) 2026 Alibaba Group Holding Ltd.
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
	"bytes"
	"strconv"
	"testing"

	"github.com/higress-group/wasm-go/pkg/iface"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/require"
)

// mockHttpContext implements the small part of wrapper.HttpContext used by
// tokenusage. The embedded interface keeps this test focused on context state
// rather than unrelated HTTP host calls.
type mockHttpContext struct {
	iface.HttpContext
	contexts   map[string]interface{}
	attributes map[string]interface{}
}

func newMockHttpContext() *mockHttpContext {
	return &mockHttpContext{
		contexts:   make(map[string]interface{}),
		attributes: make(map[string]interface{}),
	}
}

func (m *mockHttpContext) SetContext(key string, value interface{}) {
	m.contexts[key] = value
}

func (m *mockHttpContext) GetContext(key string) interface{} {
	return m.contexts[key]
}

func (m *mockHttpContext) GetBoolContext(key string, defaultValue bool) bool {
	if value, ok := m.contexts[key].(bool); ok {
		return value
	}
	return defaultValue
}

func (m *mockHttpContext) GetStringContext(key, defaultValue string) string {
	if value, ok := m.contexts[key].(string); ok {
		return value
	}
	return defaultValue
}

func (m *mockHttpContext) GetByteSliceContext(key string, defaultValue []byte) []byte {
	if value, ok := m.contexts[key].([]byte); ok {
		return value
	}
	return defaultValue
}

func (m *mockHttpContext) GetUserAttribute(key string) interface{} {
	return m.attributes[key]
}

func (m *mockHttpContext) SetUserAttribute(key string, value interface{}) {
	m.attributes[key] = value
}

func (m *mockHttpContext) SetUserAttributeMap(attributes map[string]interface{}) {
	m.attributes = attributes
}

func responsesCompletedSSE(model string, input, output, total int64) []byte {
	return []byte("event: response.completed\n" +
		"data: {\"response\":{\"model\":\"" + model + "\",\"usage\":{" +
		"\"input_tokens\":" + strconv.FormatInt(input, 10) + "," +
		"\"output_tokens\":" + strconv.FormatInt(output, 10) + "," +
		"\"total_tokens\":" + strconv.FormatInt(total, 10) +
		"}}}\n\n")
}

func requireUsage(t *testing.T, usage TokenUsage, model string, input, output, total int64) {
	t.Helper()
	require.Equal(t, model, usage.Model)
	require.Equal(t, input, usage.InputToken)
	require.Equal(t, output, usage.OutputToken)
	require.Equal(t, total, usage.TotalToken)
}

func TestGetTokenUsageResponsesCompleted(t *testing.T) {
	ctx := newMockHttpContext()
	body := responsesCompletedSSE("deepseek-v4", 123, 45, 168)

	usage := GetTokenUsage(ctx, body)

	requireUsage(t, usage, "deepseek-v4", 123, 45, 168)
}

func TestGetTokenUsageResponsesCompletedAcrossCallbacks(t *testing.T) {
	ctx := newMockHttpContext()
	body := responsesCompletedSSE("deepseek-v4", 123, 45, 168)
	split := bytes.Index(body, []byte(`"input_tokens"`))
	require.Greater(t, split, 0)

	first := body[:split]
	second := body[split:]
	firstUsage := GetTokenUsage(ctx, first)
	require.Equal(t, int64(0), firstUsage.TotalToken)
	require.True(t, ctx.GetBoolContext(ctxKeyDeltaBeginning, false))

	usage := GetTokenUsage(ctx, second)

	requireUsage(t, usage, "deepseek-v4", 123, 45, 168)
	require.False(t, ctx.GetBoolContext(ctxKeyDeltaBeginning, false))
	require.Nil(t, ctx.GetContext(ctxKeyDeltaSSEMessage))
}

func TestGetTokenUsageIgnoresEmptyStreamingCallback(t *testing.T) {
	ctx := newMockHttpContext()
	pending := []byte("event: response.completed\ndata: partial")
	ctx.SetContext(ctxKeyDeltaBeginning, true)
	ctx.SetContext(ctxKeyDeltaSSEMessage, pending)

	usage := GetTokenUsage(ctx, nil)

	require.Equal(t, int64(0), usage.TotalToken)
	require.True(t, ctx.GetBoolContext(ctxKeyDeltaBeginning, false))
	require.Equal(t, pending, ctx.GetByteSliceContext(ctxKeyDeltaSSEMessage, nil))
}

func TestGetTokenUsageResponsesCompletedCRLF(t *testing.T) {
	ctx := newMockHttpContext()
	body := bytes.ReplaceAll(responsesCompletedSSE("deepseek-v4", 123, 45, 168), []byte("\n"), []byte("\r\n"))

	usage := GetTokenUsage(ctx, body)

	requireUsage(t, usage, "deepseek-v4", 123, 45, 168)
}

func TestGetTokenUsageNonStreamingJSON(t *testing.T) {
	ctx := newMockHttpContext()
	body := []byte(`{"model":"gpt-4o","usage":{"prompt_tokens":10,"completion_tokens":4,"total_tokens":14}}`)

	usage := GetTokenUsage(ctx, body)

	requireUsage(t, usage, "gpt-4o", 10, 4, 14)
}

func TestMergeLargeResponseAPIChunksBoundsCache(t *testing.T) {
	ctx := newMockHttpContext()
	marker := []byte("event: response.completed\n")
	payload := bytes.Repeat([]byte("x"), OpenAIResponseSSECacheLimit+1024)

	mergeLargeResponseAPIChunks(ctx, marker)
	mergeLargeResponseAPIChunks(ctx, payload)

	cached := ctx.GetByteSliceContext(ctxKeyDeltaSSEMessage, nil)
	require.Len(t, cached, OpenAIResponseSSECacheLimit)
	require.Equal(t, payload[len(payload)-OpenAIResponseSSECacheLimit:], cached)
}

func TestGetTokenUsageDoesNotMergeOtherSSEEvents(t *testing.T) {
	ctx := newMockHttpContext()
	body := []byte("event: response.created\ndata: {\"response\":{\"model\":\"deepseek-v4\"}}\n\n")

	usage := GetTokenUsage(ctx, body)

	require.Equal(t, int64(0), usage.TotalToken)
	require.False(t, ctx.GetBoolContext(ctxKeyDeltaBeginning, false))
	require.Nil(t, ctx.GetContext(ctxKeyDeltaSSEMessage))
}

var _ wrapper.HttpContext = (*mockHttpContext)(nil)
