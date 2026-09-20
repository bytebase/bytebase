package aireview

import "context"

// Model is one model turn.
type Model interface {
	Chat(ctx context.Context, request *ChatRequest) (*ChatResponse, error)
}

// ChatRequest is the conversation so far and the tools the model can call.
type ChatRequest struct {
	Messages []Message
	Tools    []ToolDefinition
}

// ChatResponse is the model's reply to one ChatRequest.
type ChatResponse struct {
	Message    Message
	StopReason StopReason
	Usage      Usage
}

// Role is the sender of a message.
type Role int

const (
	RoleSystem Role = iota + 1
	RoleUser
	RoleAssistant
	RoleTool
)

// Message is one message in the conversation.
type Message struct {
	Role    Role
	Content string
	// ToolCalls is set on assistant messages only.
	ToolCalls []ToolCall
	// ToolCallID is set on tool messages only. It names the call this message answers.
	ToolCallID string
}

// ToolCall is a tool invocation the model asks for.
type ToolCall struct {
	ID   string
	Name string
	// Arguments is a JSON object.
	Arguments string
	// Replay is opaque vendor data, such as a Gemini thought signature. Vendors
	// reject a history that returns without it, so it goes back unchanged.
	Replay string
}

// StopReason is why the model ended its reply.
type StopReason int

const (
	StopReasonUnknown StopReason = iota
	StopReasonStop
	// StopReasonLength means the reply was cut at the output token limit.
	StopReasonLength
	StopReasonContentFilter
)

// Usage is the token usage of model calls.
type Usage struct {
	TotalTokens int
}
