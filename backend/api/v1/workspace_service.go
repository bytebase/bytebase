package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/mailer"
	stripeplugin "github.com/bytebase/bytebase/backend/plugin/stripe"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// WorkspaceService implements the workspace service.
type WorkspaceService struct {
	v1connect.UnimplementedWorkspaceServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
	iamManager     *iam.Manager
	profile        *config.Profile
	authService    *AuthService
}

// NewWorkspaceService creates a new WorkspaceService.
func NewWorkspaceService(
	store *store.Store,
	iamManager *iam.Manager,
	profile *config.Profile,
	licenseService *enterprise.LicenseService,
	authService *AuthService,
) *WorkspaceService {
	return &WorkspaceService{
		store:          store,
		iamManager:     iamManager,
		profile:        profile,
		licenseService: licenseService,
		authService:    authService,
	}
}

// GetWorkspace gets a workspace by name.
// Supports "workspaces/-" to resolve the current/default workspace.
func (s *WorkspaceService) GetWorkspace(ctx context.Context, req *connect.Request[v1pb.GetWorkspaceRequest]) (*connect.Response[v1pb.Workspace], error) {
	var workspaceID string

	name := req.Msg.Name
	if name == "workspaces/-" {
		// "workspaces/-" is allowed without auth (login page logo).
		// Resolve from context (authenticated) or fall back to default (self-hosted).
		workspaceID = common.GetWorkspaceIDFromContext(ctx)
		if workspaceID == "" && !s.profile.SaaS {
			ws, err := s.store.GetWorkspaceID(ctx)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			workspaceID = ws
		}
	} else {
		var err error
		workspaceID, err = common.GetWorkspaceID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid workspace name"))
		}

		// Require authentication for non-saas mode.
		if s.profile.SaaS {
			// Specific workspace requires authentication and membership.
			user, ok := GetUserFromContext(ctx)
			if !ok || user == nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
			}

			// Verify the user is a member of the requested workspace.
			ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
				WorkspaceID:    &workspaceID,
				Email:          user.Email,
				IncludeAllUser: !s.profile.SaaS,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to verify workspace membership"))
			}
			if ws == nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.Errorf("failed to verify workspace membership"))
			}
		}
	}

	result := &v1pb.Workspace{}
	if workspaceID != "" {
		ws, err := s.store.GetWorkspaceByID(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace"))
		}
		if ws == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("workspace %q not found", name))
		}
		result.Name = common.FormatWorkspace(ws.ResourceID)
		result.Title = ws.Payload.GetTitle()
		result.Logo = ws.Payload.GetLogo()
	}

	return connect.NewResponse(result), nil
}

func (s *WorkspaceService) ListWorkspaces(ctx context.Context, _ *connect.Request[v1pb.ListWorkspacesRequest]) (*connect.Response[v1pb.ListWorkspacesResponse], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	workspaces, err := s.store.ListWorkspacesByEmail(ctx, &store.FindWorkspaceMessage{
		Email:          user.Email,
		IncludeAllUser: !s.profile.SaaS,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find workspaces"))
	}

	var result []*v1pb.Workspace
	for _, ws := range workspaces {
		result = append(result, &v1pb.Workspace{
			Name:  common.FormatWorkspace(ws.ResourceID),
			Title: ws.Payload.GetTitle(),
			Logo:  ws.Payload.GetLogo(),
		})
	}
	return connect.NewResponse(&v1pb.ListWorkspacesResponse{Workspaces: result}), nil
}

func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, req *connect.Request[v1pb.UpdateWorkspaceRequest]) (*connect.Response[v1pb.Workspace], error) {
	ws := req.Msg.Workspace
	if ws == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace is required"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}
	if len(req.Msg.UpdateMask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask is required"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	patch := &store.UpdateWorkspaceMessage{
		ResourceID: workspaceID,
	}
	for _, path := range req.Msg.UpdateMask.GetPaths() {
		switch path {
		case "title":
			if ws.Title == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("title cannot be empty"))
			}
			patch.Title = &ws.Title
		case "logo":
			if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, v1pb.PlanFeature_FEATURE_CUSTOM_LOGO); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			patch.Logo = &ws.Logo
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported field: %q", path))
		}
	}

	if err := s.store.UpdateWorkspace(ctx, patch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update workspace"))
	}

	// Read back the updated workspace to return the full state.
	updated, err := s.store.GetWorkspaceByID(ctx, patch.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get updated workspace"))
	}
	return connect.NewResponse(&v1pb.Workspace{
		Name:  ws.Name,
		Title: updated.Payload.GetTitle(),
		Logo:  updated.Payload.GetLogo(),
	}), nil
}

