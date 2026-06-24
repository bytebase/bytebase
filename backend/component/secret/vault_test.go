package secret

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestResolveVaultToken(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		got, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: "plain-token"},
		})
		require.NoError(t, err)
		require.Equal(t, "plain-token", got)
	})

	t.Run("unspecified defaults to plain", func(t *testing.T) {
		got, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			TokenType:  storepb.DataSourceExternalSecret_TOKEN_TYPE_UNSPECIFIED,
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: "plain-token"},
		})
		require.NoError(t, err)
		require.Equal(t, "plain-token", got)
	})

	t.Run("environment", func(t *testing.T) {
		t.Setenv("BB_TEST_VAULT_TOKEN", "env-token")
		got, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			TokenType:  storepb.DataSourceExternalSecret_ENVIRONMENT,
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: "BB_TEST_VAULT_TOKEN"},
		})
		require.NoError(t, err)
		require.Equal(t, "env-token", got)
	})

	t.Run("environment unset", func(t *testing.T) {
		_, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			TokenType:  storepb.DataSourceExternalSecret_ENVIRONMENT,
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: "BB_TEST_VAULT_TOKEN_UNSET"},
		})
		require.Error(t, err)
	})

	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "token")
		require.NoError(t, os.WriteFile(path, []byte("  file-token\n"), 0600))
		got, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			TokenType:  storepb.DataSourceExternalSecret_FILE,
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: path},
		})
		require.NoError(t, err)
		require.Equal(t, "file-token", got)
	})

	t.Run("file missing", func(t *testing.T) {
		_, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			TokenType:  storepb.DataSourceExternalSecret_FILE,
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: filepath.Join(t.TempDir(), "nope")},
		})
		require.Error(t, err)
	})

	t.Run("file empty", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty")
		require.NoError(t, os.WriteFile(path, []byte("   \n"), 0600))
		_, err := resolveVaultToken(&storepb.DataSourceExternalSecret{
			TokenType:  storepb.DataSourceExternalSecret_FILE,
			AuthOption: &storepb.DataSourceExternalSecret_Token{Token: path},
		})
		require.Error(t, err)
	})
}
