package v1

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// GetMCPInfo reports the MCP ceiling in force in this workspace and whether
// masking narrows what a session reads under it.
//
// It is the only API a served MCP session or the consent page can read the
// ceiling from: SettingService/GetSetting is served by no ceiling, and answers
// from the generic cache besides.
func (s *WorkspaceService) GetMCPInfo(ctx context.Context, _ *connect.Request[v1pb.GetMCPInfoRequest]) (*connect.Response[v1pb.MCPInfo], error) {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if workspaceID == "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("no workspace on the request"))
	}
	settings, err := s.mcpSettingsForInfo(ctx, workspaceID)
	// Every verdict but an outage is answered, the refusing ceilings included:
	// a workspace whose ceiling nothing serves still owes the repairing admin
	// the value in force, and the consent page a policy to disclose (BOT-106).
	// capability carries the refusal by arriving as a value this build does not
	// serve — CAPABILITY_UNSPECIFIED for a stored name nothing resolves, the
	// stored number itself for one a newer release wrote.
	switch verdict := auth.ClassifyMCPCeiling(settings, err); verdict {
	case auth.MCPCeilingServes, auth.MCPCeilingDisabled, auth.MCPCeilingUnserved:
	default:
		// The store error stays in the log. This method is served to MCP
		// sessions and the tool layer renders a connect message into what the
		// model reads, so a driver error text would leave the metadata
		// database's shape in an agent's context.
		//
		// An outage has no ceiling to describe, and answering with an empty one
		// would be indistinguishable from a row nobody can resolve.
		slog.Error("failed to read the MCP setting", slog.String("workspace", workspaceID), log.BBError(err))
		return nil, connect.NewError(connect.CodeUnavailable, errors.New(verdict.Refusal()))
	}

	return connect.NewResponse(&v1pb.MCPInfo{
		Workspace:               common.FormatWorkspace(workspaceID),
		Capability:              convertToV1MCPCapability(settings.Capability),
		IgnoreMaskingExemptions: settings.IgnoreMaskingExemptions,
		DataMaskingAvailable:    s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_DATA_MASKING) == nil,
	}), nil
}

// mcpSettingsForInfo returns the MCP settings this request is answered from.
//
// On the internal MCP chain the gate already resolved them and stamped them on
// the context; reading again could answer with a ceiling the request was not
// admitted under, which is the rule mcp_gate.go states and the clamp and the
// masking check both follow. Off that chain — the consent page, which reads
// this before it discloses anything — nothing is stamped and the store is read.
func (s *WorkspaceService) mcpSettingsForInfo(ctx context.Context, workspaceID string) (*storepb.MCPSetting, error) {
	if settings, ok := mcpSettingsFromContext(ctx); ok {
		return settings, nil
	}
	return s.store.GetMCPSettingsUncached(ctx, workspaceID)
}
