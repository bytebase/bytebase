package mongodb

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(storepb.Engine_MONGODB, Diagnose)
}

// Diagnose performs syntax checking on a MongoDB shell script.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	diagnostics := []base.Diagnostic{}

	parseResult := ParseMongoShell(statement)
	if parseResult == nil {
		return diagnostics, nil
	}

	for _, err := range parseResult.Errors {
		if err != nil {
			diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(err, statement))
		}
	}

	return diagnostics, nil
}
