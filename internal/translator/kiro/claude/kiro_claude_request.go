// Package claude provides request translation functionality for Claude API to Kiro format.
// It handles parsing and transforming Claude API requests into the Kiro/Amazon Q API format,
// extracting model information, system instructions, message contents, and tool declarations.
package claude

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/JPSAUD501/CLIProxyAPI-Kiro-Plugin/internal/modelcapabilities"
	kirocommon "github.com/JPSAUD501/CLIProxyAPI-Kiro-Plugin/internal/translator/kiro/common"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// remoteWebSearchDescription is a minimal fallback for when dynamic fetch from MCP tools/list hasn't completed yet.
const remoteWebSearchDescription = "WebSearch looks up information outside the model's training data. Supports multiple queries to gather comprehensive information."

// Kiro API request structs - field order determines JSON key order

// KiroPayload is the top-level request structure for Kiro API
type KiroPayload struct {
	ConversationState            KiroConversationState `json:"conversationState"`
	ProfileArn                   string                `json:"profileArn,omitempty"`
	AdditionalModelRequestFields map[string]any        `json:"additionalModelRequestFields,omitempty"`
	SystemPrompt                 string                `json:"systemPrompt,omitempty"`
}

// KiroConversationState holds the conversation context
type KiroConversationState struct {
	ChatTriggerType string               `json:"chatTriggerType"` // Required: "MANUAL" - must be first field
	ConversationID  string               `json:"conversationId"`
	CurrentMessage  KiroCurrentMessage   `json:"currentMessage"`
	History         []KiroHistoryMessage `json:"history,omitempty"`
}

// KiroCurrentMessage wraps the current user message
type KiroCurrentMessage struct {
	UserInputMessage KiroUserInputMessage `json:"userInputMessage"`
}

// KiroHistoryMessage represents a message in the conversation history
type KiroHistoryMessage struct {
	UserInputMessage         *KiroUserInputMessage         `json:"userInputMessage,omitempty"`
	AssistantResponseMessage *KiroAssistantResponseMessage `json:"assistantResponseMessage,omitempty"`
}

// KiroImage represents an image in Kiro API format
type KiroImage struct {
	Format string          `json:"format"`
	Source KiroImageSource `json:"source"`
}

// KiroImageSource contains the image data
type KiroImageSource struct {
	Bytes string `json:"bytes"` // base64 encoded image data
}

// KiroUserInputMessage represents a user message
type KiroUserInputMessage struct {
	Content                 string                       `json:"content"`
	ModelID                 string                       `json:"modelId"`
	Origin                  string                       `json:"origin"`
	Images                  []KiroImage                  `json:"images,omitempty"`
	UserInputMessageContext *KiroUserInputMessageContext `json:"userInputMessageContext,omitempty"`
}

// KiroUserInputMessageContext contains tool-related context
type KiroUserInputMessageContext struct {
	ToolResults []KiroToolResult  `json:"toolResults,omitempty"`
	Tools       []KiroToolWrapper `json:"tools,omitempty"`
}

// KiroToolResult represents a tool execution result
type KiroToolResult struct {
	Content   []KiroTextContent `json:"content"`
	Status    string            `json:"status"`
	ToolUseID string            `json:"toolUseId"`
}

// KiroTextContent represents text content
type KiroTextContent struct {
	Text string `json:"text"`
}

// KiroToolWrapper wraps a tool specification
type KiroToolWrapper struct {
	ToolSpecification KiroToolSpecification `json:"toolSpecification"`
}

// KiroToolSpecification defines a tool's schema
type KiroToolSpecification struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema KiroInputSchema `json:"inputSchema"`
}

// KiroInputSchema wraps the JSON schema for tool input
type KiroInputSchema struct {
	JSON interface{} `json:"json"`
}

// KiroAssistantResponseMessage represents an assistant message
type KiroAssistantResponseMessage struct {
	Content  string        `json:"content"`
	ToolUses []KiroToolUse `json:"toolUses,omitempty"`
}

