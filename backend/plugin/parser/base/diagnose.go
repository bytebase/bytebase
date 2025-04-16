package base

import (
	lsp "github.com/bytebase/lsp-protocol"

	"github.com/bytebase/bytebase/backend/common"
)

// DiagnosticContext is the context for diagnosing SQL statements.
type DiagnoseContext struct {
}

type Diagnostic = lsp.Diagnostic

func ConvertSyntaxErrorToDiagnostic(err *SyntaxError, statement string) Diagnostic {
	start := *common.ConvertPositionToUTF16Position(err.Position, statement)
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
