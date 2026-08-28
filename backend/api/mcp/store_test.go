package mcp

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type testServerStore struct {
	workspaceID      string
	workspaceProfile *storepb.WorkspaceProfileSetting
	capability       storepb.MCPSetting_Capability
	capabilityErr    error

	// auditRows collects what the connection gate recorded, so a test can
	// assert on the row a denial wrote without a database.
	auditRows []*storepb.AuditLog
	auditErr  error
	// writeCtxErr is the write context's error at the moment of the write,
	// which is what shows the row survives a client that hung up.
	writeCtxErr error
	// writeCtxHasDeadline shows the detached write is still bounded: dropping
	// the request's cancellation also drops its deadline.
	writeCtxHasDeadline bool
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

func (s *testServerStore) GetMCPSettingsUncached(context.Context, string) (*storepb.MCPSetting, error) {
	if s.capabilityErr != nil {
		return nil, s.capabilityErr
	}
	return &storepb.MCPSetting{Capability: s.capability}, nil
}

func (*testServerStore) DeleteOAuth2RefreshTokensByUserAndClient(context.Context, string, string) error {
	return nil
}

func (s *testServerStore) CreateAuditLog(ctx context.Context, _ string, payload *storepb.AuditLog) error {
	s.writeCtxErr = ctx.Err()
	_, s.writeCtxHasDeadline = ctx.Deadline()
	if s.auditErr != nil {
		return s.auditErr
	}
	s.auditRows = append(s.auditRows, payload)
	return nil
}
