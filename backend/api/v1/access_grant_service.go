package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// AccessGrantService implements the access grant service.
type AccessGrantService struct {
	v1connect.UnimplementedAccessGrantServiceHandler
	store          *store.Store
	iamManager     *iam.Manager
	licenseService *enterprise.LicenseService
	webhookManager *webhook.Manager
	bus            *bus.Bus
}

// NewAccessGrantService returns a new access grant service instance.
func NewAccessGrantService(store *store.Store, iamManager *iam.Manager, licenseService *enterprise.LicenseService, webhookManager *webhook.Manager, bus *bus.Bus) *AccessGrantService {
	return &AccessGrantService{
		store:          store,
		iamManager:     iamManager,
		licenseService: licenseService,
		webhookManager: webhookManager,
		bus:            bus,
	}
}

// GetAccessGrant gets an access grant by name.
// Uses CUSTOM auth: allows access if user has bb.accessGrants.get permission,
// OR if the user is an approver on the linked issue.
func (s *AccessGrantService) GetAccessGrant(ctx context.Context, request *connect.Request[v1pb.GetAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	req := request.Msg
	projectID, accessGrantID, err := common.GetProjectIDAccessGrantID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)

	// Check IAM permission first (doesn't need the grant).
	hasPermission, err := s.iamManager.CheckPermission(ctx, permission.AccessGrantsGet, user, workspaceID, projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check permission"))
	}

	grant, err := s.store.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
		Workspace: workspaceID,
		ID:        &accessGrantID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant"))
	}
	if grant == nil {
		if hasPermission {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("access grant %q not found", req.Name))
		}
		// Don't reveal grant existence to unauthorized users.
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission to view access grant %q", req.Name))
	}

	if !hasPermission {
		if !s.isApproverForGrant(ctx, workspaceID, projectID, grant, user) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission to view access grant %q", req.Name))
		}
	}

	return connect.NewResponse(convertToAccessGrant(grant)), nil
}

// isApproverForGrant checks if the user is an approver (in any step)
// for the issue linked to the given access grant.
func (s *AccessGrantService) isApproverForGrant(ctx context.Context, workspaceID, projectID string, grant *store.AccessGrantMessage, user *store.UserMessage) bool {
	if grant.Payload == nil || grant.Payload.IssueId == 0 {
		return false
	}

	issueUID := grant.Payload.IssueId
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		Workspace:  workspaceID,
		ProjectIDs: []string{projectID},
		UID:        &issueUID,
	})
	if err != nil {
		slog.Error("failed to get issue for approver check", slog.Int64("issueUID", issueUID), log.BBError(err))
		return false
	}
	if issue == nil {
		return false
	}

	approval := issue.Payload.GetApproval()
	if approval == nil || approval.ApprovalTemplate == nil || approval.ApprovalTemplate.Flow == nil {
		return false
	}

	projectPolicy, err := s.store.GetProjectIamPolicy(ctx, workspaceID, projectID)
	if err != nil {
		slog.Error("failed to get project IAM policy for approver check", slog.String("project", projectID), log.BBError(err))
		return false
	}
	workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx, workspaceID)
	if err != nil {
		slog.Error("failed to get workspace IAM policy for approver check", log.BBError(err))
		return false
	}
	userRoles := utils.GetUserFormattedRolesMap(ctx, s.store, workspaceID, user, projectPolicy.Policy, workspacePolicy.Policy)

	for _, role := range approval.ApprovalTemplate.Flow.Roles {
		if userRoles[role] {
			return true
		}
	}

	return false
}

