package command

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
)

func TestNewClientWithOptionsUsesServiceAccountAuth(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{}
	client, err := newClient("https://example.com", "", "sa@example.com", "secret", clientOptions{
		httpClient: httpClient,
		pageSize:   -1,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, client.close())
	})

	require.Same(t, httpClient, client.httpClient)
	require.Equal(t, int32(100), client.options.pageSize)
	require.Equal(t, "sa@example.com", client.serviceAccount)
	require.Equal(t, "secret", client.serviceAccountSecret)
}

func TestNewClientWithAccessTokenUsesAccessTokenAuth(t *testing.T) {
	t.Parallel()

	client, err := newClient("https://example.com", "token", "", "", defaultClientOptions())
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, client.close())
	})

	require.Equal(t, "", client.serviceAccount)
	require.Equal(t, "", client.serviceAccountSecret)
	require.Equal(t, 120*time.Second, client.httpClient.Timeout)
}

func TestNewClientFromWorldPrefersAccessToken(t *testing.T) {
	t.Parallel()

	w := &world.World{
		URL:                  "https://example.com",
		Timeout:              5 * time.Second,
		AccessToken:          "token",
		ServiceAccount:       "sa@example.com",
		ServiceAccountSecret: "secret",
	}

	client, err := newClientFromWorld(w)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, client.close())
	})

	require.Equal(t, 5*time.Second, client.httpClient.Timeout)
	require.Equal(t, "", client.serviceAccount)
	require.Equal(t, "", client.serviceAccountSecret)
}
