package mongodb

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterGetQuerySpan(storepb.Engine_MONGODB, GetQuerySpan)
}

// GetQuerySpan returns a QuerySpan for a MongoDB shell statement.
// It classifies the statement type for access control purposes.
// MongoDB does not support column-level tracking, so only the QueryType is set.
func GetQuerySpan(_ context.Context, _ base.GetQuerySpanContext, stmt base.Statement, _ string, _ string, _ bool) (*base.QuerySpan, error) {
	queryType := GetQueryType(stmt.Text)
	return &base.QuerySpan{
		Type: queryType,
	}, nil
}
