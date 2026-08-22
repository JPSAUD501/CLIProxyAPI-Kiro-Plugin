package executor

import (
	"testing"

	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestResponsesRequestBuildsValidKiroPayload(t *testing.T) {
	t.Parallel()

	original := []byte("{\"model\":\"claude-opus-5\",\"instructions\":\"Be concise.\",\"input\":[{\"role\":\"user\",\"content\":[{\"type\":\"input_text\",\"text\":\"Run pwd\"}]}],\"tools\":[{\"type\":\"function\",\"name\":\"exec_command\",\"description\":\"Run a command\",\"parameters\":{\"type\":\"object\",\"properties\":{\"cmd\":{\"type\":\"string\"}},\"required\":[\"cmd\"]}}]}")
	intermediate := sdktranslator.TranslateRequest(
		sdktranslator.FormatOpenAIResponse,
		sdktranslator.FromString("kiro"),
		"claude-opus-5",
		original,
		false,
	)
	payload, _ := buildKiroPayloadForFormat(
		intermediate,
		"claude-opus-5",
		"arn:aws:codewhisperer:us-east-1:123456789012:profile/test",
		"AI_EDITOR",
		sdktranslator.FormatOpenAIResponse,
		nil,
	)

	parsed := gjson.ParseBytes(payload)
	if got := parsed.Get("conversationState.currentMessage.userInputMessage.content").String(); got != "Run pwd" {
		t.Fatalf("current message = %q; payload=%s", got, payload)
	}
	if got := parsed.Get("conversationState.currentMessage.userInputMessage.userInputMessageContext.tools.0.toolSpecification.name").String(); got != "exec_command" {
		t.Fatalf("tool name = %q; payload=%s", got, payload)
	}
	if got := parsed.Get("conversationState.currentMessage.userInputMessage.origin").String(); got != "AI_EDITOR" {
		t.Fatalf("origin = %q; payload=%s", got, payload)
	}
	if got := parsed.Get("profileArn").String(); got == "" {
		t.Fatalf("profileArn is empty; payload=%s", payload)
	}
}
