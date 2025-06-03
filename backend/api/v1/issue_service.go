package v1

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	metricapi "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// IssueService implements the issue service.
type IssueService struct {
	v1pb.UnimplementedIssueServiceServer
	store          *store.Store
	webhookManager *webhook.Manager
	stateCfg       *state.State
	licenseService enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
	metricReporter *metricreport.Reporter
}

// NewIssueService creates a new IssueService.
func NewIssueService(
	store *store.Store,
	webhookManager *webhook.Manager,
	stateCfg *state.State,
	licenseService enterprise.LicenseService,
	profile *config.Profile,
	iamManager *iam.Manager,
	metricReporter *metricreport.Reporter,
) *IssueService {
	return &IssueService{
		store:          store,
		webhookManager: webhookManager,
		stateCfg:       stateCfg,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
		metricReporter: metricReporter,
	}
}

// GetIssue gets a issue.
func (s *IssueService) GetIssue(ctx context.Context, request *v1pb.GetIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	issueV1, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

func (s *IssueService) getIssueFind(ctx context.Context, filter string, query string, limit, offset *int) (*store.FindIssueMessage, error) {
	issueFind := &store.FindIssueMessage{
		Limit:  limit,
		Offset: offset,
	}
	if query != "" {
		issueFind.Query = &query
	}
	if filter == "" {
		return issueFind, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse filter %v, error: %v", filter, iss.String())
	}

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
				case "creator", "subscriber":
					user, err := s.getUserByIdentifier(ctx, value.(string))
					if err != nil {
						return "", status.Errorf(codes.Internal, "failed to get user %v with error %v", value, err.Error())
					}
					if variable == "creator" {
						issueFind.CreatorID = &user.ID
					} else {
						issueFind.SubscriberID = &user.ID
					}
				case "instance":
					instanceResourceID, err := common.GetInstanceID(value.(string))
					if err != nil {
						return "", status.Errorf(codes.InvalidArgument, `invalid instance resource id "%s": %v`, value, err.Error())
					}
					issueFind.InstanceResourceID = &instanceResourceID
				case "database":
					database, err := getDatabaseMessage(ctx, s.store, value.(string))
					if err != nil {
						return "", status.Error(codes.Internal, err.Error())
					}
					if database == nil || database.Deleted {
						return "", status.Errorf(codes.InvalidArgument, `database "%q" not found`, value)
					}
					issueFind.InstanceID = &database.InstanceID
					issueFind.DatabaseName = &database.DatabaseName
				case "has_pipeline":
					hasPipeline, ok := value.(bool)
					if !ok {
						return "", status.Errorf(codes.InvalidArgument, `"has_pipeline" should be bool`)
					}
					if !hasPipeline {
						issueFind.NoPipeline = true
					}
				case "status":
					issueStatus, err := convertToAPIIssueStatus(v1pb.IssueStatus(v1pb.IssueStatus_value[value.(string)]))
					if err != nil {
						return "", status.Errorf(codes.InvalidArgument, "failed to convert to issue status, err: %v", err)
					}
					issueFind.StatusList = append(issueFind.StatusList, issueStatus)
				case "type":
					issueType, err := convertToAPIIssueType(v1pb.Issue_Type(v1pb.Issue_Type_value[value.(string)]))
					if err != nil {
						return "", status.Errorf(codes.InvalidArgument, "failed to convert to issue type, err: %v", err)
					}
					issueFind.Types = &[]base.IssueType{issueType}
				case "task_type":
					taskType, ok := value.(string)
					if !ok {
						return "", status.Errorf(codes.InvalidArgument, `"task_type" should be string`)
					}
					switch taskType {
					case "DDL":
						issueFind.TaskTypes = &[]base.TaskType{
							base.TaskDatabaseSchemaUpdate,
							base.TaskDatabaseSchemaUpdateGhost,
						}
					case "DML":
						issueFind.TaskTypes = &[]base.TaskType{
							base.TaskDatabaseDataUpdate,
						}
					case "DATA_EXPORT":
						issueFind.TaskTypes = &[]base.TaskType{
							base.TaskDatabaseDataExport,
						}
					default:
						return "", status.Errorf(codes.InvalidArgument, `unknown value %q`, value)
					}
				case "labels":
					issueFind.LabelList = append(issueFind.LabelList, value.(string))
				default:
					return "", status.Errorf(codes.InvalidArgument, "unsupport variable %q with %v operator", variable, celoperators.Equals)
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
					return "", status.Errorf(codes.InvalidArgument, "invalid list value %q for %v", value, variable)
				}
				if len(rawList) == 0 {
					return "", status.Errorf(codes.InvalidArgument, "empty list value for filter %v", variable)
				}

				switch variable {
				case "status":
					for _, raw := range rawList {
						newStatus, err := convertToAPIIssueStatus(v1pb.IssueStatus(v1pb.IssueStatus_value[raw.(string)]))
						if err != nil {
							return "", status.Errorf(codes.InvalidArgument, "failed to convert to issue status, err: %v", err)
						}
						issueFind.StatusList = append(issueFind.StatusList, newStatus)
					}
				case "type":
					types := []base.IssueType{}
					for _, raw := range rawList {
						issueType, err := convertToAPIIssueType(v1pb.Issue_Type(v1pb.Issue_Type_value[raw.(string)]))
						if err != nil {
							return "", status.Errorf(codes.InvalidArgument, "failed to convert to issue type, err: %v", err)
						}
						types = append(types, issueType)
					}
					issueFind.Types = &types
				case "labels":
					for _, label := range rawList {
						issueLabel, ok := label.(string)
						if !ok {
							return "", status.Errorf(codes.InvalidArgument, `label should be string`)
						}
						issueFind.LabelList = append(issueFind.LabelList, issueLabel)
					}
				default:
					return "", status.Errorf(codes.InvalidArgument, "unsupport variable %q with %v operator", variable, celoperators.In)
				}
			}
		default:
			return "", errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
		return "", nil
	}

	if _, err := parseFilter(ast.NativeRep().Expr()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse filter, error: %v", err)
	}
	return issueFind, nil
}

