package elasticsearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpanSeam(t *testing.T) {
	ctx := context.Background()

	// Registry dispatch, QueryType conversion, ElasticsearchAnalysis (including blocked features), and nested PathAST conversion.
	spans, err := base.GetQuerySpan(
		ctx,
		base.GetQuerySpanContext{},
		storepb.Engine_ELASTICSEARCH,
		[]base.Statement{{Text: "GET /users/_search\n{\"query\":{\"match\":{\"contact.email\":\"alice@example.com\"}},\"aggs\":{\"by_status\":{\"terms\":{\"field\":\"status\"}}}}"}},
		"",
		"",
		false,
	)
	require.NoError(t, err)
	require.Len(t, spans, 1)
	span := spans[0]
	require.NotNil(t, span)
	require.Equal(t, base.Select, span.Type)
	require.NotNil(t, span.ElasticsearchAnalysis)
	require.Equal(t, base.ElasticsearchAPIMaskSearch, span.ElasticsearchAnalysis.API)
	require.Equal(t, "users", span.ElasticsearchAnalysis.Index)
	require.Equal(t, []base.ElasticsearchBlockedFeature{base.ElasticsearchBlockedFeatureAggs}, span.ElasticsearchAnalysis.BlockedFeatures)
	require.Contains(t, span.PredicatePaths, "contact.email")
	pathStr, err := span.PredicatePaths["contact.email"].String()
	require.NoError(t, err)
	require.Equal(t, "contact.email", pathStr)

	// QueryType variants across DML, DDL, Select, and SelectInfoSchema, plus GetDoc API mapping.
	for _, tc := range []struct {
		stmt    string
		want    base.QueryType
		wantAPI base.ElasticsearchMaskableAPI
	}{
		{"POST /users/_doc/1\n{\"name\":\"alice\"}", base.DML, base.ElasticsearchAPIUnsupported},
		{"DELETE /users", base.DDL, base.ElasticsearchAPIUnsupported},
		{"GET /users/_doc/1", base.Select, base.ElasticsearchAPIMaskGetDoc},
		{"GET /_cat/indices", base.SelectInfoSchema, base.ElasticsearchAPIUnsupported},
	} {
		got, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_ELASTICSEARCH, []base.Statement{{Text: tc.stmt}}, "", "", false)
		require.NoError(t, err, tc.stmt)
		require.Len(t, got, 1, tc.stmt)
		require.Equal(t, tc.want, got[0].Type, tc.stmt)
		require.NotNil(t, got[0].ElasticsearchAnalysis, tc.stmt)
		require.Equal(t, tc.wantAPI, got[0].ElasticsearchAnalysis.API, tc.stmt)
	}

	// Syntax error propagation.
	errSpans, err := base.GetQuerySpan(ctx, base.GetQuerySpanContext{}, storepb.Engine_ELASTICSEARCH, []base.Statement{{Text: "db.users.find({})"}}, "", "", false)
	require.Error(t, err)
	require.Nil(t, errSpans)
	require.Contains(t, err.Error(), "syntax error at line 0, column 3")
}
