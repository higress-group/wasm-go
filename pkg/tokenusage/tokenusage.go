package tokenusage

import (
	"bytes"
	"slices"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	CtxKeyInputToken         = "input_token"
	CtxKeyInputTokenDetails  = "input_token_details"
	CtxKeyOutputToken        = "output_token"
	CtxKeyOutputTokenDetails = "output_token_details"
	CtxKeyTotalToken         = "total_token"
	CtxKeyModel              = "model"
	CtxKeyRequestModel       = "request_model"
)

type TokenUsage struct {
	InputToken         int64
	InputTokenDetails  map[string]int64
	OutputTokenDetails map[string]int64
	OutputToken        int64
	TotalToken         int64
	Model              string

	// Anthropic Messages
	AnthropicCacheCreationInputToken int64
	AnthropicCacheReadInputToken     int64
}

func GetTokenUsage(ctx wrapper.HttpContext, data []byte) TokenUsage {
	chunks := bytes.SplitSeq(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n"))
	u := TokenUsage{
		InputTokenDetails:  make(map[string]int64),
		OutputTokenDetails: make(map[string]int64),
	}
	for chunk := range chunks {
		// the feature strings are used to identify the usage data, like:
		// {"model":"gpt2","usage":{"prompt_tokens":1,"completion_tokens":1}}

		if !bytes.Contains(chunk, []byte(`"usage"`)) && !bytes.Contains(chunk, []byte(`"usageMetadata"`)) {
			continue
		}

		if model := wrapper.GetValueFromBody(chunk, []string{
			"model",
			"response.model", // responses
			"message.model",  // anthropic messages
			"modelVersion",   // Gemini GenerateContent
		}); model != nil {
			u.Model = model.String()
		} else if model, ok := ctx.GetUserAttribute(CtxKeyModel).(string); ok && !slices.Contains([]string{"", "unknown"}, model) { // anthropic messages
			u.Model = model
		} else if model := ctx.GetStringContext(CtxKeyRequestModel, ""); model != "" { // Openai Image Generate
			u.Model = model
		} else {
			u.Model = "unknown"
		}
		ctx.SetUserAttribute(CtxKeyModel, u.Model)

		if inputToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.prompt_tokens",            // completions , chatcompleations
			"usage.input_tokens",             // images, audio
			"response.usage.input_tokens",    // responses
			"usageMetadata.promptTokenCount", // Gemini GenerateContent
			"message.usage.input_tokens",     // Anthrophic messages
		}); inputToken != nil {
			u.InputToken = inputToken.Int()
		} else {
			inputToken, ok := ctx.GetUserAttribute(CtxKeyInputToken).(int64) // anthropic messages
			if ok && inputToken > 0 {
				u.InputToken = inputToken
			}
		}
		ctx.SetUserAttribute(CtxKeyInputToken, u.InputToken)

		if outputToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.completion_tokens",            // completions , chatcompleations
			"usage.output_tokens",                // images, audio
			"response.usage.output_tokens",       // responses
			"usageMetadata.candidatesTokenCount", // Gemini GeneratenContent
			// "message.usage.output_tokens",        // Anthropic messages
		}); outputToken != nil {
			u.OutputToken = outputToken.Int()
		} else {
			outputToken, ok := ctx.GetUserAttribute(CtxKeyOutputToken).(int64)
			if ok && outputToken > 0 {
				u.OutputToken = outputToken
			}
		}
		ctx.SetUserAttribute(CtxKeyOutputToken, u.OutputToken)

		if inputTokensDetails := wrapper.GetValueFromBody(chunk, []string{
			"usage.prompt_tokens_details",         // chatcompletions
			"response.usage.input_tokens_details", // responses
			"usage.input_tokens_details",          // Doubao
			"usageMetadata.promptTokensDetails",   // Gemini GenerateContent
		}); inputTokensDetails != nil && inputTokensDetails.IsObject() {
			for key, value := range inputTokensDetails.Map() {
				u.InputTokenDetails[key] = value.Int()
			}
		}
		if geminiCachedContentTokenCount := wrapper.GetValueFromBody(data, []string{"usageMetadata.cachedContentTokenCount"}); geminiCachedContentTokenCount != nil {
			u.InputTokenDetails["cachedContentTokenCount"] = geminiCachedContentTokenCount.Int()
		}
		if geminiToolUsePromptTokenCount := wrapper.GetValueFromBody(data, []string{"usageMetadata.toolUsePromptTokenCount"}); geminiToolUsePromptTokenCount != nil {
			u.InputTokenDetails["toolUsePromptTokenCount"] = geminiToolUsePromptTokenCount.Int()
		}
		ctx.SetUserAttribute(CtxKeyInputTokenDetails, u.InputTokenDetails)

		if outputTokensDetails := wrapper.GetValueFromBody(chunk, []string{
			"usage.completion_tokens_details",       // completions , chatcompleations
			"response.usage.output_tokens_details",  // responses
			"usage.output_tokens_details",           // doubao
			"usageMetadata.candidatesTokensDetails", // Gemini GenerateContent
		}); outputTokensDetails != nil && outputTokensDetails.IsObject() {
			for key, val := range outputTokensDetails.Map() {
				u.OutputTokenDetails[key] = val.Int()
			}
		}
		// Gemini GenerateContent
		if geminiThoughtsTokenCount := wrapper.GetValueFromBody(data, []string{"usageMetadata.thoughtsTokenCount"}); geminiThoughtsTokenCount != nil {
			u.OutputTokenDetails["thoughtsTokenCount"] = geminiThoughtsTokenCount.Int()
		}
		// Doubao Image Generate
		if doubaoGeneratedImages := wrapper.GetValueFromBody(data, []string{"usage.generated_images"}); doubaoGeneratedImages != nil {
			u.OutputTokenDetails["generated_images"] = doubaoGeneratedImages.Int()
		}
		ctx.SetUserAttribute(CtxKeyOutputTokenDetails, u.OutputTokenDetails)

		// Anthropic Messages
		if cacheCreationInputToken := wrapper.GetValueFromBody(chunk, []string{"usage.cache_creation_input_tokens"}); cacheCreationInputToken != nil {
			u.AnthropicCacheCreationInputToken = cacheCreationInputToken.Int()
			u.InputTokenDetails["cache_creation_input_tokens"] = cacheCreationInputToken.Int()
		}
		if cacheReadInputToken := wrapper.GetValueFromBody(chunk, []string{"usage.cache_read_input_tokens"}); cacheReadInputToken != nil {
			u.AnthropicCacheReadInputToken = cacheReadInputToken.Int()
			u.InputTokenDetails["cache_read_input_tokens"] = cacheReadInputToken.Int()
		}

		if totalToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.total_tokens",            // completions , chatcompleations, images, audio, responses
			"response.usage.total_tokens",   // responses
			"usageMetadata.totalTokenCount", // Gemini GenerationContent
		}); totalToken != nil {
			u.TotalToken = totalToken.Int()
		} else {
			u.TotalToken = u.InputToken + u.OutputToken + u.AnthropicCacheCreationInputToken + u.AnthropicCacheReadInputToken
		}
		ctx.SetUserAttribute(CtxKeyTotalToken, u.TotalToken)
	}
	return u
}
