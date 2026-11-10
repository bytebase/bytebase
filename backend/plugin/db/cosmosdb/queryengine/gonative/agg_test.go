package gonative

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAggregator_KnownKinds(t *testing.T) {
	for _, kind := range []string{"Count", "Sum", "Min", "Max", "Average", "Avg"} {
		t.Run(kind, func(t *testing.T) {
			require.NotNil(t, newAggregator(kind))
		})
	}
}

func TestNewAggregator_UnknownKind(t *testing.T) {
	assert.Nil(t, newAggregator("Median"))
	assert.Nil(t, newAggregator(""))
}

// feedFinal is a small helper — feed a sequence of partials then finalize.
func feedFinal(t *testing.T, a aggregator, partials ...json.RawMessage) json.RawMessage {
	t.Helper()
	for _, p := range partials {
		require.NoError(t, a.feedPartial(p))
	}
	out, err := a.finalize()
	require.NoError(t, err)
	return out
}

// ---------------------------------------------------------- Count

func TestCountAgg_EmptyFinalizesToZero(t *testing.T) {
	out := feedFinal(t, newAggregator("Count"))
	assert.Equal(t, json.RawMessage(`0`), out)
}

func TestCountAgg_SinglePartition(t *testing.T) {
	out := feedFinal(t, newAggregator("Count"), []byte(`5`))
	assert.Equal(t, json.RawMessage(`5`), out)
}

func TestCountAgg_SumsAcrossPartitions(t *testing.T) {
	out := feedFinal(t, newAggregator("Count"), []byte(`5`), []byte(`0`), []byte(`13`))
	assert.Equal(t, json.RawMessage(`18`), out)
}

// ---------------------------------------------------------- Sum

func TestSumAgg_EmptyFinalizesToUndefined(t *testing.T) {
	out := feedFinal(t, newAggregator("Sum"))
	assert.Nil(t, out, "no contributing row means no value, which the pipelines render as no row or an absent alias")
}

func TestSumAgg_SumsAcrossPartitions(t *testing.T) {
	out := feedFinal(t, newAggregator("Sum"), []byte(`10`), []byte(`20.5`), []byte(`-5`))
	// 25.5 encodes as the short form.
	assert.Equal(t, json.RawMessage(`25.5`), out)
}

// ---------------------------------------------------------- Avg

func TestAvgAgg_EmptyFinalizesToUndefined(t *testing.T) {
	out := feedFinal(t, newAggregator("Avg"))
	assert.Nil(t, out, "no contributing row means no value, which the pipelines render as no row or an absent alias")
}

func TestAvgAgg_SinglePartition(t *testing.T) {
	out := feedFinal(t, newAggregator("Avg"), []byte(`{"sum":100,"count":4}`))
	assert.Equal(t, json.RawMessage(`25`), out)
}

func TestAvgAgg_MergesAcrossPartitions(t *testing.T) {
	// (5+3)/(2+2) = 2.
	out := feedFinal(t, newAggregator("Avg"), []byte(`{"sum":5,"count":2}`), []byte(`{"sum":3,"count":2}`))
	assert.Equal(t, json.RawMessage(`2`), out)
}

func TestAvgAgg_ZeroCountPartialContributesNothing(t *testing.T) {
	out := feedFinal(t, newAggregator("Avg"),
		[]byte(`{"sum":0,"count":0}`),
		[]byte(`{"sum":10,"count":5}`),
	)
	assert.Equal(t, json.RawMessage(`2`), out)
}

// ---------------------------------------------------------- partial routing

// A MIN or MAX partial comes from item2, the only one that reports how many
// rows the partition contributed; a partition that matched none contributes
// nothing at all.
func TestAggregatePartial_MinMaxPreferItem2(t *testing.T) {
	both := aggItem{Item: json.RawMessage(`7`), Item2: json.RawMessage(`{"min":7,"count":3}`)}
	assert.JSONEq(t, `{"min":7,"count":3}`, string(aggregatePartial("Min", both)))

	onlyItem := aggItem{Item: json.RawMessage(`7`)}
	assert.JSONEq(t, `{"min":7,"count":1}`, string(aggregatePartial("Min", onlyItem)))
	assert.JSONEq(t, `{"max":7,"count":1}`, string(aggregatePartial("Max", onlyItem)))

	assert.Nil(t, aggregatePartial("Min", aggItem{}))
	assert.Equal(t, json.RawMessage(`7`), aggregatePartial("Count", onlyItem))
}

