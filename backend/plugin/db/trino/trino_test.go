package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func openTrino(t *testing.T, password string, useSSL bool) error {
	t.Helper()
	driver, err := (&Driver{}).Open(context.Background(), storepb.Engine_TRINO, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "trino.example.com",
			Port:     "8080",
			Username: "bytebase",
			UseSsl:   useSSL,
		},
		Password: password,
	})
	if driver != nil {
		// Releases the custom HTTP client the SSL path registers globally.
		t.Cleanup(func() { _ = driver.Close(context.Background()) })
	}
	return err
}

// trino-go-client refuses a password on a plaintext connection, so such a data
// source cannot connect whatever we do. The guard exists for the message: it
// names the setting to change instead of reporting a DSN parse failure.
func TestOpenRefusesAPasswordWithoutSSL(t *testing.T) {
	err := openTrino(t, "secret", false)
	require.ErrorContains(t, err, "requires SSL to authenticate with a password")
	require.NotContains(t, err.Error(), "secret", "the error must not carry the password")
}

func TestOpenAcceptsTheConfigurationsThatCanConnect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		useSSL   bool
	}{
		{"password over SSL", "secret", true},
		{"no password without SSL", "", false},
		{"no password over SSL", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := openTrino(t, tc.password, tc.useSSL)
			require.NoError(t, err)
		})
	}
}
