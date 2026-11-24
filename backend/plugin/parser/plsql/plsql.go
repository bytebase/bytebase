package plsql

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/bytebase/omni/oracle/ast"
	oraclecompletion "github.com/bytebase/omni/oracle/completion"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

func init() {
	base.Register(base.Engine{
		Split:                SplitSQL,
		Parse:                parsePLSQLStatements,
		StatementTypes:       GetStatementTypes,
		Diagnose:             Diagnose,
		Complete:             Completion,
		ValidateQuery:        validateQuery,
		QuerySpan:            GetQuerySpan,
		ResultLimit:          statementWithResultLimit,
		ExtractChanged:       extractChangedResources,
		TransformDMLToSelect: TransformDMLToSelect,
		GenerateRestoreSQL:   GenerateRestoreSQL,
	}, storepb.Engine_ORACLE)
}

// AST wraps an omni AST node and implements the base.AST interface.
type AST = base.NodeAST[ast.Node]

// GetNode extracts the omni AST node from a base.AST interface.
func GetNode(a base.AST) (ast.Node, bool) {
	return base.ExtractNode[ast.Node](a)
}

// Version is an Oracle release, such as 12.2.
type Version = analysis.Version

func ParseVersion(banner string) (*Version, error) {
	match := versionRe.FindStringSubmatch(banner)
	if len(match) >= 3 {
		firstVersion, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, errors.Errorf("failed to parse first version from banner: %s", banner)
		}
		secondVersion, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, errors.Errorf("failed to parse second version from banner: %s", banner)
		}
		return &Version{First: firstVersion, Second: secondVersion}, nil
	}
	return nil, errors.Errorf("failed to parse version from banner: %s", banner)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
// It uses omni's lexical Oracle splitter, which handles strings, comments,
// SQL*Plus separators, and PL/SQL blocks without requiring valid SQL.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.OmniStatements(statement, analysis.SplitStatements(statement)), nil
}

func parsePLSQLStatements(statement string) ([]base.ParsedStatement, error) {
	return base.ParseNonEmptyMultiNodeStatements(statement, SplitSQL, func(stmt base.Statement) ([]base.AST, error) {
		list, err := analysis.Parse(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			return nil, nil
		}
		asts := make([]base.AST, 0, len(list.Items))
		for _, node := range list.Items {
			raw, ok := node.(*ast.RawStmt)
			if !ok {
				continue
			}
			asts = append(asts, &AST{Node: raw.Stmt, Text: stmt.Text, StartPosition: stmt.Start})
		}
		return asts, nil
	})
}

func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	return base.CollectOmniStatementTypes(asts, GetNode, "expected Oracle omni AST", analysis.StatementTypeOf)
}

// Diagnose returns the script's first syntax error as a diagnostic.
func Diagnose(_ context.Context, _ base.DiagnoseContext, statement string) ([]base.Diagnostic, error) {
	return base.DiagnoseFirstError(statement, analysis.Diagnose(statement), func(e *analysis.SyntaxError) (string, int, string) {
		return e.Text, e.Offset, e.Message
	}), nil
}

// Completion returns Oracle auto-completion candidates backed by omni's
// session-aware Oracle completer.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	loadSchema, listSchemas := base.CompletionMetadataLoaders(cCtx)
	raw := oraclecompletion.CompleteSession(ctx, oraclecompletion.SessionConfig{
		ReadOnly:        cCtx.Scene == base.SceneTypeQuery,
		DefaultSchema:   cCtx.DefaultDatabase,
		LoadSchema:      loadSchema,
		ListSchemaNames: listSchemas,
	}, statement, caretLine, caretOffset)
	if raw == nil {
		return nil, nil
	}
	return base.DeduplicateCandidates(raw, func(c oraclecompletion.Candidate) (base.Candidate, bool) {
		typ := base.OmniCandidateType(c.Type)
		if c.Type == oraclecompletion.CandidateVariable {
			// Oracle package variables are user variables rather than MySQL-style system variables.
			typ = base.CandidateTypeUserVar
		}
		if typ == base.CandidateTypeNone {
			return base.Candidate{}, false
		}
		return base.Candidate{
			Type:       typ,
			Text:       c.Text,
			Definition: c.Definition,
			Comment:    c.Comment,
		}, true
	}, base.CompareCandidate), nil
}