func (s *IssueService) ListIssues(ctx context.Context, request *v1pb.ListIssuesRequest) (*v1pb.ListIssuesResponse, error) {
	if request.PageSize < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page size must be non-negative: %d", request.PageSize)
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	issueFind, err := s.getIssueFind(ctx, request.Filter, request.Query, &limitPlusOne, &offset.offset)
	if err != nil {
		return nil, err
	}
	issueFind.ProjectID = &projectID

	issues, err := s.store.ListIssueV2(ctx, issueFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search issue, error: %v", err)
	}

	var nextPageToken string
	if len(issues) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		issues = issues[:offset.limit]
	}

	converted, err := s.convertToIssues(ctx, issues)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return &v1pb.ListIssuesResponse{
		Issues:        converted,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *IssueService) SearchIssues(ctx context.Context, request *v1pb.SearchIssuesRequest) (*v1pb.SearchIssuesResponse, error) {
	if request.PageSize < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page size must be non-negative: %d", request.PageSize)
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	issueFind, err := s.getIssueFind(ctx, request.Filter, request.Query, &limitPlusOne, &offset.offset)
	if err != nil {
		return nil, err
	}
	if projectID != "-" {
		issueFind.ProjectID = &projectID
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	projectIDsFilter, err := getProjectIDsSearchFilter(ctx, user, iam.PermissionIssuesGet, s.iamManager, s.store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get projectIDs, error: %v", err)
	}
	issueFind.ProjectIDs = projectIDsFilter

	issues, err := s.store.ListIssueV2(ctx, issueFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search issue, error: %v", err)
	}

	var nextPageToken string
	if len(issues) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		issues = issues[:offset.limit]
	}

	converted, err := s.convertToIssues(ctx, issues)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return &v1pb.SearchIssuesResponse{
		Issues:        converted,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *IssueService) getUserByIdentifier(ctx context.Context, identifier string) (*store.UserMessage, error) {
	email := strings.TrimPrefix(identifier, "users/")
	if email == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid empty creator identifier")
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, `failed to find user "%s" with error: %v`, email, err.Error())
	}
	if user == nil {
		return nil, errors.Errorf("cannot found user %s", email)
	}
	return user, nil
}

// CreateIssue creates a issue.
func (s *IssueService) CreateIssue(ctx context.Context, request *v1pb.CreateIssueRequest) (*v1pb.Issue, error) {
	// Validate requests.
	if request.Issue.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "issue title is required")
	}
	if request.Issue.Type == v1pb.Issue_TYPE_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "issue type is required")
	}

	switch request.Issue.Type {
	case v1pb.Issue_GRANT_REQUEST:
		return s.createIssueGrantRequest(ctx, request)
	case v1pb.Issue_DATABASE_CHANGE:
		return s.createIssueDatabaseChange(ctx, request)
	case v1pb.Issue_DATABASE_DATA_EXPORT:
		return s.createIssueDatabaseDataExport(ctx, request)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown issue type %q", request.Issue.Type)
	}
}

