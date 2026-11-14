package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_TIDB, Diagnose)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := make([]base.Diagnostic, 0)
	syntaxError, err := analysis.Diagnose(statement)
	if err != nil {
		return nil, err
	}
	if syntaxError != nil {
		diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(toSyntaxError(statement, syntaxError), statement))
	}
	return diagnostics, nil
}

// toSyntaxError places omni's error the way convertParseError places a parse
// error from the split statement that holds it.
func toSyntaxError(statement string, e *analysis.SyntaxError) *base.SyntaxError {
	stmt := base.NewStatementFromRange(statement, base.NewByteOffsetPositionMapper(statement), e.ByteStart, e.ByteEnd, false)
	pos := ByteOffsetToRunePosition(stmt.Text, e.Offset)
	if stmt.Start != nil {
		pos.Line += stmt.Start.Line - 1
	}
	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, e.Message)
	if e.RelatedText != "" {
		msg += "\nrelated text: " + e.RelatedText
	}
	return &base.SyntaxError{
		Position:   pos,
		Message:    msg,
		RawMessage: e.Message,
	}
}
