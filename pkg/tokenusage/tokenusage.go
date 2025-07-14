package tokenusage

import (
	"bytes"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	CtxKeyInputToken  = "input_token"
	CtxKeyOutputToken = "output_token"
	CtxKeyTotalToken  = "total_token"
	CtxKeyModel       = "model"
)

type TokenUsage struct {
	InputToken  int64
	OutputToken int64
	TotalToken  int64
	Model       string
}

func GetTokenUsage(ctx wrapper.HttpContext, data []byte) (u TokenUsage) {
	chunks := bytes.SplitSeq(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n"))
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
		} else if model, ok := ctx.GetUserAttribute(CtxKeyModel).(string); ok && model != "" {
			u.Model = model
		} else {
			u.Model = "unknow"
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
			inputToken, ok := ctx.GetUserAttribute(CtxKeyInputToken).(int64)
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

		if totalToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.total_tokens",            // completions , chatcompleations, images, audio, responses
			"response.usage.total_tokens",   // responses
			"usageMetadata.totalTokenCount", // Gemini GenerationContent
		}); totalToken != nil {
			u.TotalToken = totalToken.Int()
		} else {
			u.TotalToken = u.InputToken + u.OutputToken
		}
		ctx.SetUserAttribute(CtxKeyTotalToken, u.TotalToken)
	}
	return
}
