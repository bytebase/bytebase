package base

import lsp "github.com/bytebase/lsp-protocol"

// DiagnosticContext is the context for diagnosing SQL statements.
type DiagnoseContext struct {
}

type Diagnostic = lsp.Diagnostic

func ConvertSyntaxErrorToDiagnostic(err *SyntaxError) Diagnostic {
	return Diagnostic{
		Range: lsp.Range{
			Start: lsp.Position{
				// Convert to zero-based.
				Line:      uint32(err.Line) - 1,
				Character: uint32(err.Column),
			},
			End: lsp.Position{
				// Convert to zero-based.
				Line: uint32(err.Line) - 1,
				// The end position is exclusive.
				Character: uint32(err.Column) + 1,
			},
		},
		Severity: lsp.SeverityError,
		Source:   "Syntax check",
		// Use RawMessage which created by antlr runtime, do not need our fine-tuned message
		// because we had indicated the error position in the message.
		Message: err.RawMessage,
	}
}
