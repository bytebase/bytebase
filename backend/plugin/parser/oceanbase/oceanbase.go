// Package oceanbase parses OceanBase's MySQL mode. OceanBase accepts distributed DDL
// MySQL rejects, so it carries its own omni parser and AST rather than MySQL's.
package oceanbase

import (
	"context"

	"github.com/bytebase/omni/oceanbase/analysis"
	"github.com/bytebase/omni/oceanbase/ast"
	omnicompletion "github.com/bytebase/omni/oceanbase/completion"
	omniparser "github.com/bytebase/omni/oceanbase/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.Register(base.Engine{
		Split:          SplitSQL,
		Parse:          parseStatements,
		StatementTypes: GetStatementTypes,
		Diagnose:       Diagnose,
		Complete:       Completion,
		ValidateQuery:  validateQuery,
		QuerySpan:      GetQuerySpan,
		Explain:        explainStatement,
		ResultLimit:    statementWithResultLimit,
		ExtractChanged: extractChangedResources,
	}, storepb.Engine_OCEANBASE)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical splitter which handles quotes, comments, compound
// statements (BEGIN...END, IF...END IF, etc.) and DELIMITER without
// requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omniparser.Split(statement), func(s omniparser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}

func parseStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		list, err := omniparser.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			return nil, nil
		}
		asts := make([]base.AST, len(list.Items))
		for i, node := range list.Items {
			asts[i] = &AST{Node: node, Text: stmt.Text, StartPosition: stmt.Start}
		}
		return asts, nil
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected AST for OceanBase", analysis.StatementTypeOf)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.DiagnoseFirstError(statement, analysis.Diagnose(statement), func(e *analysis.SyntaxError) (string, int, string) {
		return e.Text, e.Offset, e.Message
	}), nil
}

// Completion provides auto-complete candidates for OceanBase statements using
// the omni OceanBase completion engine.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listDBs := base.MySQLFamilyDatabaseLoaders(cCtx)
	raw := omnicompletion.CompleteSession(ctx, omnicompletion.SessionConfig{
		ReadOnly:          cCtx.Scene == base.SceneTypeQuery,
		DefaultDatabase:   cCtx.DefaultDatabase,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listDBs,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(raw, func(c omnicompletion.Candidate) (string, omnicompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}

func validateQuery(statement string) (bool, bool, error) {
	stmts, err := omniparser.Parse(statement)
	if err != nil {
		return false, false, convertError(err, base.Statement{Text: statement})
	}
	readOnly, returnsData := analysis.ValidateQuery(stmts.Items)
	return readOnly, returnsData, nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	listDBs, getDB := base.UnnamedSchemaQuerySpanCatalog(ctx, gCtx, func(d *base.UnnamedSchemaDatabase) analysis.Database { return d })
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

// explainStatement implements base.ExplainStatement. Only the JSON format has
// a reader downstream; any other format gets the engine's default plan.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, func(e error) error {
		return convertError(e, base.Statement{Text: statement})
	})
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}

// EstimateAffectedRowsFromExplainJSON returns the EST.ROWS estimate from OceanBase
// `EXPLAIN FORMAT=JSON` output.
func EstimateAffectedRowsFromExplainJSON(plan string) (int64, error) {
	return analysis.EstimateAffectedRowsFromExplainJSON(plan)
}

// statementWithResultLimit implements base.ResultLimitFunc for OceanBase. It takes
// no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "SELECT * FROM (%s) result LIMIT %d;")
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected AST for MySQL", false, func(node *AST) (analysis.StatementChanges, bool) {
		return analysis.ExtractChangedResources(node.Node, currentDatabase, node.Text), true
	})
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
