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
	RegisterParser()
}

// RegisterParser registers the CockroachDB parser.
// Returns statements.Statements (github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser/statements) on success.
func RegisterParser() {
	base.RegisterParseFunc(storepb.Engine_COCKROACHDB, func(statement string) (any, error) {
		result, err := ParseCockroachDBSQL(statement)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return result.Stmts, nil
	})
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
