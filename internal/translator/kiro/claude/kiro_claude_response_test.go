package claude

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
	"github.com/tidwall/gjson"
)

func TestBuildClaudeResponsePreservesSignedReasoningBeforeVisibleContent(t *testing.T) {
	t.Parallel()

	response := BuildClaudeResponse(
		"Answer",
		&KiroReasoningContent{ReasoningText: &KiroReasoningText{Text: "internal reasoning", Signature: "signed-by-upstream"}},
		nil,
		"claude-opus-5",
		usage.Detail{InputTokens: 10, OutputTokens: 5},
		"end_turn",
	)
	blocks := gjson.GetBytes(response, "content").Array()
	if len(blocks) != 2 {
		t.Fatalf("content blocks = %d; response=%s", len(blocks), response)
	}
	if blocks[0].Get("type").String() != "thinking" || blocks[0].Get("thinking").String() != "internal reasoning" || blocks[0].Get("signature").String() != "signed-by-upstream" {
		t.Fatalf("thinking block = %s", blocks[0].Raw)
	}
	if blocks[1].Get("type").String() != "text" || blocks[1].Get("text").String() != "Answer" {
		t.Fatalf("text block = %s", blocks[1].Raw)
	}
}
