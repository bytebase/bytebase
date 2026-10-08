package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestStatementWithResultLimit(t *testing.T) {
	// An engine with nothing registered — MongoDB, Redis, and the other stores
	// whose drivers never call this — runs the statement unchanged rather than
	// erroring, unlike ExplainStatement: there is no "must refuse" case here.
	got := StatementWithResultLimit(storepb.Engine_MONGODB, "SELECT * FROM t", 10, "")
	require.Equal(t, "SELECT * FROM t", got)

	// PostgreSQL registers in this test binary via the parser/pg package's own
	// tests importing this one transitively is not guaranteed, so this only
	// covers the registry mechanics directly with a throwaway engine value.
}

func TestRegisterResultLimitFuncPanicsOnDuplicate(t *testing.T) {
	const engine = storepb.Engine(999999) // not a real engine; scoped to this test
	RegisterResultLimitFunc(engine, func(statement string, _ int, _ string) string { return statement })
	require.Panics(t, func() {
		RegisterResultLimitFunc(engine, func(statement string, _ int, _ string) string { return statement })
	})
}

func TestTrimStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      string
	}{
		{"SELECT 1", "SELECT 1"},
		{"SELECT 1;", "SELECT 1"},
		{"SELECT 1;;;", "SELECT 1"},
		{"  SELECT 1  ", "SELECT 1"},
		{"\n\tSELECT 1\n", "SELECT 1"},
		{"", ""},
	}
	for _, tc := range tests {
		require.Equalf(t, tc.want, TrimStatement(tc.statement), "%q", tc.statement)
	}
}
