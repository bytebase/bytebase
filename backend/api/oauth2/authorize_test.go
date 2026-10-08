package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConsentWorkspaceRequiresSessionClaim(t *testing.T) {
	s := &Service{profile: &config.Profile{}}

	workspaceID, failure := s.consentWorkspace(&sessionClaims{}, &store.OAuth2ClientMessage{})

	require.Empty(t, workspaceID)
	require.Equal(t, &oauth2Failure{code: "access_denied", description: "session is missing workspace claim"}, failure)
}