func (s *IssueService) createIssueDatabaseChange(ctx context.Context, request *v1pb.CreateIssueRequest) (*v1pb.Issue, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found for id: %v", projectID)
	}

	if request.Issue.Plan == "" {
		return nil, status.Errorf(codes.InvalidArgument, "plan is required")
	}

	var planUID *int64
	_, planID, err := common.GetProjectIDPlanID(request.Issue.Plan)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}
	planUID = &plan.UID
	var rolloutUID *int
	if request.Issue.Rollout != "" {
		_, rolloutID, err := common.GetProjectIDRolloutID(request.Issue.Rollout)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		pipeline, err := s.store.GetPipelineV2ByID(ctx, rolloutID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get rollout, error: %v", err)
		}
		if pipeline == nil {
			return nil, status.Errorf(codes.NotFound, "rollout not found for id: %d", rolloutID)
		}
		rolloutUID = &pipeline.ID
	}

	issueCreateMessage := &store.IssueMessage{
		Project:     project,
		PlanUID:     planUID,
		PipelineUID: rolloutUID,
		Title:       request.Issue.Title,
		Status:      storepb.IssueStatus_OPEN,
		Type:        base.IssueDatabaseGeneral,
		Description: request.Issue.Description,
	}

	issueCreateMessage.Payload = &storepb.IssuePayload{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
			ApprovalTemplates:   nil,
			Approvers:           nil,
		},
		Labels: request.Issue.Labels,
	}

	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue, error: %v", err)
	}
	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    base.EventTypeIssueCreate,
		Comment: "",
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
	})

	converted, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}

	return converted, nil
}

func (s *IssueService) createIssueGrantRequest(ctx context.Context, request *v1pb.CreateIssueRequest) (*v1pb.Issue, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found for id: %v", projectID)
	}

	if request.Issue.GrantRequest.GetRole() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "expect grant request role")
	}
	if request.Issue.GrantRequest.GetUser() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "expect grant request user")
	}
	// Validate CEL expression if it's not empty.
	if expression := request.Issue.GrantRequest.GetCondition().GetExpression(); expression != "" {
		e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create cel environment, error: %v", err)
		}
		if _, issues := e.Compile(expression); issues != nil {
			return nil, status.Errorf(codes.InvalidArgument, "found issues in grant request condition expression, issues: %v", issues.String())
		}
	}

	issueCreateMessage := &store.IssueMessage{
		Project:     project,
		PlanUID:     nil,
		PipelineUID: nil,
		Title:       request.Issue.Title,
		Status:      storepb.IssueStatus_OPEN,
		Type:        base.IssueGrantRequest,
		Description: request.Issue.Description,
	}

	convertedGrantRequest, err := convertGrantRequest(ctx, s.store, request.Issue.GrantRequest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert GrantRequest, error: %v", err)
	}

	issueCreateMessage.Payload = &storepb.IssuePayload{
		GrantRequest: convertedGrantRequest,
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
			ApprovalTemplates:   nil,
			Approvers:           nil,
		},
		Labels: request.Issue.Labels,
	}

	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue, error: %v", err)
	}
	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    base.EventTypeIssueCreate,
		Comment: "",
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
	})

	converted, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}

	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricapi.IssueCreateMetricName,
		Value: 1,
		Labels: map[string]any{
			"type": issue.Type,
		},
	})

	return converted, nil
}

func (s *IssueService) createIssueDatabaseDataExport(ctx context.Context, request *v1pb.CreateIssueRequest) (*v1pb.Issue, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found for id: %v", projectID)
	}

	if request.Issue.Plan == "" {
		return nil, status.Errorf(codes.InvalidArgument, "plan is required")
	}

	var planUID *int64
	_, planID, err := common.GetProjectIDPlanID(request.Issue.Plan)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}
	planUID = &plan.UID
	var rolloutUID *int
	if request.Issue.Rollout != "" {
		_, rolloutID, err := common.GetProjectIDRolloutID(request.Issue.Rollout)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		pipeline, err := s.store.GetPipelineV2ByID(ctx, rolloutID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get rollout, error: %v", err)
		}
		if pipeline == nil {
			return nil, status.Errorf(codes.NotFound, "rollout not found for id: %d", rolloutID)
		}
		rolloutUID = &pipeline.ID
	}

	issueCreateMessage := &store.IssueMessage{
		Project:     project,
		PlanUID:     planUID,
		PipelineUID: rolloutUID,
		Title:       request.Issue.Title,
		Status:      storepb.IssueStatus_OPEN,
		Type:        base.IssueDatabaseDataExport,
		Description: request.Issue.Description,
	}

	issueCreateMessage.Payload = &storepb.IssuePayload{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
			ApprovalTemplates:   nil,
			Approvers:           nil,
		},
		Labels: request.Issue.Labels,
	}

	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue, error: %v", err)
	}
	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    base.EventTypeIssueCreate,
		Comment: "",
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
	})

	converted, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}

	return converted, nil
}

