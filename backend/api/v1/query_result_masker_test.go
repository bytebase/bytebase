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
// the re-sync trigger in queryRetry and the refusal in MaskResults; why they
// must agree is at the function itself.
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

// TestMaskingEnginesAgreeWithSignalProducers pins that enforcement keys on
// EngineSupportMasking rather than on the one engine that produces the signal
// today. Narrowing the predicate to POSTGRES passes every other test in this
// package, so this is the only thing standing between a later edit and an
// engine-specific gate. Add an engine as it gains a producer, so the remaining
// gap stays visible rather than being inferred from a missing test.
func TestMaskingEnginesAgreeWithSignalProducers(t *testing.T) {
	// Masking engines whose extractors do not set the signal yet. They must block
	// once one does; POSTGRES and COCKROACHDB are covered by the table above.
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
