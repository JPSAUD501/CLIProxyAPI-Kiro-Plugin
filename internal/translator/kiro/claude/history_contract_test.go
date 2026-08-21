package claude

import "testing"

func TestSanitizeKiroHistoryRepairsCompactedToolTurn(t *testing.T) {
	history := []KiroHistoryMessage{
		{AssistantResponseMessage: &KiroAssistantResponseMessage{Content: "orphan assistant"}},
		{UserInputMessage: &KiroUserInputMessage{Content: "run"}},
		{AssistantResponseMessage: &KiroAssistantResponseMessage{Content: "calling", ToolUses: []KiroToolUse{
			{ToolUseID: "matched", Name: "read"},
			{ToolUseID: "missing-result", Name: "write"},
		}}},
	}
	results := []KiroToolResult{{ToolUseID: "matched"}, {ToolUseID: "orphan"}}

	clean, current := sanitizeKiroHistory(history, results)
	if len(clean) != 2 || clean[0].UserInputMessage == nil || clean[1].AssistantResponseMessage == nil {
		t.Fatalf("history contract was not repaired: %#v", clean)
	}
	if got := len(clean[1].AssistantResponseMessage.ToolUses); got != 1 {
		t.Fatalf("tool uses = %d, want 1", got)
	}
	if got := len(current); got != 1 || current[0].ToolUseID != "matched" {
		t.Fatalf("current tool results = %#v", current)
	}
}
