package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestRedisDriver(t *testing.T) {
	redisServer := miniredis.RunT(t)

	a := require.New(t)
	ctx := context.Background()

	driver, err := newDriver(db.DriverConfig{}).Open(ctx, db.Redis, db.ConnectionConfig{
		Host: redisServer.Host(),
		Port: redisServer.Port(),
	}, db.ConnectionContext{})
	a.NoError(err)
	defer driver.Close(ctx)

	err = driver.Ping(ctx)
	a.NoError(err)
}