// KiroToolUse represents a tool invocation by the assistant
type KiroToolUse struct {
	ToolUseID      string                 `json:"toolUseId"`
	Name           string                 `json:"name"`
	Input          map[string]interface{} `json:"input"`
	IsTruncated    bool                   `json:"-"` // Internal flag, not serialized
	TruncationInfo *TruncationInfo        `json:"-"` // Truncation details, not serialized
}

// ConvertClaudeRequestToKiro converts a Claude API request to Kiro format.
// This is the main entry point for request translation.
func ConvertClaudeRequestToKiro(modelName string, inputRawJSON []byte, stream bool) []byte {
	// For Kiro, we pass through the Claude format since buildKiroPayload
	// expects Claude format and does the conversion internally.
	// The actual conversion happens in the executor when building the HTTP request.
	return inputRawJSON
}

// BuildKiroPayload constructs the Kiro API request payload from Claude format.
// Supports tool calling - tools are passed via userInputMessageContext.
// origin parameter determines which quota to use: "CLI" for Amazon Q, "AI_EDITOR" for Kiro IDE.
// Returns the payload and whether reasoning events are expected.
func BuildKiroPayload(claudeBody []byte, modelID, profileArn, origin string, capability modelcapabilities.Capability, effort string) ([]byte, bool) {
	// Normalize origin value for Kiro API compatibility
	origin = normalizeOrigin(origin)
	log.Debugf("kiro: normalized origin value: %s", origin)

	messages := gjson.GetBytes(claudeBody, "messages")

	tools := gjson.GetBytes(claudeBody, "tools")

	// Extract system prompt
	systemPrompt := extractSystemPrompt(claudeBody)

	thinkingEnabled := effort != "" && effort != "none"

	// Handle tool_choice parameter - Kiro doesn't support it natively, so we inject system prompt hints
	// Claude tool_choice values: {"type": "auto/any/tool", "name": "..."}
	toolChoiceHint := extractClaudeToolChoiceHint(claudeBody)
	if toolChoiceHint != "" {
		if systemPrompt != "" {
			systemPrompt += "\n"
		}
		systemPrompt += toolChoiceHint
		log.Debugf("kiro: injected tool_choice hint into system prompt")
	}

	// Convert Claude tools to Kiro format
	kiroTools := convertClaudeToolsToKiro(tools)

	// Process messages and build history
	history, currentUserMsg, currentToolResults := processMessages(messages, modelID, origin)
	if len(kiroTools) == 0 {
		clearClaudeToolState(history)
		currentToolResults = nil
	}

	// Build the current user content. Reasoning configuration remains a
	// top-level upstream field and is not mixed into conversation text.
	if currentUserMsg != nil {
		currentUserMsg.Content = buildFinalContent(currentUserMsg.Content, "", currentToolResults)

		// Deduplicate currentToolResults
		currentToolResults = deduplicateToolResults(currentToolResults)

		// Build userInputMessageContext with tools and tool results
		if len(kiroTools) > 0 || len(currentToolResults) > 0 {
			currentUserMsg.UserInputMessageContext = &KiroUserInputMessageContext{
				Tools:       kiroTools,
				ToolResults: currentToolResults,
			}
		}
	}

	// Build payload
	var currentMessage KiroCurrentMessage
	if currentUserMsg != nil {
		currentMessage = KiroCurrentMessage{UserInputMessage: *currentUserMsg}
	} else {
		currentMessage = KiroCurrentMessage{UserInputMessage: KiroUserInputMessage{
			Content: "Continue",
			ModelID: modelID,
			Origin:  origin,
		}}
	}

	payload := KiroPayload{
		ConversationState: KiroConversationState{
			ChatTriggerType: "MANUAL",
			ConversationID:  uuid.New().String(),
			CurrentMessage:  currentMessage,
			History:         history,
		},
		ProfileArn:                   profileArn,
		AdditionalModelRequestFields: capability.AdditionalFields(effort),
		SystemPrompt:                 systemPrompt,
	}

	result, err := json.Marshal(payload)
	if err != nil {
		log.Debugf("kiro: failed to marshal payload: %v", err)
		return nil, false
	}

	return result, thinkingEnabled
}

