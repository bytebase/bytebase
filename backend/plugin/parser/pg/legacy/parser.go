// Package legacy implements the parser for PostgreSQL.
package legacy

import (
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"

	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_POSTGRES, parsePostgresForRegistry)
}

// parsePostgresForRegistry is the ParseFunc for PostgreSQL.
// Returns []ast.Node (github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast) on success.
func parsePostgresForRegistry(statement string) (any, error) {
	nodes, err := Parse(ParseContext{}, statement)
	if err != nil {
		// For ConvertError, return as-is (will be handled by convertErrorToAdvice in sheet.go)
		if _, ok := err.(*ConvertError); ok {
			return nil, err
		}
		// For other errors, wrap in base.SyntaxError with proper line number
		line := calculatePostgresErrorLine(statement)
		return nil, &base.SyntaxError{
			Position: &storepb.Position{
				Line: int32(line),
			},
			Message: err.Error(),
		}
	}
	// Filter out nil nodes
	var res []ast.Node
	for _, node := range nodes {
		if node != nil {
			res = append(res, node)
		}
	}
	return res, nil
}

// calculatePostgresErrorLine calculates the line number where the PostgreSQL error occurred.
func calculatePostgresErrorLine(statement string) int {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_POSTGRES, statement)
	if err != nil {
		return 1
	}

	for _, singleSQL := range singleSQLs {
		if _, err := Parse(ParseContext{}, singleSQL.Text); err != nil {
			return int(singleSQL.End.GetLine())
		}
	}

	return 1
}

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
