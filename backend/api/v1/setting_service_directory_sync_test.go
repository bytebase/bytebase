package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// The WORKSPACE_PROFILE setting is readable by every workspace member
// (bb.settings.getWorkspaceProfile is in the default member role), so no secret
// may ever be converted into it. The directory sync token used to be, which is
// how it leaked to every member.
func TestWorkspaceProfileNeverExposesDirectorySyncToken(t *testing.T) {
	const tokenHash = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	got := convertToWorkspaceProfileSetting(&storepb.WorkspaceProfileSetting{
		DirectorySyncTokenHash: tokenHash,
	})

	// The hash must not appear anywhere in the serialized message, whatever
	// field a future change might place it in.
	raw, err := proto.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(raw), tokenHash, "token hash leaked into WorkspaceProfileSetting")
}

func TestWorkspaceProfileReportsTokenConfigured(t *testing.T) {
	for _, tt := range []struct {
		name  string
		store *storepb.WorkspaceProfileSetting
		want  bool
	}{
		{"hashed", &storepb.WorkspaceProfileSetting{DirectorySyncTokenHash: "abc123"}, true},
		{"absent", &storepb.WorkspaceProfileSetting{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, convertToWorkspaceProfileSetting(tt.store).GetDirectorySyncTokenConfigured())
		})
	}
}

// The ACL rejects a well-formed name for another workspace, but an empty or
// unparseable one falls back to the caller's own workspace and passes. Rotation
// invalidates a live SCIM credential, so a malformed request must be rejected
// rather than silently breaking the caller's integration — which means the
// handler has to parse the name itself.
func TestRotateDirectorySyncTokenRejectsMalformedNames(t *testing.T) {
	for _, tt := range []struct {
		name    string
		wantErr bool
	}{
		{"", true},
		{"garbage", true},
		{"workspaces/", true},
		{"workspaces/ws1/extra", true},
		{"projects/p1", true},
		{"workspaces/ws1", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := common.GetWorkspaceID(tt.name)
			if tt.wantErr {
				require.Error(t, err, "malformed name %q must not reach the rotation", tt.name)
				return
			}
			require.NoError(t, err)
		})
	}
}

// The rotate RPC is audited, and the audit log records the response. The
// plaintext token exists only in that one response, so it must be stripped
// before it is written — otherwise anyone with audit-log read access gets a
// working SCIM credential.
func TestRotateDirectorySyncTokenResponseIsRedactedForAudit(t *testing.T) {
	const token = "b17c0d9e-2222-4000-8000-fedcbafedcba"

	got, err := getResponseString(&v1pb.RotateDirectorySyncTokenResponse{Token: token})
	require.NoError(t, err)
	require.NotContains(t, got, token, "minted SCIM token written to the audit log")
}

// Guard against the next secret being dropped into this member-readable blob.
// If this fails, either redact the new field or move it out of WORKSPACE_PROFILE.
func TestWorkspaceProfileHasNoSecretShapedFields(t *testing.T) {
	// Only string/bytes fields can carry a credential. Booleans like
	// disallow_password_signin and durations like refresh_token_duration name a
	// policy, not a secret, so restricting by kind keeps this guard free of an
	// allowlist that would rot.
	secretish := []string{"token", "secret", "password", "key"}

	fields := (&v1pb.WorkspaceProfileSetting{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if field.Kind() != protoreflect.StringKind && field.Kind() != protoreflect.BytesKind {
			continue
		}
		name := string(field.Name())
		for _, needle := range secretish {
			require.NotContains(t, name, needle,
				"field %q looks like a secret but lives in WorkspaceProfileSetting, which every workspace member can read", name)
		}
	}
	// Keep the guard honest: it must still be looking at a populated message.
	require.NotNil(t, fields.ByName(protoreflect.Name("directory_sync_token_configured")))
}
