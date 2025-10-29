package audit

import (
	"unicode/utf8"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const (
	// MaxPayloadSize is the maximum size for individual Request or Response fields (120KB)
	MaxPayloadSize = 120 * 1024

	// MaxEventSize is the maximum total event size (256KB)
	MaxEventSize = 256 * 1024
)

// ValidatePayload checks if audit log payload meets size constraints
// Returns error if any field exceeds limits
func ValidatePayload(payload *storepb.AuditLog) error {
	if payload == nil {
		return common.Errorf(common.Invalid, "payload cannot be nil")
	}

	// Check individual field sizes
	requestSize := len(payload.Request)
	if requestSize > MaxPayloadSize {
		return common.Errorf(common.SizeExceeded, "request field too large: %d bytes (max %d bytes)", requestSize, MaxPayloadSize)
	}

	responseSize := len(payload.Response)
	if responseSize > MaxPayloadSize {
		return common.Errorf(common.SizeExceeded, "response field too large: %d bytes (max %d bytes)", responseSize, MaxPayloadSize)
	}

	// Check total event size (sum of all text fields)
	totalSize := requestSize + responseSize + len(payload.Method) + len(payload.Resource) + len(payload.User)
	if totalSize > MaxEventSize {
		return common.Errorf(common.SizeExceeded, "total event size too large: %d bytes (max %d bytes)", totalSize, MaxEventSize)
	}

	return nil
}

// TruncatePayloadFields performs defensive truncation on payload fields
// This should rarely execute since ValidatePayload rejects oversized events
func TruncatePayloadFields(payload *storepb.AuditLog) {
	if len(payload.Request) > MaxPayloadSize {
		payload.Request = truncateUTF8(payload.Request, MaxPayloadSize)
	}
	if len(payload.Response) > MaxPayloadSize {
		payload.Response = truncateUTF8(payload.Response, MaxPayloadSize)
	}
}

// truncateUTF8 truncates string to maxBytes while preserving UTF-8 character boundaries
// Appends "... [truncated]" marker if truncation occurred
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	const marker = "... [truncated]"
	maxBytes -= len(marker)

	// Find last valid rune boundary by walking backwards
	for maxBytes > 0 && maxBytes < len(s) && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}

	return s[:maxBytes] + marker
}
