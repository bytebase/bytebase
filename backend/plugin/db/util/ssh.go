package util

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// GetSSHClient returns a ssh client.
func GetSSHClient(cfg db.SSHConfig) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if cfg.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		if err != nil {
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	} else {
		// Establish a connection to the local ssh-agent
		conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		// Create a new instance of the ssh agent
		agentClient := agent.NewClient(conn)
		// When the agentClient connection succeeded, add them as AuthMethod
		if agentClient != nil {
			sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}
	// When there's a non empty password add the password AuthMethod.
	if cfg.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.PasswordCallback(func() (string, error) {
			return cfg.Password, nil
		}))
	}
	// Connect to the SSH Server
	sshConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port), sshConfig)
	if err != nil {
		return nil, err
	}
	return sshConn, nil
}
