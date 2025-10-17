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
	lineNumber := p.Line
	if lineNumber >= int32(len(lines)) {
		lineNumber = int32(len(lines)) - 1
	}
	u16CodeUnits := 0
	byteOffset := 0
	line := lines[lineNumber]
	for _, r := range line {
		if byteOffset >= int(p.Column) {
			break
		}
		byteOffset += len(string(r))
		u16CodeUnits++
		if r > 0xFFFF {
			// Need surrogate pair.
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
