package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// IssueService implements the issue service.
type IssueService struct {
	v1connect.UnimplementedIssueServiceHandler
	store          *store.Store
	webhookManager *webhook.Manager
	bus            *bus.Bus
	licenseService *enterprise.LicenseService
	iamManager     *iam.Manager
}

type filterIssueMessage struct {
	ApprovalStatus *v1pb.Issue_ApprovalStatus
	// Approver is the user who can approve the issue.
	Approver *store.UserMessage
}

// NewIssueService creates a new IssueService.
func NewIssueService(
	store *store.Store,
	webhookManager *webhook.Manager,
	bus *bus.Bus,
	licenseService *enterprise.LicenseService,
	iamManager *iam.Manager,
) *IssueService {
	return &IssueService{
		store:          store,
		webhookManager: webhookManager,
		bus:            bus,
		licenseService: licenseService,
		iamManager:     iamManager,
	}
}

// GetIssue gets a issue.
func (s *IssueService) GetIssue(ctx context.Context, req *connect.Request[v1pb.GetIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	issue, err := s.getIssueMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	issueV1, err := s.convertToIssue(issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}
	return connect.NewResponse(issueV1), nil
}

func (s *IssueService) getIssueFind(
	ctx context.Context,
	filter string,
	query string,
	limit,
	offset *int,
) (*store.FindIssueMessage, *filterIssueMessage, error) {
	issueFind := &store.FindIssueMessage{
		Limit:  limit,
		Offset: offset,
	}
	if query != "" {
		issueFind.Query = &query
	}
	if filter == "" {
		return issueFind, nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	filterIssue := &filterIssueMessage{}

	var parseFilter func(expr celast.Expr) (string, error)
	parseFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, parseFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "status":
					issueStatus, err := convertToAPIIssueStatus(v1pb.IssueStatus(v1pb.IssueStatus_value[value.(string)]))
					if err != nil {
						return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert to issue status, err: %v", err))
					}
					issueFind.StatusList = append(issueFind.StatusList, issueStatus)
				case "type":
					issueType, err := convertToAPIIssueType(v1pb.Issue_Type(v1pb.Issue_Type_value[value.(string)]))
					if err != nil {
						return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert to issue type, err: %v", err))
					}
					issueFind.Types = &[]storepb.Issue_Type{issueType}
				case "labels":
					issueFind.LabelList = append(issueFind.LabelList, value.(string))
				case "approval_status":
					approvalStatusValue, ok := v1pb.Issue_ApprovalStatus_value[value.(string)]
					if !ok {
						return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid approval_status %q`, value))
					}
					approvalStatus := v1pb.Issue_ApprovalStatus(approvalStatusValue)
					filterIssue.ApprovalStatus = &approvalStatus
				case "current_approver", "creator":
					user, err := s.getUserByIdentifier(ctx, value.(string))
					if err != nil {
						return "", connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user %v with error %v", value, err.Error()))
					}
					if variable == "current_approver" {
						filterIssue.Approver = user
					} else {
						issueFind.CreatorID = &user.Email
					}
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q with %v operator", variable, celoperators.Equals))
				}
			case celoperators.GreaterEquals, celoperators.LessEquals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return "", errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue)
				}
				if variable != "create_time" {
					return "", errors.Errorf(`">=" and "<=" are only supported for "create_time"`)
				}
				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return "", errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				if functionName == celoperators.GreaterEquals {
					issueFind.CreatedAtAfter = &t
				} else {
					issueFind.CreatedAtBefore = &t
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				rawList, ok := value.([]any)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid list value %q for %v", value, variable))
				}
				if len(rawList) == 0 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty list value for filter %v", variable))
				}

				switch variable {
				case "status":
					for _, raw := range rawList {
						newStatus, err := convertToAPIIssueStatus(v1pb.IssueStatus(v1pb.IssueStatus_value[raw.(string)]))
						if err != nil {
							return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert to issue status, err: %v", err))
						}
						issueFind.StatusList = append(issueFind.StatusList, newStatus)
					}
				case "type":
					types := []storepb.Issue_Type{}
					for _, raw := range rawList {
						issueType, err := convertToAPIIssueType(v1pb.Issue_Type(v1pb.Issue_Type_value[raw.(string)]))
						if err != nil {
							return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert to issue type, err: %v", err))
						}
						types = append(types, issueType)
					}
					issueFind.Types = &types
				case "labels":
					for _, label := range rawList {
						issueLabel, ok := label.(string)
						if !ok {
							return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`label should be string`))
						}
						issueFind.LabelList = append(issueFind.LabelList, issueLabel)
					}
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q with %v operator", variable, celoperators.In))
				}
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported function %q", functionName))
			}
		default:
			return "", errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
		return "", nil
	}

	if _, err := parseFilter(ast.NativeRep().Expr()); err != nil {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse filter"))
	}
	return issueFind, filterIssue, nil
}

func (s *IssueService) ListIssues(ctx context.Context, req *connect.Request[v1pb.ListIssuesRequest]) (*connect.Response[v1pb.ListIssuesResponse], error) {
	if req.Msg.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", req.Msg.PageSize))
	}

	projectID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	issueFind, issueFilter, err := s.getIssueFind(ctx, req.Msg.Filter, req.Msg.Query, &limitPlusOne, &offset.offset)
	if err != nil {
		return nil, err
	}
	issueFind.ProjectID = &projectID

	issues, err := s.store.ListIssues(ctx, issueFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to search issue"))
	}

	var nextPageToken string
	if len(issues) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		issues = issues[:offset.limit]
	}

	converted, err := s.convertToIssues(ctx, issues, issueFilter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}
	return connect.NewResponse(&v1pb.ListIssuesResponse{
		Issues:        converted,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *IssueService) SearchIssues(ctx context.Context, req *connect.Request[v1pb.SearchIssuesRequest]) (*connect.Response[v1pb.SearchIssuesResponse], error) {
	if req.Msg.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", req.Msg.PageSize))
	}

	projectID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	issueFind, issueFilter, err := s.getIssueFind(ctx, req.Msg.Filter, req.Msg.Query, &limitPlusOne, &offset.offset)
	if err != nil {
		return nil, err
	}
	if projectID != "-" {
		issueFind.ProjectID = &projectID
	}
	if issueFind.ProjectID == nil {
		user, ok := GetUserFromContext(ctx)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
		}
		projectIDsFilter, err := getProjectIDsSearchFilter(ctx, user, iam.PermissionIssuesGet, s.iamManager, s.store)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get projectIDs"))
		}
		issueFind.ProjectIDs = projectIDsFilter
	}

	issues, err := s.store.ListIssues(ctx, issueFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to search issue"))
	}

	var nextPageToken string
	if len(issues) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		issues = issues[:offset.limit]
	}

	converted, err := s.convertToIssues(ctx, issues, issueFilter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}
	return connect.NewResponse(&v1pb.SearchIssuesResponse{
		Issues:        converted,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *IssueService) getUserByIdentifier(ctx context.Context, identifier string) (*store.UserMessage, error) {
	email := strings.TrimPrefix(identifier, "users/")
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid empty creator identifier"))
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf(`failed to find user "%s" with error: %v`, email, err.Error()))
	}
	if user == nil {
		return nil, errors.Errorf("cannot found user %s", email)
	}
	return user, nil
}

// CreateIssue creates a issue.
func (s *IssueService) CreateIssue(ctx context.Context, req *connect.Request[v1pb.CreateIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	projectID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}
	if project.Setting.ForceIssueLabels && len(req.Msg.Issue.Labels) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("require issue labels"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	issue, err := s.buildIssueMessage(ctx, project, user.Email, req.Msg)
	if err != nil {
		return nil, err
	}
	issue, err = s.store.CreateIssue(ctx, issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create issue"))
	}

	// Trigger ISSUE_CREATED webhook
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Type:    storepb.Activity_ISSUE_CREATED,
		Project: webhook.NewProject(project),
		IssueCreated: &webhook.EventIssueCreated{
			Creator: &webhook.User{
				Name:  user.Name,
				Email: user.Email,
			},
			Issue: webhook.NewIssue(issue),
		},
	})

	// Trigger approval finding based on issue type
	switch issue.Type {
	case storepb.Issue_GRANT_REQUEST, storepb.Issue_DATABASE_EXPORT:
		// GRANT_REQUEST and DATABASE_EXPORT can determine approval immediately:
		// - GRANT_REQUEST only looks at issue message
		// - DATABASE_EXPORT only looks at the plan (already available)
		// Call synchronously to get approval status in the create response
		if err := approval.FindAndApplyApprovalTemplate(ctx, s.store, s.webhookManager, s.licenseService, issue); err != nil {
			slog.Error("failed to find approval template",
				slog.Int("issue_uid", issue.UID),
				slog.String("issue_title", issue.Title),
				log.BBError(err))
			// Continue anyway - non-fatal error
		}

		// Refresh issue to get updated approval payload
		uid := issue.UID
		issue, err = s.store.GetIssue(ctx, &store.FindIssueMessage{UID: &uid})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to refresh issue"))
		}

		// For GRANT_REQUEST that is auto-approved (no approval template), complete it
		if issue.Type == storepb.Issue_GRANT_REQUEST {
			approved, err := utils.CheckApprovalApproved(issue.Payload.Approval)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check if approval is approved"))
			}
			if approved {
				issue, err = s.completeGrantRequestIssue(ctx, issue, issue.Payload.GrantRequest)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal,
						errors.Wrapf(err, "failed to complete grant request"))
				}
			}
		}
	case storepb.Issue_DATABASE_CHANGE:
		// DATABASE_CHANGE needs to wait for plan check to complete
		// Trigger async approval finding via event channel
		s.bus.ApprovalCheckChan <- int64(issue.UID)
	default:
		// For other issue types, no approval finding needed
	}

	converted, err := s.convertToIssue(issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}

	return connect.NewResponse(converted), nil
}

func (s *IssueService) buildIssueMessage(ctx context.Context, project *store.ProjectMessage, userEmail string, request *v1pb.CreateIssueRequest) (*store.IssueMessage, error) {
	var planUID *int64
	var grantRequest *storepb.GrantRequest
	var title, description string

	// Type-specific validation and preparation
	switch request.Issue.Type {
	case v1pb.Issue_GRANT_REQUEST:
		// Title is required for grant requests
		if request.Issue.Title == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("issue title is required"))
		}

		// Check if grant request feature is enabled
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_REQUEST_ROLE_WORKFLOW); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied,
				errors.Errorf("role request requires approval workflow feature (available in Enterprise plan)"))
		}

		// Validate grant request fields
		if request.Issue.GrantRequest.GetRole() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect grant request role"))
		}
		if request.Issue.GrantRequest.GetUser() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect grant request user"))
		}

		// Validate CEL expression if it's not empty
		if expression := request.Issue.GrantRequest.GetCondition().GetExpression(); expression != "" {
			e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create cel environment"))
			}
			if _, issues := e.Compile(expression); issues != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("found issues in grant request condition expression, issues: %v", issues.String()))
			}
		}

		// Convert grant request (inlined from convertGrantRequest)
		grantRequestUserEmail, err := common.GetUserEmail(request.Issue.GrantRequest.User)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user email from %q", request.Issue.GrantRequest.User))
		}
		grantRequestUser, err := s.store.GetUserByEmail(ctx, grantRequestUserEmail)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get user by email %q", grantRequestUserEmail))
		}
		if grantRequestUser == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", request.Issue.GrantRequest.User))
		}
		grantRequest = &storepb.GrantRequest{
			Role:       request.Issue.GrantRequest.Role,
			User:       common.FormatUserEmail(grantRequestUser.Email),
			Condition:  request.Issue.GrantRequest.Condition,
			Expiration: request.Issue.GrantRequest.Expiration,
		}

		title = request.Issue.Title
		description = request.Issue.Description

	case v1pb.Issue_DATABASE_CHANGE, v1pb.Issue_DATABASE_EXPORT:
		// Validate and fetch plan (shared logic for both types)
		if request.Issue.Plan == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("plan is required"))
		}

		_, planID, err := common.GetProjectIDPlanID(request.Issue.Plan)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
		}

		plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID, ProjectID: &project.ResourceID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
		}
		if plan == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", planID, project.ResourceID))
		}
		planUID = &plan.UID

		// Use plan's title and description as defaults if not provided by request
		title = request.Issue.Title
		if title == "" {
			title = plan.Name
		}
		description = request.Issue.Description
		if description == "" {
			description = plan.Description
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unknown issue type %q", request.Issue.Type))
	}

	// Convert v1pb.Issue_Type to storepb.Issue_Type
	issueType, err := convertToAPIIssueType(request.Issue.Type)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to convert issue type"))
	}

	// Build the issue message (common structure)
	issue := &store.IssueMessage{
		ProjectID:    project.ResourceID,
		CreatorEmail: userEmail,
		PlanUID:      planUID,
		Title:        title,
		Status:       storepb.Issue_OPEN,
		Type:         issueType,
		Description:  description,
		Payload: &storepb.Issue{
			GrantRequest: grantRequest,
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
				ApprovalTemplate:    nil,
				Approvers:           nil,
			},
			Labels: request.Issue.Labels,
		},
	}

	return issue, nil
}

// ApproveIssue approves the approval flow of the issue.
func (s *IssueService) ApproveIssue(ctx context.Context, req *connect.Request[v1pb.ApproveIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	issue, err := s.getIssueMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", issue.ProjectID))
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("issue payload approval is nil"))
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding is not done"))
	}
	if payload.Approval.ApprovalTemplate == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("approval template is required"))
	}

	rejectedRole := utils.FindRejectedRole(payload.Approval)
	if rejectedRole != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot approve because the issue has been rejected"))
	}

	role := utils.FindNextPendingRole(payload.Approval)
	if role == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("the issue has been approved"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	canApprove := s.isUserReviewer(ctx, issue, role, user)
	if !canApprove {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot approve because the user does not have the required permission"))
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:    storepb.IssuePayloadApproval_Approver_APPROVED,
		Principal: common.FormatUserEmail(user.Email),
	})

	approved, err := utils.CheckApprovalApproved(payload.Approval)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check if the approval is approved"))
	}

	issue, err = s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: payload.Approval,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update issue"))
	}

	if _, err := s.store.CreateIssueComments(ctx, user.Email, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: req.Msg.Comment,
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{
					Status: storepb.IssuePayloadApproval_Approver_APPROVED,
				},
			},
		},
	}); err != nil {
		slog.Warn("failed to create issue comment", log.BBError(err))
	}

	approval.NotifyApprovalRequested(ctx, s.store, s.webhookManager, issue, project)

	// If the issue is a grant request and approved, complete it
	if issue.Type == storepb.Issue_GRANT_REQUEST && approved {
		issue, err = s.completeGrantRequestIssue(ctx, issue, payload.GrantRequest)
		if err != nil {
			slog.Debug("failed to complete grant request issue", log.BBError(err))
		}
	}

	issueV1, err := s.convertToIssue(issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}

	// Auto-create rollout if this approval completes the approval flow
	if issueV1.ApprovalStatus == v1pb.Issue_APPROVED {
		if issue.PlanUID != nil {
			s.bus.RolloutCreationChan <- *issue.PlanUID
		}
	}

	return connect.NewResponse(issueV1), nil
}

// RejectIssue rejects a issue.
func (s *IssueService) RejectIssue(ctx context.Context, req *connect.Request[v1pb.RejectIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	issue, err := s.getIssueMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", issue.ProjectID))
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("issue payload approval is nil"))
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding is not done"))
	}
	if payload.Approval.ApprovalTemplate == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("approval template is required"))
	}

	rejectedRole := utils.FindRejectedRole(payload.Approval)
	if rejectedRole != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot reject because the issue has been rejected"))
	}

	role := utils.FindNextPendingRole(payload.Approval)
	if role == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("the issue has been approved"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	canApprove := s.isUserReviewer(ctx, issue, role, user)
	if !canApprove {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot reject because the user does not have the required permission"))
	}
	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:    storepb.IssuePayloadApproval_Approver_REJECTED,
		Principal: common.FormatUserEmail(user.Email),
	})

	issue, err = s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: payload.Approval,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update issue"))
	}

	if _, err := s.store.CreateIssueComments(ctx, user.Email, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: req.Msg.Comment,
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{
					Status: storepb.IssuePayloadApproval_Approver_REJECTED,
				},
			},
		},
	}); err != nil {
		slog.Warn("failed to create issue comment", log.BBError(err))
	}

	// Get issue creator for webhook event
	creator, err := s.store.GetUserByEmail(ctx, issue.CreatorEmail)
	if err != nil {
		slog.Warn("failed to get issue creator, using system bot", log.BBError(err))
		creator = store.SystemBotUser
	}

	// Trigger ISSUE_SENT_BACK webhook
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Type:    storepb.Activity_ISSUE_SENT_BACK,
		Project: webhook.NewProject(project),
		SentBack: &webhook.EventIssueSentBack{
			Approver: &webhook.User{
				Name:  user.Name,
				Email: user.Email,
			},
			Creator: &webhook.User{
				Name:  creator.Name,
				Email: creator.Email,
			},
			Issue:  webhook.NewIssue(issue),
			Reason: req.Msg.Comment,
		},
	})

	issueV1, err := s.convertToIssue(issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}
	return connect.NewResponse(issueV1), nil
}

// RequestIssue requests a issue.
func (s *IssueService) RequestIssue(ctx context.Context, req *connect.Request[v1pb.RequestIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	issue, err := s.getIssueMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", issue.ProjectID))
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("issue payload approval is nil"))
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("approval template finding is not done"))
	}
	if payload.Approval.ApprovalTemplate == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("approval template is required"))
	}

	rejectedRole := utils.FindRejectedRole(payload.Approval)
	if rejectedRole == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot request issues because the issue is not rejected"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	canRequest := canRequestIssue(issue.CreatorEmail, user)
	if !canRequest {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot request issues because you are not the issue creator"))
	}

	var updatedApprovers []*storepb.IssuePayloadApproval_Approver
	for _, approver := range payload.Approval.Approvers {
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			continue
		}
		updatedApprovers = append(updatedApprovers, approver)
	}
	payload.Approval.Approvers = updatedApprovers

	issue, err = s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.Issue{
			Approval: payload.Approval,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update issue"))
	}

	approval.NotifyApprovalRequested(ctx, s.store, s.webhookManager, issue, project)

	if _, err := s.store.CreateIssueComments(ctx, user.Email, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: req.Msg.Comment,
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{
					Status: storepb.IssuePayloadApproval_Approver_PENDING,
				},
			},
		},
	}); err != nil {
		slog.Warn("failed to create issue comment", log.BBError(err))
	}

	issueV1, err := s.convertToIssue(issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}
	return connect.NewResponse(issueV1), nil
}

// UpdateIssue updates the issue.
func (s *IssueService) UpdateIssue(ctx context.Context, req *connect.Request[v1pb.UpdateIssueRequest]) (*connect.Response[v1pb.Issue], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}

	issue, err := s.getIssueMessage(ctx, req.Msg.Issue.Name)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound && req.Msg.AllowMissing {
			// When allow_missing is true and issue doesn't exist, create a new one
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionIssuesCreate, user)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionIssuesCreate))
			}

			// Extract project ID from the issue name (format: projects/{project}/issues/{issue})
			parts := strings.Split(req.Msg.Issue.Name, "/")
			if len(parts) < 4 || parts[0] != "projects" || parts[2] != "issues" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid issue name format: %q", req.Msg.Issue.Name))
			}
			projectID := parts[1]

			return s.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
				Parent: fmt.Sprintf("projects/%s", projectID),
				Issue:  req.Msg.Issue,
			}))
		}
		return nil, err
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", issue.ProjectID))
	}

	updateMasks := map[string]bool{}

	patch := &store.UpdateIssueMessage{}
	var issueCommentCreates []*store.IssueCommentMessage
	for _, path := range req.Msg.UpdateMask.Paths {
		updateMasks[path] = true
		switch path {
		case "title":
			if req.Msg.Issue.Title == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("title cannot be empty"))
			}

			patch.Title = &req.Msg.Issue.Title

			issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromTitle: &issue.Title,
							ToTitle:   &req.Msg.Issue.Title,
						},
					},
				},
			})

		case "description":
			patch.Description = &req.Msg.Issue.Description

			issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromDescription: &issue.Description,
							ToDescription:   &req.Msg.Issue.Description,
						},
					},
				},
			})

		case "labels":
			if len(req.Msg.Issue.Labels) == 0 {
				patch.RemoveLabels = true
			} else {
				if patch.PayloadUpsert == nil {
					patch.PayloadUpsert = &storepb.Issue{}
				}
				patch.PayloadUpsert.Labels = req.Msg.Issue.Labels
			}

			issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromLabels: issue.Payload.Labels,
							ToLabels:   req.Msg.Issue.Labels,
						},
					},
				},
			})
		default:
		}
	}

	issue, err = s.store.UpdateIssue(ctx, issue.UID, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update issue"))
	}

	if _, err := s.store.CreateIssueComments(ctx, user.Email, issueCommentCreates...); err != nil {
		slog.Warn("failed to create issue comments", "issue id", issue.UID, log.BBError(err))
	}

	issueV1, err := s.convertToIssue(issue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to issue"))
	}
	return connect.NewResponse(issueV1), nil
}

// BatchUpdateIssuesStatus batch updates issues status.
func (s *IssueService) BatchUpdateIssuesStatus(ctx context.Context, req *connect.Request[v1pb.BatchUpdateIssuesStatusRequest]) (*connect.Response[v1pb.BatchUpdateIssuesStatusResponse], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	if len(req.Msg.Issues) == 0 {
		return connect.NewResponse(&v1pb.BatchUpdateIssuesStatusResponse{}), nil
	}

	newStatus, err := convertToAPIIssueStatus(req.Msg.Status)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert to issue status, err: %v", err))
	}

	// Parse issue names and validate all issues belong to the same project.
	var projectID string
	issueUIDs := make([]int, 0, len(req.Msg.Issues))
	for i, issueName := range req.Msg.Issues {
		issueProjectID, issueUID, err := common.GetProjectIDIssueUID(issueName)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid issue name %q: %v", issueName, err))
		}

		// Ensure all issues belong to the same project.
		if i == 0 {
			projectID = issueProjectID
		} else if issueProjectID != projectID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("all issues must belong to the same project, found %q and %q", projectID, issueProjectID))
		}

		issueUIDs = append(issueUIDs, issueUID)
	}

	// Get project early for webhooks.
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}

	// Batch update issue statuses. This validates project membership, DONE status, and returns old statuses.
	oldIssueStatuses, err := s.store.BatchUpdateIssueStatuses(ctx, projectID, issueUIDs, newStatus)
	if err != nil {
		if common.ErrorCode(err) == common.Invalid {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to batch update issues, err: %v", err))
	}

	// Batch create issue comments.
	issueComments := make([]*store.IssueCommentMessage, 0, len(issueUIDs))
	for _, issueUID := range issueUIDs {
		oldStatus := oldIssueStatuses[issueUID]
		issueComments = append(issueComments, &store.IssueCommentMessage{
			IssueUID: issueUID,
			Payload: &storepb.IssueCommentPayload{
				Comment: req.Msg.Reason,
				Event: &storepb.IssueCommentPayload_IssueUpdate_{
					IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
						FromStatus: &oldStatus,
						ToStatus:   &newStatus,
					},
				},
			},
		})
	}
	if _, err := s.store.CreateIssueComments(ctx, user.Email, issueComments...); err != nil {
		slog.Error("failed to batch create issue comments", log.BBError(err))
	}

	// Create webhooks for each updated issue.
	for _, issueUID := range issueUIDs {
		updatedIssue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{UID: &issueUID, ProjectID: &projectID})
		if err != nil {
			slog.Error("failed to get updated issue", "issueUID", issueUID, log.BBError(err))
			continue
		}
		if updatedIssue == nil {
			slog.Error("updated issue not found", "issueUID", issueUID)
			continue
		}
	}

	return connect.NewResponse(&v1pb.BatchUpdateIssuesStatusResponse{}), nil
}

func (s *IssueService) ListIssueComments(ctx context.Context, req *connect.Request[v1pb.ListIssueCommentsRequest]) (*connect.Response[v1pb.ListIssueCommentsResponse], error) {
	if req.Msg.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", req.Msg.PageSize))
	}
	projectID, issueUID, err := common.GetProjectIDIssueUID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("%v", err.Error()))
	}
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		UID:       &issueUID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue, err: %v", err))
	}
	if issue == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("issue %q not found", req.Msg.Parent))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	issueComments, err := s.store.ListIssueComment(ctx, &store.FindIssueCommentMessage{
		IssueUID: &issue.UID,
		Limit:    &limitPlusOne,
		Offset:   &offset.offset,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list issue comments, err: %v", err))
	}
	var nextPageToken string
	if len(issueComments) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		issueComments = issueComments[:offset.limit]
	}

	return connect.NewResponse(&v1pb.ListIssueCommentsResponse{
		IssueComments: convertToIssueComments(req.Msg.Parent, issueComments),
		NextPageToken: nextPageToken,
	}), nil
}

// CreateIssueComment creates the issue comment.
func (s *IssueService) CreateIssueComment(ctx context.Context, req *connect.Request[v1pb.CreateIssueCommentRequest]) (*connect.Response[v1pb.IssueComment], error) {
	if req.Msg.IssueComment.Comment == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("issue comment is empty"))
	}
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	issue, err := s.getIssueMessage(ctx, req.Msg.Parent)
	if err != nil {
		return nil, err
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &issue.ProjectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", issue.ProjectID))
	}

	ic, err := s.store.CreateIssueComments(ctx, user.Email, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: req.Msg.IssueComment.Comment,
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create issue comment: %v", err))
	}

	return connect.NewResponse(convertToIssueComment(req.Msg.Parent, ic)), nil
}

// UpdateIssueComment updates the issue comment.
func (s *IssueService) UpdateIssueComment(ctx context.Context, req *connect.Request[v1pb.UpdateIssueCommentRequest]) (*connect.Response[v1pb.IssueComment], error) {
	if req.Msg.UpdateMask.Paths == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask is required"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	_, _, issueCommentUID, err := common.GetProjectIDIssueUIDIssueCommentUID(req.Msg.IssueComment.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid comment name %q: %v", req.Msg.IssueComment.Name, err))
	}
	issueComment, err := s.store.GetIssueComment(ctx, &store.FindIssueCommentMessage{UID: &issueCommentUID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue comment: %v", err))
	}
	if issueComment == nil {
		if req.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionIssueCommentsCreate, user)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionIssueCommentsCreate))
			}
			return s.CreateIssueComment(ctx, connect.NewRequest(&v1pb.CreateIssueCommentRequest{
				Parent:       req.Msg.Parent,
				IssueComment: req.Msg.IssueComment,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("issue comment not found"))
	}

	update := &store.UpdateIssueCommentMessage{
		UID: issueCommentUID,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "comment":
			update.Comment = &req.Msg.IssueComment.Comment
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupport update_mask: "%s"`, path))
		}
	}

	if err := s.store.UpdateIssueComment(ctx, update); err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found the issue comment %s", req.Msg.IssueComment.Name))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update the issue comment with error: %v", err.Error()))
	}
	issueComment, err = s.store.GetIssueComment(ctx, &store.FindIssueCommentMessage{UID: &issueCommentUID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue comment: %v", err))
	}

	return connect.NewResponse(convertToIssueComment(req.Msg.Parent, issueComment)), nil
}

