package executor

import (
	"testing"

	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

func TestValidateKiroRequestAcceptsCompleteLongHistory(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"start"},{"role":"assistant","content":"answer"},{"role":"user","content":"continue"}],"tools":[{"name":"read_file","description":"Read a file","input_schema":{"type":"object"}}]}`)
	if err := validateKiroRequest(body, body, sdktranslator.FormatClaude); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
}

func TestValidateKiroRequestRejectsUnsupportedNativeControl(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"reply"}],"tool_choice":{"type":"any"}}`)
	if err := validateKiroRequest(body, body, sdktranslator.FormatClaude); err == nil {
		t.Fatal("unsupported native control was accepted")
	}
}

func TestValidateKiroRequestAcceptsCompleteToolLoop(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"run it"},{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"read_file","input":{"path":"README.md"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"ok"}]}],"tools":[{"name":"read_file","description":"Read a file","input_schema":{"type":"object"}}]}`)
	if err := validateKiroRequest(body, body, sdktranslator.FormatClaude); err != nil {
		t.Fatalf("complete tool loop rejected: %v", err)
	}
}

func TestValidateKiroRequestRejectsOrphanToolResult(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"missing","content":"ok"}]}]}`)
	if err := validateKiroRequest(body, body, sdktranslator.FormatClaude); err == nil {
		t.Fatal("orphan tool result was accepted")
	}
}
