package spanner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementTypeSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_SPANNER, "CALL cancel_query('12345');")
	require.NoError(t, err)
	got, err := GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{storepb.StatementType_EXECUTE}, got)
}

func TestDiagnoseSeam(t *testing.T) {
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_SPANNER, "SELECT 1;")
	require.NoError(t, err)
	require.Empty(t, diags)
	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_SPANNER, "SELECT FROM")
	require.NoError(t, err)
	require.NotEmpty(t, diags)
}
