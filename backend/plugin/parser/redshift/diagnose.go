package redshift

import (
	"context"

	"github.com/bytebase/omni/redshift/analysis"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_REDSHIFT, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	var syntaxErr *base.SyntaxError
	if diag := analysis.Diagnose(statement); diag != nil {
		position := base.ByteOffsetToRunePosition(statement, diag.Offset)
		syntaxErr = &base.SyntaxError{
			Position:   position,
			Message:    base.FormatLineSyntaxError(position, diag.Message),
			RawMessage: diag.Message,
		}
	}
	return base.DiagnoseSyntaxError(statement, syntaxErr), nil
}
