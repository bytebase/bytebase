package v1

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNeonDiscoveryPageToken(t *testing.T) {
	token, err := encodeNeonPageToken("deployment-secret", "provider-cursor", "workspace", "projects/app", "neon-project")
	require.NoError(t, err)
	cursor, err := decodeNeonPageToken("deployment-secret", token, "workspace", "projects/app", "neon-project")
	require.NoError(t, err)
	require.Equal(t, "provider-cursor", cursor)
	data, err := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, err)
	require.NotContains(t, string(data), "provider-cursor")
	data[len(data)-1] ^= 1
	for _, tc := range []struct {
		name, secret, token, workspace, parent, project string
	}{
		{"different secret", "other-secret", token, "workspace", "projects/app", "neon-project"},
		{"different workspace", "deployment-secret", token, "other-workspace", "projects/app", "neon-project"},
		{"different parent", "deployment-secret", token, "workspace", "projects/other", "neon-project"},
		{"different project", "deployment-secret", token, "workspace", "projects/app", "neon-other"},
		{"tampered", "deployment-secret", base64.RawURLEncoding.EncodeToString(data), "workspace", "projects/app", "neon-project"},
		{"malformed", "deployment-secret", "!", "workspace", "projects/app", "neon-project"},
		{"oversized", "deployment-secret", strings.Repeat("a", 65537), "workspace", "projects/app", "neon-project"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeNeonPageToken(tc.secret, tc.token, tc.workspace, tc.parent, tc.project)
			require.Error(t, err)
		})
	}
}

func TestNeonDiscoveryPageTokenEmpty(t *testing.T) {
	token, err := encodeNeonPageToken("", "", "workspace", "", "")
	require.NoError(t, err)
	require.Empty(t, token)
	cursor, err := decodeNeonPageToken("", "", "workspace", "", "")
	require.NoError(t, err)
	require.Empty(t, cursor)
	_, err = encodeNeonPageToken("", "cursor", "workspace", "", "")
	require.Error(t, err)
}
