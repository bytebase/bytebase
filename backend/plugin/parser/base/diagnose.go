package base

import (
	"strings"

	lsp "github.com/bytebase/lsp-protocol"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DiagnosticContext is the context for diagnosing SQL statements.
type DiagnoseContext struct {
}

type Diagnostic = lsp.Diagnostic

// convertPositionToUTF16Position converts a Position to a UTF16Position in a given text.
// Position uses 1-based line and 1-based character column.
// LSP Position uses 0-based line and 0-based UTF-16 code unit offset.
// If the Position is nil, it returns a UTF16Position with line and character set to 0.
// If the line in Position is out of the end of text, replace it with the last line.
// If the column in Position is out of the end of line, replace it with the last column.
func convertPositionToUTF16Position(p *storepb.Position, text string) *lsp.Position {
	if p == nil {
		return &lsp.Position{
			Line:      0,
			Character: 0,
		}
	}
	lines := strings.Split(text, "\n")
	// Convert from 1-based to 0-based line
	lineNumber := p.Line - 1
	if lineNumber < 0 {
		lineNumber = 0
	}
	if lineNumber >= int32(len(lines)) {
		lineNumber = int32(len(lines)) - 1
	}

	// Convert from 1-based character column to 0-based UTF-16 code units
	line := lines[lineNumber]
	runes := []rune(line)

	// p.Column is 1-based, convert to 0-based character offset
	charOffset := int(p.Column) - 1
	if charOffset < 0 {
		charOffset = 0
	}
	if charOffset > len(runes) {
		charOffset = len(runes)
	}

	// Count UTF-16 code units up to the character offset
	u16CodeUnits := 0
	for i := 0; i < charOffset && i < len(runes); i++ {
		u16CodeUnits++
		if runes[i] > 0xFFFF {
			// Characters outside BMP need surrogate pairs in UTF-16
			u16CodeUnits++
		}
	}

	return &lsp.Position{
		Line:      uint32(lineNumber),
		Character: uint32(u16CodeUnits),
	}
}

func ConvertSyntaxErrorToDiagnostic(err *SyntaxError, statement string) Diagnostic {
	start := *convertPositionToUTF16Position(err.Position, statement)
	end := start
	end.Character++
	message := err.Message
	if err.RawMessage != "" {
		// Use RawMessage which created by antlr runtime, do not need our fine-tuned message
		// because we had indicated the error position in the message.
		message = err.RawMessage
	}
	return Diagnostic{
		Range: lsp.Range{
			Start: start,
			End:   end,
		},
		Severity: lsp.SeverityError,
		Source:   "Syntax check",
		Message:  message,
	}
}