func clearClaudeToolState(history []KiroHistoryMessage) {
	for i := range history {
		if assistant := history[i].AssistantResponseMessage; assistant != nil {
			assistant.ToolUses = nil
		}
		if user := history[i].UserInputMessage; user != nil && user.UserInputMessageContext != nil {
			user.UserInputMessageContext.ToolResults = nil
			if len(user.UserInputMessageContext.Tools) == 0 {
				user.UserInputMessageContext = nil
			}
		}
	}
}

// normalizeOrigin normalizes origin value for Kiro API compatibility
func normalizeOrigin(origin string) string {
	switch origin {
	case "KIRO_CLI":
		return "CLI"
	case "KIRO_AI_EDITOR":
		return "AI_EDITOR"
	case "AMAZON_Q":
		return "CLI"
	case "KIRO_IDE":
		return "AI_EDITOR"
	default:
		return origin
	}
}

// extractSystemPrompt extracts system prompt from Claude request
func extractSystemPrompt(claudeBody []byte) string {
	systemField := gjson.GetBytes(claudeBody, "system")
	if systemField.IsArray() {
		var sb strings.Builder
		for _, block := range systemField.Array() {
			if block.Get("type").String() == "text" {
				sb.WriteString(block.Get("text").String())
			} else if block.Type == gjson.String {
				sb.WriteString(block.String())
			}
		}
		return sb.String()
	}
	return systemField.String()
}

// shortenToolNameIfNeeded shortens tool names that exceed 64 characters.
// MCP tools often have long names like "mcp__server-name__tool-name".
// This preserves the "mcp__" prefix and last segment when possible.
func shortenToolNameIfNeeded(name string) string {
	const limit = 64
	if len(name) <= limit {
		return name
	}
	// For MCP tools, try to preserve prefix and last segment
	if strings.HasPrefix(name, "mcp__") {
		idx := strings.LastIndex(name, "__")
		if idx > 0 {
			cand := "mcp__" + name[idx+2:]
			if len(cand) > limit {
				return cand[:limit]
			}
			return cand
		}
	}
	return name[:limit]
}