// ApproveIssue approves the approval flow of the issue.
func (s *IssueService) ApproveIssue(ctx context.Context, request *v1pb.ApproveIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, status.Errorf(codes.Internal, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return nil, status.Errorf(codes.Internal, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot approve because the issue has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return nil, status.Errorf(codes.InvalidArgument, "the issue has been approved")
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by id %v", principalID)
	}

	policy, err := s.store.GetProjectIamPolicy(ctx, issue.Project.ResourceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project policy, error: %v", err)
	}

	workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace policy, error: %v", err)
	}

	canApprove, err := isUserReviewer(ctx, s.store, issue, step, user, policy.Policy, workspacePolicy.Policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if principal can approve step, error: %v", err)
	}
	if !canApprove {
		return nil, status.Errorf(codes.PermissionDenied, "cannot approve because the user does not have the required permission")
	}

	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:      storepb.IssuePayloadApproval_Approver_APPROVED,
		PrincipalId: int32(principalID),
	})

	approved, err := utils.CheckApprovalApproved(payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if the approval is approved, error: %v", err)
	}

	newApprovers, err := utils.HandleIncomingApprovalSteps(payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle incoming approval steps, error: %v", err)
	}
	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: payload.Approval,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	// Grant the privilege if the issue is approved.
	if approved && issue.Type == base.IssueGrantRequest {
		if err := utils.UpdateProjectPolicyFromGrantIssue(ctx, s.store, issue, payload.GrantRequest); err != nil {
			return nil, err
		}
		// TODO(p0ny): Post project IAM policy update activity.
	}

	if err := func() error {
		p := &storepb.IssueCommentPayload{
			Comment: request.Comment,
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{
					Status: storepb.IssueCommentPayload_Approval_APPROVED,
				},
			},
		}
		if _, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
			IssueUID: issue.UID,
			Payload:  p,
		}, user.ID); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		slog.Warn("failed to create issue comment", log.BBError(err))
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 1 {
			return nil
		}
		approvalStep := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
		if approvalStep == nil {
			return nil
		}

		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   s.store.GetSystemBotUser(ctx),
			Type:    base.EventTypeIssueApprovalCreate,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			IssueApprovalCreate: &webhook.EventIssueApprovalCreate{
				ApprovalStep: approvalStep,
			},
		})

		return nil
	}(); err != nil {
		slog.Error("failed to create approval step pending activity after creating issue", log.BBError(err))
	}

	func() {
		if !approved {
			return
		}

		// notify issue approved
		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   s.store.GetSystemBotUser(ctx),
			Type:    base.EventTypeIssueApprovalPass,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
		})

		// notify pipeline rollout
		if err := func() error {
			if issue.PipelineUID == nil {
				return nil
			}
			stages, err := s.store.ListStageV2(ctx, *issue.PipelineUID)
			if err != nil {
				return errors.Wrapf(err, "failed to list stages")
			}
			if len(stages) == 0 {
				return nil
			}

			policy, err := GetValidRolloutPolicyForStage(ctx, s.store, stages[0])
			if err != nil {
				return err
			}
			s.webhookManager.CreateEvent(ctx, &webhook.Event{
				Actor:   user,
				Type:    base.EventTypeIssueRolloutReady,
				Comment: "",
				Issue:   webhook.NewIssue(issue),
				Project: webhook.NewProject(issue.Project),
				IssueRolloutReady: &webhook.EventIssueRolloutReady{
					RolloutPolicy: policy,
					StageName:     stages[0].Environment,
				},
			})
			return nil
		}(); err != nil {
			slog.Error("failed to create rollout release notification activity", log.BBError(err))
		}
	}()

	// If the issue is a grant request and approved, we will always auto close it.
	if issue.Type == base.IssueGrantRequest {
		if err := func() error {
			payload := issue.Payload
			approved, err := utils.CheckApprovalApproved(payload.Approval)
			if err != nil {
				return errors.Wrap(err, "failed to check if the approval is approved")
			}
			if approved {
				if err := webhook.ChangeIssueStatus(ctx, s.store, s.webhookManager, issue, storepb.IssueStatus_DONE, s.store.GetSystemBotUser(ctx), ""); err != nil {
					return errors.Wrap(err, "failed to update issue status")
				}
			}
			return nil
		}(); err != nil {
			slog.Debug("failed to update issue status to done if grant request issue is approved", log.BBError(err))
		}
	}

	issueV1, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// RejectIssue rejects a issue.