func (s *WorkspaceService) GetIamPolicy(ctx context.Context, _ *connect.Request[v1pb.GetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	policy, err := s.store.GetWorkspaceIamPolicy(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find iam policy"))
	}

	v1Policy, err := convertToV1IamPolicy(policy)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1Policy), nil
}

// DeleteWorkspace soft-deletes a workspace. SaaS only.
// Cancels any active Stripe subscription before deleting, then switches
// to the next available workspace and returns new auth tokens.
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, req *connect.Request[v1pb.DeleteWorkspaceRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	if !s.profile.SaaS {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("workspace deletion is only supported in SaaS mode"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	workspaceID, err := common.GetWorkspaceID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if contextWorkspaceID := common.GetWorkspaceIDFromContext(ctx); workspaceID != contextWorkspaceID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot delete workspace %q from workspace %q", workspaceID, contextWorkspaceID))
	}

	// Cancel active Stripe subscription if any.
	sub, err := s.store.GetSubscriptionByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check subscription"))
	}
	if sub != nil && sub.Payload != nil && sub.Payload.StripeSubscriptionId != "" {
		if _, err := stripeplugin.CancelSubscription(sub.Payload.StripeSubscriptionId, workspaceID, true); err != nil {
			slog.Warn("failed to cancel Stripe subscription during workspace deletion",
				slog.String("workspace", workspaceID),
				log.BBError(err),
			)
		}
	}

	if err := s.store.DeleteWorkspace(ctx, workspaceID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to delete workspace"))
	}

	// Find the next workspace and switch to it.
	nextWS, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		Email: user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find next workspace"))
	}
	if nextWS == nil {
		// No remaining workspace — clear auth cookies and return empty response.
		// Frontend redirects to login which will provision a new workspace.
		resp := connect.NewResponse(&v1pb.LoginResponse{})
		s.authService.clearSessionAndSetCookies(ctx, req.Header(), resp.Header(), workspaceID)
		return resp, nil
	}

	isWeb := auth.GetRefreshTokenFromCookie(req.Header()) != ""
	return s.authService.switchWorkspaceInternal(ctx, user, nextWS.ResourceID, isWeb, req.Header())
}

// LeaveWorkspace removes the calling user from a workspace's IAM bindings,
// then switches to the next available workspace.
func (s *WorkspaceService) LeaveWorkspace(ctx context.Context, req *connect.Request[v1pb.LeaveWorkspaceRequest]) (*connect.Response[v1pb.LoginResponse], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	workspaceID, err := common.GetWorkspaceID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Verify the user is a member of the target workspace.
	ws, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		WorkspaceID: &workspaceID,
		Email:       user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to verify workspace membership"))
	}
	if ws == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("not a member of workspace %q", workspaceID))
	}

	memberIdentifier := common.FormatUserEmail(user.Email)

	// Simulate removing the user from all direct IAM bindings and check
	// that at least one active admin would remain. We exclude the caller
	// from the admin list to also account for admin-via-group bindings.
	policyMessage, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to get workspace IAM policy"))
	}
	simulatedPolicy := &storepb.IamPolicy{}
	for _, binding := range policyMessage.Policy.Bindings {
		var filtered []string
		for _, m := range binding.Members {
			if m != memberIdentifier {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) > 0 {
			simulatedPolicy.Bindings = append(simulatedPolicy.Bindings, &storepb.Binding{
				Role:      binding.Role,
				Members:   filtered,
				Condition: binding.Condition,
			})
		}
	}
	admins := utils.GetUsersByRoleInIAMPolicy(ctx, s.store, workspaceID, store.WorkspaceAdminRole, !s.profile.SaaS, simulatedPolicy)
	// Filter out the caller — they may still appear as admin via group bindings
	// that haven't been removed yet.
	var otherAdmins []*store.UserMessage
	for _, admin := range admins {
		if admin.Email != user.Email {
			otherAdmins = append(otherAdmins, admin)
		}
	}
	if !containsActiveEndUser(otherAdmins) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot leave workspace: you are the last admin"))
	}

	// Remove the user from all groups in this workspace.
	if err := s.store.RemoveMemberFromAllGroups(ctx, workspaceID, memberIdentifier); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to remove user from groups"))
	}

	// Remove the user from all direct IAM bindings.
	if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: workspaceID,
		Member:    memberIdentifier,
		Roles:     []string{},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update IAM policy"))
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	// Only switch workspace if the user is leaving their current workspace.
	currentWorkspaceID := common.GetWorkspaceIDFromContext(ctx)
	if workspaceID != currentWorkspaceID {
		// Leaving a different workspace — no token switch needed.
		return connect.NewResponse(&v1pb.LoginResponse{}), nil
	}

	// Find the next workspace to switch to.
	nextWS, err := s.store.FindWorkspace(ctx, &store.FindWorkspaceMessage{
		Email: user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to find next workspace"))
	}
	if nextWS == nil {
		resp := connect.NewResponse(&v1pb.LoginResponse{})
		s.authService.clearSessionAndSetCookies(ctx, req.Header(), resp.Header(), workspaceID)
		return resp, nil
	}

	isWeb := auth.GetRefreshTokenFromCookie(req.Header()) != ""
	return s.authService.switchWorkspaceInternal(ctx, user, nextWS.ResourceID, isWeb, req.Header())
}

