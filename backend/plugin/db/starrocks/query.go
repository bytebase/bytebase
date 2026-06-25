package starrocks

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bytebase/omni/starrocks/ast"
	"github.com/bytebase/omni/starrocks/parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func makeValueByTypeName(typeName string, _ *sql.ColumnType) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "DATETIME", "TIMESTAMP":
		return new(sql.NullString)
	case "BOOL":
		return new(sql.NullBool)
	case "INT", "INTEGER", "TINYINT", "SMALLINT", "MEDIUMINT", "BIGINT", "INT2", "INT4", "INT8":
		return new(sql.NullInt64)
	case "FLOAT", "DOUBLE", "FLOAT4", "FLOAT8":
		return new(sql.NullFloat64)
	case "BIT", "VARBIT":
		return new([]byte)
	default:
		return new(sql.NullString)
	}
}

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			if typeName == "TIMESTAMP" || typeName == "DATETIME" {
				_, scale, _ := columnType.DecimalSize()
				return util.BuildTimestampOrStringRowValue(raw.String, scale)
			}
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
	default:
	}
	return util.NullRowValue
}

func getStatementWithResultLimit(statement string, limit int) string {
	trimmedStatement := strings.TrimSpace(statement)
	if strings.HasPrefix(strings.ToUpper(trimmedStatement), "SHOW") {
		return statement
	}

	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return stmt
}

func getStatementWithResultLimitInline(statement string, limitCount int) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("empty statement")
	}

	file, errs := parser.Parse(statement)
	if len(errs) > 0 {
		return "", errors.New(errs[0].Error())
	}
	if file == nil || len(file.Stmts) != 1 {
		stmtCount := 0
		if file != nil {
			stmtCount = len(file.Stmts)
		}
		return "", errors.Errorf("expected exactly one statement, got %d", stmtCount)
	}

	switch stmt := file.Stmts[0].(type) {
	case *ast.SelectStmt:
		return rewriteSelectLimit(statement, stmt, limitCount)
	case *ast.SetOpStmt:
		return rewriteSetOpLimit(statement, stmt, limitCount)
	case *ast.ParenSelect:
		switch inner := stmt.Sel.(type) {
		case *ast.SelectStmt:
			return rewriteSelectLimit(statement, inner, limitCount)
		case *ast.SetOpStmt:
			return rewriteSetOpLimit(statement, inner, limitCount)
		}
		return statement, nil
	default:
		return statement, nil
	}
}

func hasUnparsedTail(sql string, locEnd int) bool {
	pos := locEnd
	for pos < len(sql) {
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}
		for pos < len(sql) && sql[pos] == ';' {
			pos++
		}
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}
		if pos >= len(sql) {
			return false
		}
		if strings.HasPrefix(sql[pos:], "--") || strings.HasPrefix(sql[pos:], "#") {
			nl := strings.IndexByte(sql[pos:], '\n')
			if nl < 0 {
				return false
			}
			pos += nl + 1
			continue
		}
		if strings.HasPrefix(sql[pos:], "/*") {
			end := strings.Index(sql[pos:], "*/")
			if end < 0 {
				return false
			}
			pos += end + 2
			continue
		}
		return true
	}
	return false
}