func validateQuery(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertError(err, base.Statement{Text: statement})
	}
	return readOnly, returnsData, nil
}

func GetQuerySpan(ctx context.Context, gCtx base.GetQuerySpanContext, stmt base.Statement, database, _ string, _ bool) (*base.QuerySpan, error) {
	querySpan, err := base.OmniResolvedSpanResult(analysis.ResolveQuerySpan(stmt.Text, analysis.ResolveOptions{
		Instance:        gCtx.InstanceID,
		DefaultDatabase: database,
		Catalog:         newCatalog(ctx, gCtx),
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span from statement: %s", stmt.Text)
	}
	return querySpan, nil
}

// newCatalog reads synced metadata for omni. A nil metadata function stays
// nil, so omni fails where the extractor did.
func newCatalog(ctx context.Context, gCtx base.GetQuerySpanContext) analysis.Catalog {
	var catalog analysis.Catalog
	if gCtx.GetDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(instance, name string) (analysis.Database, error) {
			_, database, err := gCtx.GetDatabaseMetadataFunc(ctx, instance, name)
			if err != nil {
				return nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return nil, nil
			}
			return &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	if gCtx.GetLinkedDatabaseMetadataFunc != nil {
		catalog.GetLinkedDatabase = func(instance, link, schema string) (string, string, analysis.Database, error) {
			linkedInstance, name, database, err := gCtx.GetLinkedDatabaseMetadataFunc(ctx, instance, link, schema)
			if err != nil {
				return "", "", nil, base.ToOmniNotFoundError(err)
			}
			if database == nil {
				return linkedInstance, name, nil, nil
			}
			return linkedInstance, name, &base.UnnamedSchemaDatabase{Database: database}, nil
		}
	}
	return catalog
}

// statementWithResultLimit implements base.ResultLimitFunc for Oracle.
// engineVersion picks the clause: ROWNUM before version 12, FETCH NEXT from
// 12c on. A FETCH NEXT rewrite that fails falls back to ROWNUM.
func statementWithResultLimit(statement string, limit int, engineVersion string) string {
	stmt, err := analysis.RewriteResultLimit(statement, limit, engineVersion)
	if err != nil {
		slog.Error("failed to add FETCH NEXT clause, falling back to ROWNUM",
			slog.String("statement", statement), log.BBError(err))
		return analysis.RowNumResultLimit(statement, limit)
	}
	return stmt
}

// extractChangedResources treats currentDatabase as the current schema, as
// Oracle does.
func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	var indexTable analysis.IndexTableFunc
	if dbMetadata != nil {
		indexTable = func(schema, index string) (string, bool) {
			foundIndex := dbMetadata.GetSchemaMetadata(schema).GetIndex(index)
			if foundIndex == nil {
				return "", false
			}
			return foundIndex.GetTableProto().GetName(), true
		}
	}
	return base.ExtractChangedResourcesFromASTs(dbMetadata, asts, false, func(a base.AST) (analysis.StatementChanges, bool, error) {
		wrapped, ok := a.(*AST)
		if !ok || wrapped == nil {
			return analysis.StatementChanges{}, false, nil
		}
		return analysis.ExtractChangedResources(wrapped.Node, currentDatabase, wrapped.Text, indexTable), true, nil
	})
}

func convertError(err error, stmt base.Statement) error {
	return base.ConvertSyntaxError(stmt, err, true,
		func(pe *oracleparser.ParseError) (int, string) { return pe.Position, pe.Message },
		func(pe *oracleparser.ParseError, pos *storepb.Position) string {
			return base.FormatLineColSyntaxError(pos, pe.Message, "")
		},
	)
}
