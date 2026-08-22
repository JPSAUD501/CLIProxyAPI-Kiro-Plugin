package modelcapabilities

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseEffortSchema(t *testing.T) {
	schema := json.RawMessage(`{"properties":{"output_config":{"properties":{"effort":{"type":"string","enum":["low","medium","high","xhigh","max"],"default":"xhigh"}}},"thinking":{"properties":{"type":{"enum":["adaptive","disabled"]}}},"max_tokens":{"type":"integer","minimum":1024,"maximum":128000}}}`)
	got := Parse("claude-opus-5", schema)
	if got.EffortPath != EffortPathOutputConfig || got.DefaultEffort != "xhigh" {
		t.Fatalf("unexpected capability: %#v", got)
	}
	want := []string{"low", "medium", "high", "xhigh", "max"}
	if !reflect.DeepEqual(got.EffortLevels, want) {
		t.Fatalf("levels = %#v, want %#v", got.EffortLevels, want)
	}
	if !got.SupportsMaxTokens || got.MinimumOutputTokens != 1024 || got.MaximumOutputTokens != 128000 {
		t.Fatalf("max token capability = %#v", got)
	}
	wantFields := map[string]any{
		"output_config": map[string]any{"effort": "high"},
		"max_tokens":    int64(4096),
	}
	if fields := got.AdditionalFieldsForRequest("high", 4096); !reflect.DeepEqual(fields, wantFields) {
		t.Fatalf("additional fields = %#v, want %#v", fields, wantFields)
	}
}

func TestParseDoesNotInferEffortFromThinkingOnly(t *testing.T) {
	schema := json.RawMessage(`{"properties":{"thinking":{"properties":{"type":{"enum":["adaptive","disabled"]}}}}}`)
	got := Parse("model", schema)
	if got.EffortPath != EffortPathNone || len(got.EffortLevels) != 0 {
		t.Fatalf("unexpected inferred effort: %#v", got)
	}
}

func TestAdditionalFieldsUsesDeclaredPath(t *testing.T) {
	capability := Capability{EffortPath: EffortPathReasoning, EffortLevels: []string{"none", "high"}}
	want := map[string]any{"reasoning": map[string]any{"effort": "high"}}
	if got := capability.AdditionalFields("high"); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields = %#v, want %#v", got, want)
	}
}