func rewriteSelectLimit(sql string, stmt *ast.SelectStmt, limitCount int) (string, error) {
	if stmt.Limit != nil {
		limitLoc := literalLoc(stmt.Limit)
		if limitLoc.Start < 0 {
			return "", errors.New("cannot rewrite non-constant LIMIT expression")
		}

		if countStart, countEnd, ok := findCommaLimitCount(sql, limitLoc); ok {
			existingCount, _ := strconv.Atoi(sql[countStart:countEnd])
			if existingCount > 0 && existingCount <= limitCount {
				return sql, nil
			}
			return sql[:countStart] + fmt.Sprintf("%d", limitCount) + sql[countEnd:], nil
		}

		existingLimit := extractLimitValue(stmt.Limit)
		if existingLimit >= 0 && existingLimit <= limitCount {
			return sql, nil
		}
		return sql[:limitLoc.Start] + fmt.Sprintf("%d", limitCount) + sql[limitLoc.End:], nil
	}

	if stmt.Into != nil {
		return sql[:stmt.Into.Loc.Start] + fmt.Sprintf("LIMIT %d ", limitCount) + sql[stmt.Into.Loc.Start:], nil
	}
	if hasUnparsedTail(sql, stmt.Loc.End) {
		return "", errors.New("statement has unparsed tail content")
	}
	return sql[:stmt.Loc.End] + fmt.Sprintf(" LIMIT %d", limitCount) + sql[stmt.Loc.End:], nil
}

func rewriteSetOpLimit(sql string, stmt *ast.SetOpStmt, limitCount int) (string, error) {
	if stmt.Limit != nil {
		limitLoc := literalLoc(stmt.Limit)
		if limitLoc.Start < 0 {
			return "", errors.New("cannot rewrite non-constant LIMIT expression")
		}

		if countStart, countEnd, ok := findCommaLimitCount(sql, limitLoc); ok {
			existingCount, _ := strconv.Atoi(sql[countStart:countEnd])
			if existingCount > 0 && existingCount <= limitCount {
				return sql, nil
			}
			return sql[:countStart] + fmt.Sprintf("%d", limitCount) + sql[countEnd:], nil
		}

		existingLimit := extractLimitValue(stmt.Limit)
		if existingLimit >= 0 && existingLimit <= limitCount {
			return sql, nil
		}
		return sql[:limitLoc.Start] + fmt.Sprintf("%d", limitCount) + sql[limitLoc.End:], nil
	}
	if hasUnparsedTail(sql, stmt.Loc.End) {
		return "", errors.New("statement has unparsed tail content")
	}
	return sql[:stmt.Loc.End] + fmt.Sprintf(" LIMIT %d", limitCount) + sql[stmt.Loc.End:], nil
}

func extractLimitValue(node ast.Node) int {
	lit, ok := node.(*ast.Literal)
	if !ok || lit.Kind != ast.LitInt {
		return -1
	}
	val, err := strconv.Atoi(lit.Value)
	if err != nil {
		return -1
	}
	return val
}

func literalLoc(node ast.Node) ast.Loc {
	lit, ok := node.(*ast.Literal)
	if !ok {
		return ast.Loc{Start: -1, End: -1}
	}
	return lit.Loc
}

func findCommaLimitCount(sql string, limitLoc ast.Loc) (countStart, countEnd int, found bool) {
	pos := limitLoc.End
	pos = skipWhitespaceAndComments(sql, pos)
	if pos >= len(sql) || sql[pos] != ',' {
		return 0, 0, false
	}
	pos++
	pos = skipWhitespaceAndComments(sql, pos)
	countStart = pos
	for pos < len(sql) && sql[pos] >= '0' && sql[pos] <= '9' {
		pos++
	}
	if pos == countStart {
		return 0, 0, false
	}
	return countStart, pos, true
}

func skipWhitespaceAndComments(sql string, pos int) int {
	for pos < len(sql) {
		for pos < len(sql) && (sql[pos] == ' ' || sql[pos] == '\t' || sql[pos] == '\n' || sql[pos] == '\r') {
			pos++
		}
		if strings.HasPrefix(sql[pos:], "--") || strings.HasPrefix(sql[pos:], "#") {
			nl := strings.IndexByte(sql[pos:], '\n')
			if nl < 0 {
				return len(sql)
			}
			pos += nl + 1
			continue
		}
		if strings.HasPrefix(sql[pos:], "/*") {
			end := strings.Index(sql[pos:], "*/")
			if end < 0 {
				return len(sql)
			}
			pos += end + 2
			continue
		}
		return pos
	}
	return pos
}
