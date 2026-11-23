package bigquery

import (
	"context"
	"testing"

	odriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// captureSession records the options the adapter hands the driver's session.
type captureSession struct {
	opts odriver.QueryOptions
}

func (s *captureSession) Query(_ context.Context, _ string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.opts = opts
	return nil, nil
}

func (*captureSession) Close() error { return nil }

// A masked query must reach the driver as masked: the driver withholds rows for
// statements whose columns it cannot trace, which is the only place a runtime
// statement — EXECUTE IMMEDIATE, a stored procedure — can be kept from a masked
// caller.
func TestQueryConnCarriesMaskingMode(t *testing.T) {
	session := &captureSession{}
	d := &Driver{}
	d.SetSession(session)

	_, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{
		Limit:                5,
		MaximumSQLResultSize: 1024,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.True(t, session.opts.MaskingEnabled)
	require.Equal(t, 5, session.opts.Limit)
	require.Equal(t, int64(1024), session.opts.MaxResultBytes)
}
