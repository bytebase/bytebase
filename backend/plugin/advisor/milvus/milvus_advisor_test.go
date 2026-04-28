package milvus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/milvus"
)

func TestTableDisallowDDLAdvisor(t *testing.T) {
	advices := runRule(t, &storepb.SQLReviewRule{
		Type:  storepb.SQLReviewRule_TABLE_DISALLOW_DDL,
		Level: storepb.SQLReviewRule_ERROR,
	}, "create collection c1 with {\"dimension\":4}; drop collection c1; release collection c1;")

	require.Len(t, advices, 3)
	for _, advice := range advices {
		require.Equal(t, advisorcode.TableDisallowDDL.Int32(), advice.Code)
	}
	require.Contains(t, advices[1].Content, "destructive/disruptive")
}

func TestTableDisallowDMLAdvisor(t *testing.T) {
	advices := runRule(t, &storepb.SQLReviewRule{
		Type:  storepb.SQLReviewRule_TABLE_DISALLOW_DML,
		Level: storepb.SQLReviewRule_ERROR,
	}, "insert into c1 values {\"id\":1}; upsert into c1 values {\"id\":2}; delete from c1 where id > 0;")

	require.Len(t, advices, 3)
	for _, advice := range advices {
		require.Equal(t, advisorcode.TableDisallowDML.Int32(), advice.Code)
	}
}

func TestIndexTypeAllowListAdvisor(t *testing.T) {
	t.Run("allowlist violation", func(t *testing.T) {
		advices := runRule(t, &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST,
			Level: storepb.SQLReviewRule_ERROR,
			Payload: &storepb.SQLReviewRule_StringArrayPayload{
				StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"HNSW"}},
			},
		}, `create index on c1 field vector with {"indexType":"IVF_FLAT","metricType":"L2"};`)

		require.Len(t, advices, 1)
		require.Equal(t, advisorcode.IndexTypeNotAllowed.Int32(), advices[0].Code)
		require.Contains(t, advices[0].Content, "allowlist")
	})

	t.Run("metric incompatibility", func(t *testing.T) {
		advices := runRule(t, &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST,
			Level: storepb.SQLReviewRule_ERROR,
		}, `create index on c1 field vector with {"indexType":"BIN_IVF_FLAT","metricType":"L2"};`)

		require.Len(t, advices, 1)
		require.Equal(t, advisorcode.IndexTypeNotAllowed.Int32(), advices[0].Code)
		require.Contains(t, advices[0].Content, "incompatible")
	})

	t.Run("compatible configuration", func(t *testing.T) {
		advices := runRule(t, &storepb.SQLReviewRule{
			Type:  storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST,
			Level: storepb.SQLReviewRule_ERROR,
			Payload: &storepb.SQLReviewRule_StringArrayPayload{
				StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{List: []string{"HNSW", "IVF_FLAT"}},
			},
		}, `create index on c1 field vector with {"indexType":"HNSW","metricType":"COSINE"};`)

		require.Empty(t, advices)
	})
}

func runRule(t *testing.T, rule *storepb.SQLReviewRule, statement string) []*storepb.Advice {
	t.Helper()
	parsed, err := base.ParseStatements(storepb.Engine_MILVUS, statement)
	require.NoError(t, err)

	adviceList, err := advisor.Check(context.Background(), storepb.Engine_MILVUS, rule.Type, advisor.Context{
		DBType:            storepb.Engine_MILVUS,
		Rule:              rule,
		ParsedStatements:  parsed,
		CurrentDatabase:   "milvus",
		EnablePriorBackup: true,
	})
	require.NoError(t, err)
	return adviceList
}
