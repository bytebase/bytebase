package tidb

import (
	"context"

	"github.com/bytebase/omni/tidb/analysis"
	"github.com/bytebase/omni/tidb/ast"
	omnicompletion "github.com/bytebase/omni/tidb/completion"
	omniparser "github.com/bytebase/omni/tidb/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, omnicompletion.Engine)
	engine.Parse = parseTiDBStatements
	engine.StatementTypes = GetStatementTypes
	engine.QuerySpan = GetQuerySpan
	engine.ExtractChanged = extractChangedResources
	engine.TransformDMLToSelect = TransformDMLToSelect
	engine.GenerateRestoreSQL = GenerateRestoreSQL
	base.Register(engine, storepb.Engine_TIDB)
}

// AST wraps an omni/tidb AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits a script through the registered TiDB engine.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_TIDB, statement)
}

func parseTiDBStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		list, err := omniparser.Parse(stmt.Text)
		if err != nil {
			return nil, convertParseError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			return nil, nil
		}
		asts := make([]base.AST, len(list.Items))
		for i, node := range list.Items {
			asts[i] = &AST{
				Node:          node,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			}
		}
		return asts, nil
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected TiDB AST", analysis.StatementTypeOf)
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	_, getDB := base.UnnamedSchemaQuerySpanCatalog(ctx, gCtx, wrapDatabase)
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase: database,
		Catalog:         analysis.Catalog{GetDatabase: getDB},
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected TiDB AST", false, func(wrapped *AST) (analysis.StatementChanges, bool) {
		return analysis.ExtractChangedResources(wrapped.Node, database, wrapped.Text), true
	})
}

func convertParseError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