func (s *IssueService) RejectIssue(ctx context.Context, request *v1pb.RejectIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, status.Errorf(codes.Internal, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return nil, status.Errorf(codes.Internal, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot reject because the issue has been rejected")
	}

	step := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if step == nil {
		return nil, status.Errorf(codes.InvalidArgument, "the issue has been approved")
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by id %v", principalID)
	}

	policy, err := s.store.GetProjectIamPolicy(ctx, issue.Project.ResourceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project policy, error: %v", err)
	}

	workspacePolicy, err := s.store.GetWorkspaceIamPolicy(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace policy, error: %v", err)
	}

	canApprove, err := isUserReviewer(ctx, s.store, issue, step, user, policy.Policy, workspacePolicy.Policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check if principal can reject step, error: %v", err)
	}
	if !canApprove {
		return nil, status.Errorf(codes.PermissionDenied, "cannot reject because the user does not have the required permission")
	}
	payload.Approval.Approvers = append(payload.Approval.Approvers, &storepb.IssuePayloadApproval_Approver{
		Status:      storepb.IssuePayloadApproval_Approver_REJECTED,
		PrincipalId: int32(principalID),
	})

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: payload.Approval,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	if err := func() error {
		p := &storepb.IssueCommentPayload{
			Comment: request.Comment,
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{
					Status: storepb.IssueCommentPayload_Approval_REJECTED,
				},
			},
		}
		_, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
			IssueUID: issue.UID,
			Payload:  p,
		}, user.ID)
		return err
	}(); err != nil {
		slog.Warn("failed to create issue comment", log.BBError(err))
	}

	issueV1, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// RequestIssue requests a issue.
func (s *IssueService) RequestIssue(ctx context.Context, request *v1pb.RequestIssueRequest) (*v1pb.Issue, error) {
	issue, err := s.getIssueMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	payload := issue.Payload
	if payload.Approval == nil {
		return nil, status.Errorf(codes.Internal, "issue payload approval is nil")
	}
	if !payload.Approval.ApprovalFindingDone {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
	}
	if payload.Approval.ApprovalFindingError != "" {
		return nil, status.Errorf(codes.FailedPrecondition, "approval template finding failed: %v", payload.Approval.ApprovalFindingError)
	}
	if len(payload.Approval.ApprovalTemplates) != 1 {
		return nil, status.Errorf(codes.Internal, "expecting one approval template but got %v", len(payload.Approval.ApprovalTemplates))
	}

	rejectedStep := utils.FindRejectedStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
	if rejectedStep == nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot request issues because the issue is not rejected")
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user by id %v", principalID)
	}

	canRequest := canRequestIssue(issue.Creator, user)
	if !canRequest {
		return nil, status.Errorf(codes.PermissionDenied, "cannot request issues because you are not the issue creator")
	}

	var updatedApprovers []*storepb.IssuePayloadApproval_Approver
	for _, approver := range payload.Approval.Approvers {
		if approver.Status == storepb.IssuePayloadApproval_Approver_REJECTED {
			continue
		}
		updatedApprovers = append(updatedApprovers, approver)
	}
	payload.Approval.Approvers = updatedApprovers

	newApprovers, err := utils.HandleIncomingApprovalSteps(payload.Approval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle incoming approval steps, error: %v", err)
	}
	payload.Approval.Approvers = append(payload.Approval.Approvers, newApprovers...)

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
		PayloadUpsert: &storepb.IssuePayload{
			Approval: payload.Approval,
		},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	if err := func() error {
		if len(payload.Approval.ApprovalTemplates) != 1 {
			return nil
		}
		approvalStep := utils.FindNextPendingStep(payload.Approval.ApprovalTemplates[0], payload.Approval.Approvers)
		if approvalStep == nil {
			return nil
		}

		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   s.store.GetSystemBotUser(ctx),
			Type:    base.EventTypeIssueApprovalCreate,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			IssueApprovalCreate: &webhook.EventIssueApprovalCreate{
				ApprovalStep: approvalStep,
			},
		})

		return nil
	}(); err != nil {
		slog.Error("failed to create approval step pending activity after re-request issue review", log.BBError(err))
	}

	if err := func() error {
		p := &storepb.IssueCommentPayload{
			Comment: request.Comment,
			Event: &storepb.IssueCommentPayload_Approval_{
				Approval: &storepb.IssueCommentPayload_Approval{
					Status: storepb.IssueCommentPayload_Approval_PENDING,
				},
			},
		}
		if _, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
			IssueUID: issue.UID,
			Payload:  p,
		}, user.ID); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		slog.Warn("failed to create issue comment", log.BBError(err))
	}

	issueV1, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// UpdateIssue updates the issue.
