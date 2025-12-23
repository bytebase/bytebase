package v1

import (
	"context"
	"log/slog"
	"slices"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// PlanService represents a service for managing plan.
type PlanService struct {
	v1connect.UnimplementedPlanServiceHandler
	store          *store.Store
	sheetManager   *sheet.Manager
	licenseService *enterprise.LicenseService
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewPlanService returns a plan service instance.
func NewPlanService(store *store.Store, sheetManager *sheet.Manager, licenseService *enterprise.LicenseService, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile *config.Profile, iamManager *iam.Manager) *PlanService {
	return &PlanService{
		store:          store,
		sheetManager:   sheetManager,
		licenseService: licenseService,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// GetPlan gets a plan.
func (s *PlanService) GetPlan(ctx context.Context, request *connect.Request[v1pb.GetPlanRequest]) (*connect.Response[v1pb.Plan], error) {
	req := request.Msg
	projectID, planID, err := common.GetProjectIDPlanID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		UID:       &planID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan, error: %v", err))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", planID, projectID))
	}
	convertedPlan, err := convertToPlan(ctx, s.store, plan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to plan, error: %v", err))
	}
	return connect.NewResponse(convertedPlan), nil
}

// ListPlans lists plans.
func (s *PlanService) ListPlans(ctx context.Context, request *connect.Request[v1pb.ListPlansRequest]) (*connect.Response[v1pb.ListPlansResponse], error) {
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

	find := &store.FindPlanMessage{
		Limit:     &limitPlusOne,
		Offset:    &offset.offset,
		ProjectID: &projectID,
	}
	plans, err := s.store.ListPlans(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list plans, error: %v", err))
	}

	var nextPageToken string
	// has more pages
	if len(plans) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get next page token, error: %v", err))
		}
		plans = plans[:offset.limit]
	}

	convertedPlans, err := convertToPlans(ctx, s.store, plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to plans, error: %v", err))
	}

	return connect.NewResponse(&v1pb.ListPlansResponse{
		Plans:         convertedPlans,
		NextPageToken: nextPageToken,
	}), nil
}

// SearchPlans searches plans.
func (s *PlanService) SearchPlans(ctx context.Context, request *connect.Request[v1pb.SearchPlansRequest]) (*connect.Response[v1pb.SearchPlansResponse], error) {
	req := request.Msg
	projectID, err := common.GetProjectID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	projectIDsFilter, err := getProjectIDsSearchFilter(ctx, user, iam.PermissionPlansGet, s.iamManager, s.store)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get projectIDs, error: %v", err))
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

	find := &store.FindPlanMessage{
		Limit:      &limitPlusOne,
		Offset:     &offset.offset,
		ProjectIDs: projectIDsFilter,
	}
	if projectID != "-" {
		find.ProjectID = &projectID
	}
	filterQ, err := s.store.GetListPlanFilter(ctx, req.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	plans, err := s.store.ListPlans(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list plans, error: %v", err))
	}

	var nextPageToken string
	// has more pages
	if len(plans) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get next page token, error: %v", err))
		}
		plans = plans[:offset.limit]
	}

	convertedPlans, err := convertToPlans(ctx, s.store, plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to plans, error: %v", err))
	}

	return connect.NewResponse(&v1pb.SearchPlansResponse{
		Plans:         convertedPlans,
		NextPageToken: nextPageToken,
	}), nil
}

func getProjectIDsSearchFilter(ctx context.Context, user *store.UserMessage, permission iam.Permission, iamManager *iam.Manager, stores *store.Store) (*[]string, error) {
	ok, err := iamManager.CheckPermission(ctx, permission, user)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check permission %q", permission)
	}
	if ok {
		return nil, nil
	}
	projects, err := stores.ListProjects(ctx, &store.FindProjectMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list projects")
	}

	var projectIDs []string
	for _, project := range projects {
		ok, err := iamManager.CheckPermission(ctx, permission, user, project.ResourceID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check permission %q", permission)
		}
		if ok {
			projectIDs = append(projectIDs, project.ResourceID)
		}
	}
	return &projectIDs, nil
}

