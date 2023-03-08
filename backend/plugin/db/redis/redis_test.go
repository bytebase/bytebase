package redis

import (
	"context"
	"strings"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestRedisDriver(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	defer redisClient.Close()

	a := require.New(t)
	ctx := context.Background()

	connStr := strings.Split(redisClient.Options().Addr, ":")
	a.Equal(2, len(connStr))
	host := connStr[0]
	port := connStr[1]

	driver, err := newDriver(db.DriverConfig{}).Open(ctx, db.Redis, db.ConnectionConfig{
		Host: host,
		Port: port,
	}, db.ConnectionContext{})
	a.NoError(err)
	defer driver.Close(ctx)

	err = driver.Ping(ctx)
	a.NoError(err)
}
