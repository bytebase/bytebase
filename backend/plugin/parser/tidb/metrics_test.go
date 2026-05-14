package tidb

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestClassifyOmniParseError pins the dispatcher fallback classifier
// against the actual omni error strings — empirically derived against
// omni v0.0.0-20260513072939-39c04c4cca0f. The classifier must label each
// known Tier-4 grammar gap correctly so the
// tidb_dispatcher_omni_fallback_total{reason} counter drives the
// invariant #8 retirement gate (otherwise we ship Option B blind).
//
// Critical empirical finding (motivating the err+sql two-arg signature):
// CREATE SEQUENCE seq; produces an omni error string of "unexpected token
// after CREATE (line 1, column 8)" — the keyword SEQUENCE never appears
// in the error. A classifier that matched only against err.Error() would
// silently mis-label SEQUENCE rejections as "unknown" and lose the entire
// SEQUENCE telemetry signal. The sql parameter is required, not optional.
func TestClassifyOmniParseError(t *testing.T) {
	cases := []struct {
		name     string
		sql      string
		wantOK   bool   // expect parse to fail
		expected string // counter label
	}{
		{
			name:     "FLASHBACK keyword in error msg AND input",
			sql:      "FLASHBACK TABLE foo TO BEFORE DROP;",
			wantOK:   false,
			expected: "flashback",
		},
		{
			name:     "SEQUENCE keyword ONLY in input (omni err says 'after CREATE')",
			sql:      "CREATE SEQUENCE seq;",
			wantOK:   false,
			expected: "sequence",
		},
		{
			name:     "BATCH keyword in error msg AND input",
			sql:      "BATCH ON id LIMIT 100 DELETE FROM t WHERE 1=1;",
			wantOK:   false,
			expected: "batch_dml",
		},
		{
			name:     "lowercase batch in input is matched (case-insensitive)",
			sql:      "batch on id limit 100 delete from t where 1=1;",
			wantOK:   false,
			expected: "batch_dml",
		},
		{
			name:     "genuine syntax error → unknown (no Tier-4 keyword present)",
			sql:      "SELECT FROM WHERE;",
			wantOK:   false,
			expected: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseTiDBOmni(tc.sql)
			require.Error(t, err, "test setup expects this input to be rejected by current omni")
			got := classifyOmniParseError(err, tc.sql)
			require.Equal(t, tc.expected, got,
				"classifier must return the correct label for omni err %q + sql %q",
				err.Error(), tc.sql)
		})
	}
}

// TestClassifyOmniParseError_NilErr pins the contract that a nil error
// returns "unknown" — the classifier is called only on the fallback path
// in practice, but defensive against future refactors that might call it
// on success.
func TestClassifyOmniParseError_NilErr(t *testing.T) {
	require.Equal(t, "unknown", classifyOmniParseError(nil, "anything"))
}

// TestClassifyOmniParseError_NonOmniErr pins behavior for an error type
// that is not an *omniparser.ParseError — the haystack is just
// err.Error(), so non-omni errors get classified normally too. Useful for
// future-proofing if the dispatcher ever wraps the omni error.
func TestClassifyOmniParseError_NonOmniErr(t *testing.T) {
	plainErr := errors.New("some FLASHBACK-shaped wrapper")
	require.Equal(t, "flashback",
		classifyOmniParseError(plainErr, "irrelevant"),
		"classifier matches the haystack regardless of error type")
}
