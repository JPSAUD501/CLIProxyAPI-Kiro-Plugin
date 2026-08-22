package executor

import "testing"

func TestCountClaudeInputTokensGrowsWithConversation(t *testing.T) {
	short, err := countClaudeInputTokens([]byte(`{"messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	long, err := countClaudeInputTokens([]byte(`{"system":"be concise","messages":[{"role":"user","content":"hello"},{"role":"assistant","content":"hello there"},{"role":"user","content":"continue with more detail"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if short <= 0 || long <= short {
		t.Fatalf("local counts short=%d long=%d", short, long)
	}
}
