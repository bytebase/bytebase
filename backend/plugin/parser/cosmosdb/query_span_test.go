package cosmosdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpanSeam(t *testing.T) {
	ctx := context.Background()

	// Registry dispatch, projection SourceFieldPaths, bracket string access,
	// and mergeSelectors folding (ItemSelector + ArraySelector) into base.ArraySelector.
	spans, err := base.GetQuerySpan(
		ctx,
		base.GetQuerySpanContext{},
		storepb.Engine_COSMOSDB,
		[]base.Statement{{Text: `SELECT c.category, c.salary["value"] FROM container c WHERE c.addresses[1].country = 'Canada'`}},
		"",
		"",
		false,
	)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	span := spans[0]
	require.Len(t, span.Results, 1)
	require.False(t, span.Results[0].SelectAsterisk)
	require.Len(t, span.Results[0].SourceFieldPaths["category"], 1)
	catStr, err := span.Results[0].SourceFieldPaths["category"][0].String()
	require.NoError(t, err)
	require.Equal(t, "container.category", catStr)
	require.Len(t, span.Results[0].SourceFieldPaths["value"], 1)
	valStr, err := span.Results[0].SourceFieldPaths["value"][0].String()
	require.NoError(t, err)
	require.Equal(t, "container.salary.value", valStr)
	require.Contains(t, span.PredicatePaths, "container.addresses[1].country")

	// SelectStar case.
	starSpans, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_COSMOSDB, []base.Statement{{Text: "SELECT TOP 5 * FROM c"}}, "", "", false)
	require.NoError(t, err)
	require.Len(t, starSpans, 1)
	require.Len(t, starSpans[0].Results, 1)
	require.True(t, starSpans[0].Results[0].SelectAsterisk)
	require.Empty(t, starSpans[0].Results[0].SourceFieldPaths)

	// Empty statement returns empty results; invalid syntax returns error.
	emptySpans, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_COSMOSDB, []base.Statement{{Text: "  "}}, "", "", false)
	require.NoError(t, err)
	require.Len(t, emptySpans, 1)
	require.Empty(t, emptySpans[0].Results)

	_, err = base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_COSMOSDB, []base.Statement{{Text: "SELCT * FRM c"}}, "", "", false)
	require.Error(t, err)
}