// CreatePlan creates a new plan.
func (s *PlanService) CreatePlan(ctx context.Context, request *connect.Request[v1pb.CreatePlanRequest]) (*connect.Response[v1pb.Plan], error) {
	req := request.Msg
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}
	projectID, err := common.GetProjectID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}

	// Validate plan specs
	databaseGroup, err := validateSpecs(ctx, s.store, projectID, req.Plan.Specs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to validate plan specs, error: %v", err))
	}

	planMessage := &store.PlanMessage{
		ProjectID:   projectID,
		PipelineUID: nil,
		Name:        req.Plan.Title,
		Description: req.Plan.Description,
		Config:      convertPlan(req.Plan),
	}

	plan, err := s.store.CreatePlan(ctx, planMessage, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan, error: %v", err))
	}

	// Don't create plan checks if the plan comes from releases.
	planCheckRuns, err := getPlanCheckRunsFromPlan(project, plan, databaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for plan, error: %v", err))
	}
	if err := s.store.CreatePlanCheckRuns(ctx, plan, planCheckRuns...); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan check runs, error: %v", err))
	}
	// Tickle plan check scheduler.
	s.stateCfg.PlanCheckTickleChan <- 0

	convertedPlan, err := convertToPlan(ctx, s.store, plan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to plan, error: %v", err))
	}
	return connect.NewResponse(convertedPlan), nil
}

// UpdatePlan updates a plan.
func (s *PlanService) UpdatePlan(ctx context.Context, request *connect.Request[v1pb.UpdatePlanRequest]) (*connect.Response[v1pb.Plan], error) {
	req := request.Msg
	if req.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}
	if len(req.UpdateMask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must not be empty"))
	}
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	projectID, planID, err := common.GetProjectIDPlanID(req.Plan.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project %q, err: %v", projectID, err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}
	oldPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: &projectID, UID: &planID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan %q: %v", req.Plan.Name, err))
	}
	if oldPlan == nil {
		if req.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionPlansCreate, user, projectID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionPlansCreate))
			}
			return s.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
				Parent: common.FormatProject(projectID),
				Plan:   req.Plan,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %q not found", req.Plan.Name))
	}

	if storePlanConfigHasRelease(oldPlan.Config) && slices.Contains(req.UpdateMask.Paths, "specs") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("disallowed to update the plan specs because the plan is created from a release"))
	}

	ok, err = func() (bool, error) {
		if oldPlan.Creator == user.Email {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionPlansUpdate, user, oldPlan.ProjectID)
	}()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission, error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied to update plan"))
	}

	planUpdate := &store.UpdatePlanMessage{
		UID: oldPlan.UID,
	}

	var planCheckRunsTrigger bool
	var databaseGroup *v1pb.DatabaseGroup

	for _, path := range req.UpdateMask.Paths {
		switch path {
		case "title":
			title := req.Plan.Title
			planUpdate.Name = &title
		case "description":
			description := req.Plan.Description
			planUpdate.Description = &description
		case "state":
			deleted := req.Plan.State == v1pb.State_DELETED
			planUpdate.Deleted = &deleted
		case "specs":
			// Block all spec changes if plan has a rollout (pipeline).
			if oldPlan.PipelineUID != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot update specs for plan that has a rollout"))
			}

			// Validate the new specs.
			dg, err := validateSpecs(ctx, s.store, oldPlan.ProjectID, req.Plan.Specs)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to validate plan specs, error: %v", err))
			}
			databaseGroup = dg

			// Convert and store new specs.
			allSpecs := convertPlanSpecs(req.GetPlan().GetSpecs())
			planUpdate.Specs = &allSpecs

			// Trigger plan check runs.
			planCheckRunsTrigger = true

			// Evict approvals if issue exists to request re-approval.
			issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &oldPlan.UID})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue: %v", err))
			}
			if issue != nil {
				if _, err := s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
					PayloadUpsert: &storepb.Issue{
						Approval: &storepb.IssuePayloadApproval{
							ApprovalFindingDone: false,
						},
					},
				}); err != nil {
					slog.Error("failed to update issue to refind approval", log.BBError(err))
				} else {
					s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
				}
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update_mask path %q", path))
		}
	}

	if err := s.store.UpdatePlan(ctx, planUpdate); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update plan %q: %v", req.Plan.Name, err))
	}

	updatedPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		UID:       &oldPlan.UID,
		ProjectID: &oldPlan.ProjectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get updated plan %q: %v", req.Plan.Name, err))
	}
	if updatedPlan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("updated plan %q not found", req.Plan.Name))
	}

	if planCheckRunsTrigger {
		planCheckRuns, err := getPlanCheckRunsFromPlan(project, updatedPlan, databaseGroup)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for plan, error: %v", err))
		}
		if err := s.store.CreatePlanCheckRuns(ctx, updatedPlan, planCheckRuns...); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan check runs, error: %v", err))
		}
		// Tickle plan check scheduler.
		s.stateCfg.PlanCheckTickleChan <- 0
	}

	convertedPlan, err := convertToPlan(ctx, s.store, updatedPlan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert to plan, error: %v", err))
	}
	return connect.NewResponse(convertedPlan), nil
}