func (s *IssueService) UpdateIssue(ctx context.Context, request *v1pb.UpdateIssueRequest) (*v1pb.Issue, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	issue, err := s.getIssueMessage(ctx, request.Issue.Name)
	if err != nil {
		return nil, err
	}

	updateMasks := map[string]bool{}

	patch := &store.UpdateIssueMessage{}
	var webhookEvents []*webhook.Event
	var issueCommentCreates []*store.IssueCommentMessage
	for _, path := range request.UpdateMask.Paths {
		updateMasks[path] = true
		switch path {
		case "approval_finding_done":
			if request.Issue.ApprovalFindingDone {
				return nil, status.Errorf(codes.InvalidArgument, "cannot set approval_finding_done to true")
			}
			payload := issue.Payload
			if payload.Approval == nil {
				return nil, status.Errorf(codes.Internal, "issue payload approval is nil")
			}
			if !payload.Approval.ApprovalFindingDone {
				return nil, status.Errorf(codes.FailedPrecondition, "approval template finding is not done")
			}

			if patch.PayloadUpsert == nil {
				patch.PayloadUpsert = &storepb.IssuePayload{}
			}
			patch.PayloadUpsert.Approval = &storepb.IssuePayloadApproval{
				ApprovalFindingDone: false,
			}

			if issue.PlanUID != nil {
				plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
				}
				if plan == nil {
					return nil, status.Errorf(codes.NotFound, "plan %q not found", *issue.PlanUID)
				}

				planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, plan)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
				}
				if err := s.store.CreatePlanCheckRuns(ctx, plan, planCheckRuns...); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
				}
			}

		case "title":
			if request.Issue.Title == "" {
				return nil, status.Errorf(codes.InvalidArgument, "title cannot be empty")
			}

			patch.Title = &request.Issue.Title

			issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromTitle: &issue.Title,
							ToTitle:   &request.Issue.Title,
						},
					},
				},
			})

			webhookEvents = append(webhookEvents, &webhook.Event{
				Actor:   user,
				Type:    base.EventTypeIssueUpdate,
				Comment: "",
				Issue:   webhook.NewIssue(issue),
				Project: webhook.NewProject(issue.Project),
				IssueUpdate: &webhook.EventIssueUpdate{
					Path: path,
				},
			})

		case "description":
			patch.Description = &request.Issue.Description

			issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromDescription: &issue.Description,
							ToDescription:   &request.Issue.Description,
						},
					},
				},
			})

			webhookEvents = append(webhookEvents, &webhook.Event{
				Actor:   user,
				Type:    base.EventTypeIssueUpdate,
				Comment: "",
				Issue:   webhook.NewIssue(issue),
				Project: webhook.NewProject(issue.Project),
				IssueUpdate: &webhook.EventIssueUpdate{
					Path: path,
				},
			})

		case "subscribers":
			var subscribers []*store.UserMessage
			for _, subscriber := range request.Issue.Subscribers {
				subscriberEmail, err := common.GetUserEmail(subscriber)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "failed to get user email from %v, error: %v", subscriber, err)
				}
				user, err := s.store.GetUserByEmail(ctx, subscriberEmail)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get user %v, error: %v", subscriberEmail, err)
				}
				if user == nil {
					return nil, status.Errorf(codes.NotFound, "user %v not found", subscriber)
				}
				subscribers = append(subscribers, user)
			}
			patch.Subscribers = &subscribers

		case "labels":
			if len(request.Issue.Labels) == 0 {
				patch.RemoveLabels = true
			} else {
				if patch.PayloadUpsert == nil {
					patch.PayloadUpsert = &storepb.IssuePayload{}
				}
				patch.PayloadUpsert.Labels = request.Issue.Labels
			}

			issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
				IssueUID: issue.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_IssueUpdate_{
						IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
							FromLabels: issue.Payload.Labels,
							ToLabels:   request.Issue.Labels,
						},
					},
				},
			})
		}
	}

	issue, err = s.store.UpdateIssueV2(ctx, issue.UID, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update issue, error: %v", err)
	}

	if updateMasks["approval_finding_done"] {
		s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
	}

	for _, e := range webhookEvents {
		s.webhookManager.CreateEvent(ctx, e)
	}
	for _, create := range issueCommentCreates {
		if _, err := s.store.CreateIssueComment(ctx, create, user.ID); err != nil {
			slog.Warn("failed to create issue comment", "issue id", issue.UID, log.BBError(err))
		}
	}

	issueV1, err := s.convertToIssue(ctx, issue)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to issue, error: %v", err)
	}
	return issueV1, nil
}

