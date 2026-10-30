package oceanbase

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/require"
)

type closeCounter struct{ closed int }

func (c *closeCounter) Close() error {
	c.closed++
	return nil
}

type noopConnector struct{}

func (noopConnector) Connect(context.Context) (driver.Conn, error) { return nil, driver.ErrBadConn }
func (noopConnector) Driver() driver.Driver                        { return nil }

func TestDriverCloseClosesCloudSQLDialer(t *testing.T) {
	dialer := &closeCounter{}
	d := &Driver{db: sql.OpenDB(noopConnector{}), cloudSQLDialer: dialer}
	require.NoError(t, d.Close(context.Background()))
	require.Equal(t, 1, dialer.closed)
}
