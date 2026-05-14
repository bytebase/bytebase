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
	// Phase 1.5 §1.5.N+1 dispatcher flip: register the Option B
	// omni-first/pingcap-fallback dispatcher (parseTiDBStatementsOmni in
	// dispatcher.go). See plans/2026-04-23-omni-tidb-completion-plan.md
	// §1.5.0 invariant #8 for the contract.
	base.RegisterParseStatementsFunc(storepb.Engine_TIDB, parseTiDBStatementsOmni)
	base.RegisterGetStatementTypes(storepb.Engine_TIDB, GetStatementTypes)
}

// ParseTiDBForSyntaxCheck parses TiDB SQL for syntax checking purposes.
// Returns []base.AST with *AST instances.
//
// Per-statement parse + line-tracking is delegated to
// parsePingCapSingleStatement (in dispatcher.go) so the dispatcher's
// pingcap-fallback path and this canonical pre-flip path produce
// structurally identical *AST values. (nil, nil) from the helper
// signals "non-1 node count, skip" — same semantic as the pre-
// refactor inline `if len(nodes) != 1 { continue }`.
func ParseTiDBForSyntaxCheck(statement string) ([]base.AST, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, err
	}

	var results []base.AST
	for _, singleSQL := range singleSQLs {
		ast, err := parsePingCapSingleStatement(singleSQL)
		if err != nil {
			return nil, err
		}
		if ast == nil {
			continue
		}
		results = append(results, ast)
	}

	return results, nil
}

// applyTiDBLineTracking sets OriginTextPosition on a freshly-parsed pingcap
// node and walks CREATE TABLE columns to set their per-column line numbers
// via SetLineForMySQLCreateTableStmt. Used by both ParseTiDBForSyntaxCheck
// (the canonical pre-flip parsing path) and OmniAST.AsPingCapAST (the
// post-flip bridge for un-migrated advisors), so consumers reading
// node.OriginTextPosition() see consistent values across both paths.
//
// baseLine is the 0-based line index of the statement's first line in the
// original (pre-split) SQL — i.e., base.Statement.BaseLine().
//
// Returns the 1-based actualStartLine (`baseLine + leadingNewlinesStripped +
// 1`) so callers can use it for *AST.StartPosition.
func applyTiDBLineTracking(node ast.StmtNode, baseLine int, originalText string) (int, error) {
	// The native TiDB parser may strip leading newlines from its internal
	// Text(); count how many to recover the absolute line.
	nativeText := node.Text()
	leadingNewlinesStripped := strings.Count(originalText, "\n") - strings.Count(nativeText, "\n")
	actualStartLine := baseLine + leadingNewlinesStripped + 1

	node.SetOriginTextPosition(actualStartLine)
	if n, ok := node.(*ast.CreateTableStmt); ok {
		if err := SetLineForMySQLCreateTableStmt(n); err != nil {
			return actualStartLine, errors.Wrapf(err, "failed to set line for create table statement at line %d", actualStartLine)
		}
	}
	return actualStartLine, nil
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
	line, err := strconv.ParseInt(res[0][1], 10, 32)
	if err != nil {
		return parserErr
	}
	column, err := strconv.ParseInt(res[0][2], 10, 32)
	if err != nil {
		return parserErr
	}
	return &base.SyntaxError{
		Position: common.ConvertTiDBParserErrorPositionToPosition(int32(line), int32(column)),
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
