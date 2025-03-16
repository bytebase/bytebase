package util

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
			// Create a new instance of the ssh agent
			agentClient := agent.NewClient(conn)
			// When the agentClient connection succeeded, add them as AuthMethod
			if agentClient != nil {
				sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(agentClient.Signers))
			}
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

type NoDeadlineConn struct{ net.Conn }

func (*NoDeadlineConn) SetDeadline(time.Time) error      { return nil }
func (*NoDeadlineConn) SetReadDeadline(time.Time) error  { return nil }
func (*NoDeadlineConn) SetWriteDeadline(time.Time) error { return nil }
