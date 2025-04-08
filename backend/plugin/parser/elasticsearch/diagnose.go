package elasticsearch

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/proto/generated-go/store"
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
			diagnostics = append(diagnostics, base.Diagnostic{
				Range: lsp.Range{
					Start: lsp.Position{
						// Convert to zero-based.
						Line:      uint32(err.Line),
						Character: uint32(err.Column),
					},
					End: lsp.Position{
						// Convert to zero-based.
						Line: uint32(err.Line),
						// The end position is exclusive.
						Character: uint32(err.Column) + 1,
					},
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