// An aggregate over a partition that matched no row has no partial for the
// gateway to send, and the aggregate still has to finalize.
func TestFeedAggregatePartial_SkipsAnAbsentPartial(t *testing.T) {
	a := newAggregator("Min")
	require.NoError(t, feedAggregatePartial(a, "Min", aggItem{}))
	out, err := a.finalize()
	require.NoError(t, err)
	assert.Nil(t, out)

	require.NoError(t, feedAggregatePartial(a, "Min", aggItem{Item2: json.RawMessage(`{"count":0}`)}))
	out, err = a.finalize()
	require.NoError(t, err)
	assert.Nil(t, out)
}

// ---------------------------------------------------------- Min

func TestMinAgg_EmptyFinalizesToUndefined(t *testing.T) {
	out := feedFinal(t, newAggregator("Min"))
	assert.Nil(t, out, "no contributing row means no value, which the pipelines render as no row or an absent alias")
}

func TestMinAgg_SinglePartitionNumeric(t *testing.T) {
	out := feedFinal(t, newAggregator("Min"), []byte(`{"min":42,"count":5}`))
	assert.Equal(t, json.RawMessage(`42`), out)
}

func TestMinAgg_AcrossPartitions_TakesSmallest(t *testing.T) {
	out := feedFinal(t, newAggregator("Min"),
		[]byte(`{"min":100,"count":5}`),
		[]byte(`{"min":7,"count":3}`),
		[]byte(`{"min":50,"count":2}`),
	)
	assert.Equal(t, json.RawMessage(`7`), out)
}

func TestMinAgg_IgnoresZeroCountPartitions(t *testing.T) {
	out := feedFinal(t, newAggregator("Min"),
		[]byte(`{"count":0}`), // empty partition — "min" missing, count=0 → skip
		[]byte(`{"min":99,"count":1}`),
	)
	assert.Equal(t, json.RawMessage(`99`), out)
}

func TestMinAgg_MixedTypesUsesCosmosOrdering(t *testing.T) {
	// null < bool < number < string; Min should pick the smallest type-class.
	out := feedFinal(t, newAggregator("Min"),
		[]byte(`{"min":"apple","count":1}`),
		[]byte(`{"min":42,"count":1}`),
	)
	assert.Equal(t, json.RawMessage(`42`), out)
}

// ---------------------------------------------------------- Max

func TestMaxAgg_EmptyFinalizesToUndefined(t *testing.T) {
	out := feedFinal(t, newAggregator("Max"))
	assert.Nil(t, out, "no contributing row means no value, which the pipelines render as no row or an absent alias")
}

func TestMaxAgg_AcrossPartitions_TakesLargest(t *testing.T) {
	out := feedFinal(t, newAggregator("Max"),
		[]byte(`{"max":100,"count":5}`),
		[]byte(`{"max":8982000,"count":3}`),
		[]byte(`{"max":50,"count":2}`),
	)
	assert.Equal(t, json.RawMessage(`8982000`), out)
}

func TestMaxAgg_IgnoresZeroCountPartitions(t *testing.T) {
	out := feedFinal(t, newAggregator("Max"),
		[]byte(`{"count":0}`),
		[]byte(`{"max":7,"count":1}`),
	)
	assert.Equal(t, json.RawMessage(`7`), out)
}

// ---------------------------------------------------------- Error paths

func TestAggregators_RejectBadPartials(t *testing.T) {
	for _, kind := range []string{"Count", "Sum", "Avg", "Min", "Max"} {
		t.Run(kind, func(t *testing.T) {
			a := newAggregator(kind)
			err := a.feedPartial([]byte(`not json`))
			require.Error(t, err)
		})
	}
}

// ---------------------------------------------------------- Format regression

func TestNumberRawMessage_IntegerFormatting(t *testing.T) {
	assert.Equal(t, json.RawMessage(`18`), numberRawMessage(18))
	assert.Equal(t, json.RawMessage(`-5`), numberRawMessage(-5))
	assert.Equal(t, json.RawMessage(`0`), numberRawMessage(0))
	assert.Equal(t, json.RawMessage(`1.5`), numberRawMessage(1.5))
}

// MIN and MAX fold their partials through compareItems, so the composite
// classes reach them too: two ranges reporting the same array agree on the
// answer, while two reporting different arrays have none the values can give.
func TestMinAgg_FoldsEqualComposites(t *testing.T) {
	out := feedFinal(t, newAggregator("Min"),
		[]byte(`{"min":[1,2],"count":2}`),
		[]byte(`{"min":[1,2],"count":3}`),
	)
	assert.Equal(t, json.RawMessage(`[1,2]`), out)
}

func TestMinAgg_RefusesTwoDifferentComposites(t *testing.T) {
	a := newAggregator("Min")
	require.NoError(t, a.feedPartial([]byte(`{"min":[1,2],"count":1}`)))
	err := a.feedPartial([]byte(`{"min":[1,3],"count":1}`))
	assert.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}
