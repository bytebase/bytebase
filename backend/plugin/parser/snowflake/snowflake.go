package snowflake

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/snowflake/analysis"
	omniast "github.com/bytebase/omni/snowflake/ast"
	"github.com/bytebase/omni/snowflake/diagnostics"
	omniparser "github.com/bytebase/omni/snowflake/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var _ base.AST = (*AST)(nil)

func init() {
	base.Register(base.Engine{
		Split:           SplitSQL,
		Parse:           parseSnowflakeStatements,
		StatementRanges: GetStatementRanges,
		Diagnose:        Diagnose,
		ValidateQuery:   analysis.ValidateQuery,
		QuerySpan:       GetQuerySpan,
		Explain:         explainStatement,
		ResultLimit:     statementWithResultLimit,
	}, storepb.Engine_SNOWFLAKE)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[omniast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (omniast.Node, bool) {
	node, ok := base.ExtractNode[omniast.Node](a)
	return node, ok && node != nil
}

// SplitSQL splits the input into statements, each with its trailing ';'.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.OmniStatements(statement, analysis.SplitStatements(statement)), nil
}

func parseSnowflakeStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseSegmentedStatements(statement, SplitSQL, func(stmt base.Statement) (*AST, error) {
		node, err := parseStatementNode(stmt.Text)
		if err != nil {
			return nil, convertParseError(err, stmt)
		}
		return &AST{
			Node: node,
			Text: stmt.Text,
			// Keep the 1-based line-only ASTStartPosition shape expected by callers.
			StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + 1},
		}, nil
	})
}

func parseStatementNode(text string) (omniast.Node, error) {
	file, err := omniparser.Parse(text)
	if err != nil {
		return nil, err
	}
	if file == nil || len(file.Stmts) == 0 {
		return nil, errors.New("statement yielded no parse tree")
	}
	return file.Stmts[0], nil
}

func convertParseError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *omniparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *omniparser.ParseError, pos *storepb.Position) string {
			return fmt.Sprintf("Syntax error at line %d:%d \n%s", pos.Line, pos.Column, pe.Message)
		},
	)
}

// GetStatementRanges returns the UTF-16 line/character ranges of statements in
// the input, formatted for the LSP protocol.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	return base.MapByteRangesToLSP(statement, analysis.StatementRanges(statement)), nil
}

// Diagnose returns syntax diagnostics for the given Snowflake statement.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.ConvertOffsetDiagnostics(statement, diagnostics.Analyze(statement), func(d diagnostics.Diagnostic) (int, string) {
		return d.Range.Start.Offset, d.Message
	}), nil
}

// GetQuerySpan returns the query span for the given statement.
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	stmt base.Statement,
	database, schema string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	span, err := analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		DefaultDatabase:     database,
		DefaultSchema:       schema,
		IgnoreCaseSensitive: ignoreCaseSensitive,
		ResolveTable: func(database, schema, table string) (*analysis.Table, error) {
			return resolveTable(ctx, gCtx, database, schema, table)
		},
	})
	if err != nil {
		if mixErr := base.ConvertOmniMixTablesError(err); mixErr != nil {
			return nil, mixErr
		}
		var resolveErr *analysis.ResolveError
		if !errors.As(err, &resolveErr) {
			return nil, err
		}
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(resolveErr.Err, &resourceNotFound) {
			return &base.QuerySpan{
				SourceColumns: base.OmniColumnSet(resolveErr.SourceColumns),
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, resolveErr.Err
	}
	return base.OmniResolvedSpan(span), nil
}

// resolveTable finds a table in the synced metadata of the instance's
// databases. An empty database name searches every database.
func resolveTable(ctx context.Context, gCtx base.GetQuerySpanContext, normalizedDatabaseName, normalizedSchemaName, normalizedTableName string) (*analysis.Table, error) {
	allDatabases, err := gCtx.ListDatabaseNamesFunc(ctx, gCtx.InstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all databases")
	}

	for _, databaseName := range allDatabases {
		if normalizedDatabaseName != "" && normalizedDatabaseName != databaseName {
			continue
		}
		_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, gCtx.InstanceID, normalizedDatabaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %s", normalizedDatabaseName)
		}
		if database == nil {
			return nil, errors.Errorf("database metadata for database %q not found (database not synced)", normalizedDatabaseName)
		}
		allSchemaNames := database.ListSchemaNames()
		for _, schemaSchema := range allSchemaNames {
			if normalizedSchemaName != "" && normalizedSchemaName != schemaSchema {
				continue
			}
			schema := database.GetSchemaMetadata(normalizedSchemaName)
			if schema == nil {
				return nil, errors.Errorf(`schema %s.%s is not found`, normalizedDatabaseName, normalizedSchemaName)
			}
			allTableNames := schema.ListTableNames()
			for _, tableName := range allTableNames {
				if normalizedTableName != tableName {
					continue
				}
				tableSchema := schema.GetTable(normalizedTableName)
				if tableSchema == nil {
					return nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
				}
				var columns []string
				for _, column := range tableSchema.GetProto().GetColumns() {
					columns = append(columns, column.Name)
				}
				return &analysis.Table{
					Name:    tableSchema.GetProto().Name,
					Columns: columns,
				}, nil
			}
		}
	}
	return nil, errors.Errorf(`table %s.%s.%s is not found`, normalizedDatabaseName, normalizedSchemaName, normalizedTableName)
}

// explainStatement implements base.ExplainStatement. Snowflake spells its
// structured plan EXPLAIN USING JSON; any other format gets the tabular plan.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	return analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "SELECT * FROM (%s) LIMIT %d")
}
