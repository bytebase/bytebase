package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

func instanceOn(engine storepb.Engine) *store.InstanceMessage {
	return &store.InstanceMessage{
		ResourceID: "inst",
		Metadata:   &storepb.Instance{Engine: engine},
	}
}

func unresolvedSpan() *parserbase.QuerySpan {
	return &parserbase.QuerySpan{
		Type: parserbase.Select,
		UnresolvedColumnsError: &parserbase.UnresolvedColumnsError{
			Relations: []parserbase.ColumnResource{
				{Database: "db", Schema: "public", Table: "t"},
			},
		},
	}
}

// TestMaskingBlockedByUnresolvedColumns pins the single predicate behind both
// the re-sync trigger in queryRetry and the refusal in MaskResults.
//
// These two conditions were originally written out separately and drifted apart
// twice during review: once when only one of them gained the engine check, and
// once when only the other gained a result-shape check. Each divergence is a
// real defect — a re-sync that fires where the refusal will not burns a full
// schema sync on the request path for nothing, and a refusal that fires where
// the re-sync did not denies a query that one sync would have repaired. Routing
// both through this function is what makes them agree; this test is what keeps
// the function honest.
func TestMaskingBlockedByUnresolvedColumns(t *testing.T) {
	testCases := []struct {
		name     string
		span     *parserbase.QuerySpan
		instance *store.InstanceMessage
		want     bool
	}{
		{
			name:     "unresolved columns on a masking engine blocks",
			span:     unresolvedSpan(),
			instance: instanceOn(storepb.Engine_POSTGRES),
			want:     true,
		},
		{
			// CockroachDB shares the PostgreSQL span extractor, so it receives the
			// signal, but it never masks. Acting on it would sync per query and
			// then refuse nothing.
			name:     "engine that never masks does not block",
			span:     unresolvedSpan(),
			instance: instanceOn(storepb.Engine_COCKROACHDB),
			want:     false,
		},
		{
			name:     "resolved span does not block",
			span:     &parserbase.QuerySpan{Type: parserbase.Select},
			instance: instanceOn(storepb.Engine_POSTGRES),
			want:     false,
		},
		{
			name:     "nil span does not block",
			span:     nil,
			instance: instanceOn(storepb.Engine_POSTGRES),
			want:     false,
		},
		{
			name:     "nil instance does not block",
			span:     unresolvedSpan(),
			instance: nil,
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, maskingBlockedByUnresolvedColumns(tc.span, tc.instance))
		})
	}
}

// TestMaskingEnginesAgreeWithSignalProducers records which masking-capable
// engines actually produce the unresolved-columns signal today. Only the
// PostgreSQL extractor sets it, so every other masking engine still returns
// unmasked rows against a snapshot with no column metadata.
//
// This is a scope marker, not a passing grade: when another engine gains the
// signal, add it here so the remaining gap stays visible instead of being
// inferred from the absence of a test.
func TestMaskingEnginesAgreeWithSignalProducers(t *testing.T) {
	// The pg extractor is registered for both of these; only POSTGRES masks.
	require.True(t, maskingBlockedByUnresolvedColumns(unresolvedSpan(), instanceOn(storepb.Engine_POSTGRES)),
		"PostgreSQL both produces the signal and masks, so it must act on it")
	require.False(t, maskingBlockedByUnresolvedColumns(unresolvedSpan(), instanceOn(storepb.Engine_COCKROACHDB)),
		"CockroachDB produces the signal but never masks, so acting on it is pure cost")

	// Masking engines whose extractors do not set the signal yet. They would
	// block if they ever did, which is the intended direction of travel.
	for _, engine := range []storepb.Engine{
		storepb.Engine_MYSQL,
		storepb.Engine_ORACLE,
		storepb.Engine_MSSQL,
		storepb.Engine_REDSHIFT,
	} {
		require.True(t, maskingBlockedByUnresolvedColumns(unresolvedSpan(), instanceOn(engine)),
			"%s masks, so it must act on the signal once its extractor sets one", engine)
	}
}