func (s *IssueService) getIssueMessage(ctx context.Context, name string) (*store.IssueMessage, error) {
	projectID, issueUID, err := common.GetProjectIDIssueUID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
		UID:       &issueUID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get issue"))
	}
	if issue == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("issue %d not found in project %s", issueUID, projectID))
	}
	return issue, nil
}

// completeGrantRequestIssue grants privilege and closes a grant request issue.
// Called when:
// 1. Issue created without approval template (auto-approved)
// 2. Issue approval flow completes
//
// Returns the updated issue with DONE status.
func (s *IssueService) completeGrantRequestIssue(ctx context.Context, issue *store.IssueMessage, grantRequest *storepb.GrantRequest) (*store.IssueMessage, error) {
	// Grant the privilege
	if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, s.store, issue, grantRequest); err != nil {
		return nil, err
	}

	// Update issue status to DONE
	newStatus := storepb.Issue_DONE
	updatedIssue, err := s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{Status: &newStatus})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	// Create issue comment documenting the status change
	if _, err := s.store.CreateIssueComments(ctx, common.SystemBotEmail, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Event: &storepb.IssueCommentPayload_IssueUpdate_{
				IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
					FromStatus: &issue.Status,
					ToStatus:   &updatedIssue.Status,
				},
			},
		},
	}); err != nil {
		// Non-fatal: log warning but continue
		slog.Warn("failed to create issue comment after changing the issue status", log.BBError(err))
	}

	return updatedIssue, nil
}

func (s *IssueService) isUserReviewer(ctx context.Context, issue *store.IssueMessage, role string, user *store.UserMessage) bool {
	roles := s.getUserRoleMap(ctx, issue.ProjectID, user)
	return roles[role]
}

func canRequestIssue(issueCreatorEmail string, user *store.UserMessage) bool {
	return issueCreatorEmail == user.Email
}
