// Package common provides shared constants and utilities for Kiro translator.
package common

const (
	// KiroMaxToolDescLen is the maximum description length for Kiro API tools.
	// Kiro API limit is 10240 bytes, leave room for "..."
	KiroMaxToolDescLen = 10237

	// ToolCompressionTargetSize is the target total size for compressed tools (20KB).
	// If tools exceed this size, compression will be applied.
	ToolCompressionTargetSize = 20 * 1024 // 20KB

	// MinToolDescriptionLength is the minimum description length after compression.
	// Descriptions will not be shortened below this length.
	MinToolDescriptionLength = 50

	// ThinkingStartTag is the start tag for thinking blocks in responses.
	ThinkingStartTag = "<thinking>"

	// ThinkingEndTag is the end tag for thinking blocks in responses.
	ThinkingEndTag = "</thinking>"

	// CodeFenceMarker is the markdown code fence marker.
	CodeFenceMarker = "```"

	// AltCodeFenceMarker is the alternative markdown code fence marker.
	AltCodeFenceMarker = "~~~"

	// InlineCodeMarker is the markdown inline code marker (backtick).
	InlineCodeMarker = "`"

	// DefaultAssistantContentWithTools is the fallback content for assistant messages
	// that have tool_use but no text content. Kiro API requires non-empty content.
	// IMPORTANT: Use a minimal neutral string that the model won't mimic in responses.
	// Previously "I'll help you with that." which caused the model to parrot it back.
	DefaultAssistantContentWithTools = "."

	// DefaultAssistantContent is the fallback content for assistant messages
	// that have no content at all. Kiro API requires non-empty content.
	// IMPORTANT: Use a minimal neutral string that the model won't mimic in responses.
	// Previously "I understand." which could leak into model behavior.
	DefaultAssistantContent = "."

	// DefaultUserContentWithToolResults is the fallback content for user messages
	// that have only tool_result (no text). Kiro API requires non-empty content.
	DefaultUserContentWithToolResults = "Tool results provided."

	// DefaultUserContent is the fallback content for user messages
	// that have no content at all. Kiro API requires non-empty content.
	DefaultUserContent = "Continue"
)
