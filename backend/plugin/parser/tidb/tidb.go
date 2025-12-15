package tidb

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	tidbparser "github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pkg/errors"

	// The packege parser_driver has to be imported.
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"

	parser "github.com/bytebase/parser/tidb"

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
				// singleSQL.BaseLine is 0-based line number of the first line in the original statement
				// Absolute line (1-based) = BaseLine (0-based) + errorLine (1-based)
				se.Position.Line = int32(singleSQL.BaseLine) + se.Position.Line
			}
			return nil, syntaxErr
		}

		if len(nodes) != 1 {
			continue
		}

		node := nodes[0]
		node.SetText(nil, singleSQL.Text)
		node.SetOriginTextPosition(int(singleSQL.End.GetLine()))
		if n, ok := node.(*ast.CreateTableStmt); ok {
			if err := SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, errors.Wrapf(err, "failed to set line for create table statement at line %d", singleSQL.End.GetLine())
			}
		}
		results = append(results, &AST{
			StartPosition: &storepb.Position{Line: int32(singleSQL.BaseLine) + 1},
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

func ANTLRParseTiDB(statement string) ([]*base.ParseResult, error) {
	statement, err := DealWithDelimiter(statement)
	if err != nil {
		return nil, err
	}
	list, err := parseInputStream(antlr.NewInputStream(statement), statement)
	// HACK(p0ny): the callee may end up in an infinite loop, we print the statement here to help debug.
	if err != nil && strings.Contains(err.Error(), "split SQL statement timed out") {
		slog.Info("split SQL statement timed out", "statement", statement)
	}
	return list, err
}

func parseSingleStatement(baseLine int, statement string) (antlr.Tree, *antlr.CommonTokenStream, error) {
	input := antlr.NewInputStream(statement)
	lexer := parser.NewTiDBLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	p := parser.NewTiDBParser(stream)

	startPosition := &storepb.Position{Line: int32(baseLine) + 1}
	lexerErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{
		Statement:     statement,
		StartPosition: startPosition,
	}
	p.RemoveErrorListeners()
	p.AddErrorListener(parserErrorListener)

	p.BuildParseTrees = true

	tree := p.Script()

	if lexerErrorListener.Err != nil {
		return nil, nil, lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return nil, nil, parserErrorListener.Err
	}

	return tree, stream, nil
}

func parseInputStream(input *antlr.InputStream, statement string) ([]*base.ParseResult, error) {
	var result []*base.ParseResult
	lexer := parser.NewTiDBLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

	list, err := splitTiDBStatement(stream, statement)
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		list[len(list)-1].Text = tidbAddSemicolonIfNeeded(list[len(list)-1].Text)
	}

	baseLine := 0
	for _, s := range list {
		tree, tokens, err := parseSingleStatement(baseLine, s.Text)
		if err != nil {
			return nil, err
		}

		if isEmptyStatement(tokens) {
			continue
		}

		result = append(result, &base.ParseResult{
			Tree:     tree,
			Tokens:   tokens,
			BaseLine: s.BaseLine,
		})
		baseLine = int(s.End.GetLine())
	}

	return result, nil
}

func isEmptyStatement(tokens *antlr.CommonTokenStream) bool {
	for _, token := range tokens.GetAllTokens() {
		if token.GetChannel() == antlr.TokenDefaultChannel && token.GetTokenType() != parser.TiDBParserSEMICOLON_SYMBOL && token.GetTokenType() != parser.TiDBParserEOF {
			return false
		}
	}
	return true
}

// DealWithDelimiter converts the delimiter statement to comment, also converts the following statement's delimiter to semicolon(`;`).
func DealWithDelimiter(statement string) (string, error) {
	has, list, err := hasDelimiter(statement)
	if err != nil {
		return "", err
	}
	if has {
		var result []string
		delimiter := `;`
		for _, sql := range list {
			if IsDelimiter(sql.Text) {
				delimiter, err = ExtractDelimiter(sql.Text)
				if err != nil {
					return "", err
				}
				result = append(result, "-- "+sql.Text)
				continue
			}
			// TODO(rebelice): after deal with delimiter, we may cannot get the right line number, fix it.
			if delimiter != ";" && !sql.Empty {
				result = append(result, fmt.Sprintf("%s;", strings.TrimSuffix(sql.Text, delimiter)))
			} else {
				result = append(result, sql.Text)
			}
		}

		statement = strings.Join(result, "\n")
	}
	return statement, nil
}

// IsDelimiter returns true if the statement is a delimiter statement.
func IsDelimiter(stmt string) bool {
	delimiterRegex := `(?i)^\s*DELIMITER\s+`
	re := regexp.MustCompile(delimiterRegex)
	return re.MatchString(stmt)
}

// ExtractDelimiter extracts the delimiter from the delimiter statement.
func ExtractDelimiter(stmt string) (string, error) {
	delimiterRegex := `(?i)^\s*DELIMITER\s+(?P<DELIMITER>[^\s\\]+)\s*`
	re := regexp.MustCompile(delimiterRegex)
	matchList := re.FindStringSubmatch(stmt)
	index := re.SubexpIndex("DELIMITER")
	if index >= 0 && index < len(matchList) {
		return matchList[index], nil
	}
	return "", errors.Errorf("cannot extract delimiter from %q", stmt)
}

func hasDelimiter(statement string) (bool, []base.Statement, error) {
	// use splitTiDBMultiSQL to check if the statement has delimiter
	t := tokenizer.NewTokenizer(statement)
	list, err := t.SplitTiDBMultiSQL()
	if err != nil {
		return false, nil, errors.Errorf("failed to split multi sql: %v", err)
	}

	for _, sql := range list {
		if IsDelimiter(sql.Text) {
			return true, list, nil
		}
	}

	return false, list, nil
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
	firstLine := node.OriginTextPosition() - strings.Count(node.Text(), "\n")
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

func tidbAddSemicolonIfNeeded(sql string) string {
	lexer := parser.NewTiDBLexer(antlr.NewInputStream(sql))
	lexerErrorListener := &base.ParseErrorListener{
		Statement: sql,
	}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	if lexerErrorListener.Err != nil {
		// If the lexer fails, we cannot add semicolon.
		return sql
	}
	tokens := stream.GetAllTokens()
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i].GetChannel() != antlr.TokenDefaultChannel || tokens[i].GetTokenType() == parser.TiDBParserEOF {
			continue
		}

		// The last default channel token is a semicolon.
		if tokens[i].GetTokenType() == parser.TiDBParserSEMICOLON_SYMBOL {
			return sql
		}

		var result []string
		result = append(result, stream.GetTextFromInterval(antlr.NewInterval(0, tokens[i].GetTokenIndex())))
		result = append(result, ";")
		result = append(result, stream.GetTextFromInterval(antlr.NewInterval(tokens[i].GetTokenIndex()+1, tokens[len(tokens)-1].GetTokenIndex())))
		return strings.Join(result, "")
	}
	return sql
}
