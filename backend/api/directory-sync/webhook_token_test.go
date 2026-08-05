package directorysync

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// The rotate handler stores common.HashDirectorySyncToken(token); this path must
// accept the plaintext behind it. Computing the expected hash independently here
// (rather than calling the shared helper) means a change to the digest or the
// encoding shows up as a failure instead of both sides moving together.
func TestCheckDirectorySyncTokenAcceptsWhatRotateStores(t *testing.T) {
	const token = "3f4e5d6c-4444-4000-8000-112233445566"

	sum := sha256.Sum256([]byte(token))
	stored := common.HashDirectorySyncToken(token)
	require.Equal(t, hex.EncodeToString(sum[:]), stored,
		"stored hash must stay hex-encoded SHA-256; the SQL migration writes the same encoding")

	require.NoError(t, checkDirectorySyncToken(
		&storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: stored}, token))

	// A rotation replaces the hash, so the previous token must stop working.
	rotated := common.HashDirectorySyncToken("a-different-token")
	require.Error(t, checkDirectorySyncToken(
		&storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: rotated}, token),
		"the pre-rotation token must be rejected after rotation")
}

func TestCheckDirectorySyncToken(t *testing.T) {
	const token = "7a1b9f30-1111-4000-8000-abcdefabcdef"
	hashOf := func(s string) string {
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:])
	}

	for _, tt := range []struct {
		name      string
		setting   *storepb.WorkspaceProfileSetting
		presented string
		wantErr   bool
	}{
		{
			// The migration hashes the existing token in place, so a token an IdP
			// was already configured with keeps authenticating.
			name:      "accepts the plaintext behind the stored hash",
			setting:   &storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: hashOf(token)},
			presented: token,
		},
		{
			name:      "rejects a different token",
			setting:   &storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: hashOf(token)},
			presented: "not-the-token",
			wantErr:   true,
		},
		{
			// Presenting the stored hash itself must not authenticate, otherwise
			// reading the database would be as good as holding the token.
			name:      "rejects the hash itself",
			setting:   &storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: hashOf(token)},
			presented: hashOf(token),
			wantErr:   true,
		},
		{
			name:      "rejects when no token is configured",
			setting:   &storepb.WorkspaceProfileSetting{},
			presented: token,
			wantErr:   true,
		},
		{
			name:      "rejects an empty presentation",
			setting:   &storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: hashOf(token)},
			presented: "",
			wantErr:   true,
		},
		{
			// An empty hash must not be treated as "matches the empty token".
			name:      "rejects an empty presentation against an empty hash",
			setting:   &storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: ""},
			presented: "",
			wantErr:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDirectorySyncToken(tt.setting, tt.presented)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
