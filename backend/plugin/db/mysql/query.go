package mysql

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var nullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
}

func rowsToQueryResult(rows *sql.Rows, limit int64) (*v1pb.QueryResult, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// DatabaseTypeName returns the database system name of the column type.
	// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
	var columnTypeNames []string
	for _, v := range columnTypes {
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}
	result := &v1pb.QueryResult{
		ColumnNames:     columnNames,
		ColumnTypeNames: columnTypeNames,
	}
	columnLength := len(columnNames)

	if columnLength > 0 {
		for rows.Next() {
			values := make([]any, columnLength)
			for i, v := range columnTypeNames {
				values[i] = makeValueByTypeName(v)
			}

			if err := rows.Scan(values...); err != nil {
				return nil, err
			}

			row := &v1pb.QueryRow{}
			for i := 0; i < columnLength; i++ {
				rowValue := nullRowValue
				switch raw := values[i].(type) {
				case *sql.NullString:
					if raw.Valid {
						if columnTypeNames[i] == "TIMESTAMP" || columnTypeNames[i] == "DATETIME" {
							t, err := time.Parse(time.DateTime, raw.String)
							if err != nil {
								return nil, err
							}
							rowValue = &v1pb.RowValue{
								Kind: &v1pb.RowValue_TimestampValue{
									TimestampValue: timestamppb.New(t),
								},
							}
						} else {
							rowValue = &v1pb.RowValue{
								Kind: &v1pb.RowValue_StringValue{
									StringValue: raw.String,
								},
							}
						}
					}
				case *sql.NullInt64:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_Int64Value{
								Int64Value: raw.Int64,
							},
						}
					}
				case *[]byte:
					if len(*raw) > 0 {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_BytesValue{
								BytesValue: *raw,
							},
						}
					}
				case *sql.NullBool:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_BoolValue{
								BoolValue: raw.Bool,
							},
						}
					}
				case *sql.NullFloat64:
					if raw.Valid {
						rowValue = &v1pb.RowValue{
							Kind: &v1pb.RowValue_DoubleValue{
								DoubleValue: raw.Float64,
							},
						}
					}
				}

				row.Values = append(row.Values, rowValue)
			}

			result.Rows = append(result.Rows, row)
			n := len(result.Rows)
			if (n&(n-1) == 0) && int64(proto.Size(result)) > limit {
				result.Error = common.FormatMaximumSQLResultSizeMessage(limit)
				break
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func makeValueByTypeName(typeName string) any {
	switch typeName {
	case "VARCHAR", "TEXT", "UUID", "DATE":
		return new(sql.NullString)
	case "TIMESTAMP", "DATETIME":
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
		firstOption := limitClause.LimitOptions().LimitOption(0)
		userLimitText := firstOption.GetText()
		limit, _ := strconv.Atoi(userLimitText)
		if limit == 0 || r.limitCount < limit {
			limit = r.limitCount
		}
		r.rewriter.ReplaceDefault(firstOption.GetStart().GetTokenIndex(), firstOption.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
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
		}
	}
}
