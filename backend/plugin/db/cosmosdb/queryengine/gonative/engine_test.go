package gonative_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db/cosmosdb/queryengine/gonative"
)

func TestDefaultReturnsNonNilEngine(t *testing.T) {
	e := gonative.Default()
	require.NotNil(t, e)
}

func TestDefaultAdvertisesStage1Features(t *testing.T) {
	// Stage 1 advertises aggregate-family features so the gateway emits plans
	// for VALUE + aliased + multi-aggregate queries.
	feats := gonative.Default().SupportedFeatures()
	assert.Contains(t, feats, "Aggregate")
	assert.Contains(t, feats, "NonValueAggregate")
	assert.Contains(t, feats, "MultipleAggregates")
}

func TestDefaultCreateQueryPipelineRejectsUnparseablePlan(t *testing.T) {
	_, err := gonative.Default().CreateQueryPipeline("SELECT * FROM c", "not json", "{}")
	require.Error(t, err)
}

func TestDefaultCreateQueryPipelineRejectsUnsupportedPlan(t *testing.T) {
	// A plan carrying hasNonStreamingOrderBy (vector/hybrid-search scoring)
	// is out of the engine's scope — the design spec explicitly excludes it.
	plan := `{"partitionedQueryExecutionInfoVersion":2,"queryInfo":{"hasNonStreamingOrderBy":true}}`
	pkranges := `{"PartitionKeyRanges":[{"id":"0"}]}`
	_, err := gonative.Default().CreateQueryPipeline("SELECT ...", plan, pkranges)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gonative.ErrUnsupportedPlanFeature))
}

func TestDefaultCreateReadManyPipelineReturnsUnsupported(t *testing.T) {
	_, err := gonative.Default().CreateReadManyPipeline(nil, "{}", "", 0, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gonative.ErrUnsupportedPlanFeature))
}

func TestDefaultReturnsDistinctEngineInstances(t *testing.T) {
	// Each caller gets its own instance — enables per-engine caches in later stages.
	a := gonative.Default()
	b := gonative.Default()
	assert.NotSame(t, a, b)
}
