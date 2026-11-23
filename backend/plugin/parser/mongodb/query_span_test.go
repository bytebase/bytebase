package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpanSeam(t *testing.T) {
	ctx := context.Background()

	// Registry dispatch, Type, MongoDBAnalysis, and dotPathToPathAST nested path conversion.
	spans, err := base.GetQuerySpan(
		ctx,
		base.GetQuerySpanContext{},
		storepb.Engine_MONGODB,
		[]base.Statement{{Text: `db["users"].findOne({"contact.phone": "123"})`}},
		"",
		"",
		false,
	)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	span := spans[0]
	require.Equal(t, base.Select, span.Type)
	require.NotNil(t, span.MongoDBAnalysis)
	require.Equal(t, base.MongoDBMaskableAPIFindOne, span.MongoDBAnalysis.API)
	require.Contains(t, span.PredicatePaths, "contact.phone")
	pathStr, err := span.PredicatePaths["contact.phone"].String()
	require.NoError(t, err)
	require.Equal(t, "contact.phone", pathStr)

	// Non-read operation returns DML with no masking analysis or predicate paths.
	dmlSpans, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_MONGODB, []base.Statement{{Text: `db.users.insertOne({name: "bob"})`}}, "", "", false)
	require.NoError(t, err)
	require.Len(t, dmlSpans, 1)
	require.Equal(t, base.DML, dmlSpans[0].Type)
	require.Nil(t, dmlSpans[0].MongoDBAnalysis)
	require.Empty(t, dmlSpans[0].PredicatePaths)

	// Syntax error propagates error through registry.
	_, err = base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_MONGODB, []base.Statement{{Text: `db.users.find({`}}, "", "", false)
	require.Error(t, err)
}