func ensureKiroInputSchema(parameters interface{}) interface{} {
	if parameters != nil {
		return parameters
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// convertClaudeToolsToKiro converts Claude tools to Kiro format
func convertClaudeToolsToKiro(tools gjson.Result) []KiroToolWrapper {
	var kiroTools []KiroToolWrapper
	if !tools.IsArray() {
		return kiroTools
	}

	for _, tool := range tools.Array() {
		name := tool.Get("name").String()
		description := tool.Get("description").String()
		inputSchemaResult := tool.Get("input_schema")
		var inputSchema interface{}
		if inputSchemaResult.Exists() && inputSchemaResult.Type != gjson.Null {
			inputSchema = inputSchemaResult.Value()
		}
		inputSchema = ensureKiroInputSchema(inputSchema)

		// Shorten tool name if it exceeds 64 characters (common with MCP tools)
		originalName := name
		name = shortenToolNameIfNeeded(name)
		if name != originalName {
			log.Debugf("kiro: shortened tool name from '%s' to '%s'", originalName, name)
		}

		// CRITICAL FIX: Kiro API requires non-empty description
		if strings.TrimSpace(description) == "" {
			description = fmt.Sprintf("Tool: %s", name)
			log.Debugf("kiro: tool '%s' has empty description, using default: %s", name, description)
		}

		// Rename web_search → remote_web_search for Kiro API compatibility
		if name == "web_search" {
			name = "remote_web_search"
			// Prefer dynamically fetched description, fall back to hardcoded constant
			if cached := GetWebSearchDescription(); cached != "" {
				description = cached
			} else {
				description = remoteWebSearchDescription
			}
			log.Debugf("kiro: renamed tool web_search → remote_web_search")
		}

		// Truncate long descriptions (individual tool limit)
		if len(description) > kirocommon.KiroMaxToolDescLen {
			truncLen := kirocommon.KiroMaxToolDescLen - 30
			for truncLen > 0 && !utf8.RuneStart(description[truncLen]) {
				truncLen--
			}
			description = description[:truncLen] + "... (description truncated)"
		}

		kiroTools = append(kiroTools, KiroToolWrapper{
			ToolSpecification: KiroToolSpecification{
				Name:        name,
				Description: description,
				InputSchema: KiroInputSchema{JSON: inputSchema},
			},
		})
	}

	// Apply dynamic compression if total tools size exceeds threshold
	// This prevents 500 errors when Claude Code sends too many tools
	kiroTools = compressToolsIfNeeded(kiroTools)

	return kiroTools
}

// processMessages processes Claude messages and builds Kiro history
func processMessages(messages gjson.Result, modelID, origin string) ([]KiroHistoryMessage, *KiroUserInputMessage, []KiroToolResult) {
	var history []KiroHistoryMessage
	var currentUserMsg *KiroUserInputMessage
	var currentToolResults []KiroToolResult

	// Merge adjacent messages with the same role
	messagesArray := kirocommon.MergeAdjacentMessages(messages.Array())

	// FIX: Kiro API requires history to start with a user message.
	// Some clients (e.g., OpenClaw) send conversations starting with an assistant message,
	// which is valid for the Claude API but causes "Improperly formed request" on Kiro.
	// Prepend a placeholder user message so the history alternation is correct.
	if len(messagesArray) > 0 && messagesArray[0].Get("role").String() == "assistant" {
		placeholder := `{"role":"user","content":"."}`
		messagesArray = append([]gjson.Result{gjson.Parse(placeholder)}, messagesArray...)
		log.Infof("kiro: messages started with assistant role, prepended placeholder user message for Kiro API compatibility")
	}

	for i, msg := range messagesArray {
		role := msg.Get("role").String()
		isLastMessage := i == len(messagesArray)-1

		if role == "user" {
			userMsg, toolResults := BuildUserMessageStruct(msg, modelID, origin)
			// CRITICAL: Kiro API requires content to be non-empty for ALL user messages
			// This includes both history messages and the current message.
			// When user message contains only tool_result (no text), content will be empty.
			// This commonly happens in compaction requests from OpenCode.
			if strings.TrimSpace(userMsg.Content) == "" {
				if len(toolResults) > 0 {
					userMsg.Content = kirocommon.DefaultUserContentWithToolResults
				} else {
					userMsg.Content = kirocommon.DefaultUserContent
				}
				log.Debugf("kiro: user content was empty, using default: %s", userMsg.Content)
			}
			if isLastMessage {
				currentUserMsg = &userMsg
				currentToolResults = toolResults
			} else {
				// For history messages, embed tool results in context
				if len(toolResults) > 0 {
					userMsg.UserInputMessageContext = &KiroUserInputMessageContext{
						ToolResults: toolResults,
					}
				}
				history = append(history, KiroHistoryMessage{
					UserInputMessage: &userMsg,
				})
			}
		} else if role == "assistant" {
			assistantMsg := BuildAssistantMessageStruct(msg)
			if isLastMessage {
				history = append(history, KiroHistoryMessage{
					AssistantResponseMessage: &assistantMsg,
				})
				// Create a "Continue" user message as currentMessage
				currentUserMsg = &KiroUserInputMessage{
					Content: "Continue",
					ModelID: modelID,
					Origin:  origin,
				}
			} else {
				history = append(history, KiroHistoryMessage{
					AssistantResponseMessage: &assistantMsg,
				})
			}
		}
	}

	history, currentToolResults = sanitizeKiroHistory(history, currentToolResults)

	return history, currentUserMsg, currentToolResults
}

const kiroMaxHistoryMessages = 50

// sanitizeKiroHistory keeps only complete alternating turns and binds tool
// results to tool uses from the immediately preceding assistant message.
func sanitizeKiroHistory(history []KiroHistoryMessage, currentResults []KiroToolResult) ([]KiroHistoryMessage, []KiroToolResult) {
	clean := make([]KiroHistoryMessage, 0, len(history))
	expectUser := true
	for _, message := range history {
		if expectUser {
			if message.UserInputMessage == nil || message.AssistantResponseMessage != nil {
				continue
			}
			clean = append(clean, message)
			expectUser = false
			continue
		}
		if message.AssistantResponseMessage == nil || message.UserInputMessage != nil {
			continue
		}
		clean = append(clean, message)
		expectUser = true
	}
	if len(clean)%2 != 0 {
		clean = clean[:len(clean)-1]
	}
	if len(clean) > kiroMaxHistoryMessages {
		start := len(clean) - kiroMaxHistoryMessages
		if start%2 != 0 {
			start++
		}
		clean = clean[start:]
	}

	for i := 1; i < len(clean); i += 2 {
		assistant := clean[i].AssistantResponseMessage
		var results []KiroToolResult
		if i+1 < len(clean) {
			results = toolResultsFromClaudeUser(clean[i+1].UserInputMessage)
		} else {
			results = currentResults
		}
		resultIDs := make(map[string]struct{}, len(results))
		for _, result := range results {
			if result.ToolUseID != "" {
				resultIDs[result.ToolUseID] = struct{}{}
			}
		}
		seen := make(map[string]struct{}, len(assistant.ToolUses))
		toolUses := assistant.ToolUses[:0]
		for _, toolUse := range assistant.ToolUses {
			if toolUse.ToolUseID == "" || strings.TrimSpace(toolUse.Name) == "" {
				continue
			}
			if _, duplicate := seen[toolUse.ToolUseID]; duplicate {
				continue
			}
			if _, matched := resultIDs[toolUse.ToolUseID]; !matched {
				continue
			}
			seen[toolUse.ToolUseID] = struct{}{}
			toolUses = append(toolUses, toolUse)
		}
		assistant.ToolUses = toolUses
	}

	var previousToolUses map[string]struct{}
	for i := range clean {
		if assistant := clean[i].AssistantResponseMessage; assistant != nil {
			previousToolUses = make(map[string]struct{}, len(assistant.ToolUses))
			for _, toolUse := range assistant.ToolUses {
				previousToolUses[toolUse.ToolUseID] = struct{}{}
			}
			continue
		}
		filterClaudeUserToolResults(clean[i].UserInputMessage, previousToolUses)
		previousToolUses = nil
	}
	currentResults = filterClaudeToolResults(currentResults, previousToolUses)
	return clean, currentResults
}

func toolResultsFromClaudeUser(user *KiroUserInputMessage) []KiroToolResult {
	if user == nil || user.UserInputMessageContext == nil {
		return nil
	}
	return user.UserInputMessageContext.ToolResults
}

func filterClaudeUserToolResults(user *KiroUserInputMessage, allowed map[string]struct{}) {
	if user == nil || user.UserInputMessageContext == nil {
		return
	}
	user.UserInputMessageContext.ToolResults = filterClaudeToolResults(user.UserInputMessageContext.ToolResults, allowed)
	if len(user.UserInputMessageContext.ToolResults) == 0 && len(user.UserInputMessageContext.Tools) == 0 {
		user.UserInputMessageContext = nil
	}
}

func filterClaudeToolResults(results []KiroToolResult, allowed map[string]struct{}) []KiroToolResult {
	filtered := make([]KiroToolResult, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		if result.ToolUseID == "" {
			continue
		}
		if _, ok := allowed[result.ToolUseID]; !ok {
			continue
		}
		if _, duplicate := seen[result.ToolUseID]; duplicate {
			continue
		}
		seen[result.ToolUseID] = struct{}{}
		filtered = append(filtered, result)
	}
	return filtered
}

// buildFinalContent builds the final content with system prompt
func buildFinalContent(content, systemPrompt string, toolResults []KiroToolResult) string {
	var contentBuilder strings.Builder

	if systemPrompt != "" {
		contentBuilder.WriteString("--- SYSTEM PROMPT ---\n")
		contentBuilder.WriteString(systemPrompt)
		contentBuilder.WriteString("\n--- END SYSTEM PROMPT ---\n\n")
	}

	contentBuilder.WriteString(content)
	finalContent := contentBuilder.String()

	// CRITICAL: Kiro API requires content to be non-empty
	if strings.TrimSpace(finalContent) == "" {
		if len(toolResults) > 0 {
			finalContent = "Tool results provided."
		} else {
			finalContent = "Continue"
		}
		log.Debugf("kiro: content was empty, using default: %s", finalContent)
	}

	return finalContent
}

// deduplicateToolResults removes duplicate tool results
func deduplicateToolResults(toolResults []KiroToolResult) []KiroToolResult {
	if len(toolResults) == 0 {
		return toolResults
	}

	seenIDs := make(map[string]bool)
	unique := make([]KiroToolResult, 0, len(toolResults))
	for _, tr := range toolResults {
		if !seenIDs[tr.ToolUseID] {
			seenIDs[tr.ToolUseID] = true
			unique = append(unique, tr)
		} else {
			log.Debugf("kiro: skipping duplicate toolResult in currentMessage: %s", tr.ToolUseID)
		}
	}
	return unique
}

// extractClaudeToolChoiceHint extracts tool_choice from Claude request and returns a system prompt hint.
// Claude tool_choice values:
// - {"type": "auto"}: Model decides (default, no hint needed)
// - {"type": "any"}: Must use at least one tool
// - {"type": "tool", "name": "..."}: Must use specific tool
func extractClaudeToolChoiceHint(claudeBody []byte) string {
	toolChoice := gjson.GetBytes(claudeBody, "tool_choice")
	if !toolChoice.Exists() {
		return ""
	}

	toolChoiceType := toolChoice.Get("type").String()
	switch toolChoiceType {
	case "any":
		return "[INSTRUCTION: You MUST use at least one of the available tools to respond. Do not respond with text only - always make a tool call.]"
	case "tool":
		toolName := toolChoice.Get("name").String()
		if toolName != "" {
			return fmt.Sprintf("[INSTRUCTION: You MUST use the tool named '%s' to respond. Do not use any other tool or respond with text only.]", toolName)
		}
	case "auto":
		// Default behavior, no hint needed
		return ""
	}

	return ""
}

// BuildUserMessageStruct builds a user message and extracts tool results
func BuildUserMessageStruct(msg gjson.Result, modelID, origin string) (KiroUserInputMessage, []KiroToolResult) {
	content := msg.Get("content")
	var contentBuilder strings.Builder
	var toolResults []KiroToolResult
	var images []KiroImage

	// Track seen toolUseIds to deduplicate
	seenToolUseIDs := make(map[string]bool)

	if content.IsArray() {
		for _, part := range content.Array() {
			partType := part.Get("type").String()
			switch partType {
			case "text":
				contentBuilder.WriteString(part.Get("text").String())
			case "image":
				mediaType := part.Get("source.media_type").String()
				data := part.Get("source.data").String()

				format := ""
				if idx := strings.LastIndex(mediaType, "/"); idx != -1 {
					format = mediaType[idx+1:]
				}

				if format != "" && data != "" {
					images = append(images, KiroImage{
						Format: format,
						Source: KiroImageSource{
							Bytes: data,
						},
					})
				}
			case "tool_result":
				toolUseID := part.Get("tool_use_id").String()

				// Skip duplicate toolUseIds
				if seenToolUseIDs[toolUseID] {
					log.Debugf("kiro: skipping duplicate tool_result with toolUseId: %s", toolUseID)
					continue
				}
				seenToolUseIDs[toolUseID] = true

				isError := part.Get("is_error").Bool()
				resultContent := part.Get("content")

				var textContents []KiroTextContent

				// Check if this tool_result contains error from our SOFT_LIMIT_REACHED tool_use
				// The client will return an error when trying to execute a tool with marker input
				resultStr := resultContent.String()
				isSoftLimitError := strings.Contains(resultStr, "SOFT_LIMIT_REACHED") ||
					strings.Contains(resultStr, "_status") ||
					strings.Contains(resultStr, "truncated") ||
					strings.Contains(resultStr, "missing required") ||
					strings.Contains(resultStr, "invalid input") ||
					strings.Contains(resultStr, "Error writing file")

				if isError && isSoftLimitError {
					// Replace error content with SOFT_LIMIT_REACHED guidance
					log.Infof("kiro: detected SOFT_LIMIT_REACHED in tool_result for %s, replacing with guidance", toolUseID)
					softLimitMsg := `SOFT_LIMIT_REACHED

Your previous tool call was incomplete due to API output size limits.
The content was PARTIALLY transmitted but NOT executed.

REQUIRED ACTION:
1. Split your content into smaller chunks (max 300 lines per call)
2. For file writes: Create file with first chunk, then use append for remaining
3. Do NOT regenerate content you already attempted - continue from where you stopped

STATUS: This is NOT an error. Continue with smaller chunks.`
					textContents = append(textContents, KiroTextContent{Text: softLimitMsg})
					// Mark as SUCCESS so Claude doesn't treat it as a failure
					isError = false
				} else if resultContent.IsArray() {
					for _, item := range resultContent.Array() {
						if item.Get("type").String() == "text" {
							textContents = append(textContents, KiroTextContent{Text: item.Get("text").String()})
						} else if item.Type == gjson.String {
							textContents = append(textContents, KiroTextContent{Text: item.String()})
						}
					}
				} else if resultContent.Type == gjson.String {
					textContents = append(textContents, KiroTextContent{Text: resultContent.String()})
				}

				if len(textContents) == 0 {
					textContents = append(textContents, KiroTextContent{Text: "Tool use was cancelled by the user"})
				}

				status := "success"
				if isError {
					status = "error"
				}

				toolResults = append(toolResults, KiroToolResult{
					ToolUseID: toolUseID,
					Content:   textContents,
					Status:    status,
				})
			}
		}
	} else {
		contentBuilder.WriteString(content.String())
	}

	userMsg := KiroUserInputMessage{
		Content: contentBuilder.String(),
		ModelID: modelID,
		Origin:  origin,
	}

	if len(images) > 0 {
		userMsg.Images = images
	}

	return userMsg, toolResults
}

// BuildAssistantMessageStruct builds an assistant message with tool uses
func BuildAssistantMessageStruct(msg gjson.Result) KiroAssistantResponseMessage {
	content := msg.Get("content")
	var contentBuilder strings.Builder
	var toolUses []KiroToolUse

	if content.IsArray() {
		for _, part := range content.Array() {
			partType := part.Get("type").String()
			switch partType {
			case "text":
				contentBuilder.WriteString(part.Get("text").String())
			case "tool_use":
				toolUseID := part.Get("id").String()
				toolName := part.Get("name").String()
				toolInput := part.Get("input")

				var inputMap map[string]interface{}
				if toolInput.IsObject() {
					inputMap = make(map[string]interface{})
					toolInput.ForEach(func(key, value gjson.Result) bool {
						inputMap[key.String()] = value.Value()
						return true
					})
				}

				// Rename web_search → remote_web_search to match convertClaudeToolsToKiro
				if toolName == "web_search" {
					toolName = "remote_web_search"
				}

				toolUses = append(toolUses, KiroToolUse{
					ToolUseID: toolUseID,
					Name:      toolName,
					Input:     inputMap,
				})
			}
		}
	} else {
		contentBuilder.WriteString(content.String())
	}

	// CRITICAL FIX: Kiro API requires non-empty content for assistant messages
	// This can happen with compaction requests where assistant messages have only tool_use
	// (no text content). Without this fix, Kiro API returns "Improperly formed request" error.
	finalContent := contentBuilder.String()
	if strings.TrimSpace(finalContent) == "" {
		if len(toolUses) > 0 {
			finalContent = kirocommon.DefaultAssistantContentWithTools
		} else {
			finalContent = kirocommon.DefaultAssistantContent
		}
		log.Debugf("kiro: assistant content was empty, using default: %s", finalContent)
	}

	return KiroAssistantResponseMessage{
		Content:  finalContent,
		ToolUses: toolUses,
	}
}