// ListAccessGrants lists access grants in a project.
func (s *AccessGrantService) ListAccessGrants(ctx context.Context, request *connect.Request[v1pb.ListAccessGrantsRequest]) (*connect.Response[v1pb.ListAccessGrantsResponse], error) {
	req := request.Msg
	projectID, err := common.GetProjectID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.PageToken,
		limit:   int(req.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindAccessGrantMessage{
		Workspace: common.GetWorkspaceIDFromContext(ctx),
		ProjectID: &projectID,
		Limit:     &limitPlusOne,
		Offset:    &offset.offset,
	}

	if req.Filter != "" {
		filterQ, err := store.GetListAccessGrantFilter(req.Filter)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse filter"))
		}
		find.FilterQ = filterQ
	}

	orderByKeys, err := store.GetAccessGrantOrders(req.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.OrderByKeys = orderByKeys

	grants, err := s.store.ListAccessGrants(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list access grants"))
	}

	resp := &v1pb.ListAccessGrantsResponse{}
	if len(grants) == limitPlusOne {
		nextPageToken, err := offset.getNextPageToken()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		resp.NextPageToken = nextPageToken
		grants = grants[:offset.limit]
	}
	for _, grant := range grants {
		resp.AccessGrants = append(resp.AccessGrants, convertToAccessGrant(grant))
	}

	return connect.NewResponse(resp), nil
}

// CreateAccessGrant creates an access grant.
func (s *AccessGrantService) CreateAccessGrant(ctx context.Context, request *connect.Request[v1pb.CreateAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	if err := s.licenseService.IsFeatureEnabled(ctx, common.GetWorkspaceIDFromContext(ctx), v1pb.PlanFeature_FEATURE_JIT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	req := request.Msg
	projectID, err := common.GetProjectID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	ag := req.AccessGrant
	if ag == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("access_grant is required"))
	}
	if ag.Creator == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("creator is required"))
	}
	if len(ag.Targets) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("targets is required"))
	}
	if ag.Query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query is required"))
	}
	if ag.Reason == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("reason is required"))
	}

	// Validate the query is a read-only statement (SELECT).
	instanceID, _, err := common.GetInstanceDatabaseID(ag.Targets[0])
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid target %q", ag.Targets[0]))
	}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &instanceID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance %q", instanceID))
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}
	if ok, _, err := parserbase.ValidateSQLForEditor(instance.Metadata.GetEngine(), ag.Query); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid query"))
	} else if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only read-only statements are allowed in access grants"))
	}

	creatorEmail, err := common.GetUserEmail(ag.Creator)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid creator"))
	}

	var expireTime *time.Time
	var requestedDuration *durationpb.Duration
	switch exp := ag.Expiration.(type) {
	case *v1pb.AccessGrant_ExpireTime:
		if exp.ExpireTime == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expire_time is required"))
		}
		expireTime = new(exp.ExpireTime.AsTime())
	case *v1pb.AccessGrant_Ttl:
		if exp.Ttl == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("ttl is required"))
		}
		// Store the requested duration; expire_time will be computed at activation time.
		requestedDuration = exp.Ttl
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expiration (expire_time or ttl) is required"))
	}

	// Step 1: Create the access grant.
	grant, err := s.store.CreateAccessGrant(ctx, &store.AccessGrantMessage{
		ProjectID:  projectID,
		Creator:    creatorEmail,
		Status:     storepb.AccessGrant_PENDING,
		ExpireTime: expireTime,
		Payload: &storepb.AccessGrantPayload{
			Targets:           ag.Targets,
			Query:             ag.Query,
			Unmask:            ag.Unmask,
			Reason:            ag.Reason,
			RequestedDuration: requestedDuration,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create access grant"))
	}

	// Step 2: Create the associated issue.
	issue, err := s.store.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    projectID,
		CreatorEmail: creatorEmail,
		Title:        fmt.Sprintf("JIT access request by %s", creatorEmail),
		Type:         storepb.Issue_ACCESS_GRANT,
		Description:  ag.Reason,
		Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			},
			AccessGrantId: grant.ID,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create issue"))
	}

	// Step 3: Update access grant payload with the issue ID.
	grant.Payload.IssueId = int64(issue.UID)
	grant, err = s.store.UpdateAccessGrant(ctx, grant.ID, &store.UpdateAccessGrantMessage{
		Payload: grant.Payload,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update access grant payload"))
	}

	// Step 4: Post-create: webhook, approval finding, auto-approve.
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{Workspace: workspaceID, ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project %v", projectID))
	}
	if project != nil {
		issue, err := postCreateIssue(ctx, s.store, s.webhookManager, s.licenseService, s.bus, project, creatorEmail, creatorEmail, issue)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if issue.Status == storepb.Issue_DONE {
			// Refresh the grant if issue is completed.
			grant, err = s.store.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
				Workspace: common.GetWorkspaceIDFromContext(ctx),
				ID:        &grant.ID,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant %v", grant.ID))
			}
		}
	}

	return connect.NewResponse(convertToAccessGrant(grant)), nil
}

// ActivateAccessGrant activates a pending access grant.
func (s *AccessGrantService) ActivateAccessGrant(ctx context.Context, request *connect.Request[v1pb.ActivateAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	if err := s.licenseService.IsFeatureEnabled(ctx, common.GetWorkspaceIDFromContext(ctx), v1pb.PlanFeature_FEATURE_JIT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	grant, err := activateAccessGrant(ctx, s.store, request.Msg.Name, false /* do not refresh expire time */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant"))
	}

	return connect.NewResponse(convertToAccessGrant(grant)), nil
}

func activateAccessGrant(ctx context.Context, stores *store.Store, accessGrantName string, refreshExpireTime bool) (*store.AccessGrantMessage, error) {
	projectID, accessGrantID, err := common.GetProjectIDAccessGrantID(accessGrantName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	grant, err := stores.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
		Workspace: common.GetWorkspaceIDFromContext(ctx),
		ID:        &accessGrantID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant"))
	}
	if grant == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("access grant %q not found", accessGrantName))
	}

	update := &store.UpdateAccessGrantMessage{
		Status: new(storepb.AccessGrant_ACTIVE),
	}

	// If the grant was created with a TTL, compute expire_time at activation time.
	if refreshExpireTime && grant.Payload != nil && grant.Payload.RequestedDuration != nil {
		update.ExpireTime = new(time.Now().Add(grant.Payload.RequestedDuration.AsDuration()))
	}

	updated, err := stores.UpdateAccessGrant(ctx, accessGrantID, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to activate access grant"))
	}

	return updated, nil
}