func (s *WorkspaceService) SetIamPolicy(ctx context.Context, req *connect.Request[v1pb.SetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	request := req.Msg

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	policyMessage, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace iam policy"))
	}
	if request.Etag != "" && request.Etag != policyMessage.Etag {
		return nil, connect.NewError(connect.CodeAborted, errors.New("there is concurrent update to the workspace iam policy, please refresh and try again"))
	}

	if err := validateIAMPolicy(ctx, s.store, !s.profile.SaaS, request, policyMessage); err != nil {
		return nil, err
	}

	iamPolicy, err := convertToStoreIamPolicy(request.Policy)
	if err != nil {
		return nil, err
	}

	users := utils.GetUsersByRoleInIAMPolicy(
		ctx,
		s.store,
		common.GetWorkspaceIDFromContext(ctx),
		store.WorkspaceAdminRole,
		!s.profile.SaaS,
		iamPolicy,
	)
	if !containsActiveEndUser(users) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("workspace must have at least one admin"))
	}

	// Guard: count members in the new policy BEFORE saving.
	// Allow over-limit workspaces to reduce seats incrementally (e.g. after license downgrade).
	userLimit := s.licenseService.GetUserLimit(ctx, workspaceID)
	newCount, err := countUsersInIamPolicy(ctx, s.store, workspaceID, iamPolicy, s.profile.SaaS)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to count users in IAM policy"))
	}
	if newCount > userLimit {
		oldCount, err := countUsersInIamPolicy(ctx, s.store, workspaceID, policyMessage.Policy, s.profile.SaaS)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to count users in current IAM policy"))
		}
		if newCount >= oldCount {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf("workspace has %d users, exceeding the limit of %d", newCount, userLimit))
		}
	}

	payloadBytes, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal iam policy"))
	}
	patch := &store.UpdatePolicyMessage{
		ResourceType: storepb.Policy_WORKSPACE,
		Resource:     request.Resource,
		Type:         storepb.Policy_IAM,
		Workspace:    workspaceID,
		Payload:      new(string(payloadBytes)),
	}

	if _, err := s.store.UpdatePolicy(ctx, patch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}

	policy, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find iam policy"))
	}

	deltas := findIamPolicyDeltas(policyMessage.Policy, policy.Policy)

	if setServiceData, ok := common.GetSetServiceDataFromContext(ctx); ok {
		p, err := convertToProtoAny(deltas)
		if err != nil {
			slog.Warn("audit: failed to convert to anypb.Any")
		}
		setServiceData(p)
	}

	// send invite emails to newly added members.
	go s.sendInviteEmails(context.WithoutCancel(ctx), workspaceID, policyMessage.Policy, deltas)

	v1Policy, err := convertToV1IamPolicy(policy)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1Policy), nil
}

