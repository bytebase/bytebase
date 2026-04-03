package mongodb

import (
	"context"
	"errors"

	omniparser "github.com/bytebase/omni/mongo/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_MONGODB, Diagnose)
}

// Diagnose performs syntax checking on a MongoDB shell script.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	_, err := ParseMongoShell(statement)
	if err == nil {
		return []base.Diagnostic{}, nil
	}

	var parseErr *omniparser.ParseError
	if errors.As(err, &parseErr) {
		syntaxErr := &base.SyntaxError{
			Position: &storepb.Position{
				Line:   int32(parseErr.Line),
				Column: int32(parseErr.Column),
			},
			RawMessage: parseErr.Message,
			Message:    parseErr.Message,
		}
		return []base.Diagnostic{
			base.ConvertSyntaxErrorToDiagnostic(syntaxErr, statement),
		}, nil
	}

	// Non-parse error — return as a generic diagnostic.
	return []base.Diagnostic{
		{
			Message: err.Error(),
		},
	}, nil
}
