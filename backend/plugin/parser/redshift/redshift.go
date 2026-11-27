package redshift

import (
	"context"

	omniredshift "github.com/bytebase/omni/redshift"
	"github.com/bytebase/omni/redshift/analysis"
	"github.com/bytebase/omni/redshift/ast"
	redshiftcompletion "github.com/bytebase/omni/redshift/completion"
	omniredshiftparser "github.com/bytebase/omni/redshift/parser"
	pkgerrors "github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine, redshiftcompletion.Engine)
	engine.Parse = parseRedshiftStatements
	engine.StatementTypes = GetStatementTypes
	engine.QuerySpan = GetQuerySpan
	engine.ExtractChanged = extractChangedResources
	base.Register(engine, storepb.Engine_REDSHIFT)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

func parseRedshiftStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseMultiNodeStatements(statement, splitSQL, func(stmt base.Statement) ([]base.AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		parsed, err := omniredshift.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, 0, len(parsed))
		for _, parsedStmt := range parsed {
			if parsedStmt.Empty() {
				continue
			}
			asts = append(asts, &AST{
				Node:          parsedStmt.AST,
				Text:          stmt.Text,
				StartPosition: stmt.Start,
			})
		}
		return asts, nil
	})
}

func getStatementTypesWithPosition(asts []base.AST) ([]base.StatementTypeWithPosition, error) {
	return base.CollectSingleStatementTypesWithPosition(asts, "expected Redshift omni AST", false, func(a *AST) (storepb.StatementType, string, bool) {
		if a == nil || a.Node == nil {
			return 0, "", false
		}
		return base.OmniStatementType(analysis.StatementTypeOf(a.Node)), a.Text, true
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := getStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	return base.ExtractTypesFromPositions(results), nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, schema string, _ bool) (*base.QuerySpan, error) {
	meta, err := base.RequireDatabaseMetadata(ctx, gCtx, database)
	if err != nil {
		return nil, err
	}
	searchPath := meta.GetSearchPath()
	if schema != "" {
		searchPath = []string{schema}
	}

	resolved, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		SearchPath:          searchPath,
		GetDatabaseMetadata: base.QuerySpanProtoMetadataFunc(ctx, gCtx, false),
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		err = convertError(err, stmt)
		return nil, pkgerrors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}

	span := base.OmniQuerySpan(resolved.Type, resolved.SourceColumns, resolved.PredicateColumns, resolved.Results, resolved.NotFoundError)
	span.FunctionNotSupportedError = base.OmniFunctionNotSupportedError(resolved.FunctionNotSupportedError)
	return span, nil
}

func extractChangedResources(database string, schema string, dbMetadata *model.DatabaseMetadata, _ []base.AST, statement string) (*base.ChangeSummary, error) {
	changes, err := analysis.ExtractChangedResources(statement, database, schema)
	if err != nil {
		return nil, err
	}
	summary := &base.ChangeSummary{
		ChangedResources: model.NewChangedResources(dbMetadata),
	}
	base.AppendStatementChanges(summary, *changes, false)
	return summary, nil
}

func splitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_REDSHIFT, statement)
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniredshiftparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniredshiftparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