// BatchUpdateIssuesStatus batch updates issues status.
func (s *IssueService) BatchUpdateIssuesStatus(ctx context.Context, request *v1pb.BatchUpdateIssuesStatusRequest) (*v1pb.BatchUpdateIssuesStatusResponse, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	var issueIDs []int
	var issues []*store.IssueMessage
	for _, issueName := range request.Issues {
		issue, err := s.getIssueMessage(ctx, issueName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find issue %v, err: %v", issueName, err)
		}
		if issue == nil {
			return nil, status.Errorf(codes.NotFound, "cannot find issue %v", issueName)
		}
		issueIDs = append(issueIDs, issue.UID)
		issues = append(issues, issue)

		// Check if there is any running/pending task runs.
		if issue.PipelineUID != nil {
			taskRunStatusList := []base.TaskRunStatus{base.TaskRunRunning, base.TaskRunPending}
			taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{PipelineUID: issue.PipelineUID, Status: &taskRunStatusList})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to list task runs, err: %v", err)
			}
			if len(taskRuns) > 0 {
				return nil, status.Errorf(codes.FailedPrecondition, "cannot update status because there are running/pending task runs for issue %q", issueName)
			}
		}
	}

	if len(issueIDs) == 0 {
		return &v1pb.BatchUpdateIssuesStatusResponse{}, nil
	}

	newStatus, err := convertToAPIIssueStatus(request.Status)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert to issue status, err: %v", err)
	}

	if err := s.store.BatchUpdateIssueStatuses(ctx, issueIDs, newStatus); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch update issues, err: %v", err)
	}

	if err := func() error {
		var errs error
		for _, issue := range issues {
			updatedIssue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issue.UID})
			if err != nil {
				errs = multierr.Append(errs, errors.Wrapf(err, "failed to get issue %v", issue.UID))
				continue
			}

			func() {
				s.webhookManager.CreateEvent(ctx, &webhook.Event{
					Actor:   user,
					Type:    base.EventTypeIssueStatusUpdate,
					Comment: request.Reason,
					Issue:   webhook.NewIssue(updatedIssue),
					Project: webhook.NewProject(updatedIssue.Project),
				})
			}()

			func() {
				fromStatus := convertToIssueStatus(issue.Status)
				if _, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
					IssueUID: issue.UID,
					Payload: &storepb.IssueCommentPayload{
						Comment: request.Reason,
						Event: &storepb.IssueCommentPayload_IssueUpdate_{
							IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
								FromStatus: convertToIssueCommentPayloadIssueUpdateIssueStatus(&fromStatus),
								ToStatus:   convertToIssueCommentPayloadIssueUpdateIssueStatus(&request.Status),
							},
						},
					},
				}, user.ID); err != nil {
					errs = multierr.Append(errs, errors.Wrapf(err, "failed to create issue comment after change the issue status"))
					return
				}
			}()
		}
		return errs
	}(); err != nil {
		slog.Error("failed to create activity after changing the issue status", log.BBError(err))
	}

	return &v1pb.BatchUpdateIssuesStatusResponse{}, nil
}

func (s *IssueService) ListIssueComments(ctx context.Context, request *v1pb.ListIssueCommentsRequest) (*v1pb.ListIssueCommentsResponse, error) {
	if request.PageSize < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page size must be non-negative: %d", request.PageSize)
	}
	_, issueUID, err := common.GetProjectIDIssueUID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, err: %v", err)
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
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
		return nil, status.Errorf(codes.Internal, "failed to list issue comments, err: %v", err)
	}
	var nextPageToken string
	if len(issueComments) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		issueComments = issueComments[:offset.limit]
	}

	return &v1pb.ListIssueCommentsResponse{
		IssueComments: convertToIssueComments(request.Parent, issueComments),
		NextPageToken: nextPageToken,
	}, nil
}

