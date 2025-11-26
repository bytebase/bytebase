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
}

// parseCockroachDBForRegistry is the ParseFunc for CockroachDB.
// Returns []*base.AST with CockroachDBStmt populated.
func parseCockroachDBForRegistry(statement string) ([]*base.AST, error) {
	result, err := ParseCockroachDBSQL(statement)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	var asts []*base.AST
	for _, stmt := range result.Stmts {
		asts = append(asts, &base.AST{
			OriginalText:    stmt.SQL,
			CockroachDBStmt: stmt,
		})
	}
	return asts, nil
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
