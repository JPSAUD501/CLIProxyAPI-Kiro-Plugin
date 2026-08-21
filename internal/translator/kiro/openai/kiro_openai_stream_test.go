package openai

import (
	"context"
	"testing"

	"github.com/tidwall/gjson"
)

func TestToolCallIndexesFollowToolBlocks(t *testing.T) {
	t.Parallel()

	chunks := [][]byte{
		[]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\"}}"),
		[]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"call_first\",\"name\":\"exec_command\",\"input\":{}}}"),
		[]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"cmd\\\":\\\"pwd\\\"}\"}}"),
		[]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}"),
		[]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":2,\"content_block\":{\"type\":\"tool_use\",\"id\":\"call_second\",\"name\":\"read_file\",\"input\":{}}}"),
		[]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":2,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\":\\\"README.md\\\"}\"}}"),
	}

	var state any
	var toolChunks []gjson.Result
	for _, chunk := range chunks {
		for _, output := range ConvertKiroStreamToOpenAI(context.Background(), "model", nil, nil, chunk, &state) {
			parsed := gjson.Parse(output)
			if parsed.Get("choices.0.delta.tool_calls.0").Exists() {
				toolChunks = append(toolChunks, parsed)
			}
		}
	}

	if len(toolChunks) != 4 {
		t.Fatalf("tool chunk count = %d, want 4", len(toolChunks))
	}
	wantIndexes := []int64{0, 0, 1, 1}
	for i, want := range wantIndexes {
		if got := toolChunks[i].Get("choices.0.delta.tool_calls.0.index").Int(); got != want {
			t.Fatalf("tool chunk %d index = %d, want %d", i, got, want)
		}
	}
	if got := toolChunks[1].Get("choices.0.delta.tool_calls.0.function.arguments").String(); got != "{\"cmd\":\"pwd\"}" {
		t.Fatalf("first tool arguments = %q", got)
	}
	if got := toolChunks[3].Get("choices.0.delta.tool_calls.0.function.arguments").String(); got != "{\"path\":\"README.md\"}" {
		t.Fatalf("second tool arguments = %q", got)
	}
}

func TestToolArgumentsWithoutStartAreNotEmitted(t *testing.T) {
	t.Parallel()

	var state any
	outputs := ConvertKiroStreamToOpenAI(
		context.Background(),
		"model",
		nil,
		nil,
		[]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":7,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"cmd\\\":\\\"pwd\\\"}\"}}"),
		&state,
	)
	if len(outputs) != 0 {
		t.Fatalf("orphan arguments produced %d chunks", len(outputs))
	}
}