// CreateIssueComment creates the issue comment.
func (s *IssueService) CreateIssueComment(ctx context.Context, request *v1pb.CreateIssueCommentRequest) (*v1pb.IssueComment, error) {
	if request.IssueComment.Comment == "" {
		return nil, status.Errorf(codes.InvalidArgument, "issue comment is empty")
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	issue, err := s.getIssueMessage(ctx, request.Parent)
	if err != nil {
		return nil, err
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   user,
		Type:    base.EventTypeIssueCommentCreate,
		Comment: request.IssueComment.Comment,
		Issue:   webhook.NewIssue(issue),
		Project: webhook.NewProject(issue.Project),
	})

	ic, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
		IssueUID: issue.UID,
		Payload: &storepb.IssueCommentPayload{
			Comment: request.IssueComment.Comment,
		},
	}, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue comment: %v", err)
	}

	// Add issue commenter to issue subscribers.
	hasSubscriber := false
	for _, subscriber := range issue.Subscribers {
		if subscriber.ID == user.ID {
			hasSubscriber = true
			break
		}
	}
	if !hasSubscriber {
		issue.Subscribers = append(issue.Subscribers, user)
		if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
			Subscribers: &issue.Subscribers,
		}); err != nil {
			return nil, err
		}
	}

	return convertToIssueComment(request.Parent, ic), nil
}

// UpdateIssueComment updates the issue comment.
func (s *IssueService) UpdateIssueComment(ctx context.Context, request *v1pb.UpdateIssueCommentRequest) (*v1pb.IssueComment, error) {
	if request.UpdateMask.Paths == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}

	_, _, issueCommentUID, err := common.GetProjectIDIssueUIDIssueCommentUID(request.IssueComment.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid comment name %q: %v", request.IssueComment.Name, err)
	}
	issueComment, err := s.store.GetIssueComment(ctx, &store.FindIssueCommentMessage{UID: &issueCommentUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue comment: %v", err)
	}
	if issueComment == nil {
		return nil, status.Errorf(codes.NotFound, "issue comment not found")
	}

	update := &store.UpdateIssueCommentMessage{
		UID: issueCommentUID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "comment":
			update.Comment = &request.IssueComment.Comment
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask: "%s"`, path)
		}
	}

	if err := s.store.UpdateIssueComment(ctx, update); err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return nil, status.Errorf(codes.NotFound, "cannot found the issue comment %s", request.IssueComment.Name)
		}
		return nil, status.Errorf(codes.Internal, "failed to update the issue comment with error: %v", err.Error())
	}
	issueComment, err = s.store.GetIssueComment(ctx, &store.FindIssueCommentMessage{UID: &issueCommentUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue comment: %v", err)
	}

	return convertToIssueComment(request.Parent, issueComment), nil
}

func (s *IssueService) getIssueMessage(ctx context.Context, name string) (*store.IssueMessage, error) {
	issueID, err := common.GetIssueID(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{UID: &issueID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get issue, error: %v", err)
	}
	if issue == nil {
		return nil, status.Errorf(codes.NotFound, "issue %d not found", issueID)
	}
	return issue, nil
}

func canRequestIssue(issueCreator *store.UserMessage, user *store.UserMessage) bool {
	return issueCreator.ID == user.ID
}

func isUserReviewer(ctx context.Context, stores *store.Store, issue *store.IssueMessage, step *storepb.ApprovalStep, user *store.UserMessage, policies ...*storepb.IamPolicy) (bool, error) {
	if len(step.Nodes) != 1 {
		return false, errors.Errorf("expecting one node but got %v", len(step.Nodes))
	}
	if step.Type != storepb.ApprovalStep_ANY {
		return false, errors.Errorf("expecting ANY step type but got %v", step.Type)
	}
	node := step.Nodes[0]
	if node.Type != storepb.ApprovalNode_ANY_IN_GROUP {
		return false, errors.Errorf("expecting ANY_IN_GROUP node type but got %v", node.Type)
	}

	// Check project policy about self approval.
	if !issue.Project.Setting.AllowSelfApproval && issue.Creator.ID == user.ID {
		return false, errors.Errorf("creator cannot self approve")
	}

	roles := utils.GetUserFormattedRolesMap(ctx, stores, user, policies...)
	return roles[node.Role], nil
}
