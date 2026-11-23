package spanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSpannerValidateQuerySeam(t *testing.T) {
	require.True(t, base.HasQueryValidator(storepb.Engine_SPANNER))

	readOnly, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_SPANNER, "SELECT 1")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, allQuery)

	readOnly, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_SPANNER, "SELECT * FROM t FOR UPDATE")
	require.NoError(t, err)
	require.False(t, readOnly)
	require.False(t, allQuery)

	readOnly, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_SPANNER, "SELECT FROM")
	require.Error(t, err)
	require.False(t, readOnly)
	require.False(t, allQuery)
}
