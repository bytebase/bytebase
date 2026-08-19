package mcp

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type testServerStore struct {
	workspaceID      string
	workspaceProfile *storepb.WorkspaceProfileSetting
	capability       storepb.WorkspaceProfileSetting_MCPCapability
	capabilityErr    error
}

func newTestServerStore() *testServerStore {
	return &testServerStore{
		workspaceID:      "ws-test",
		workspaceProfile: &storepb.WorkspaceProfileSetting{},
		capability:       storepb.WorkspaceProfileSetting_READ_WRITE,
	}
}

func (s *testServerStore) GetWorkspaceID(context.Context) (string, error) {
	return s.workspaceID, nil
}

func (s *testServerStore) GetWorkspaceProfileSetting(context.Context, string) (*storepb.WorkspaceProfileSetting, error) {
	return s.workspaceProfile, nil
}

func (s *testServerStore) GetMCPCapabilityUncached(context.Context, string) (storepb.WorkspaceProfileSetting_MCPCapability, error) {
	return s.capability, s.capabilityErr
}

func (*testServerStore) DeleteOAuth2RefreshTokensByUserAndClient(context.Context, string, string) error {
	return nil
}
