package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestBigQueryValidateQuerySeam(t *testing.T) {
	require.True(t, base.HasQueryValidator(storepb.Engine_BIGQUERY))

	readOnly, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_BIGQUERY, "SELECT 1")
	require.NoError(t, err)
	require.True(t, readOnly)
	require.True(t, allQuery)

	readOnly, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_BIGQUERY, "SELECT 1; DELETE FROM t WHERE TRUE")
	require.NoError(t, err)
	require.False(t, readOnly)
	require.False(t, allQuery)

	readOnly, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_BIGQUERY, "SELECT FROM")
	require.Error(t, err)
	require.False(t, readOnly)
	require.False(t, allQuery)
}