// ListPlanCheckRuns lists plan check runs for the plan.
func (s *PlanService) ListPlanCheckRuns(ctx context.Context, request *connect.Request[v1pb.ListPlanCheckRunsRequest]) (*connect.Response[v1pb.ListPlanCheckRunsResponse], error) {
	req := request.Msg
	projectID, planUID, err := common.GetProjectIDPlanID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindPlanCheckRunMessage{
		PlanUID: &planUID,
	}
	// Parse filter if provided
	if req.Filter != "" {
		if err := s.parsePlanCheckRunFilter(req.Filter, find); err != nil {
			return nil, err
		}
	}
	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list plan check runs, error: %v", err))
	}
	converted := convertToPlanCheckRuns(projectID, planUID, planCheckRuns)

	return connect.NewResponse(&v1pb.ListPlanCheckRunsResponse{
		PlanCheckRuns: converted,
	}), nil
}

// parsePlanCheckRunFilter parses the filter for plan check runs.
func (*PlanService) parsePlanCheckRunFilter(filter string, find *store.FindPlanCheckRunMessage) error {
	if filter == "" {
		return nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var parseFilter func(expr celast.Expr) error
	parseFilter = func(expr celast.Expr) error {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalAnd:
				// Handle AND operator by recursively parsing left and right expressions
				for _, arg := range expr.AsCall().Args() {
					if err := parseFilter(arg); err != nil {
						return err
					}
				}
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "status":
					statusStr, ok := value.(string)
					if !ok {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("status value must be a string"))
					}
					// Convert v1pb status to store status
					v1Status := v1pb.PlanCheckRun_Status_value[statusStr]
					if v1Status == 0 && statusStr != "STATUS_UNSPECIFIED" {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid status value: %s", statusStr))
					}
					storeStatus := convertToStorePlanCheckRunStatus(v1pb.PlanCheckRun_Status(v1Status))
					if find.Status == nil {
						find.Status = &[]store.PlanCheckRunStatus{}
					}
					*find.Status = append(*find.Status, storeStatus)
				case "result_status":
					resultStatusStr, ok := value.(string)
					if !ok {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("result_status value must be a string"))
					}
					// Convert v1pb result status to store result status
					v1ResultStatus := v1pb.Advice_Level_value[resultStatusStr]
					if v1ResultStatus == 0 && resultStatusStr != "STATUS_UNSPECIFIED" {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid result_status value: %s", resultStatusStr))
					}
					storeResultStatus := convertToStoreResultStatus(v1pb.Advice_Level(v1ResultStatus))
					if find.ResultStatus == nil {
						find.ResultStatus = &[]storepb.Advice_Status{}
					}
					*find.ResultStatus = append(*find.ResultStatus, storeResultStatus)
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported filter variable: %s", variable))
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "status":
					rawList, ok := value.([]any)
					if !ok {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid list value %q for %v", value, variable))
					}
					if len(rawList) == 0 {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty list value for filter %v", variable))
					}
					if find.Status == nil {
						find.Status = &[]store.PlanCheckRunStatus{}
					}
					for _, raw := range rawList {
						statusStr, ok := raw.(string)
						if !ok {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("status value must be a string"))
						}
						// Convert v1pb status to store status
						v1Status := v1pb.PlanCheckRun_Status_value[statusStr]
						if v1Status == 0 && statusStr != "STATUS_UNSPECIFIED" {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid status value: %s", statusStr))
						}
						storeStatus := convertToStorePlanCheckRunStatus(v1pb.PlanCheckRun_Status(v1Status))
						*find.Status = append(*find.Status, storeStatus)
					}
				case "result_status":
					rawList, ok := value.([]any)
					if !ok {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid list value %q for %v", value, variable))
					}
					if len(rawList) == 0 {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty list value for filter %v", variable))
					}
					if find.ResultStatus == nil {
						find.ResultStatus = &[]storepb.Advice_Status{}
					}
					for _, raw := range rawList {
						resultStatusStr, ok := raw.(string)
						if !ok {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("result_status value must be a string"))
						}
						// Convert v1pb result status to store result status
						v1ResultStatus := v1pb.Advice_Level_value[resultStatusStr]
						if v1ResultStatus == 0 && resultStatusStr != "STATUS_UNSPECIFIED" {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid result_status value: %s", resultStatusStr))
						}
						storeResultStatus := convertToStoreResultStatus(v1pb.Advice_Level(v1ResultStatus))
						*find.ResultStatus = append(*find.ResultStatus, storeResultStatus)
					}
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported filter variable: %s", variable))
				}
			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported operator: %s", functionName))
			}
		default:
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid filter expression"))
		}
		return nil
	}

	return parseFilter(ast.NativeRep().Expr())
}

