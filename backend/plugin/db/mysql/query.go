package mysql

import (
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/omni/mysql/ast"
	omnimysqlparser "github.com/bytebase/omni/mysql/parser"
	"github.com/pkg/errors"

	antlrmysql "github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
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
	case "BIT", "VARBIT", "BINARY", "VARBINARY":
		return new([]byte)
	case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION":
		return new([]byte)
	default:
		return new(sql.NullString)
	}
}

func convertValue(typeName string, columnType *sql.ColumnType, value any) *v1pb.RowValue {
	switch raw := value.(type) {
	case *sql.NullString:
		if raw.Valid {
			_, scale, _ := columnType.DecimalSize()
			if typeName == "TIMESTAMP" || typeName == "DATETIME" {
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
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// MySQL 5.7 doesn't support WITH clause.
		return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return stmt
}

func getStatementWithResultLimitInline(statement string, limitCount int) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("empty statement")
	}

	list, err := mysqlparser.ParseMySQLOmni(statement)
	if err != nil {
		if stmt, legacyErr := getStatementWithResultLimitInlineLegacy(statement, limitCount); legacyErr == nil {
			return stmt, nil
		}
		return "", err
	}
	if len(list.Items) != 1 {
		return "", errors.Errorf("expected exactly one statement, got %d", len(list.Items))
	}

	stmt, ok := list.Items[0].(*ast.SelectStmt)
	if !ok {
		return statement, nil
	}

	return rewriteMySQLSelectLimit(statement, stmt, limitCount)
}

func getStatementWithResultLimitInlineLegacy(statement string, limitCount int) (string, error) {
	list, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		return "", err
	}
	if len(list) != 1 {
		return "", errors.Errorf("expected exactly one statement, got %d", len(list))
	}

	listener := &mysqlRewriter{
		limitCount:     limitCount,
		outerMostQuery: true,
	}

	for _, stmt := range list {
		listener.rewriter = *antlr.NewTokenStreamRewriter(stmt.Tokens)
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
		if listener.err != nil {
			return "", errors.Wrapf(listener.err, "statement: %s", statement)
		}
	}
	return listener.rewriter.GetTextDefault(), nil
}

type mysqlRewriter struct {
	*antlrmysql.BaseMySQLParserListener

	rewriter       antlr.TokenStreamRewriter
	err            error
	outerMostQuery bool
	limitCount     int
}

