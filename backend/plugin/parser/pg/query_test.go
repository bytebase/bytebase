package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditorSeam(t *testing.T) {
	for _, engine := range []storepb.Engine{storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB} {
		t.Run(engine.String(), func(t *testing.T) {
			valid, allQuery, err := base.ValidateSQLForEditor(engine, "SELECT * FROM t;")
			require.NoError(t, err)
			require.True(t, valid)
			require.True(t, allQuery)

			valid, allQuery, err = base.ValidateSQLForEditor(engine, "SET work_mem = '64MB'; SELECT 1;")
			require.NoError(t, err)
			require.True(t, valid)
			require.False(t, allQuery)

			valid, allQuery, err = base.ValidateSQLForEditor(engine, "DELETE FROM t;")
			require.NoError(t, err)
			require.False(t, valid)
			require.False(t, allQuery)

			_, _, err = base.ValidateSQLForEditor(engine, "SELECT * FROM")
			require.Error(t, err)
		})
	}
}
