package starrocks

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/parser/mysql"

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
				t, err := time.Parse(time.DateTime, raw.String)
				if err != nil {
					slog.Error("failed to parse time value", log.BBError(err))
					return util.NullRowValue
				}
				return &v1pb.RowValue{
					Kind: &v1pb.RowValue_TimestampValue{
						TimestampValue: &v1pb.RowValue_Timestamp{
							GoogleTimestamp: timestamppb.New(t),
							Accuracy:        int32(scale),
						},
					},
				}
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
	}
	return util.NullRowValue
}

func getStatementWithResultLimit(statement string, limit int) string {
	// SHOW statements don't need LIMIT clause wrapping as they typically return small result sets
	// and wrapping them in SELECT * FROM (...) causes syntax errors
	trimmedStatement := strings.TrimSpace(statement)
	if strings.HasPrefix(strings.ToUpper(trimmedStatement), "SHOW") {
		return statement
	}

	stmt, err := getStatementWithResultLimitForMySQL(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// MySQL 5.7 doesn't support WITH clause.
		return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return stmt
}

// singleStatement must be a selectStatement for mysql.
func getStatementWithResultLimitForMySQL(statement string, limitCount int) (string, error) {
	list, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		return "", err
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
	*mysql.BaseMySQLParserListener

	rewriter       antlr.TokenStreamRewriter
	err            error
	outerMostQuery bool
	limitCount     int
}

func (r *mysqlRewriter) EnterQueryExpression(ctx *mysql.QueryExpressionContext) {
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
			// No query expression body or parens found, skip adding LIMIT
		}
	}
}
