package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainSeam(t *testing.T) {
	got, err := base.ExplainStatement(storepb.Engine_MONGODB, `db.orders.find({a: 1})`, base.ExplainFormatJSON)
	require.NoError(t, err)
	require.Equal(t, `db.orders.find({a: 1}).explain("queryPlanner")`, got)

	_, err = base.ExplainStatement(storepb.Engine_MONGODB, `db.orders.insertOne({a: 1})`, base.ExplainFormatJSON)
	require.Error(t, err)

	format, executed, isExplain := DescribeExplain(`db.orders.find({}).explain("executionStats")`)
	require.True(t, isExplain)
	require.True(t, executed)
	require.Equal(t, "json", format)

	format, executed, isExplain = DescribeExplain(`db.orders.find({})`)
	require.False(t, isExplain)
	require.False(t, executed)
	require.Empty(t, format)
}
