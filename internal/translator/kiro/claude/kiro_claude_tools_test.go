package claude

import "testing"

func TestProcessToolUseEventRepresentsZeroArgumentCallAsEmptyObject(t *testing.T) {
	t.Parallel()

	event := map[string]interface{}{
		"toolUseEvent": map[string]interface{}{
			"toolUseId": "call_1",
			"name":      "get_number",
			"stop":      true,
		},
	}
	toolUses, state, err := ProcessToolUseEvent(event, nil, make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}
	if state != nil {
		t.Fatalf("completed tool state = %#v", state)
	}
	if len(toolUses) != 1 || toolUses[0].Input == nil || len(toolUses[0].Input) != 0 {
		t.Fatalf("tool uses = %#v", toolUses)
	}
}
