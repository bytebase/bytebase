package mariadb

import (
	"context"

	"github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/ast"
	omnicompletion "github.com/bytebase/omni/mariadb/completion"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	engine := base.OmniEngine(analysis.Engine)
	engine.StatementTypes = GetStatementTypes
	engine.Complete = Completion
	engine.QuerySpan = GetQuerySpan
	engine.ExtractChanged = extractChangedResources
	engine.TransformDMLToSelect = TransformDMLToSelect
	engine.GenerateRestoreSQL = GenerateRestoreSQL
	base.Register(engine, storepb.Engine_MARIADB)
}

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// SplitSQL splits a script through the registered MariaDB engine.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.SplitMultiSQL(storepb.Engine_MARIADB, statement)
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected AST for MariaDB", analysis.StatementTypeOf)
}

// Completion provides auto-complete candidates for MariaDB statements using the
// omni MariaDB completion engine.
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
			return analysis.StatementChanges{}, false, errors.New("expected AST for MariaDB")
		}
		return analysis.ExtractChangedResources(node, currentDatabase, base.ASTSource(a)), true, nil
	})
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *mariadbparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *mariadbparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, pe.RelatedText)
		},
	)
}
