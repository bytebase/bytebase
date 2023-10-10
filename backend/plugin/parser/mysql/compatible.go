package mysql

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// extractTiDBUnsupportedStmts returns a list of unsupported statements in TiDB extracted from the `stmts`,
// and returns the remaining statements supported by TiDB from `stmts`.
func extractTiDBUnsupportedStmts(stmts string) ([]string, string, error) {
	var unsupportStmts []string
	var supportedStmts bytes.Buffer
	// We use our bb tokenizer to help us split the multi-statements into statement list.
	singleSQLs, err := SplitSQL(stmts)
	if err != nil {
		return nil, "", errors.Wrapf(err, "cannot split multi sql %q via bytebase parser", stmts)
	}
	for _, sql := range singleSQLs {
		content := sql.Text
		if isTiDBUnsupportStmt(content) {
			unsupportStmts = append(unsupportStmts, content)
			continue
		}
		_, _ = supportedStmts.WriteString(content)
		_, _ = supportedStmts.WriteString("\n")
	}
	return unsupportStmts, supportedStmts.String(), nil
}

// isTiDBUnsupportStmt returns true if this statement is unsupported in TiDB.
func isTiDBUnsupportStmt(stmt string) bool {
	if _, err := ParseTiDB(stmt, "", ""); err != nil {
		return true
	}
	return false
}

// ParseTiDB parses the given SQL statement and returns the AST.
func ParseTiDB(sql string, charset string, collation string) ([]ast.StmtNode, error) {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

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
		Line:    line,
		Column:  column,
		Message: parserErr.Error(),
	}
}

// IsTiDBUnsupportDDLStmt checks whether the `stmt` is unsupported DDL statement in TiDB, the following statements are unsupported:
// 1. `CREATE TRIGGER`
// 2. `CREATE EVENT`
// 3. `CREATE FUNCTION`
// 4. `CREATE PROCEDURE`.
func IsTiDBUnsupportDDLStmt(stmt string) bool {
	objects := []string{
		"TRIGGER",
		"EVENT",
		"FUNCTION",
		"PROCEDURE",
	}
	createRegexFmt := "(?i)^\\s*CREATE\\s+(DEFINER=(`(.)+`|(.)+)@(`(.)+`|(.)+)(\\s)+)?%s\\s+"
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
