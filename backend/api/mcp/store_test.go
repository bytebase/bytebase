package mcp

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type testServerStore struct {
	workspaceID      string
	workspaceProfile *storepb.WorkspaceProfileSetting
	capability       storepb.MCPSetting_Capability
	capabilityErr    error
}

func newTestServerStore() *testServerStore {
	return &testServerStore{
		workspaceID:      "ws-test",
		workspaceProfile: &storepb.WorkspaceProfileSetting{},
		capability:       storepb.MCPSetting_READ_WRITE,
	}
}

func (s *testServerStore) GetWorkspaceID(context.Context) (string, error) {
	return s.workspaceID, nil
}

func (s *testServerStore) GetWorkspaceProfileSetting(context.Context, string) (*storepb.WorkspaceProfileSetting, error) {
	return s.workspaceProfile, nil
}

func (s *testServerStore) GetMCPSettingsUncached(context.Context, string) (store.MCPSettings, error) {
	return store.MCPSettings{Capability: s.capability}, s.capabilityErr
}

func (*testServerStore) DeleteOAuth2RefreshTokensByUserAndClient(context.Context, string, string) error {
	return nil
}
