package oracle

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	oracleast "github.com/bytebase/omni/oracle/ast"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/utils"
)

const dbVersion12 = 12

// ========== Type Conversion Functions ==========

// makeValueByTypeName creates appropriate Go types for Oracle column types.
// DATE: date.
// TIMESTAMPDTY: timestamp.
// TIMESTAMPTZ_DTY: timestamp with time zone.
// TIMESTAMPLTZ_DTY: timezone with local time zone.
func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INT2", "INT4", "INT8":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE", "FLOAT4", "FLOAT8":
		return new(sql.NullFloat64)
	case "BIT", "VARBIT":
		return new([]byte)
	case "DATE", "TIMESTAMPDTY", "TIMESTAMPLTZ_DTY", "TIMESTAMPTZ_DTY":
		return new(sql.NullTime)
	default:
		return new(sql.NullString)
	}
}

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: raw.String,
				},
			}
		}
	case *sql.NullInt64:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_Int64Value{
					Int64Value: raw.Int64,
				},
			}
		}
	case *[]byte:
		if len(*raw) > 0 {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_BytesValue{
					BytesValue: *raw,
				},
			}
		}
	case *sql.NullBool:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_BoolValue{
					BoolValue: raw.Bool,
				},
			}
		}
	case *sql.NullFloat64:
		if raw.Valid {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_DoubleValue{
					DoubleValue: raw.Float64,
				},
			}
		}
	case *sql.NullTime:
		if raw.Valid {
			return convertTimestamp(typeName, columnType, raw.Time)
		}
	default:
	}
	return util.NullRowValue
}

// convertTimestamp handles Oracle timestamp type conversions.
// The go-ora driver retrieves the database timezone from the wire protocol and appends it to the timestamp.
// To ensure consistency with Oracle Date expectations, we handle different timestamp types appropriately.
// https://github.com/sijms/go-ora/blob/2962e725e7a756a667a546fb360ef09afd4c8bd0/v2/parameter.go#L616
func convertTimestamp(typeName string, columnType *sql.ColumnType, t time.Time) *v1pb.RowValue {
	_, scale, _ := columnType.DecimalSize()

	switch typeName {
	case "DATE", "TIMESTAMPDTY":
		// Strip timezone information for DATE and TIMESTAMP types
		timeStripped := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(timeStripped),
					Accuracy:        int32(scale),
				},
			},
		}

	case "TIMESTAMPLTZ_DTY":
		// Handle local timezone timestamp
		// This timestamp is not consistent with sqlplus likely due to db and session timezone.
		// TODO(d): fix the go-ora library.
		s := t.Format("2006-01-02 15:04:05.000000000")
		parsedTime, err := time.Parse(time.DateTime, s)
		if err != nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(parsedTime),
					Accuracy:        int32(scale),
				},
			},
		}

	default:
		// Handle timestamp with timezone
		zone, offset := t.Zone()
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampTzValue{
				TimestampTzValue: &v1pb.RowValue_TimestampTZ{
					GoogleTimestamp: timestamppb.New(t),
					Zone:            zone,
					Offset:          int32(offset),
					Accuracy:        int32(scale),
				},
			},
		}
	}
}

// ========== Limit Handling Functions ==========

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
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", util.TrimStatement(statement), limitCount)
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
	list, err := plsqlparser.ParsePLSQLOmni(statement)
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

// ========== Skip Limit Logic for DUAL Queries ==========

// skipAddLimit checks if the statement needs a limit clause.
// For Oracle, we think the statement like "SELECT xxx FROM DUAL" does not need a limit clause.
// More details, xxx can not be a subquery.
func skipAddLimit(stmt string) (bool, error) {
	list, err := plsqlparser.ParsePLSQLOmni(stmt)
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
		selectStmt.Pivot == nil &&
		selectStmt.Unpivot == nil &&
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
