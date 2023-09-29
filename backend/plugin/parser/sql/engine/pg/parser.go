// Package pg implements the parser for PostgreSQL.
package pg

import (
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

const (
	operatorLike    string = "~~"
	operatorNotLike string = "!~~"
	// DeparseIndentString is the string for each indent level.
	deparseIndentString = "    "
)

// ParseContext is the context for parsing.
type ParseContext struct {
}

// DeparseContext is the contxt for restoring.
type DeparseContext struct {
	// IndentLevel is indent level for current line.
	// The parser deparses statements with the indent level for pretty format.
	IndentLevel int
}

// WriteIndent is the helper function to write indent string.
func (ctx DeparseContext) WriteIndent(buf *strings.Builder, indent string) error {
	for i := 0; i < ctx.IndentLevel; i++ {
		if _, err := buf.WriteString(indent); err != nil {
			return err
		}
	}
	return nil
}

// Parse implements the parser.Parser interface.
func Parse(_ ParseContext, statement string) ([]ast.Node, error) {
	res, err := pgquery.Parse(statement)
	if err != nil {
		return nil, err
	}

	// sikp the setting line stage
	if len(res.Stmts) == 0 {
		return nil, nil
	}

	// setting line stage
	textList, err := splitSQL(statement)
	if err != nil {
		return nil, err
	}
	if len(res.Stmts) != len(textList) {
		return nil, errors.Errorf("split multi-SQL failed: the length should be %d, but get %d. stmt: \"%s\"", len(res.Stmts), len(textList), statement)
	}

	var nodeList []ast.Node

	for i, stmt := range res.Stmts {
		node, err := convert(stmt.Stmt, textList[i])
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, node)
	}
	return nodeList, nil
}

// splitSQL splits the given SQL statement into multiple SQL statements.
func splitSQL(statement string) ([]base.SingleSQL, error) {
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitPostgreSQLMultiSQL()
	if err != nil {
		return nil, err
	}
	var results []base.SingleSQL
	for _, sql := range list {
		if sql.Empty {
			continue
		}
		results = append(results, sql)
	}
	return results, nil
}

// Deparse implements the parser.Deparse interface.
func Deparse(context DeparseContext, node ast.Node) (string, error) {
	buf := &strings.Builder{}
	if err := deparseImpl(context, node, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
