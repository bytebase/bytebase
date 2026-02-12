package v1

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// AccessGrantService implements the access grant service.
type AccessGrantService struct {
	v1connect.UnimplementedAccessGrantServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
	webhookManager *webhook.Manager
	bus            *bus.Bus
}

// NewAccessGrantService returns a new access grant service instance.
func NewAccessGrantService(store *store.Store, licenseService *enterprise.LicenseService, webhookManager *webhook.Manager, bus *bus.Bus) *AccessGrantService {
	return &AccessGrantService{
		store:          store,
		licenseService: licenseService,
		webhookManager: webhookManager,
		bus:            bus,
	}
}

// GetAccessGrant gets an access grant by name.
func (s *AccessGrantService) GetAccessGrant(ctx context.Context, request *connect.Request[v1pb.GetAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	req := request.Msg
	projectID, accessGrantID, err := common.GetProjectIDAccessGrantID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	grant, err := s.store.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
		ID:        &accessGrantID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant"))
	}
	if grant == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("access grant %q not found", req.Name))
	}

	return connect.NewResponse(convertToAccessGrant(grant)), nil
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
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_JIT); err != nil {
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
		t := exp.ExpireTime.AsTime()
		expireTime = &t
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
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project != nil {
		if _, err := postCreateIssue(ctx, s.store, s.webhookManager, s.licenseService, s.bus, project, creatorEmail, creatorEmail, issue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(convertToAccessGrant(grant)), nil
}

// ActivateAccessGrant activates a pending access grant.
func (s *AccessGrantService) ActivateAccessGrant(ctx context.Context, request *connect.Request[v1pb.ActivateAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_JIT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	req := request.Msg
	projectID, accessGrantID, err := common.GetProjectIDAccessGrantID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	grant, err := s.store.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
		ID:        &accessGrantID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get access grant"))
	}
	if grant == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("access grant %q not found", req.Name))
	}
	if grant.Status != storepb.AccessGrant_PENDING {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("access grant %q is not in PENDING status", req.Name))
	}

	status := storepb.AccessGrant_ACTIVE
	update := &store.UpdateAccessGrantMessage{
		Status: &status,
	}

	// If the grant was created with a TTL, compute expire_time at activation time.
	if grant.Payload != nil && grant.Payload.RequestedDuration != nil {
		expireTime := time.Now().Add(grant.Payload.RequestedDuration.AsDuration())
		update.ExpireTime = &expireTime
	}

	updated, err := s.store.UpdateAccessGrant(ctx, accessGrantID, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to activate access grant"))
	}

	return connect.NewResponse(convertToAccessGrant(updated)), nil
}

// RevokeAccessGrant revokes an active access grant.
func (s *AccessGrantService) RevokeAccessGrant(ctx context.Context, request *connect.Request[v1pb.RevokeAccessGrantRequest]) (*connect.Response[v1pb.AccessGrant], error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_JIT); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	req := request.Msg
	projectID, accessGrantID, err := common.GetProjectIDAccessGrantID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	grant, err := s.store.GetAccessGrant(ctx, &store.FindAccessGrantMessage{
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

	status := storepb.AccessGrant_REVOKED
	updated, err := s.store.UpdateAccessGrant(ctx, accessGrantID, &store.UpdateAccessGrantMessage{
		Status: &status,
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
			ag.Issue = common.FormatIssue(msg.ProjectID, int(p.IssueId))
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