// convertToStorePlanCheckRunStatus converts v1pb.PlanCheckRun_Status to store.PlanCheckRunStatus.
func convertToStorePlanCheckRunStatus(status v1pb.PlanCheckRun_Status) store.PlanCheckRunStatus {
	switch status {
	case v1pb.PlanCheckRun_CANCELED:
		return store.PlanCheckRunStatusCanceled
	case v1pb.PlanCheckRun_DONE:
		return store.PlanCheckRunStatusDone
	case v1pb.PlanCheckRun_FAILED:
		return store.PlanCheckRunStatusFailed
	case v1pb.PlanCheckRun_RUNNING:
		return store.PlanCheckRunStatusRunning
	default:
		return store.PlanCheckRunStatusRunning
	}
}

// convertToStoreResultStatus converts v1pb.Advice_Status to storepb.Advice_Status.
func convertToStoreResultStatus(status v1pb.Advice_Level) storepb.Advice_Status {
	switch status {
	case v1pb.Advice_ERROR:
		return storepb.Advice_ERROR
	case v1pb.Advice_WARNING:
		return storepb.Advice_WARNING
	case v1pb.Advice_SUCCESS:
		return storepb.Advice_SUCCESS
	default:
		return storepb.Advice_STATUS_UNSPECIFIED
	}
}

// RunPlanChecks runs plan checks for a plan.
func (s *PlanService) RunPlanChecks(ctx context.Context, request *connect.Request[v1pb.RunPlanChecksRequest]) (*connect.Response[v1pb.RunPlanChecksResponse], error) {
	req := request.Msg
	projectID, planID, err := common.GetProjectIDPlanID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		UID:       &planID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan, error: %v", err))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan not found"))
	}
	if storePlanConfigHasRelease(plan.Config) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot run plan checks because plan %q has release", plan.Name))
	}
	var databaseGroup *v1pb.DatabaseGroup
	for _, spec := range plan.Config.GetSpecs() {
		if c, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig); ok {
			if len(c.ChangeDatabaseConfig.Targets) == 1 {
				if _, _, err := common.GetProjectIDDatabaseGroupID(c.ChangeDatabaseConfig.Targets[0]); err == nil {
					dg, err := getDatabaseGroupByName(ctx, s.store, c.ChangeDatabaseConfig.Targets[0], v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_BASIC)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database group %q: %v", c.ChangeDatabaseConfig.Targets[0], err))
					}
					if dg == nil {
						return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database group %q not found", c.ChangeDatabaseConfig.Targets[0]))
					}
					databaseGroup = dg
					break
				}
			}
		}
	}
	planCheckRuns, err := getPlanCheckRunsFromPlan(project, plan, databaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for plan, error: %v", err))
	}
	if err := s.store.CreatePlanCheckRuns(ctx, plan, planCheckRuns...); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan check runs, error: %v", err))
	}

	// Tickle plan check scheduler.
	s.stateCfg.PlanCheckTickleChan <- 0

	return connect.NewResponse(&v1pb.RunPlanChecksResponse{}), nil
}

