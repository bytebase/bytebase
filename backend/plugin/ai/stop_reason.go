package ai

import (
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Each mapping sends a value its provider's API reference does not list to
// OTHER, never to TURN_COMPLETED: a caller trusts a completed turn, and nobody
// has read what an unlisted value means.

// openAIChatCompletionsStopReason maps choices[].finish_reason. A refusal
// arrives as message.refusal beside a finish_reason of stop.
func openAIChatCompletionsStopReason(finishReason string, refusal string) v1pb.AIChatResponse_StopReason {
	if refusal != "" {
		return v1pb.AIChatResponse_BLOCKED
	}
	switch finishReason {
	case "":
		return v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED
	case "stop", "tool_calls", "function_call":
		return v1pb.AIChatResponse_TURN_COMPLETED
	case "length":
		return v1pb.AIChatResponse_TOKEN_LIMIT
	case "content_filter":
		return v1pb.AIChatResponse_BLOCKED
	default:
		return v1pb.AIChatResponse_OTHER
	}
}

// openAIResponsesStopReason maps status. The caller returns an error for an
// incomplete response and checks for a refusal, which arrives as a content part.
func openAIResponsesStopReason(status string) v1pb.AIChatResponse_StopReason {
	switch status {
	case "":
		return v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED
	case "completed":
		return v1pb.AIChatResponse_TURN_COMPLETED
	default:
		// failed, cancelled, queued, and in_progress land here.
		return v1pb.AIChatResponse_OTHER
	}
}

// claudeStopReason maps stop_reason.
func claudeStopReason(stopReason string) v1pb.AIChatResponse_StopReason {
	switch stopReason {
	case "":
		return v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED
	case "end_turn", "stop_sequence", "tool_use":
		return v1pb.AIChatResponse_TURN_COMPLETED
	case "max_tokens", "model_context_window_exceeded":
		return v1pb.AIChatResponse_TOKEN_LIMIT
	case "refusal":
		return v1pb.AIChatResponse_BLOCKED
	default:
		// pause_turn lands here.
		return v1pb.AIChatResponse_OTHER
	}
}

// geminiStopReason maps candidates[].finishReason.
func geminiStopReason(finishReason string) v1pb.AIChatResponse_StopReason {
	switch finishReason {
	case "", "FINISH_REASON_UNSPECIFIED":
		return v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED
	case "STOP":
		return v1pb.AIChatResponse_TURN_COMPLETED
	case "MAX_TOKENS", "CONTINUATION":
		return v1pb.AIChatResponse_TOKEN_LIMIT
	case "SAFETY", "RECITATION", "LANGUAGE", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII",
		"IMAGE_SAFETY", "IMAGE_PROHIBITED_CONTENT", "IMAGE_RECITATION",
		"ESCALATION", "PUP_LIMITED_DISABLED":
		return v1pb.AIChatResponse_BLOCKED
	case "MALFORMED_FUNCTION_CALL":
		return v1pb.AIChatResponse_MALFORMED_TOOL_CALL
	default:
		// OTHER, IMAGE_OTHER, NO_IMAGE, UNEXPECTED_TOOL_CALL, TOO_MANY_TOOL_CALLS,
		// MISSING_THOUGHT_SIGNATURE, and MALFORMED_RESPONSE land here.
		return v1pb.AIChatResponse_OTHER
	}
}

// geminiPromptBlockStopReason maps promptFeedback.blockReason, which Gemini
// sets when it blocked the prompt and returned no candidate. Its OTHER is a
// block for a reason Gemini does not name.
func geminiPromptBlockStopReason(blockReason string) v1pb.AIChatResponse_StopReason {
	switch blockReason {
	case "", "BLOCK_REASON_UNSPECIFIED":
		return v1pb.AIChatResponse_STOP_REASON_UNSPECIFIED
	case "SAFETY", "OTHER", "BLOCKLIST", "PROHIBITED_CONTENT", "IMAGE_SAFETY":
		return v1pb.AIChatResponse_BLOCKED
	default:
		return v1pb.AIChatResponse_OTHER
	}
}
