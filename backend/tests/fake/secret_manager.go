package fake

import (
	"fmt"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SecretManager is a fake implementation of secret manager.
type SecretManager struct {
	port   int
	echo   *echo.Echo
	client *http.Client
}

// NewSecretManager creates a new fake implementation of secret manager.
func NewSecretManager(port int) *SecretManager {
	e := newEchoServer()
	sm := &SecretManager{
		port:   port,
		echo:   e,
		client: &http.Client{},
	}
	e.GET("/secrets/hello-secret-id:access", func(c echo.Context) error {
		// Base64 format of "bytebase".
		data := `{
			"payload": {
				"data": "Ynl0ZWJhc2U="
			}
		}`
		return c.String(http.StatusOK, data)
	})
	return sm
}

// Run starts the secret manager server.
func (sm *SecretManager) Run() error {
	return sm.echo.Start(fmt.Sprintf(":%d", sm.port))
}

// Close shuts down the secret manager server.
func (sm *SecretManager) Close() error {
	return sm.echo.Close()
}

// ListenerAddr returns the secret manager server listener address.
func (sm *SecretManager) ListenerAddr() net.Addr {
	return sm.echo.ListenerAddr()
}