// sendInviteEmails sends invite emails to newly added members. Errors are logged, never returned.
func (s *WorkspaceService) sendInviteEmails(ctx context.Context, workspaceID string, oldPolicy *storepb.IamPolicy, deltas []*v1pb.BindingDelta) {
	emailSetting, err := resolvePreLoginEmailSetting(ctx, s.store, workspaceID)
	if err != nil || emailSetting == nil {
		return
	}

	sender, err := mailer.NewSender(emailSetting)
	if err != nil {
		slog.Warn("failed to create mail sender for invite emails", log.BBError(err))
		return
	}

	// Get workspace URL and title.
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, workspaceID)
	if err != nil {
		slog.Warn("failed to get external URL for invite emails", log.BBError(err))
		return
	}
	if externalURL == "" {
		slog.Warn("empty external URL, skip send invite emails")
		return
	}
	workspaceTitle := workspaceID
	if ws, err := s.store.GetWorkspaceByID(ctx, workspaceID); err == nil && ws != nil && ws.Payload.GetTitle() != "" {
		workspaceTitle = ws.Payload.GetTitle()
	}

	// Build set of members who already existed in the old policy (any role).
	existingMembers := make(map[string]bool)
	for _, binding := range oldPolicy.GetBindings() {
		for _, member := range binding.Members {
			if email, ok := strings.CutPrefix(member, common.UserNamePrefix); ok {
				existingMembers[email] = true
			} else if groupName, ok := strings.CutPrefix(member, common.GroupPrefix); ok {
				// Expand group to individual members.
				group, err := s.store.GetGroupByName(ctx, workspaceID, common.GroupPrefix+groupName)
				if err != nil || group == nil || group.Payload == nil {
					continue
				}
				for _, m := range group.Payload.Members {
					if email, ok := strings.CutPrefix(m.Member, common.UserNamePrefix); ok {
						existingMembers[email] = true
					}
				}
			}
		}
	}

	// Collect truly new user emails and their roles. Map deduplicates by email.
	invites := make(map[string]string) // email -> roleName
	for _, delta := range deltas {
		if delta.Action != v1pb.BindingDelta_ADD {
			continue
		}
		roleName := delta.Role
		if role, err := s.store.GetRole(ctx, &store.FindRoleMessage{Workspace: workspaceID, ResourceID: &delta.Role}); err == nil && role != nil {
			roleName = role.Name
		}

		// Direct user member.
		if email, ok := strings.CutPrefix(delta.Member, common.UserNamePrefix); ok {
			if !existingMembers[email] {
				invites[email] = roleName
			}
			continue
		}
		// Group member — expand to individual users.
		if groupName, ok := strings.CutPrefix(delta.Member, common.GroupPrefix); ok {
			group, err := s.store.GetGroupByName(ctx, workspaceID, common.GroupPrefix+groupName)
			if err != nil || group == nil || group.Payload == nil {
				continue
			}
			for _, m := range group.Payload.Members {
				if email, ok := strings.CutPrefix(m.Member, common.UserNamePrefix); ok {
					if !existingMembers[email] {
						invites[email] = roleName
					}
				}
			}
		}
	}
	if len(invites) == 0 {
		return
	}

	for email, roleName := range invites {
		loginLink := fmt.Sprintf("%s?workspace=%s&email=%s", externalURL, url.QueryEscape(workspaceID), url.QueryEscape(email))
		subject := fmt.Sprintf("[Bytebase] You've been invited to workspace %q", workspaceTitle)
		body := fmt.Sprintf("Hi,\n\nYou've been added to the Bytebase workspace %q as %s.\n\nSign in to get started:\n%s\n\n— Bytebase", workspaceTitle, roleName, loginLink)
		if err := sender.Send(ctx, &mailer.SendRequest{
			To:       []string{email},
			Subject:  subject,
			TextBody: body,
		}); err != nil {
			slog.Warn("failed to send invite email", slog.String("to", email), log.BBError(err))
		}
	}
}

func containsActiveEndUser(users []*store.UserMessage) bool {
	for _, user := range users {
		if user.Type == storepb.PrincipalType_END_USER && !user.MemberDeleted {
			return true
		}
	}
	return false
}
