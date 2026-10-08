package plsql

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"unicode"

	oracleast "github.com/bytebase/omni/oracle/ast"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/utils"
)

const dbVersion12 = 12

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_ORACLE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Oracle.
// engineVersion picks the clause: ROWNUM before version 12, FETCH NEXT from
// 12c on — see isOracle11gOrEarlier.
func statementWithResultLimit(statement string, limit int, engineVersion string) string {
	return addResultLimit(statement, limit, engineVersion)
}

// addResultLimit adds a limit clause to the statement based on Oracle version
func addResultLimit(stmt string, limit int, engineVersion string) string {
	// Check if we should skip adding limit (e.g., for simple DUAL queries)
	if shouldSkipLimit(stmt) {
		return stmt
	}

	// Determine Oracle version
	if isOracle11gOrEarlier(engineVersion) {
		return addLimitFor11g(stmt, limit)
	}
	return addLimitFor12cAndLater(stmt, limit)
}

// shouldSkipLimit checks if the statement needs a limit clause
func shouldSkipLimit(stmt string) bool {
	ok, err := skipAddLimit(stmt)
	return err == nil && ok
}

// isOracle11gOrEarlier checks if the Oracle version is 11g or earlier
func isOracle11gOrEarlier(engineVersion string) bool {
	versionIdx := strings.Index(engineVersion, ".")
	if versionIdx < 0 {
		return true // Default to 11g behavior for invalid version
	}
	versionNumber, err := strconv.Atoi(engineVersion[:versionIdx])
	if err != nil {
		return true // Default to 11g behavior for parsing errors
	}
	return versionNumber < dbVersion12
}

// addLimitFor11g adds a ROWNUM-based limit for Oracle 11g and earlier versions.
// Uses the legacy approach with subquery and ROWNUM.
func addLimitFor11g(statement string, limitCount int) string {
	if !isSelectOrWithStatement(statement) {
		return statement
	}
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", base.TrimStatement(statement), limitCount)
}

// isSelectOrWithStatement checks if the statement is a SELECT or WITH statement
func isSelectOrWithStatement(statement string) bool {
	trimmedStatement := strings.ToLower(strings.TrimLeftFunc(statement, unicode.IsSpace))
	return strings.HasPrefix(trimmedStatement, "select") || strings.HasPrefix(trimmedStatement, "with")
}

// addLimitFor12cAndLater adds a FETCH NEXT clause for Oracle 12c and later versions.
// Uses the modern SQL standard approach, falling back to 11g approach on error.
func addLimitFor12cAndLater(statement string, limit int) string {
	if !isSelectOrWithStatement(statement) {
		return statement
	}

	stmt, err := addFetchNextClause(statement, limit)
	if err != nil {
		slog.Error("failed to add FETCH NEXT clause, falling back to ROWNUM",
			slog.String("statement", statement), log.BBError(err))
		return addLimitFor11g(statement, limit)
	}
	return stmt
}

// addFetchNextClause adds a FETCH NEXT clause to a SELECT statement using AST parsing.
// This provides more precise placement of the limit clause compared to simple string wrapping.
func addFetchNextClause(statement string, limitCount int) (string, error) {
	list, err := ParsePLSQL(statement)
	if err != nil {
		return "", err
	}
	if list == nil || len(list.Items) == 0 {
		return "", errors.New("no parse results")
	}
	if len(list.Items) > 1 {
		return "", errors.Errorf("expected single statement, got %d statements", len(list.Items))
	}
	raw, ok := list.Items[0].(*oracleast.RawStmt)
	if !ok {
		return "", errors.Errorf("expected raw statement, got %T", list.Items[0])
	}
	selectStmt, ok := raw.Stmt.(*oracleast.SelectStmt)
	if !ok {
		return statement, nil
	}

	res, err := rewriteOracleSelectFetch(statement, selectStmt, limitCount)
	if err != nil {
		return "", err
	}
	// https://stackoverflow.com/questions/27987882/how-can-i-solve-ora-00911-invalid-character-error
	res = strings.TrimRightFunc(res, utils.IsSpaceOrSemicolon)

	return res, nil
}

func rewriteOracleSelectFetch(sql string, selectStmt *oracleast.SelectStmt, limitCount int) (string, error) {
	target := rightmostOracleSetSelect(selectStmt)
	if target.FetchFirst != nil {
		return rewriteOracleFetchClause(sql, target.FetchFirst, limitCount)
	}
	if target.ForUpdate != nil && target.ForUpdate.Loc.Start > 0 && target.ForUpdate.Loc.Start <= len(sql) {
		return sql[:target.ForUpdate.Loc.Start] + fmt.Sprintf("FETCH NEXT %d ROWS ONLY ", limitCount) + sql[target.ForUpdate.Loc.Start:], nil
	}

	loc := oracleast.NodeLoc(target)
	if loc.End < 0 || loc.End > len(sql) {
		return "", errors.Errorf("invalid SELECT end position %d", loc.End)
	}
	return sql[:loc.End] + fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", limitCount) + sql[loc.End:], nil
}

func rightmostOracleSetSelect(selectStmt *oracleast.SelectStmt) *oracleast.SelectStmt {
	if selectStmt.Op != oracleast.SETOP_NONE && selectStmt.Rarg != nil {
		return rightmostOracleSetSelect(selectStmt.Rarg)
	}
	return selectStmt
}

