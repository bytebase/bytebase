package base

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestWithSearchPath(t *testing.T) {
	statement := WithSearchPath("DELETE FROM t;", []string{"app", `we"ird`})
	require.Equal(t, "SET LOCAL search_path TO \"app\", \"we\"\"ird\";\nDELETE FROM t;", statement)
}

func TestRegister(t *testing.T) {
	const (
		engineA = storepb.Engine(999901)
		engineB = storepb.Engine(999902)
	)

	// Disjoint capabilities may be registered across separate calls and shared across engines.
	Register(Engine{
		ValidateQuery: func(string) (bool, bool, error) { return true, false, nil },
	}, engineA, engineB)
	Register(Engine{
		Explain: func(stmt string, _ ExplainFormat) (string, error) { return "EXPLAIN " + stmt, nil },
	}, engineA)

	require.True(t, HasQueryValidator(engineA))
	require.True(t, HasQueryValidator(engineB))
	require.True(t, HasExplainStatement(engineA))
	require.False(t, HasExplainStatement(engineB))

	planned, err := ExplainStatement(engineA, "SELECT 1", ExplainFormatDefault)
	require.NoError(t, err)
	require.Equal(t, "EXPLAIN SELECT 1", planned)

	// Duplicate registration of the same capability panics with capability name.
	require.PanicsWithValue(t, "parser/base: Explain registered twice for 999901", func() {
		Register(Engine{
			Explain: func(stmt string, _ ExplainFormat) (string, error) { return stmt, nil },
		}, engineA)
	})
}
