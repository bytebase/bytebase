package testcontainer

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// SSHServer is an in-process SSH bastion for tests that reach a database
// through an SSH tunnel. It runs in the test binary rather than in a container,
// so the addresses it forwards to are the ones the test itself can dial.
type SSHServer struct {
	host       string
	port       string
	privateKey string
}

func (*SSHServer) GetUser() string { return "bytebase" }

func (s *SSHServer) GetHost() string { return s.host }
func (s *SSHServer) GetPort() string { return s.port }

// GetPrivateKey is the PEM-encoded key the server accepts, for any user.
func (s *SSHServer) GetPrivateKey() string { return s.privateKey }

// NewSSHServer starts an SSH server that forwards TCP connections and stops it
// when the test ends.
func NewSSHServer(t testing.TB) *SSHServer {
	t.Helper()

	_, hostKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	hostSigner, err := ssh.NewSignerFromKey(hostKey)
	require.NoError(t, err)

	clientPublicKey, clientKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	authorizedKey, err := ssh.NewPublicKey(clientPublicKey)
	require.NoError(t, err)
	clientKeyPEM, err := ssh.MarshalPrivateKey(clientKey, "")
	require.NoError(t, err)

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if !bytes.Equal(key.Marshal(), authorizedKey.Marshal()) {
				return nil, errors.New("unknown public key")
			}
			return nil, nil
		},
	}
	config.AddHostKey(hostSigner)

	host, port := NewTCPServer(t, func(conn net.Conn) { serveSSHConn(conn, config) })
	return &SSHServer{host: host, port: port, privateKey: string(pem.EncodeToMemory(clientKeyPEM))}
}

// NewTCPServer listens on a loopback port and runs handle on each connection it
// accepts. A connection stays open after handle returns, until the test ends,
// so a handle that does nothing is a server that never answers.
func NewTCPServer(t testing.TB, handle func(net.Conn)) (host, port string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, port, err = net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	var mu sync.Mutex
	var conns []net.Conn
	t.Cleanup(func() {
		_ = listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range conns {
			_ = conn.Close()
		}
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
			go handle(conn)
		}
	}()
	return host, port
}

func serveSSHConn(conn net.Conn, config *ssh.ServerConfig) {
	_, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(requests)
	for newChannel := range channels {
		if newChannel.ChannelType() != "direct-tcpip" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only direct-tcpip is supported")
			continue
		}
		go forwardSSHChannel(newChannel)
	}
}

// forwardSSHChannel serves one direct-tcpip channel as sshd does: it dials the
// requested address and closes both sides once either one ends.
func forwardSSHChannel(newChannel ssh.NewChannel) {
	// RFC 4254 section 7.2.
	var request struct {
		DestAddr   string
		DestPort   uint32
		OriginAddr string
		OriginPort uint32
	}
	if err := ssh.Unmarshal(newChannel.ExtraData(), &request); err != nil {
		_ = newChannel.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	target, err := net.Dial("tcp", net.JoinHostPort(request.DestAddr, strconv.FormatUint(uint64(request.DestPort), 10)))
	if err != nil {
		_ = newChannel.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	defer target.Close()

	channel, requests, err := newChannel.Accept()
	if err != nil {
		return
	}
	defer channel.Close()
	go ssh.DiscardRequests(requests)

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(channel, target)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(target, channel)
		done <- struct{}{}
	}()
	<-done
}