// RevokeAccessGrant revokes an active access grant.
func (s *AccessGrantService) RevokeAccessGrant(ctx context.Context, request *connect.Request[v1pb.RevokeAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	if err := s.licenseService.IsFeatureEnabled(ctx, common.GetWorkspaceIDFromContext(ctx), v1pb.PlanFeature_FEATURE_JIT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	req := request.Msg
	projectID, accessGrantID, err := common.GetProjectIDAccessGrantID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	grant, err := s.store.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
		Workspace: common.GetWorkspaceIDFromContext(ctx),
		ID:        &accessGrantID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant"))
	}
	if grant == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("access grant %q not found", req.Name))
	}
	if grant.Status != storepb.AccessGrant_ACTIVE {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("access grant %q is not in ACTIVE status", req.Name))
	}

	updated, err := s.store.UpdateAccessGrant(ctx, accessGrantID, &store.UpdateAccessGrantMessage{
		Status: new(storepb.AccessGrant_REVOKED),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to revoke access grant"))
	}

	return connect.NewResponse(convertToAccessGrant(updated)), nil
}

// SearchMyAccessGrants searches access grants created by the caller.
func (s *AccessGrantService) SearchMyAccessGrants(ctx context.Context, request *connect.Request[v1pb.SearchMyAccessGrantsRequest]) (*connect.Response[v1pb.SearchMyAccessGrantsResponse], error) {
	req := request.Msg
	projectID, err := common.GetProjectID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found"))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.PageToken,
		limit:   int(req.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindAccessGrantMessage{
		Workspace: common.GetWorkspaceIDFromContext(ctx),
		ProjectID: &projectID,
		Creator:   &user.Email,
		Limit:     &limitPlusOne,
		Offset:    &offset.offset,
	}

	if req.Filter != "" {
		filterQ, err := store.GetListAccessGrantFilter(req.Filter)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse filter"))
		}
		find.FilterQ = filterQ
	}

	orderByKeys, err := store.GetAccessGrantOrders(req.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.OrderByKeys = orderByKeys

	grants, err := s.store.ListAccessGrants(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to search access grants"))
	}

	resp := &v1pb.SearchMyAccessGrantsResponse{}
	if len(grants) == limitPlusOne {
		nextPageToken, err := offset.getNextPageToken()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		resp.NextPageToken = nextPageToken
		grants = grants[:offset.limit]
	}
	for _, grant := range grants {
		resp.AccessGrants = append(resp.AccessGrants, convertToAccessGrant(grant))
	}

	return connect.NewResponse(resp), nil
}

func convertToAccessGrant(msg *store.AccessGrantMessage) *v1pb.AccessGrant {
	ag := &v1pb.AccessGrant{
		Name:       common.FormatAccessGrant(msg.ProjectID, msg.ID),
		Creator:    common.FormatUserEmail(msg.Creator),
		Status:     convertToAccessGrantStatus(msg.Status),
		CreateTime: timestamppb.New(msg.CreatedAt),
		UpdateTime: timestamppb.New(msg.UpdatedAt),
	}
	if msg.ExpireTime != nil {
		ag.Expiration = &v1pb.AccessGrant_ExpireTime{ExpireTime: timestamppb.New(*msg.ExpireTime)}
	}
	if p := msg.Payload; p != nil {
		ag.Targets = p.Targets
		ag.Query = p.Query
		ag.Unmask = p.Unmask
		if p.IssueId != 0 {
			ag.Issue = common.FormatIssue(msg.ProjectID, p.IssueId)
		}
		if ag.Expiration == nil && p.RequestedDuration != nil {
			ag.Expiration = &v1pb.AccessGrant_Ttl{
				Ttl: p.RequestedDuration,
			}
		}
	}
	return ag
}

func convertToAccessGrantStatus(status storepb.AccessGrant_Status) v1pb.AccessGrant_Status {
	switch status {
	case storepb.AccessGrant_PENDING:
		return v1pb.AccessGrant_PENDING
	case storepb.AccessGrant_ACTIVE:
		return v1pb.AccessGrant_ACTIVE
	case storepb.AccessGrant_REVOKED:
		return v1pb.AccessGrant_REVOKED
	default:
		return v1pb.AccessGrant_STATUS_UNSPECIFIED
	}
}
