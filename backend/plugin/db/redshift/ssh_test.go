package redshift

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestPingThroughSSHTunnelReturnsWhenContextEnds(t *testing.T) {
	t.Parallel()
	sshServer := testcontainer.NewSSHServer(t)
	// The server accepts the connection and never answers the startup message.
	host, port := testcontainer.NewTCPServer(t, func(net.Conn) {})

	background := context.Background()
	driver, err := (&Driver{}).Open(background, storepb.Engine_REDSHIFT, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:          host,
			Port:          port,
			Username:      "bytebase",
			SshHost:       sshServer.GetHost(),
			SshPort:       sshServer.GetPort(),
			SshUser:       sshServer.GetUser(),
			SshPrivateKey: sshServer.GetPrivateKey(),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(background) })

	ctx, cancel := context.WithTimeout(background, 500*time.Millisecond)
	defer cancel()
	pinged := make(chan error, 1)
	go func() { pinged <- driver.Ping(ctx) }()

	select {
	case err := <-pinged:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "Ping was still blocked long after its context ended")
	}
}
