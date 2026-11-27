package mysql

import (
	"context"

	"github.com/bytebase/omni/mysql/analysis"
	"github.com/bytebase/omni/mysql/ast"
	omnicompletion "github.com/bytebase/omni/mysql/completion"
	mysqlparser "github.com/bytebase/omni/mysql/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, omnicompletion.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.QuerySpan = GetQuerySpan
	engine.ExtractChanged = extractChangedResources
	engine.TransformDMLToSelect = TransformDMLToSelect
	engine.GenerateRestoreSQL = GenerateRestoreSQL
	base.Register(engine, storepb.Engine_MYSQL)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits a script through the registered MySQL engine.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_MYSQL, statement)
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected AST for MySQL", analysis.StatementTypeOf)
}

// StatementTypeWithPosition contains a statement type and its position information.
type StatementTypeWithPosition = base.StatementTypeWithPosition

// GetStatementTypesWithPositions returns statement types with position information,
// preserving statement order. Line numbers are one-based.
func GetStatementTypesWithPositions(asts []base.AST) ([]StatementTypeWithPosition, error) {
	// STATEMENT_TYPE_UNSPECIFIED entries are included so the SDL gate fails closed.
	return base.CollectStatementTypesWithPosition(asts, func(a base.AST) ([]storepb.StatementType, string, bool, error) {
		node, ok := GetNode(a)
		if !ok {
			return nil, "", false, errors.New("expected AST for MySQL")
		}
		if node == nil {
			return nil, "", false, nil
		}
		return []storepb.StatementType{base.OmniStatementType(analysis.StatementTypeOf(node))}, base.ASTSource(a), true, nil
	})
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	listDBs, getDB := base.UnnamedSchemaQuerySpanCatalog(ctx, gCtx, wrapDatabase)
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		Catalog:             analysis.Catalog{ListDatabaseNames: listDBs, GetDatabase: getDB},
	}))
	if err != nil {
		return nil, convertError(err, stmt)
	}
	return querySpan, nil
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (analysis.StatementChanges, bool, error) {
		node, ok := GetNode(a)
		if !ok {
			return analysis.StatementChanges{}, false, errors.New("expected AST for MySQL")
		}
		return analysis.ExtractChangedResources(node, currentDatabase, base.ASTSource(a)), true, nil
	})
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *mysqlparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *mysqlparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
