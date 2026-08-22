package modelcapabilities

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseEffortSchema(t *testing.T) {
	schema := json.RawMessage(`{"properties":{"output_config":{"properties":{"effort":{"type":"string","enum":["low","medium","high","xhigh","max"],"default":"xhigh"}}},"thinking":{"properties":{"type":{"enum":["adaptive","disabled"]}}}}}`)
	got := Parse("claude-opus-5", schema)
	if got.EffortPath != EffortPathOutputConfig || got.DefaultEffort != "xhigh" {
		t.Fatalf("unexpected capability: %#v", got)
	}
	want := []string{"low", "medium", "high", "xhigh", "max"}
	if !reflect.DeepEqual(got.EffortLevels, want) {
		t.Fatalf("levels = %#v, want %#v", got.EffortLevels, want)
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
