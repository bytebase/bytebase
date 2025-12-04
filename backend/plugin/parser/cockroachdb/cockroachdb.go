package cockroachdb

import (
	"strings"

	crrawparser "github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser/statements"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_COCKROACHDB, parseCockroachDBForRegistry)
	base.RegisterParseStatementsFunc(storepb.Engine_COCKROACHDB, parseCockroachDBStatements)
}

// parseCockroachDBForRegistry is the ParseFunc for CockroachDB.
// Returns []base.AST with *CockroachDBAST instances.
func parseCockroachDBForRegistry(statement string) ([]base.AST, error) {
	result, err := ParseCockroachDBSQL(statement)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	var asts []base.AST
	for _, stmt := range result.Stmts {
		asts = append(asts, &AST{
			StartPosition: &storepb.Position{Line: 1},
			Stmt:          stmt,
		})
	}
	return asts, nil
}

// parseCockroachDBStatements is the ParseStatementsFunc for CockroachDB.
// Returns []Statement with both text and AST populated.
func parseCockroachDBStatements(statement string) ([]base.Statement, error) {
	// First split to get SingleSQL with text and positions (uses PostgreSQL splitter)
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_COCKROACHDB, statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	asts, err := parseCockroachDBForRegistry(statement)
	if err != nil {
		return nil, err
	}

	// Combine: SingleSQL provides text/positions, AST provides parsed tree
	var statements []base.Statement
	astIndex := 0
	for _, sql := range singleSQLs {
		stmt := base.Statement{
			Text:            sql.Text,
			Empty:           sql.Empty,
			StartPosition:   sql.Start,
			EndPosition:     sql.End,
			ByteOffsetStart: sql.ByteOffsetStart,
			ByteOffsetEnd:   sql.ByteOffsetEnd,
		}
		if !sql.Empty && astIndex < len(asts) {
			stmt.AST = asts[astIndex]
			astIndex++
		}
		statements = append(statements, stmt)
	}

	return statements, nil
}

type ParseResult struct {
	Stmts statements.Statements
}

// ParseCockroachDBSQL parses the given CockroachDB statement by using LALR(1) parser. Returns the AST if no error.
func ParseCockroachDBSQL(statement string) (*ParseResult, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + ";"
	stmts, err := crrawparser.Parse(statement)
	if err != nil {
		return nil, err
	}
	result := &ParseResult{
		Stmts: stmts,
	}

	return result, nil
}

func SplitSQLStatement(statement string) ([]string, error) {
	statement = strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon) + ";"
	stmts, err := crrawparser.Parse(statement)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, stmt := range stmts {
		sql := stmt.SQL
		if !strings.HasSuffix(sql, ";") {
			sql += ";"
		}
		result = append(result, sql)
	}
	return result, nil
}
