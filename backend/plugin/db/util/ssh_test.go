package util

import (
	"context"
	"crypto/rand"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func newSSHClient(t *testing.T) *ssh.Client {
	t.Helper()
	sshServer := testcontainer.NewSSHServer(t)
	client, err := GetSSHClient(&storepb.DataSource{
		SshHost:       sshServer.GetHost(),
		SshPort:       sshServer.GetPort(),
		SshUser:       sshServer.GetUser(),
		SshPrivateKey: sshServer.GetPrivateKey(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// openTunnel returns a tunneled connection to a TCP server that runs handle
// on it, with the address the tunnel was asked to reach.
func openTunnel(t *testing.T, handle func(net.Conn)) (net.Conn, string) {
	t.Helper()
	addr := net.JoinHostPort(testcontainer.NewTCPServer(t, handle))
	conn, err := DialSSHTunnel(context.Background(), newSSHClient(t), "tcp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn, addr
}

func silent(net.Conn) {}

func echo(conn net.Conn) { _, _ = io.Copy(conn, conn) }

// within fails the test unless f returns in time, so a call the connection
// fails to interrupt reads as a failure rather than a hung test binary.
func within[T any](t *testing.T, timeout time.Duration, f func() T) T {
	t.Helper()
	done := make(chan T, 1)
	go func() { done <- f() }()
	select {
	case v := <-done:
		return v
	case <-time.After(timeout):
		require.FailNow(t, "the call was still blocked", "after %s", timeout)
		var zero T
		return zero
	}
}

func requireTimeout(t *testing.T, err error) {
	t.Helper()
	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	require.True(t, netErr.Timeout(), "want a timeout, got %v", err)
}

func TestSSHTunnelConnRelaysBothDirections(t *testing.T) {
	t.Parallel()
	conn, _ := openTunnel(t, echo)

	// Larger than the ssh channel window, so the relay has to keep pace in both directions.
	sent := make([]byte, 4<<20)
	_, err := rand.Read(sent)
	require.NoError(t, err)
	go func() { _, _ = conn.Write(sent) }()

	received := make([]byte, len(sent))
	err = within(t, 30*time.Second, func() error {
		_, err := io.ReadFull(conn, received)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, sent, received)
}

func TestSSHTunnelConnReportsDialedAddress(t *testing.T) {
	t.Parallel()
	conn, addr := openTunnel(t, silent)

	require.Equal(t, "tcp", conn.RemoteAddr().Network())
	require.Equal(t, addr, conn.RemoteAddr().String())
}

func TestSSHTunnelConnReadDeadlineInterruptsBlockedRead(t *testing.T) {
	t.Parallel()
	conn, _ := openTunnel(t, echo)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(100*time.Millisecond)))
	err := within(t, 10*time.Second, func() error {
		_, err := conn.Read(make([]byte, 1))
		return err
	})
	requireTimeout(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Time{}))
	_, err = conn.Write([]byte("ping"))
	require.NoError(t, err)
	reply := make([]byte, 4)
	_, err = io.ReadFull(conn, reply)
	require.NoError(t, err, "the connection must stay usable once the deadline is cleared")
	require.Equal(t, "ping", string(reply))
}

func TestSSHTunnelConnWriteDeadlineInterruptsBlockedWrite(t *testing.T) {
	t.Parallel()
	conn, _ := openTunnel(t, silent)

	require.NoError(t, conn.SetWriteDeadline(time.Now().Add(500*time.Millisecond)))
	chunk := make([]byte, 1<<20)
	err := within(t, 30*time.Second, func() error {
		// The peer never reads, so the write blocks once the buffers on the way to it fill up.
		for {
			if _, err := conn.Write(chunk); err != nil {
				return err
			}
		}
	})
	requireTimeout(t, err)
}

func TestSSHTunnelConnCloseInterruptsBlockedRead(t *testing.T) {
	t.Parallel()
	conn, _ := openTunnel(t, silent)

	readErr := make(chan error, 1)
	go func() {
		_, err := conn.Read(make([]byte, 1))
		readErr <- err
	}()
	require.NoError(t, conn.Close())
	require.Error(t, within(t, 10*time.Second, func() error { return <-readErr }))
}

func TestSSHTunnelConnCloseDeliversAcceptedWrites(t *testing.T) {
	t.Parallel()
	received := make(chan []byte, 1)
	conn, _ := openTunnel(t, func(conn net.Conn) {
		data, _ := io.ReadAll(conn)
		received <- data
	})

	sent := make([]byte, 256<<10)
	_, err := rand.Read(sent)
	require.NoError(t, err)
	_, err = conn.Write(sent)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	require.Equal(t, sent, within(t, 10*time.Second, func() []byte { return <-received }),
		"the peer must see every accepted byte, then the end of the stream")
}

func TestSSHTunnelConnReadsEOFWhenPeerCloses(t *testing.T) {
	t.Parallel()
	conn, _ := openTunnel(t, func(conn net.Conn) {
		_, _ = conn.Write([]byte("bye"))
		_ = conn.Close()
	})

	var data []byte
	err := within(t, 10*time.Second, func() (err error) {
		data, err = io.ReadAll(conn)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "bye", string(data))
}

func TestDialSSHTunnelReturnsWhenContextEnds(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := DialSSHTunnel(ctx, newSSHClient(t), "tcp", net.JoinHostPort(testcontainer.NewTCPServer(t, silent)))
	require.ErrorIs(t, err, context.Canceled)
}
