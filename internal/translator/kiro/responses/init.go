// Package responses registers native OpenAI Responses translation for Kiro.
package responses

import (
	"context"

	kiroopenai "github.com/JPSAUD501/CLIProxyAPI-Kiro-Plugin/internal/translator/kiro/openai"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/translator/builtin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var registry = builtin.Registry()

type streamState struct {
	claudeToOpenAI    any
	openAIToResponses any
}

func init() {
	kiroFormat := sdktranslator.FromString("kiro")
	sdktranslator.Register(
		sdktranslator.FormatOpenAIResponse,
		kiroFormat,
		convertResponsesRequestToClaude,
		sdktranslator.ResponseTransform{
			Stream: func(ctx context.Context, model string, originalRequest, request, response []byte, param *any) [][]byte {
				return convertClaudeStreamToResponses(ctx, model, originalRequest, request, response, param)
			},
			NonStream: func(ctx context.Context, model string, originalRequest, request, response []byte, param *any) []byte {
				return convertClaudeNonStreamToResponses(ctx, model, originalRequest, request, response)
			},
		},
	)
}

func convertClaudeStreamToResponses(ctx context.Context, model string, originalRequest, claudeRequest, response []byte, param *any) [][]byte {
	state, ok := (*param).(*streamState)
	if !ok {
		state = &streamState{}
		*param = state
	}

	openAIRequest := registry.TranslateRequest(
		sdktranslator.FormatOpenAIResponse,
		sdktranslator.FormatOpenAI,
		model,
		originalRequest,
		true,
	)
	openAIStrings := kiroopenai.ConvertKiroStreamToOpenAI(ctx, model, openAIRequest, claudeRequest, response, &state.claudeToOpenAI)

	var outputs [][]byte
	for _, chunk := range openAIStrings {
		outputs = append(outputs, registry.TranslateStream(
			ctx,
			sdktranslator.FormatOpenAI,
			sdktranslator.FormatOpenAIResponse,
			model,
			originalRequest,
			openAIRequest,
			[]byte(chunk),
			&state.openAIToResponses,
		)...)
	}

	// The plugin host adds Chat Completions' terminal marker after the
	// translator returns, so the Kiro-to-OpenAI translator intentionally does
	// not emit it. Responses translation happens one layer earlier and needs
	// that marker now: the built-in OpenAI-to-Responses translator defers
	// response.completed until it receives data: [DONE].
	eventType, _ := kiroopenai.ParseClaudeEvent(response)
	if eventType == "message_stop" {
		outputs = append(outputs, registry.TranslateStream(
			ctx,
			sdktranslator.FormatOpenAI,
			sdktranslator.FormatOpenAIResponse,
			model,
			originalRequest,
			openAIRequest,
			[]byte("data: [DONE]"),
			&state.openAIToResponses,
		)...)
	}
	return outputs
}

func convertClaudeNonStreamToResponses(ctx context.Context, model string, originalRequest, claudeRequest, response []byte) []byte {
	openAIRequest := registry.TranslateRequest(
		sdktranslator.FormatOpenAIResponse,
		sdktranslator.FormatOpenAI,
		model,
		originalRequest,
		false,
	)
	openAIResponse := []byte(kiroopenai.ConvertKiroNonStreamToOpenAI(ctx, model, openAIRequest, claudeRequest, response, nil))
	return registry.TranslateNonStream(
		ctx,
		sdktranslator.FormatOpenAI,
		sdktranslator.FormatOpenAIResponse,
		model,
		originalRequest,
		openAIRequest,
		openAIResponse,
		nil,
	)
}

func convertResponsesRequestToClaude(model string, body []byte, stream bool) []byte {
	body = normalizeResponsesStringInput(body)
	return registry.TranslateRequest(
		sdktranslator.FormatOpenAIResponse,
		sdktranslator.FormatClaude,
		model,
		body,
		stream,
	)
}

func normalizeResponsesStringInput(body []byte) []byte {
	input := gjson.GetBytes(body, "input")
	if input.Type != gjson.String {
		return body
	}
	message := []byte(`[{"role":"user","content":[{"type":"input_text","text":""}]}]`)
	message, err := sjson.SetBytes(message, "0.content.0.text", input.String())
	if err != nil {
		return body
	}
	normalized, err := sjson.SetRawBytes(body, "input", message)
	if err != nil {
		return body
	}
	return normalized
}
