package redshift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnoseSeam(t *testing.T) {
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_REDSHIFT, "SELECT * FROM sales QUALIFY row_number() OVER () = 1;")
	require.NoError(t, err)
	require.Empty(t, diags)

	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_REDSHIFT, "SELECT '你好';\nSELECT * FROM")
	require.NoError(t, err)
	require.Len(t, diags, 1)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)
}
