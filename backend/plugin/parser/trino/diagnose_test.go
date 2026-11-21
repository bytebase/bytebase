package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnoseSeam(t *testing.T) {
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_TRINO, "SELECT * FROM users")
	require.NoError(t, err)
	require.Empty(t, diags)

	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_TRINO, "SELECT * FFROM users")
	require.NoError(t, err)
	require.NotEmpty(t, diags)

	_, err = parseTrinoSQL("SELECT * FFROM users")
	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.NotNil(t, syntaxErr.Position)
}
