package parser

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/parser/ast"
)

// SingleSQL is a separate SQL split from multi-SQL.
type SingleSQL struct {
	Text     string
	LastLine int
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engineType EngineType, statement string) ([]SingleSQL, error) {
	switch engineType {
	case Postgres:
		t := newTokenizer(statement)
		return t.splitPostgreSQLMultiSQL()
	case MySQL, TiDB:
		t := newTokenizer(statement)
		return t.splitMySQLMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SplitMultiSQLStream splits statement stream into a slice of the single SQL.
func SplitMultiSQLStream(engineType EngineType, src io.Reader, f func(string) error) ([]SingleSQL, error) {
	switch engineType {
	case Postgres:
		t := newStreamTokenizer(src, f)
		return t.splitPostgreSQLMultiSQL()
	case MySQL, TiDB:
		t := newStreamTokenizer(src, f)
		return t.splitMySQLMultiSQL()
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SetLineForCreateTableStmt sets the line for columns and table constraints in CREATE TABLE statements.
func SetLineForCreateTableStmt(engineType EngineType, node *ast.CreateTableStmt) error {
	switch engineType {
	case Postgres:
		t := newTokenizer(node.Text())
		firstLine := node.LastLine() - strings.Count(node.Text(), "\n")
		return t.setLineForPGCreateTableStmt(node, firstLine)
	default:
		return errors.Errorf("engine type is not supported: %s", engineType)
	}
}

// SetLineForMySQLCreateTableStmt sets the line for columns and table constraints in MySQL CREATE TABLE statments.
// This is a temporary function. Because we do not convert tidb AST to our AST. So we have to implement this.
// TODO(rebelice): remove it.
func SetLineForMySQLCreateTableStmt(node *tidbast.CreateTableStmt) error {
	// exclude CREATE TABLE ... AS and CREATE TABLE ... LIKE statement.
	if len(node.Cols) == 0 {
		return nil
	}
	firstLine := node.OriginTextPosition() - strings.Count(node.Text(), "\n")
	return newTokenizer(node.Text()).setLineForMySQLCreateTableStmt(node, firstLine)
}

// ExtractTiDBUnsupportStmts returns a list of unsupported statements in TiDB extracted from the `stmts`,
// and returns the remaining statements supported by TiDB from `stmts`.
func ExtractTiDBUnsupportStmts(stmts string) ([]string, string, error) {
	var unsupportStmts []string
	var supportedStmts bytes.Buffer
	// We use our bb tokenizer to help us split the multi-statements into statement list.
	singleSQLs, err := SplitMultiSQL(MySQL, stmts)
	if err != nil {
		return nil, "", errors.Wrapf(err, "cannot split multi sql %q via bytebase parser", stmts)
	}
	for _, sql := range singleSQLs {
		content := sql.Text
		if isTiDBUnsupportStmt(content) {
			unsupportStmts = append(unsupportStmts, content)
			continue
		}
		_, _ = supportedStmts.Write([]byte(content))
		_, _ = supportedStmts.Write([]byte("\n"))
	}
	return unsupportStmts, supportedStmts.String(), nil
}

// isTiDBUnsupportStmt returns true if this statement is unsupported in TiDB.
func isTiDBUnsupportStmt(stmt string) bool {
	if isTiDBUnsupportDDLStmt(stmt) {
		return true
	}
	// Match DELIMITER statement
	// Now, we assume that all input comes from our mysqldump, and the tokenizer can split the mysqldump DELIMITER statement
	// in one singleSQL correctly, so we can handle it easily here by checking the prefix.
	return IsDelimiter(stmt)
}

// isTiDBUnsupportStmt checks whether the `stmt` is unsupported DDL statement in TiDB, the following statements are unsupported:
// 1. `CREATE TRIGGER`
// 2. `CREATE EVENT`
// 3. `CREATE FUNCTION`
// 4. `CREATE PROCEDURE`.
func isTiDBUnsupportDDLStmt(stmt string) bool {
	objects := []string{
		"TRIGGER",
		"EVENT",
		"FUNCTION",
		"PROCEDURE",
	}
	createRegexFmt := "(?i)^\\s*CREATE\\s+(DEFINER=`(.)+`@`(.)+`(\\s)+)?%s\\s+"
	dropRegexFmt := "(?i)^\\s*DROP\\s+%s\\s+"
	for _, obj := range objects {
		createRegexp := fmt.Sprintf(createRegexFmt, obj)
		re := regexp.MustCompile(createRegexp)
		if re.MatchString(stmt) {
			return true
		}
		dropRegexp := fmt.Sprintf(dropRegexFmt, obj)
		re = regexp.MustCompile(dropRegexp)
		if re.MatchString(stmt) {
			return true
		}
	}
	return false
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

// ExtractDatabaseList extracts all databases from statement.
// TODO(rebelice): this function only works for single table in FROM clause, fix it.
//
//	e.g. SELECT a, b FROM t;
func ExtractDatabaseList(engineType EngineType, statement string) ([]string, error) {
	switch engineType {
	case MySQL, TiDB:
		return extractMySQLDatabaseList(statement)
	default:
		return nil, errors.Errorf("engine type is not supported: %s", engineType)
	}
}

func newMySQLParser() *tidbparser.Parser {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

func extractMySQLDatabaseList(statement string) ([]string, error) {
	databaseMap := make(map[string]bool)

	p := newMySQLParser()
	nodeList, _, err := p.Parse(statement, "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parser statement %q", statement)
	}

	for _, node := range nodeList {
		databaseList := extractDatabaseListFromNode(node)
		for _, database := range databaseList {
			databaseMap[database] = true
		}
	}

	var databaseList []string
	for database := range databaseMap {
		databaseList = append(databaseList, database)
	}
	sort.Slice(databaseList, func(i, j int) bool {
		return databaseList[i] < databaseList[j]
	})
	return databaseList, nil
}

// extractDatabaseListFromNode extracts all the database from node.
// TODO(rebelice): this function only works for single table in FROM clause, fix it.
//
//	e.g. SELECT a, b FROM t;
func extractDatabaseListFromNode(in tidbast.Node) []string {
	switch node := in.(type) {
	case *tidbast.SelectStmt:
		if node.From != nil {
			return extractDatabaseListFromNode(node.From.TableRefs)
		}
	case *tidbast.Join:
		var res []string
		res = append(res, extractDatabaseListFromNode(node.Left)...)
		res = append(res, extractDatabaseListFromNode(node.Right)...)
		return res
	case *tidbast.TableSource:
		if tableName, ok := node.Source.(*tidbast.TableName); ok {
			return []string{tableName.Schema.O}
		}
	}
	return nil
}
