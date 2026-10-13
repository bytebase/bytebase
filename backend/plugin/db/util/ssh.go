//nolint:revive
package util

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// GetSSHClient returns a ssh client.
func GetSSHClient(ds *storepb.DataSource) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            ds.GetSshUser(),
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if ds.GetSshPrivateKey() != "" {
		signer, err := ssh.ParsePrivateKey([]byte(ds.GetSshPrivateKey()))
		if err != nil {
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	} else {
		// Users may use ssh-agent to store the private key with passphrase,
		// we will try to connect to the ssh-agent to get the private key.
		if conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			defer conn.Close()
			agentClient := agent.NewClient(conn)
			sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}
	// When there's a non empty password add the password AuthMethod.
	if ds.GetSshPassword() != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.PasswordCallback(func() (string, error) {
			return ds.GetSshPassword(), nil
		}))
	}
	// Connect to the SSH Server
	sshConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", ds.GetSshHost(), ds.GetSshPort()), sshConfig)
	if err != nil {
		return nil, err
	}
	return sshConn, nil
}

const sshPortSize = 100

// PortFIFO is the fifo for SSH client port queue.
var PortFIFO = make(chan int, sshPortSize)

func init() {
	for i := 0; i < sshPortSize; i++ {
		PortFIFO <- i + 6113
	}
}

// NoDeadlineConn ignores deadlines, so a read or write blocked on it cannot be
// interrupted. A driver that relies on deadlines dials with DialSSHTunnel.
type NoDeadlineConn struct{ net.Conn }

func (*NoDeadlineConn) SetDeadline(time.Time) error      { return nil }
func (*NoDeadlineConn) SetReadDeadline(time.Time) error  { return nil }
func (*NoDeadlineConn) SetWriteDeadline(time.Time) error { return nil }

// DialSSHTunnel dials addr through client. Unlike the ssh channel it wraps, the
// connection it returns honors deadlines and reports addr as its remote
// address. A driver needs both to stop a query whose context ended: pgx sets a
// deadline to release the blocked read, then dials RemoteAddr to send the
// cancel request.
func DialSSHTunnel(ctx context.Context, client *ssh.Client, network, addr string) (net.Conn, error) {
	channel, err := client.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	return newSSHTunnelConn(channel, sshTunnelAddr{network: network, address: addr}), nil
}

// sshTunnelFlushTimeout bounds how long Close waits for a peer that stopped
// reading to take the bytes Write already accepted.
const sshTunnelFlushTimeout = time.Second

// sshTunnelConn relays an ssh channel through a net.Pipe. The caller reads and
// writes the near end of the pipe, which is what supplies the deadlines.
type sshTunnelConn struct {
	net.Conn
	channel net.Conn
	remote  net.Addr

	// flushed is closed once the channel has every accepted write and is closed.
	flushed   chan struct{}
	closeOnce sync.Once
}

func newSSHTunnelConn(channel net.Conn, remote net.Addr) *sshTunnelConn {
	near, far := net.Pipe()
	c := &sshTunnelConn{Conn: near, channel: channel, remote: remote, flushed: make(chan struct{})}
	go func() {
		_, _ = io.Copy(far, channel)
		_ = far.Close()
	}()
	go func() {
		defer close(c.flushed)
		_, _ = io.Copy(channel, far)
		_ = far.Close()
		_ = channel.Close()
	}()
	return c
}

func (c *sshTunnelConn) LocalAddr() net.Addr  { return c.channel.LocalAddr() }
func (c *sshTunnelConn) RemoteAddr() net.Addr { return c.remote }

// Close closes the channel after the writes accepted so far, as closing a TCP
// socket still sends what is buffered.
func (c *sshTunnelConn) Close() error {
	c.closeOnce.Do(func() {
		_ = c.Conn.Close()
		timer := time.NewTimer(sshTunnelFlushTimeout)
		defer timer.Stop()
		select {
		case <-c.flushed:
		case <-timer.C:
			_ = c.channel.Close()
		}
	})
	return nil
}

type sshTunnelAddr struct{ network, address string }

func (a sshTunnelAddr) Network() string { return a.network }
func (a sshTunnelAddr) String() string  { return a.address }
