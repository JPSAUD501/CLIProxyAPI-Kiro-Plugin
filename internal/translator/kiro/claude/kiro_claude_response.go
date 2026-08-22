// Package claude translates Kiro responses to the Anthropic Messages shape.
package claude

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
)

// BuildClaudeResponse preserves text and tool arguments returned by Kiro. Kiro
// does not provide an Anthropic-verifiable thinking signature in this buffered
// response path, so the plugin does not manufacture a thinking block.
func BuildClaudeResponse(content string, toolUses []KiroToolUse, model string, usageInfo usage.Detail, stopReason string) []byte {
	contentBlocks := make([]map[string]interface{}, 0, len(toolUses)+1)
	if content != "" {
		contentBlocks = append(contentBlocks, map[string]interface{}{"type": "text", "text": content})
	}
	for _, toolUse := range toolUses {
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type":  "tool_use",
			"id":    toolUse.ToolUseID,
			"name":  toolUse.Name,
			"input": toolUse.Input,
		})
	}
	if len(contentBlocks) == 0 {
		contentBlocks = append(contentBlocks, map[string]interface{}{"type": "text", "text": ""})
	}
	stopReason = NormalizeStopReason(stopReason)
	if stopReason == "" {
		stopReason = "end_turn"
		if len(toolUses) > 0 {
			stopReason = "tool_use"
		}
	}

	response := map[string]interface{}{
		"id":          "msg_" + uuid.New().String()[:24],
		"type":        "message",
		"role":        "assistant",
		"model":       model,
		"content":     contentBlocks,
		"stop_reason": stopReason,
		"usage": map[string]interface{}{
			"input_tokens":  usageInfo.InputTokens,
			"output_tokens": usageInfo.OutputTokens,
		},
	}
	result, _ := json.Marshal(response)
	return result
}

// NormalizeStopReason converts Kiro's enum spelling to Anthropic's wire values.
func NormalizeStopReason(stopReason string) string {
	return strings.ToLower(strings.TrimSpace(stopReason))
}