// BatchCancelPlanCheckRuns cancels a list of plan check runs.
func (s *PlanService) BatchCancelPlanCheckRuns(ctx context.Context, request *connect.Request[v1pb.BatchCancelPlanCheckRunsRequest]) (*connect.Response[v1pb.BatchCancelPlanCheckRunsResponse], error) {
	req := request.Msg
	if len(req.PlanCheckRuns) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("plan check runs cannot be empty"))
	}

	projectID, planID, err := common.GetProjectIDPlanID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	var planCheckRunIDs []int
	for _, planCheckRun := range req.PlanCheckRuns {
		_, _, planCheckRunID, err := common.GetProjectIDPlanIDPlanCheckRunID(planCheckRun)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		planCheckRunIDs = append(planCheckRunIDs, planCheckRunID)
	}

	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		UIDs: &planCheckRunIDs,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list plan check runs, error: %v", err))
	}

	// Validate that all plan check runs belong to the plan specified in the parent.
	for _, planCheckRun := range planCheckRuns {
		if planCheckRun.PlanUID != planID {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan check run %d not found in plan %d", planCheckRun.UID, planID))
		}
	}

	// Check if any of the given plan check runs are not running.
	for _, planCheckRun := range planCheckRuns {
		switch planCheckRun.Status {
		case store.PlanCheckRunStatusRunning:
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("planCheckRun %v(%v) is not running", planCheckRun.UID, planCheckRun.Type))
		}
	}
	// Cancel the plan check runs.
	for _, planCheckRun := range planCheckRuns {
		if cancelFunc, ok := s.stateCfg.RunningPlanCheckRunsCancelFunc.Load(planCheckRun.UID); ok {
			cancelFunc.(context.CancelFunc)()
		}
	}
	// Update the status of the plan check runs to canceled.
	if err := s.store.BatchCancelPlanCheckRuns(ctx, planCheckRunIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to batch patch task run status to canceled, error: %v", err))
	}

	return connect.NewResponse(&v1pb.BatchCancelPlanCheckRunsResponse{}), nil
}

