package tidb

import (
	"regexp"
	"strconv"
	"strings"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pkg/errors"

	// The packege parser_driver has to be imported.
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

func init() {
	base.RegisterParseFunc(storepb.Engine_TIDB, ParseTiDBForSyntaxCheck)
	base.RegisterParseStatementsFunc(storepb.Engine_TIDB, parseTiDBStatements)
	base.RegisterGetStatementTypes(storepb.Engine_TIDB, GetStatementTypes)
}

// ParseTiDBForSyntaxCheck parses TiDB SQL for syntax checking purposes.
// Returns []base.AST with *TiDBAST instances.
func ParseTiDBForSyntaxCheck(statement string) ([]base.AST, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, err
	}

	p := newTiDBParser()
	var results []base.AST
	for _, singleSQL := range singleSQLs {
		nodes, _, err := p.Parse(singleSQL.Text, "", "")
		if err != nil {
			// Convert parser error to SyntaxError with proper position
			syntaxErr := convertParserError(err)
			// Adjust the line number to be absolute (relative to the full statement)
			// The TiDB parser reports line numbers relative to singleSQL.Text (starting at 1)
			// We need to add the offset to get the absolute line number
			if se, ok := syntaxErr.(*base.SyntaxError); ok && se.Position != nil {
				// errorLine is 1-based relative to singleSQL.Text
				// singleSQL.BaseLine() is 0-based line number of the first line in the original statement
				// Absolute line (1-based) = BaseLine (0-based) + errorLine (1-based)
				se.Position.Line = int32(singleSQL.BaseLine()) + se.Position.Line
			}
			return nil, syntaxErr
		}

		if len(nodes) != 1 {
			continue
		}

		node := nodes[0]
		// node.Text() includes leading whitespace from singleSQL.Text.
		// This maintains consistency: Statement.Start points to first char of Statement.Text,
		// and AST position matches Statement position.
		// Trim only at display points (e.g., error messages) where needed.

		// Calculate the start line. The native TiDB parser may strip leading newlines
		// from its internal Text(), so we count how many were stripped.
		nativeText := node.Text()
		leadingNewlinesStripped := strings.Count(singleSQL.Text, "\n") - strings.Count(nativeText, "\n")
		actualStartLine := singleSQL.BaseLine() + leadingNewlinesStripped + 1

		node.SetOriginTextPosition(actualStartLine)
		if n, ok := node.(*ast.CreateTableStmt); ok {
			if err := SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, errors.Wrapf(err, "failed to set line for create table statement at line %d", actualStartLine)
			}
		}
		results = append(results, &AST{
			StartPosition: &storepb.Position{Line: int32(actualStartLine)},
			Node:          node,
		})
	}

	return results, nil
}

// parseTiDBStatements is the ParseStatementsFunc for TiDB.
// Returns []ParsedStatement with both text and AST populated.
func parseTiDBStatements(statement string) ([]base.ParsedStatement, error) {
	// First split to get Statement with text and positions
	stmts, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, err
	}

	// Then parse to get ASTs
	asts, err := ParseTiDBForSyntaxCheck(statement)
	if err != nil {
		return nil, err
	}

	// Combine: Statement provides text/positions, AST provides parsed tree
	var result []base.ParsedStatement
	astIndex := 0
	for _, stmt := range stmts {
		ps := base.ParsedStatement{
			Statement: stmt,
		}
		if !stmt.Empty && astIndex < len(asts) {
			ps.AST = asts[astIndex]
			astIndex++
		}
		result = append(result, ps)
	}

	return result, nil
}

func newTiDBParser() *tidbparser.Parser {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

// ParseTiDB parses the given SQL statement and returns the AST.
func ParseTiDB(sql string, charset string, collation string) ([]ast.StmtNode, error) {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)
	mode, err := mysql.GetSQLMode(mysql.DefaultSQLMode)
	if err != nil {
		return nil, errors.Errorf("failed to get sql mode: %v", err)
	}
	mode = mysql.DelSQLMode(mode, mysql.ModeNoZeroDate)
	mode = mysql.DelSQLMode(mode, mysql.ModeNoZeroInDate)
	p.SetSQLMode(mode)

	nodes, _, err := p.Parse(sql, charset, collation)
	if err != nil {
		return nil, convertParserError(err)
	}
	return nodes, nil
}

var (
	lineColumnRegex = regexp.MustCompile(`line (\d+) column (\d+)`)
)

func convertParserError(parserErr error) error {
	// line 1 column 15 near "TO world;"
	res := lineColumnRegex.FindAllStringSubmatch(parserErr.Error(), -1)
	if len(res) != 1 {
		return parserErr
	}
	if len(res[0]) != 3 {
		return parserErr
	}
	line, err := strconv.Atoi(res[0][1])
	if err != nil {
		return parserErr
	}
	column, err := strconv.Atoi(res[0][2])
	if err != nil {
		return parserErr
	}
	return &base.SyntaxError{
		Position: common.ConvertTiDBParserErrorPositionToPosition(line, column),
		Message:  parserErr.Error(),
	}
}

// SetLineForMySQLCreateTableStmt sets the line for columns and table constraints in MySQL CREATE TABLE statments.
// This is a temporary function. Because we do not convert tidb AST to our AST. So we have to implement this.
// TODO(rebelice): remove it.
func SetLineForMySQLCreateTableStmt(node *ast.CreateTableStmt) error {
	// exclude CREATE TABLE ... AS and CREATE TABLE ... LIKE statement.
	if len(node.Cols) == 0 {
		return nil
	}
	// OriginTextPosition() now stores the first line of the statement (1-based),
	// so we can use it directly as firstLine.
	firstLine := node.OriginTextPosition()
	return tokenizer.NewTokenizer(node.Text()).SetLineForMySQLCreateTableStmt(node, firstLine)
}

// TypeString returns the string representation of the type for MySQL.
func TypeString(tp byte) string {
	switch tp {
	case mysql.TypeTiny:
		return "tinyint"
	case mysql.TypeShort:
		return "smallint"
	case mysql.TypeInt24:
		return "mediumint"
	case mysql.TypeLong:
		return "int"
	case mysql.TypeLonglong:
		return "bigint"
	case mysql.TypeFloat:
		return "float"
	case mysql.TypeDouble:
		return "double"
	case mysql.TypeNewDecimal:
		return "decimal"
	case mysql.TypeVarchar:
		return "varchar"
	case mysql.TypeBit:
		return "bit"
	case mysql.TypeTimestamp:
		return "timestamp"
	case mysql.TypeDatetime:
		return "datetime"
	case mysql.TypeDate:
		return "date"
	case mysql.TypeDuration:
		return "time"
	case mysql.TypeJSON:
		return "json"
	case mysql.TypeEnum:
		return "enum"
	case mysql.TypeSet:
		return "set"
	case mysql.TypeTinyBlob:
		return "tinyblob"
	case mysql.TypeMediumBlob:
		return "mediumblob"
	case mysql.TypeLongBlob:
		return "longblob"
	case mysql.TypeBlob:
		return "blob"
	case mysql.TypeVarString:
		return "varbinary"
	case mysql.TypeString:
		return "binary"
	case mysql.TypeGeometry:
		return "geometry"
	default:
		return "unknown"
	}
}
