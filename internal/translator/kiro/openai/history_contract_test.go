package openai

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBuildKiroPayloadUsesOfficialSystemPromptAndCompleteHistory(t *testing.T) {
	messages := make([]map[string]any, 0, 62)
	messages = append(messages, map[string]any{"role": "system", "content": "Keep answers short."})
	for i := 0; i < 30; i++ {
		messages = append(messages,
			map[string]any{"role": "user", "content": fmt.Sprintf("question %d", i)},
			map[string]any{"role": "assistant", "content": fmt.Sprintf("answer %d", i)},
		)
	}
	messages = append(messages, map[string]any{"role": "user", "content": "latest"})
	request, err := json.Marshal(map[string]any{"model": "claude-opus-5", "messages": messages})
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := BuildKiroPayloadFromOpenAI(request, "claude-opus-5", "profile", "AI_EDITOR", nil, nil)
	var payload KiroPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid payload: %v", err)
	}
	if payload.SystemPrompt == "" || payload.ConversationState.CurrentMessage.UserInputMessage.Content != "latest" {
		t.Fatalf("system prompt or current message was not mapped: %#v", payload)
	}
	if len(payload.ConversationState.History) != kiroMaxHistoryMessages {
		t.Fatalf("history length = %d, want %d", len(payload.ConversationState.History), kiroMaxHistoryMessages)
	}
	for i, message := range payload.ConversationState.History {
		if i%2 == 0 && message.UserInputMessage == nil {
			t.Fatalf("history[%d] must be a user message", i)
		}
		if i%2 == 1 && message.AssistantResponseMessage == nil {
			t.Fatalf("history[%d] must be an assistant message", i)
		}
	}
}

func TestSanitizeKiroHistoryKeepsOnlyAdjacentToolPairs(t *testing.T) {
	history := []KiroHistoryMessage{
		{UserInputMessage: &KiroUserInputMessage{Content: "run"}},
		{AssistantResponseMessage: &KiroAssistantResponseMessage{Content: "calling", ToolUses: []KiroToolUse{
			{ToolUseID: "matched", Name: "read"},
			{ToolUseID: "missing-result", Name: "write"},
		}}},
	}
	results := []KiroToolResult{{ToolUseID: "matched"}, {ToolUseID: "orphan"}}

	clean, current := sanitizeKiroHistory(history, results)
	if got := len(clean[1].AssistantResponseMessage.ToolUses); got != 1 {
		t.Fatalf("tool uses = %d, want 1", got)
	}
	if got := len(current); got != 1 || current[0].ToolUseID != "matched" {
		t.Fatalf("current tool results = %#v", current)
	}
}
