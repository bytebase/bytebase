package mcp

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type testServerStore struct {
	workspaceID      string
	workspaceProfile *storepb.WorkspaceProfileSetting
	setting          *store.SettingMessage
}

func newTestServerStore() *testServerStore {
	return &testServerStore{
		workspaceID:      "ws-test",
		workspaceProfile: &storepb.WorkspaceProfileSetting{},
	}
}

func (s *testServerStore) GetWorkspaceID(context.Context) (string, error) {
	return s.workspaceID, nil
}

func (s *testServerStore) GetWorkspaceProfileSetting(context.Context, string) (*storepb.WorkspaceProfileSetting, error) {
	return s.workspaceProfile, nil
}

func (s *testServerStore) GetSettingUncached(context.Context, string, storepb.SettingName) (*store.SettingMessage, error) {
	return s.setting, nil
}

func (*testServerStore) DeleteOAuth2RefreshTokensByUserAndClient(context.Context, string, string) error {
	return nil
}
