package tokenusage

import (
	"testing"

	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/require"
)

type testHTTPContext struct {
	wrapper.HttpContext
	values         map[string]interface{}
	userAttributes map[string]interface{}
}

func newTestHTTPContext() *testHTTPContext {
	return &testHTTPContext{
		values:         make(map[string]interface{}),
		userAttributes: make(map[string]interface{}),
	}
}

func (c *testHTTPContext) SetContext(key string, value interface{}) {
	c.values[key] = value
}

func (c *testHTTPContext) GetContext(key string) interface{} {
	return c.values[key]
}

func (c *testHTTPContext) GetBoolContext(key string, defaultValue bool) bool {
	value, ok := c.values[key].(bool)
	if !ok {
		return defaultValue
	}
	return value
}

func (c *testHTTPContext) GetStringContext(key, defaultValue string) string {
	value, ok := c.values[key].(string)
	if !ok {
		return defaultValue
	}
	return value
}

func (c *testHTTPContext) GetByteSliceContext(key string, defaultValue []byte) []byte {
	value, ok := c.values[key].([]byte)
	if !ok {
		return defaultValue
	}
	return value
}

func (c *testHTTPContext) GetUserAttribute(key string) interface{} {
	return c.userAttributes[key]
}

func (c *testHTTPContext) SetUserAttribute(key string, value interface{}) {
	c.userAttributes[key] = value
}

func TestGetTokenUsageSplitResponsesCompletedEvent(t *testing.T) {
	ctx := newTestHTTPContext()
	chunks := [][]byte{
		[]byte("event: response.completed\n"),
		[]byte(`data: {"response":{"id":"resp_test","model":"test-model","instructions":"large payload"`),
		[]byte(`,"usage":{"input_tokens":75907,`),
		[]byte(`"output_tokens":199,"total_tokens":76106}}`),
		[]byte(`,"sequence_number":63,"type":"response.completed"}` + "\n\n"),
	}

	var got TokenUsage
	for _, chunk := range chunks {
		usage := GetTokenUsage(ctx, chunk)
		if usage.TotalToken > 0 {
			got = usage
		}
	}

	require.Equal(t, int64(75907), got.InputToken)
	require.Equal(t, int64(199), got.OutputToken)
	require.Equal(t, int64(76106), got.TotalToken)
	require.Equal(t, "test-model", got.Model)
}
