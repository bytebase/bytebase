package tidb

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// ExtractTiDBUnsupportedStmts returns a list of unsupported statements in TiDB extracted from the `stmts`,
// and returns the remaining statements supported by TiDB from `stmts`.
func ExtractTiDBUnsupportedStmts(stmts string) ([]string, string, error) {
	var unsupportStmts []string
	var supportedStmts bytes.Buffer
	// We use our bb tokenizer to help us split the multi-statements into statement list.
	singleSQLs, err := mysqlparser.SplitSQL(stmts)
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
