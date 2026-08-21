package claude

import (
	"context"
	"testing"

	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

func TestAnthropicToolStreamPreservesCanonicalEvents(t *testing.T) {
	t.Parallel()

	events := [][]byte{
		BuildClaudeContentBlockStartEvent(0, "tool_use", "tooluse_1", "exec_command"),
		BuildClaudeInputJsonDeltaEvent("{\"cmd\":\"pwd\"}", 0),
		BuildClaudeContentBlockStopEvent(0),
	}
	var state any
	for _, event := range events {
		outputs := sdktranslator.TranslateStream(
			context.Background(),
			sdktranslator.FromString("kiro"),
			sdktranslator.FormatClaude,
			"claude-opus-5",
			nil,
			nil,
			event,
			&state,
		)
		if len(outputs) != 1 {
			t.Fatalf("output count = %d, want 1", len(outputs))
		}
		if string(outputs[0]) != string(event) {
			t.Fatalf("Anthropic event changed\nwant: %s\ngot:  %s", event, outputs[0])
		}
	}
}
