package elasticsearch

import (
	"context"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_ELASTICSEARCH, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	var diagnostics []base.Diagnostic
	parseResult, _ := ParseElasticsearchREST(statement)
	if parseResult == nil {
		return nil, nil
	}
	for _, err := range parseResult.Errors {
		if err != nil {
			diagnostics = append(diagnostics, base.ConvertSyntaxErrorToDiagnostic(err, statement))
		}
	}

	return diagnostics, nil
}
