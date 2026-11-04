package gonative

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos/queryengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------ canonicalJSON

func TestCanonicalJSON_SortsObjectKeys(t *testing.T) {
	a, err := canonicalJSON([]byte(`{"a": 1, "b": 2}`))
	require.NoError(t, err)
	b, err := canonicalJSON([]byte(`{"b": 2, "a": 1}`))
	require.NoError(t, err)
	assert.Equal(t, string(a), string(b),
		"objects with same members but different key order should canonicalize equal")
}

func TestCanonicalJSON_DropsExtraneousWhitespace(t *testing.T) {
	a, err := canonicalJSON([]byte(`  { "k" : "v" }  `))
	require.NoError(t, err)
	b, err := canonicalJSON([]byte(`{"k":"v"}`))
	require.NoError(t, err)
	assert.Equal(t, string(a), string(b))
}

func TestCanonicalJSON_DistinctScalars(t *testing.T) {
	a, err := canonicalJSON([]byte(`"AE"`))
	require.NoError(t, err)
	b, err := canonicalJSON([]byte(`"AD"`))
	require.NoError(t, err)
	assert.NotEqual(t, string(a), string(b))
}

func TestCanonicalJSON_NumericEquivalence(t *testing.T) {
	// JSON 1 and 1.0 are both valid numerics; Go unmarshal treats both as float64,
	// then re-marshal emits the shortest representation. Both → "1" (same canonical bytes).
	a, err := canonicalJSON([]byte(`1`))
	require.NoError(t, err)
	b, err := canonicalJSON([]byte(`1.0`))
	require.NoError(t, err)
	assert.Equal(t, string(a), string(b))
}

// ------------------------------------------------------------ distinct pipeline

// feedSingle exercises the pipeline through a full cycle with one partition's
// worth of documents and returns the emitted rows.
func feedSingle(t *testing.T, p *distinctPipeline, docs []json.RawMessage) [][]byte {
	t.Helper()

	// First Run() requests partitions.
	res, err := p.Run()
	require.NoError(t, err)
	require.NotEmpty(t, res.Requests, "expected initial partition requests")

	// Simulate the SDK delivering the partition body.
	body := map[string]any{"Documents": docs, "_count": len(docs)}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.NoError(t, p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: raw},
	}))

	// Flush pending items.
	var out [][]byte
	for {
		r, err := p.Run()
		require.NoError(t, err)
		out = append(out, r.Items...)
		if r.IsCompleted || (len(r.Items) == 0 && len(r.Requests) == 0 && p.IsComplete()) {
			break
		}
	}
	return out
}

func rawMsgs(ss ...string) []json.RawMessage {
	out := make([]json.RawMessage, len(ss))
	for i, s := range ss {
		out[i] = json.RawMessage(s)
	}
	return out
}

func TestDistinctPipeline_SinglePartitionEmitsAll(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	p, err := newDistinctPipeline(plan, []string{"0"})
	require.NoError(t, err)
	p.query = `SELECT DISTINCT c.country FROM c`

	items := feedSingle(t, p, rawMsgs(`{"country":"AD"}`, `{"country":"AE"}`, `{"country":"DE"}`))
	assert.Len(t, items, 3)
	assert.True(t, p.IsComplete())
}

func TestDistinctPipeline_DedupesWithinPartition(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	p, err := newDistinctPipeline(plan, []string{"0"})
	require.NoError(t, err)

	items := feedSingle(t, p, rawMsgs(`{"country":"AE"}`, `{"country":"AE"}`, `{"country":"AD"}`))
	assert.Len(t, items, 2)
}

func TestDistinctPipeline_DedupesAcrossPartitions(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	p, err := newDistinctPipeline(plan, []string{"0", "1", "2"})
	require.NoError(t, err)

	// Kick off the request fan-out.
	res, err := p.Run()
	require.NoError(t, err)
	require.Len(t, res.Requests, 3)

	mkBody := func(docs ...string) []byte {
		b, _ := json.Marshal(map[string]any{"Documents": rawMsgs(docs...)})
		return b
	}
	require.NoError(t, p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: mkBody(`{"c":"AE"}`, `{"c":"US"}`)},
		{PartitionKeyRangeID: "1", Data: mkBody(`{"c":"US"}`, `{"c":"GB"}`)},
		{PartitionKeyRangeID: "2", Data: mkBody(`{"c":"AE"}`, `{"c":"DE"}`)},
	}))

	var out [][]byte
	for i := 0; i < 5 && !p.IsComplete(); i++ {
		r, err := p.Run()
		require.NoError(t, err)
		out = append(out, r.Items...)
	}
	// Four unique countries: AE, US, GB, DE.
	assert.Len(t, out, 4)
}

func TestDistinctPipeline_TreatsKeyReorderedObjectsAsDuplicates(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	p, err := newDistinctPipeline(plan, []string{"0"})
	require.NoError(t, err)

	items := feedSingle(t, p, rawMsgs(
		`{"a": 1, "b": 2}`,
		`{"b": 2, "a": 1}`, // same value, different key order
	))
	assert.Len(t, items, 1, "canonicalization must treat these as the same")
}

func TestDistinctPipeline_DistinctValueScalars(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	plan.QueryInfo.HasSelectValue = true
	p, err := newDistinctPipeline(plan, []string{"0"})
	require.NoError(t, err)

	// DISTINCT VALUE emits bare scalars in Documents[].
	items := feedSingle(t, p, rawMsgs(`"1"`, `"2"`, `"1"`, `"3"`))
	assert.Len(t, items, 3)
}

func TestDistinctPipeline_RejectsOrderedDistinctType(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Ordered"
	_, err := newDistinctPipeline(plan, []string{"0"})
	require.ErrorIs(t, err, ErrUnsupportedPlanFeature)
}

// A value already emitted from an earlier page must not be emitted again when
// a later page of the same range repeats it.
func TestDistinctPipeline_DedupesAcrossPages(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	p, err := newDistinctPipeline(plan, []string{"0"})
	require.NoError(t, err)

	rows, read := drivePages(t, p, map[string][]page{
		"0": {
			{body: mkDistinctBody(t, `"AE"`, `"BR"`), continuation: "0-page-2"},
			{body: mkDistinctBody(t, `"BR"`, `"CN"`)},
		},
	})
	got := make([]string, 0, len(rows))
	for _, r := range rows {
		got = append(got, string(r))
	}
	assert.Equal(t, []string{`"AE"`, `"BR"`, `"CN"`}, got)
	assert.Equal(t, map[string]int{"0": 2}, read)
}

// mkDistinctBody wraps raw JSON values into a DISTINCT partition response.
func mkDistinctBody(t *testing.T, values ...string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{"Documents": rawMsgs(values...)})
	require.NoError(t, err)
	return body
}

// Verifies the pipeline accepts high-cardinality inputs unchanged — the seen-set
// grows unbounded, matching the .NET SDK's UnorderedDistinctMap behavior.
func TestDistinctPipeline_HighCardinality(t *testing.T) {
	plan := &planDoc{}
	plan.QueryInfo.DistinctType = "Unordered"
	p, err := newDistinctPipeline(plan, []string{"0"})
	require.NoError(t, err)

	_, _ = p.Run()
	docs := make([]json.RawMessage, 100)
	for i := range docs {
		docs[i] = json.RawMessage(fmt.Sprintf(`%d`, i))
	}
	body, _ := json.Marshal(map[string]any{"Documents": docs})
	require.NoError(t, p.ProvideData([]queryengine.QueryResult{
		{PartitionKeyRangeID: "0", Data: body},
	}))
}
