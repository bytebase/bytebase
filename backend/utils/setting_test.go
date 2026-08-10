package utils

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type testDiscoveryExternalURLStore struct {
	workspaceID      string
	workspaceIDErr   error
	workspaceProfile *storepb.WorkspaceProfileSetting
}

func (s *testDiscoveryExternalURLStore) GetWorkspaceID(context.Context) (string, error) {
	return s.workspaceID, s.workspaceIDErr
}

func (s *testDiscoveryExternalURLStore) GetWorkspaceProfileSetting(context.Context, string) (*storepb.WorkspaceProfileSetting, error) {
	return s.workspaceProfile, nil
}

func TestGetDiscoveryExternalURLRequiresConfiguredURL(t *testing.T) {
	externalURL, err := GetDiscoveryExternalURL(
		context.Background(),
		&testDiscoveryExternalURLStore{},
		&config.Profile{SaaS: true},
	)

	require.Empty(t, externalURL)
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestGetDiscoveryExternalURLPropagatesWorkspaceLookupError(t *testing.T) {
	externalURL, err := GetDiscoveryExternalURL(
		context.Background(),
		&testDiscoveryExternalURLStore{workspaceIDErr: errors.New("database unavailable")},
		&config.Profile{},
	)

	require.Empty(t, externalURL)
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetDiscoveryExternalURLUsesWorkspaceSetting(t *testing.T) {
	externalURL, err := GetDiscoveryExternalURL(
		context.Background(),
		&testDiscoveryExternalURLStore{
			workspaceID:      "ws-test",
			workspaceProfile: &storepb.WorkspaceProfileSetting{ExternalUrl: "https://workspace.example.com"},
		},
		&config.Profile{},
	)

	require.NoError(t, err)
	require.Equal(t, "https://workspace.example.com", externalURL)
}
