package openai

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/JPSAUD501/CLIProxyAPI-Kiro-Plugin/internal/modelcapabilities"
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

	raw, _ := BuildKiroPayloadFromOpenAI(request, "claude-opus-5", "profile", "AI_EDITOR", modelcapabilities.Capability{}, "")
	var payload KiroPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid payload: %v", err)
	}
	if payload.SystemPrompt == "" || payload.ConversationState.CurrentMessage.UserInputMessage.Content != "latest" {
		t.Fatalf("system prompt or current message was not mapped: %#v", payload)
	}
	if len(payload.ConversationState.History) != 60 {
		t.Fatalf("history length = %d, want all 60 messages", len(payload.ConversationState.History))
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
