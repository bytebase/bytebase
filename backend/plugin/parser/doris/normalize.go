package doris

import (
	"strings"

	parser "github.com/bytebase/parser/doris"
)

// NormalizeMultipartIdentifier extracts parts from a MultipartIdentifierContext.
func NormalizeMultipartIdentifier(ctx parser.IMultipartIdentifierContext) []string {
	if ctx == nil {
		return nil
	}
	var result []string
	for _, part := range ctx.AllErrorCapturingIdentifier() {
		if part == nil {
			continue
		}
		id := part.Identifier()
		if id == nil {
			continue
		}
		result = append(result, NormalizeIdentifier(id))
	}
	return result
}

// NormalizeIdentifier extracts the identifier text from an IdentifierContext.
// It removes backticks from quoted identifiers.
func NormalizeIdentifier(ctx parser.IIdentifierContext) string {
	if ctx == nil {
		return ""
	}

	text := ctx.GetText()
	// Check if it's a backtick-quoted identifier and remove the backticks
	if strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`") && len(text) >= 2 {
		return text[1 : len(text)-1]
	}

	return text
}
