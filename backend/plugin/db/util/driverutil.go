// Package util provides the library for database drivers.
//
//nolint:revive
package util

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/types/known/structpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/utils"
)

func BuildAffectedRowsResult(affectedRows int64, messages []*v1pb.QueryResult_Message) *v1pb.QueryResult {
	return &v1pb.QueryResult{
		ColumnNames:     []string{"Affected Rows"},
		ColumnTypeNames: []string{"INT"},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{
						Kind: &v1pb.RowValue_Int64Value{
							Int64Value: affectedRows,
						},
					},
				},
			},
		},
		Messages: messages,
	}
}

var NullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
}

// TrimStatement trims the unused characters from a statement before running it.
func TrimStatement(statement string) string {
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace)
}

func FormatMaximumSQLResultSizeMessage(limit int64) string {
	return fmt.Sprintf("Output of query exceeds max allowed output size of %dMB", limit/1024/1024)
}
