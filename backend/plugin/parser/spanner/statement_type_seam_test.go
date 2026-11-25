package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementTypeSeam(t *testing.T) {
	stmts, err := ParseStatements("CALL cancel_query('12345');")
	require.NoError(t, err)
	got, err := GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{storepb.StatementType_EXECUTE}, got)
}

func TestDiagnoseSeam(t *testing.T) {
	require.Empty(t, Diagnose("SELECT 1;"))
	require.NotEmpty(t, Diagnose("SELECT FROM"))
}