func validateSpecs(ctx context.Context, s *store.Store, projectID string, specs []*v1pb.Plan_Spec) (*v1pb.DatabaseGroup, error) {
	if len(specs) == 0 {
		return nil, errors.Errorf("the plan has zero spec")
	}
	configTypeCount := map[string]int{}
	seenID := map[string]bool{}

	var releaseCount, sheetCount int
	var sheetSha256s []string
	var releaseString string
	var instanceIDs []string
	var databaseGroups []string
	var databaseNames []string
	var databaseGroup *v1pb.DatabaseGroup

	for _, spec := range specs {
		id := spec.GetId()
		if id == "" {
			return nil, errors.Errorf("spec id cannot be empty")
		}
		if seenID[id] {
			return nil, errors.Errorf("found duplicate spec id %v", id)
		}
		seenID[id] = true

		switch config := spec.Config.(type) {
		case *v1pb.Plan_Spec_CreateDatabaseConfig:
			configTypeCount["create_database"]++
			if target := config.CreateDatabaseConfig.Target; target != "" {
				instanceID, err := common.GetInstanceID(target)
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid instance name %q: %v", target, err))
				}
				instanceIDs = append(instanceIDs, instanceID)
			}
		case *v1pb.Plan_Spec_ChangeDatabaseConfig:
			configTypeCount["change_database"]++
			var databaseTarget, databaseGroupTarget int
			for _, target := range config.ChangeDatabaseConfig.Targets {
				if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
					databaseTarget++
					databaseNames = append(databaseNames, target)
				} else if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
					databaseGroupTarget++
					databaseGroups = append(databaseGroups, target)
				} else {
					return nil, errors.Errorf("invalid target %v", target)
				}
			}
			// Disallow mixing database and database group targets in the same spec.
			if databaseTarget > 0 && databaseGroupTarget > 0 {
				return nil, errors.Errorf("found databaseTarget and databaseGroupTarget, expect only one kind")
			}
			// Track if this spec uses release or sheet.
			if config.ChangeDatabaseConfig.Release != "" {
				releaseCount++
				releaseString = config.ChangeDatabaseConfig.Release
			}
			if config.ChangeDatabaseConfig.Sheet != "" {
				sheetCount++
				if _, sha, err := common.GetProjectResourceIDSheetSha256(config.ChangeDatabaseConfig.Sheet); err == nil {
					sheetSha256s = append(sheetSha256s, sha)
				}
			}
		case *v1pb.Plan_Spec_ExportDataConfig:
			configTypeCount["export_data"]++
			databaseNames = append(databaseNames, config.ExportDataConfig.Targets...)
			if config.ExportDataConfig.Sheet != "" {
				if _, sha, err := common.GetProjectResourceIDSheetSha256(config.ExportDataConfig.Sheet); err == nil {
					sheetSha256s = append(sheetSha256s, sha)
				}
			}
		default:
			return nil, errors.Errorf("invalid spec type")
		}
	}
	if len(configTypeCount) > 1 {
		return nil, errors.Errorf("plan contains multiple types of spec configurations (%v), but each plan must contain only one type", len(configTypeCount))
	}
	// Disallow mixing ChangeDatabaseConfig specs with release and sheet.
	if releaseCount > 0 && sheetCount > 0 {
		return nil, errors.Errorf("plan contains both release and sheet based change database configs, but each plan must use only one approach")
	}
	// Allow at most one ChangeDatabaseConfig with release.
	if releaseCount > 1 {
		return nil, errors.Errorf("plan contains multiple change database configs with release, but only one is allowed")
	}

	// Allow at most one instance.
	if len(instanceIDs) > 1 {
		return nil, errors.Errorf("plan contains targets on multiple instances, but only one instance is allowed")
	}

	// Allow at most one database group.
	if len(databaseGroups) > 1 {
		return nil, errors.Errorf("plan contains multiple database groups, but only one is allowed")
	}

	// Don't allow mixing database group and databases.
	if len(databaseGroups) > 0 && len(databaseNames) > 0 {
		return nil, errors.Errorf("plan contains both database group and databases, but only one is allowed")
	}

	// Validate resources existence.
	if len(instanceIDs) == 1 {
		instanceID := instanceIDs[0]
		instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{
			ResourceID: &instanceID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get instance %q: %v", instanceID, err))
		}
		if instance == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
		}
	}

	if len(databaseGroups) == 1 {
		name := databaseGroups[0]
		groupProjectID, _, err := common.GetProjectIDDatabaseGroupID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid database group name %q", name))
		}
		if groupProjectID != projectID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database group %q (project %q) does not belong to plan project %q", name, groupProjectID, projectID))
		}

		dg, err := getDatabaseGroupByName(ctx, s, name, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_BASIC)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database group %q: %v", name, err))
		}
		if dg == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database group %q not found", name))
		}
		databaseGroup = dg
	}

	for _, name := range databaseNames {
		instanceID, dbName, err := common.GetInstanceDatabaseID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid database name %q", name))
		}
		db, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &dbName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get database %q: %v", name, err))
		}
		if db == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", name))
		}

		if db.ProjectID != projectID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database %q (project %q) does not belong to plan project %q", name, db.ProjectID, projectID))
		}
	}

	// Validate sheets existence.
	if len(sheetSha256s) > 0 {
		exist, err := s.HasSheets(ctx, sheetSha256s...)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check sheets: %v", err))
		}
		if !exist {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("some sheets are not found"))
		}
	}

	// Validate release existence.
	if releaseString != "" {
		releaseProjectID, releaseUID, err := common.GetProjectReleaseUID(releaseString)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid release name %q", releaseString))
		}
		if releaseProjectID != projectID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("release %q (project %q) does not belong to plan project %q", releaseString, releaseProjectID, projectID))
		}
		release, err := s.GetReleaseByUID(ctx, releaseUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get release %d: %v", releaseUID, err))
		}
		if release == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %d not found", releaseUID))
		}
	}
	return databaseGroup, nil
}

func storePlanConfigHasRelease(plan *storepb.PlanConfig) bool {
	for _, spec := range plan.GetSpecs() {
		if c, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig); ok {
			if c.ChangeDatabaseConfig.Release != "" {
				return true
			}
		}
	}
	return false
}
