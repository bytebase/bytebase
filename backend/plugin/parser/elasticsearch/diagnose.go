package elasticsearch

import (
	"context"

	lsp "github.com/bytebase/lsp-protocol"

	es "github.com/bytebase/omni/elasticsearch"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterDiagnoseFunc(store.Engine_ELASTICSEARCH, Diagnose)
}

func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	omniDiags, err := es.Diagnose(statement)
	if err != nil {
		return nil, err
	}
	if omniDiags == nil {
		return nil, nil
	}

	var diagnostics []base.Diagnostic
	for _, d := range omniDiags {
		diagnostics = append(diagnostics, base.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      uint32(d.Range.Start.Line),
					Character: uint32(d.Range.Start.Column),
				},
				End: lsp.Position{
					Line:      uint32(d.Range.End.Line),
					Character: uint32(d.Range.End.Column),
				},
			},
			Severity: lsp.DiagnosticSeverity(d.Severity),
			Source:   "Syntax check",
			Message:  d.Message,
		})
	}
	return diagnostics, nil
}