func rewriteOracleFetchClause(sql string, fetch *oracleast.FetchFirstClause, limitCount int) (string, error) {
	if fetch.Count == nil {
		if fetch.Loc.Start < 0 || fetch.Loc.End < fetch.Loc.Start || fetch.Loc.End > len(sql) {
			return "", errors.Errorf("invalid FETCH position %d:%d", fetch.Loc.Start, fetch.Loc.End)
		}
		if hasOracleFetchKeyword(sql, fetch.Loc) {
			return sql, nil
		}
		return sql[:fetch.Loc.End] + fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", limitCount) + sql[fetch.Loc.End:], nil
	}
	if fetch.Percent {
		return "", errors.Errorf("cannot rewrite PERCENT FETCH expression")
	}

	existingLimit := extractOracleFetchCount(fetch.Count)
	if existingLimit > 0 && existingLimit <= limitCount {
		return sql, nil
	}

	loc := oracleast.NodeLoc(fetch.Count)
	loc = trimOracleLocSpace(sql, loc)
	if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(sql) {
		if existingLimit <= 0 {
			return "", errors.Errorf("cannot rewrite non-constant FETCH expression")
		}
		return sql[:loc.Start] + fmt.Sprintf("%d", limitCount) + sql[loc.End:], nil
	}
	return "", errors.Errorf("cannot rewrite FETCH expression")
}

func hasOracleFetchKeyword(sql string, loc oracleast.Loc) bool {
	segment := sql[loc.Start:loc.End]
	lexer := oracleparser.NewLexer(segment)
	for {
		tok := lexer.NextToken()
		if tok.Loc == tok.End && tok.End >= len(segment) {
			return false
		}
		if tok.Loc < 0 || tok.End > len(segment) || tok.End <= tok.Loc {
			return false
		}
		if strings.EqualFold(segment[tok.Loc:tok.End], "FETCH") {
			return true
		}
	}
}

func extractOracleFetchCount(node oracleast.Node) int {
	limit, ok := node.(*oracleast.NumberLiteral)
	if !ok || limit.IsFloat || limit.Ival <= 0 {
		return 0
	}
	return int(limit.Ival)
}

func trimOracleLocSpace(sql string, loc oracleast.Loc) oracleast.Loc {
	for loc.Start < loc.End && loc.Start < len(sql) && unicode.IsSpace(rune(sql[loc.Start])) {
		loc.Start++
	}
	for loc.End > loc.Start && loc.End <= len(sql) && unicode.IsSpace(rune(sql[loc.End-1])) {
		loc.End--
	}
	return loc
}

// skipAddLimit checks if the statement needs a limit clause.
// For Oracle, we think the statement like "SELECT xxx FROM DUAL" does not need a limit clause.
// More details, xxx can not be a subquery.
func skipAddLimit(stmt string) (bool, error) {
	list, err := ParsePLSQL(stmt)
	if err != nil {
		return false, err
	}
	if list == nil || len(list.Items) == 0 {
		return false, nil
	}
	// Multiple statements should not skip limit
	if len(list.Items) > 1 {
		return false, nil
	}
	raw, ok := list.Items[0].(*oracleast.RawStmt)
	if !ok {
		return false, nil
	}
	selectStmt, ok := raw.Stmt.(*oracleast.SelectStmt)
	if !ok {
		return false, nil
	}
	if !isSimpleOracleSelect(selectStmt) {
		return false, nil
	}
	if !isOracleSelectFromDual(selectStmt) {
		return false, nil
	}
	return !hasOracleSubqueriesInSelection(selectStmt), nil
}

func isSimpleOracleSelect(selectStmt *oracleast.SelectStmt) bool {
	return selectStmt.WithClause == nil &&
		!selectStmt.Distinct &&
		!selectStmt.UniqueKw &&
		!selectStmt.All &&
		selectStmt.Into == nil &&
		selectStmt.IntoVars == nil &&
		selectStmt.WhereClause == nil &&
		selectStmt.Hierarchical == nil &&
		selectStmt.GroupClause == nil &&
		selectStmt.HavingClause == nil &&
		selectStmt.ModelClause == nil &&
		len(selectStmt.WindowDefs) == 0 &&
		selectStmt.QualifyClause == nil &&
		selectStmt.OrderBy == nil &&
		selectStmt.ForUpdate == nil &&
		selectStmt.FetchFirst == nil &&
		selectStmt.Op == oracleast.SETOP_NONE
}

func isOracleSelectFromDual(selectStmt *oracleast.SelectStmt) bool {
	if selectStmt.FromClause == nil || len(selectStmt.FromClause.Items) != 1 {
		return false
	}
	tableRef, ok := selectStmt.FromClause.Items[0].(*oracleast.TableRef)
	if !ok || tableRef.Name == nil {
		return false
	}
	return tableRef.Alias == nil &&
		tableRef.Name.Schema == "" &&
		tableRef.Name.DBLink == "" &&
		strings.EqualFold(tableRef.Name.Name, "DUAL")
}

func hasOracleSubqueriesInSelection(selectStmt *oracleast.SelectStmt) bool {
	if selectStmt.TargetList == nil || len(selectStmt.TargetList.Items) == 0 {
		return true
	}
	for _, item := range selectStmt.TargetList.Items {
		target, ok := item.(*oracleast.ResTarget)
		if !ok || target.Expr == nil {
			return true
		}
		if _, ok := target.Expr.(*oracleast.Star); ok {
			return true
		}
		hasSubquery := false
		oracleast.Inspect(target.Expr, func(node oracleast.Node) bool {
			if _, ok := node.(*oracleast.SubqueryExpr); ok {
				hasSubquery = true
				return false
			}
			return true
		})
		if hasSubquery {
			return true
		}
	}
	return false
}
