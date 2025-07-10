package elasticsearch

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"

	"github.com/bytebase/bytebase/backend/common"
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
			// TODO(zp): unify the position in diagnose.
			start := *common.ConvertPositionToUTF16Position(err.Position, statement)
			end := start
			end.Character++
			diagnostics = append(diagnostics, base.Diagnostic{
				Range: lsp.Range{
					Start: start,
					End:   end,
				},
				Severity: lsp.SeverityError,
				Source:   "Syntax check",
				// Use RawMessage which created by antlr runtime, do not need our fine-tuned message
				// because we had indicated the error position in the message.
				Message: err.Error(),
			})
		}
	}

	return diagnostics, nil
}
