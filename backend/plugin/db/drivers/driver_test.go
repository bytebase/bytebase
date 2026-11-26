package drivers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type estimatingTestDriver struct{}

func (estimatingTestDriver) Ping(context.Context) error { return nil }
func (estimatingTestDriver) Close() error               { return nil }
func (estimatingTestDriver) CountAffectedRows(context.Context, string) (int64, error) {
	return 3, nil
}

func TestDriverCapabilities(t *testing.T) {
	plain := &Driver{OmniAdapter: util.OmniAdapter{Driver: estimatingTestDriver{}}}
	require.Nil(t, db.ProberOf(plain))
	_, ok := db.As[db.TaskRunSessionReader](plain)
	require.False(t, ok)
	estimator, ok := db.As[db.RowEstimator](plain)
	require.True(t, ok)
	rows, err := estimator.CountAffectedRows(context.Background(), "DELETE FROM t")
	require.NoError(t, err)
	require.Equal(t, int64(3), rows)

	probing := &Driver{
		spec:        spec{probe: func(*Driver) util.OmniProbeOptions { return util.OmniProbeOptions{} }},
		OmniAdapter: util.OmniAdapter{Driver: estimatingTestDriver{}},
	}
	prober := db.ProberOf(probing)
	require.NotNil(t, prober)
	// The advisor asks the probe it holds for the driver's other capabilities.
	_, ok = db.As[db.RowEstimator](prober)
	require.True(t, ok)
}
