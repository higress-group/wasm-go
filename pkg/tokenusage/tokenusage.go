package tokenusage

import (
	"bytes"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

type TokenUsage struct {
	InputToken  int64
	OutputToken int64
	TotalToken  int64
	Model       string
}

func GetTokenUsage(data []byte) (u TokenUsage) {
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
			"modelVersion",   // Gemini GenerateContent
		}); model != nil {
			u.Model = model.String()
		} else {
			u.Model = "unknown"
		}
		if inputToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.prompt_tokens",            // completions , chatcompleations
			"usage.input_tokens",             // images, audio
			"response.usage.input_tokens",    // responses
			"usageMetadata.promptTokenCount", // Gemini GenerateContent
		}); inputToken != nil {
			u.InputToken = inputToken.Int()
		}
		if outputToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.completion_tokens",            // completions , chatcompleations
			"usage.output_tokens",                // images, audio
			"response.usage.output_tokens",       // responses
			"usageMetadata.candidatesTokenCount", // Gemini GeneratenContent
		}); outputToken != nil {
			u.OutputToken = outputToken.Int()
		}

		if totalToken := wrapper.GetValueFromBody(chunk, []string{
			"usage.total_tokens",            // completions , chatcompleations, images, audio, responses
			"response.usage.total_tokens",   // responses
			"usageMetadata.totalTokenCount", // Gemini GenerationContent
		}); totalToken != nil {
			u.TotalToken = totalToken.Int()
		} else {
			u.TotalToken = u.InputToken + u.OutputToken
		}
	}
	return
}
