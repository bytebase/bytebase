package parser

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

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
	regexFmt := "(?i)^CREATE\\s+(DEFINER=`(.)+`@`(.)+`(\\s)+)?%s\\s+"
	for _, obj := range objects {
		regex := fmt.Sprintf(regexFmt, obj)
		re := regexp.MustCompile(regex)
		if re.MatchString(stmt) {
			return true
		}
	}
	return false
}

// IsDelimiter returns true if the statement is a delimiter statement.
func IsDelimiter(stmt string) bool {
	delimiterRegex := `(?i)^DELIMITER\s+`
	re := regexp.MustCompile(delimiterRegex)
	return re.MatchString(stmt)
}
