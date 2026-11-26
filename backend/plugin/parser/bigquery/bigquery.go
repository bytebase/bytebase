// Package bigquery registers omni's BigQuery analysis engine with the query-span
// extractor and the statement-type mapping.
//
// Masking note: the query-span extractor is masking-critical (bytebase's
// masker applies per-output-column maskers positionally and fails OPEN, so
// under-attributed lineage returns sensitive columns unmasked). The
// differential corpus (test-data/query-span/standard.yaml, recorded from the
// legacy ANTLR resolver) and the leak-pin tests in this package pin it.
package bigquery

import (
	"context"

	"github.com/bytebase/omni/bigquery/analysis"
	"github.com/bytebase/omni/bigquery/ast"
	"github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.QuerySpan = GetQuerySpan
	base.Register(engine, storepb.Engine_BIGQUERY)
}

// GetStatementTypes maps each parsed statement onto the storepb
// statement type that approval rules and risk levels are written against. A
// statement omni cannot parse or classify is UNSPECIFIED rather than dropped, so
// base.IsAllDML does not read it as DML.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	types := make([]storepb.StatementType, 0, len(asts))
	for _, a := range asts {
		node, ok := base.ExtractNode[ast.Node](a)
		if !ok || node == nil {
			types = append(types, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED)
			continue
		}
		types = append(types, base.OmniStatementType(analysis.StatementTypeOf(node)))
	}
	return types, nil
}

// GetQuerySpan is the registered query-span handler.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, defaultDatabase, _ string, _ bool) (*base.QuerySpan, error) {
	getMeta := base.QuerySpanProtoMetadataFunc(ctx, gCtx, true)
	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: defaultDatabase,
		// The written dataset is the DATABASE (default fill-in, single
		// unnamed metadata schema), system queries keep their table-level
		// resources, SET classifies as Select (the legacy listener's "treat
		// SAFE SET as select"), and set-operation star-merge names are
		// upper-cased (the legacy BigQuery extractor's rendering).
		SetStatementIsSelect:   true,
		UppercaseStarMergeName: true,
		GetDatabaseMetadata: func(database string) (*metadata.DatabaseSchemaMetadata, error) {
			if getMeta == nil {
				return nil, &analysis.ResourceNotFoundError{Database: &database}
			}
			return getMeta(database)
		},
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		return nil, err
	}
	return base.OmniQuerySpan(span.Type, span.SourceColumns, nil, span.Results, span.NotFoundError), nil
}