func (r *mysqlRewriter) EnterQueryExpression(ctx *antlrmysql.QueryExpressionContext) {
	if !r.outerMostQuery {
		return
	}
	r.outerMostQuery = false
	limitClause := ctx.LimitClause()
	if limitClause != nil && limitClause.LimitOptions() != nil && len(limitClause.LimitOptions().AllLimitOption()) > 0 {
		userLimitOption := limitClause.LimitOptions().LimitOption(0)
		if limitClause.LimitOptions().COMMA_SYMBOL() != nil {
			userLimitOption = limitClause.LimitOptions().LimitOption(1)
		}

		userLimitText := userLimitOption.GetText()
		limit, _ := strconv.Atoi(userLimitText)
		if limit == 0 || r.limitCount < limit {
			limit = r.limitCount
		}
		r.rewriter.ReplaceDefault(userLimitOption.GetStart().GetTokenIndex(), userLimitOption.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
		return
	}

	if ctx.OrderClause() != nil {
		r.rewriter.InsertAfterDefault(ctx.OrderClause().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
	} else {
		switch {
		case ctx.QueryExpressionBody() != nil:
			r.rewriter.InsertAfterDefault(ctx.QueryExpressionBody().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
		case ctx.QueryExpressionParens() != nil:
			r.rewriter.InsertAfterDefault(ctx.QueryExpressionParens().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
		default:
			// No action needed for other query expression types.
		}
	}
}

func rewriteMySQLSelectLimit(sql string, stmt *ast.SelectStmt, limitCount int) (string, error) {
	if stmt.Limit != nil && stmt.Limit.Count != nil {
		existingLimit := extractMySQLLimit(stmt.Limit.Count)
		if existingLimit >= 0 && existingLimit <= limitCount {
			return sql, nil
		}
		loc := nodeLocOf(stmt.Limit.Count)
		loc = trimMySQLLocSpace(sql, loc)
		if loc.Start >= 0 && loc.End > loc.Start && loc.End <= len(sql) {
			if existingLimit < 0 && stmt.Into == nil {
				return "", errors.Errorf("cannot rewrite non-constant LIMIT expression")
			}
			return sql[:loc.Start] + fmt.Sprintf("%d", limitCount) + sql[loc.End:], nil
		}
		return "", errors.Errorf("cannot rewrite non-constant LIMIT expression")
	}

	insertPos, beforeClause := findMySQLLimitInsertPosition(sql, stmt)
	if insertPos < 0 || insertPos > len(sql) {
		return "", errors.Errorf("invalid LIMIT insert position %d", insertPos)
	}
	if beforeClause {
		return sql[:insertPos] + fmt.Sprintf("LIMIT %d ", limitCount) + sql[insertPos:], nil
	}
	return sql[:insertPos] + fmt.Sprintf(" LIMIT %d", limitCount) + sql[insertPos:], nil
}

func findMySQLLimitInsertPosition(sql string, stmt *ast.SelectStmt) (int, bool) {
	if stmt.Into != nil && stmt.Into.Loc.Start > 0 {
		intoStart := findMySQLKeywordBefore(sql, stmt.Into.Loc.Start, "INTO")
		if intoStart >= maxMySQLLocEndBeforeTailClauses(stmt) {
			return intoStart, true
		}
	}
	if stmt.ForUpdate != nil && stmt.ForUpdate.Loc.Start > 0 {
		return stmt.ForUpdate.Loc.Start, true
	}
	if stmt.Loc.End > 0 {
		return stmt.Loc.End, false
	}
	return -1, false
}

func extractMySQLLimit(node ast.Node) int {
	limit, ok := node.(*ast.IntLit)
	if !ok {
		return -1
	}
	return int(limit.Value)
}

func nodeLocOf(node ast.Node) ast.Loc {
	return mysqlNodeLoc(node)
}

func trimMySQLLocSpace(sql string, loc ast.Loc) ast.Loc {
	for loc.Start < loc.End && loc.Start < len(sql) && (sql[loc.Start] == ' ' || sql[loc.Start] == '\t' || sql[loc.Start] == '\n' || sql[loc.Start] == '\r') {
		loc.Start++
	}
	for loc.End > loc.Start && loc.End <= len(sql) && (sql[loc.End-1] == ' ' || sql[loc.End-1] == '\t' || sql[loc.End-1] == '\n' || sql[loc.End-1] == '\r') {
		loc.End--
	}
	return loc
}

func maxMySQLLocEndBeforeTailClauses(node ast.Node) int {
	maxEnd := -1
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if n == node {
			return true
		}
		if _, ok := n.(*ast.IntoClause); ok {
			return false
		}
		if _, ok := n.(*ast.ForUpdate); ok {
			return false
		}
		if loc := mysqlNodeLoc(n); loc.End > maxEnd {
			maxEnd = loc.End
		}
		return true
	})
	return maxEnd
}

func mysqlNodeLoc(node ast.Node) ast.Loc {
	if node == nil {
		return ast.Loc{Start: -1, End: -1}
	}
	value := reflect.ValueOf(node)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return ast.Loc{Start: -1, End: -1}
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return ast.Loc{Start: -1, End: -1}
	}
	field := elem.FieldByName("Loc")
	if !field.IsValid() || !field.CanInterface() {
		return ast.Loc{Start: -1, End: -1}
	}
	loc, ok := field.Interface().(ast.Loc)
	if !ok {
		return ast.Loc{Start: -1, End: -1}
	}
	return loc
}

func findMySQLKeywordBefore(sql string, offset int, keyword string) int {
	if offset > len(sql) {
		offset = len(sql)
	}
	if len(keyword) == 0 {
		return offset
	}
	keyword = strings.ToUpper(keyword)
	tokens := omnimysqlparser.Tokenize(sql[:offset])
	for i := len(tokens) - 1; i >= 0; i-- {
		token := tokens[i]
		if token.End <= offset && omnimysqlparser.TokenName(token.Type) == keyword {
			return token.Loc
		}
	}
	return offset
}
