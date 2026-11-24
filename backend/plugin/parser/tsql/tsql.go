package tsql

import (
	"context"

	omnimssql "github.com/bytebase/omni/mssql"
	"github.com/bytebase/omni/mssql/analysis"
	"github.com/bytebase/omni/mssql/ast"
	mssqlcompletion "github.com/bytebase/omni/mssql/completion"
	mssqlparser "github.com/bytebase/omni/mssql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.Register(base.Engine{
		Split:                SplitSQL,
		Parse:                parseTSQLStatements,
		StatementTypes:       GetStatementTypes,
		Diagnose:             Diagnose,
		Complete:             Completion,
		ValidateQuery:        ValidateSQLForEditor,
		QuerySpan:            GetQuerySpan,
		ResultLimit:          statementWithResultLimit,
		ExtractChanged:       extractChangedResources,
		TransformDMLToSelect: TransformDMLToSelect,
		GenerateRestoreSQL:   GenerateRestoreSQL,
	}, storepb.Engine_MSSQL)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// ParseTSQL parses SQL using omni's parser.
func ParseTSQL(sql string) ([]omnimssql.Statement, error) {
	return omnimssql.Parse(sql)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits the given T-SQL script into contiguous statements via omni.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omnimssql.Split(statement), func(s omnimssql.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty
	}, false), nil
}

func parseTSQLStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		parsed, err := ParseTSQL(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		asts := make([]base.AST, 0, len(parsed))
		for _, os := range parsed {
			if os.Empty() {
				continue
			}
			asts = append(asts, &AST{Node: os.AST, Text: stmt.Text, StartPosition: stmt.Start})
		}
		return asts, nil
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected omni AST for MSSQL", analysis.StatementTypeOf)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.DiagnoseFirstError(statement, analysis.Diagnose(statement), func(e *analysis.SyntaxError) (string, int, string) {
		return e.Text, e.Offset, e.Message
	}), nil
}

// Completion returns T-SQL auto-completion candidates backed by omni's
// session-aware MSSQL completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadDB, listDBs := base.CompletionMetadataLoaders(cCtx)
	raw := mssqlcompletion.CompleteSession(ctx, mssqlcompletion.SessionConfig{
		DefaultDatabase:   cCtx.DefaultDatabase,
		DefaultSchema:     cCtx.DefaultSchema,
		LoadDatabase:      loadDB,
		ListDatabaseNames: listDBs,
	}, statement, caretLine, caretOffset)
	return base.ConvertOmniCandidates(raw, func(c mssqlcompletion.Candidate) (string, mssqlcompletion.CandidateType, string, string) {
		return c.Text, c.Type, c.Definition, c.Comment
	}, base.CompareCandidate), nil
}

// ValidateSQLForEditor validates that every statement in the SQL is a read-only
// SELECT (no SELECT INTO) or sets read-only session options. Returns (valid,
// allQuery, err); a session option returns no data, so it clears allQuery.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	stmts, err := ParseTSQL(statement)
	if err != nil {
		return false, false, err
	}
	valid, allQuery := analysis.ValidateQuery(stmts)
	return valid, allQuery, nil
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, omnimssql.StatementWithResultLimit, "WITH result AS (%s) SELECT TOP %d * FROM result;")
}

func extractChangedResources(currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	return base.ExtractTypedChangedResources(dbMetadata, asts, "expected AST for MSSQL", false, func(node *AST) (analysis.StatementChanges, bool) {
		if node.Node == nil {
			return analysis.StatementChanges{}, false
		}
		return analysis.ExtractChangedResources(node.Node, currentDatabase, currentSchema, node.Text), true
	})
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, false,
		func(pe *mssqlparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *mssqlparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineSyntaxError(pos, pe.Message)
		},
	)
}
