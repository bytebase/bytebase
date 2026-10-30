package sharepoint

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestLiveSharePoint uploads a small export into a new subfolder of a real
// destination. The application needs write access to that destination.
func TestLiveSharePoint(t *testing.T) {
	cfg := Config{
		TenantID:     os.Getenv("SHAREPOINT_TEST_TENANT_ID"),
		ClientID:     os.Getenv("SHAREPOINT_TEST_CLIENT_ID"),
		ClientSecret: os.Getenv("SHAREPOINT_TEST_CLIENT_SECRET"),
	}
	rawURL := os.Getenv("SHAREPOINT_TEST_URL")
	if cfg.TenantID == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || rawURL == "" {
		t.Skip("set SHAREPOINT_TEST_TENANT_ID, SHAREPOINT_TEST_CLIENT_ID, SHAREPOINT_TEST_CLIENT_SECRET, and SHAREPOINT_TEST_URL to run")
	}
	ctx := t.Context()
	canonical, err := NormalizeURL(rawURL)
	require.NoError(t, err)
	client, err := NewClient(cfg)
	require.NoError(t, err)
	loc, err := client.Resolve(ctx, canonical)
	require.NoError(t, err)
	t.Logf("resolved %s", loc.WebURL)

	folderName := "bytebase-live-test " + time.Now().UTC().Format("20060102T150405Z")
	files := []File{
		{Name: "statement.sql", Content: []byte("SELECT 1;\n")},
		{Name: "result.csv", Content: bytes.Repeat([]byte("a,b\n"), uploadChunkSize/4+1)},
	}
	folder, err := client.Upload(ctx, loc, folderName, files)
	require.NoError(t, err)
	t.Logf("uploaded to %s", folder.WebURL)

	again, err := client.Upload(ctx, loc, folderName, files)
	require.NoError(t, err)
	require.Equal(t, folder, again)
}
