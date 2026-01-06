package v1

import (
	"context"
	"log/slog"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/runner/plancheck"

	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/webhook"
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
	bus            *bus.Bus
	iamManager     *iam.Manager
	webhookManager *webhook.Manager
	licenseService *enterprise.LicenseService
}

// NewPlanService returns a plan service instance.
func NewPlanService(store *store.Store, bus *bus.Bus, iamManager *iam.Manager, webhookManager *webhook.Manager, licenseService *enterprise.LicenseService) *PlanService {
	return &PlanService{
		store:          store,
		bus:            bus,
		iamManager:     iamManager,
		webhookManager: webhookManager,
		licenseService: licenseService,
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %d not found in project %s", planID, projectID))
	}
	convertedPlan, err := convertToPlan(ctx, s.store, plan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to plan"))
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

	if req.Filter != "" {
		filterQ, err := store.GetListPlanFilter(req.Filter)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		find.FilterQ = filterQ
	}

	plans, err := s.store.ListPlans(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list plans"))
	}

	var nextPageToken string
	// has more pages
	if len(plans) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		plans = plans[:offset.limit]
	}

	convertedPlans, err := convertToPlans(ctx, s.store, plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to plans"))
	}

	return connect.NewResponse(&v1pb.ListPlansResponse{
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}

	// Validate plan specs
	databaseGroup, err := validateSpecs(ctx, s.store, projectID, req.Plan.Specs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to validate plan specs"))
	}

	planMessage := &store.PlanMessage{
		ProjectID:   projectID,
		Name:        req.Plan.Title,
		Description: req.Plan.Description,
		Config:      convertPlan(req.Plan),
	}

	plan, err := s.store.CreatePlan(ctx, planMessage, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create plan"))
	}

	planCheckRun, err := getPlanCheckRunFromPlan(project, plan, databaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan check run for plan"))
	}
	if planCheckRun != nil {
		if err := s.store.CreatePlanCheckRun(ctx, planCheckRun); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create plan check run"))
		}
	}
	// Tickle plan check scheduler.
	s.bus.PlanCheckTickleChan <- 0

	convertedPlan, err := convertToPlan(ctx, s.store, plan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to plan"))
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

	// Disallow updating CREATE_DATABASE plan specs
	if storePlanConfigHasCreateDatabase(oldPlan.Config) && slices.Contains(req.UpdateMask.Paths, "specs") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("disallowed to update the plan specs for CREATE_DATABASE plans"))
	}

	ok, err = func() (bool, error) {
		if oldPlan.Creator == user.Email {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionPlansUpdate, user, oldPlan.ProjectID)
	}()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check permission"))
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
			// Block all spec changes if plan has a rollout (pipeline).
			if oldPlan.Config != nil && oldPlan.Config.GetHasRollout() {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot update specs for plan that has a rollout"))
			}

			// Validate the new specs.
			dg, err := validateSpecs(ctx, s.store, oldPlan.ProjectID, req.Plan.Specs)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to validate plan specs"))
			}
			databaseGroup = dg

			// Convert and store new specs.
			allSpecs := convertPlanSpecs(req.GetPlan().GetSpecs())
			config := proto.CloneOf(oldPlan.Config)
			config.Specs = allSpecs
			planUpdate.Config = config

			// Trigger plan check runs.
			planCheckRunsTrigger = true

			// Evict approvals if issue exists to request re-approval.
			issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &oldPlan.UID})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue: %v", err))
			}
			if issue != nil {
				// Reset approval finding status
				updatedIssue, err := s.store.UpdateIssue(ctx, issue.UID, &store.UpdateIssueMessage{
					PayloadUpsert: &storepb.Issue{
						Approval: &storepb.IssuePayloadApproval{
							ApprovalFindingDone: false,
						},
					},
				})
				if err != nil {
					slog.Error("failed to reset approval finding status after plan update", log.BBError(err))
				}

				// DATABASE_CHANGE: Don't trigger ApprovalCheckChan here - plan update creates new plan check run,
				// which will trigger approval finding on completion
				// DATABASE_EXPORT: Re-run approval finding synchronously (no plan checks for export data)
				if updatedIssue.Type == storepb.Issue_DATABASE_EXPORT {
					if err := approval.FindAndApplyApprovalTemplate(ctx, s.store, s.webhookManager, s.licenseService, updatedIssue); err != nil {
						slog.Error("failed to find approval template after plan update",
							slog.Int("issue_uid", updatedIssue.UID),
							slog.String("issue_title", updatedIssue.Title),
							log.BBError(err))
						// Continue anyway - non-fatal error
					}
				}
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update_mask path %q", path))
		}
	}

	updatedPlan, err := s.store.UpdatePlan(ctx, planUpdate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update plan %q: %v", req.Plan.Name, err))
	}

	if planCheckRunsTrigger {
		planCheckRun, err := getPlanCheckRunFromPlan(project, updatedPlan, databaseGroup)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan check run for plan"))
		}
		if planCheckRun != nil {
			if err := s.store.CreatePlanCheckRun(ctx, planCheckRun); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create plan check run"))
			}
		}
		// Tickle plan check scheduler.
		s.bus.PlanCheckTickleChan <- 0
	}

	convertedPlan, err := convertToPlan(ctx, s.store, updatedPlan)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to plan"))
	}
	return connect.NewResponse(convertedPlan), nil
}

// GetPlanCheckRun gets the plan check run for the plan.
func (s *PlanService) GetPlanCheckRun(ctx context.Context, request *connect.Request[v1pb.GetPlanCheckRunRequest]) (*connect.Response[v1pb.PlanCheckRun], error) {
	req := request.Msg
	projectID, planUID, err := common.GetProjectIDPlanIDFromPlanCheckRun(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	planCheckRun, err := s.store.GetPlanCheckRun(ctx, planUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan check run"))
	}
	if planCheckRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan check run not found for plan %d", planUID))
	}

	converted := convertToPlanCheckRun(projectID, planUID, planCheckRun)
	return connect.NewResponse(converted), nil
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{
		UID:       &planID,
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan"))
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
	planCheckRun, err := getPlanCheckRunFromPlan(project, plan, databaseGroup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan check run for plan"))
	}
	if planCheckRun != nil {
		if err := s.store.CreatePlanCheckRun(ctx, planCheckRun); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create plan check run"))
		}
	}

	// Tickle plan check scheduler.
	s.bus.PlanCheckTickleChan <- 0

	return connect.NewResponse(&v1pb.RunPlanChecksResponse{}), nil
}

// CancelPlanCheckRun cancels the plan check run for a plan.
func (s *PlanService) CancelPlanCheckRun(ctx context.Context, request *connect.Request[v1pb.CancelPlanCheckRunRequest]) (*connect.Response[v1pb.CancelPlanCheckRunResponse], error) {
	req := request.Msg
	projectID, planUID, err := common.GetProjectIDPlanIDFromPlanCheckRun(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %v not found", projectID))
	}

	planCheckRun, err := s.store.GetPlanCheckRun(ctx, planUID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get plan check run"))
	}
	if planCheckRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan check run not found for plan %d", planUID))
	}

	if planCheckRun.Status != store.PlanCheckRunStatusRunning && planCheckRun.Status != store.PlanCheckRunStatusAvailable {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("plan check run is not running or available"))
	}

	// Cancel in-flight plan check run if running.
	if cancelFunc, ok := s.bus.RunningPlanCheckRunsCancelFunc.Load(planCheckRun.UID); ok {
		cancelFunc.(context.CancelFunc)()
	}

	// Broadcast cancel signal to all replicas for HA.
	if err := s.store.SendSignal(ctx, storepb.Signal_CANCEL_PLAN_CHECK_RUN, int32(planCheckRun.UID)); err != nil {
		slog.Warn("failed to send cancel signal", log.BBError(err))
	}

	// Update the status to canceled.
	if err := s.store.BatchCancelPlanCheckRuns(ctx, []int{planCheckRun.UID}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to cancel plan check run"))
	}

	return connect.NewResponse(&v1pb.CancelPlanCheckRunResponse{}), nil
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
			for _, target := range config.ExportDataConfig.Targets {
				if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
					databaseNames = append(databaseNames, target)
				} else if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
					databaseGroups = append(databaseGroups, target)
				} else {
					return nil, errors.Errorf("invalid target %v", target)
				}
			}
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

func storePlanConfigHasCreateDatabase(plan *storepb.PlanConfig) bool {
	for _, spec := range plan.GetSpecs() {
		if _, ok := spec.Config.(*storepb.PlanConfig_Spec_CreateDatabaseConfig); ok {
			return true
		}
	}
	return false
}

// Converters section - ordered with callers before callees.

// getPlanCheckRunFromPlan returns the plan check run for a plan.
func getPlanCheckRunFromPlan(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) (*store.PlanCheckRunMessage, error) {
	targets, err := plancheck.DeriveCheckTargets(project, plan, databaseGroup)
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
	}, nil
}

func convertToPlans(ctx context.Context, s *store.Store, plans []*store.PlanMessage) ([]*v1pb.Plan, error) {
	v1Plans := make([]*v1pb.Plan, len(plans))
	for i := range plans {
		p, err := convertToPlan(ctx, s, plans[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert plan")
		}
		v1Plans[i] = p
	}
	return v1Plans, nil
}

func convertToPlan(ctx context.Context, s *store.Store, plan *store.PlanMessage) (*v1pb.Plan, error) {
	p := &v1pb.Plan{
		Name:                    common.FormatPlan(plan.ProjectID, plan.UID),
		Title:                   plan.Name,
		Description:             plan.Description,
		Creator:                 common.FormatUserEmail(plan.Creator),
		Specs:                   convertToPlanSpecs(plan.ProjectID, plan.Config.Specs), // Use specs field for output
		CreateTime:              timestamppb.New(plan.CreatedAt),
		UpdateTime:              timestamppb.New(plan.UpdatedAt),
		State:                   convertDeletedToState(plan.Deleted),
		PlanCheckRunStatusCount: map[string]int32{},
	}

	issue, err := s.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &plan.UID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get issue by plan uid %d", plan.UID)
	}
	if issue != nil {
		p.Issue = common.FormatIssue(issue.ProjectID, issue.UID)
	}
	if plan.Config != nil {
		p.HasRollout = plan.Config.HasRollout
	}
	planCheckRun, err := s.GetPlanCheckRun(ctx, int64(plan.UID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan check run for plan uid %d", plan.UID)
	}
	if planCheckRun != nil {
		p.PlanCheckRunStatusCount[string(planCheckRun.Status)]++
		for _, result := range planCheckRun.Result.Results {
			p.PlanCheckRunStatusCount[storepb.Advice_Status_name[int32(result.Status)]]++
		}
	}
	return p, nil
}

func convertPlan(plan *v1pb.Plan) *storepb.PlanConfig {
	if plan == nil {
		return nil
	}

	// At this point, plan.Specs should always be populated
	// (either originally or converted from steps at API entry point)
	return &storepb.PlanConfig{
		Specs: convertPlanSpecs(plan.Specs),
	}
}

func convertToPlanCheckRun(projectID string, planUID int64, run *store.PlanCheckRunMessage) *v1pb.PlanCheckRun {
	return &v1pb.PlanCheckRun{
		Name:       common.FormatPlanCheckRun(projectID, planUID),
		Status:     convertToPlanCheckRunStatus(run.Status),
		Results:    convertToPlanCheckRunResults(run.Result.GetResults()),
		Error:      run.Result.Error,
		CreateTime: timestamppb.New(run.CreatedAt),
	}
}

func convertToPlanSpecs(projectID string, specs []*storepb.PlanConfig_Spec) []*v1pb.Plan_Spec {
	v1Specs := make([]*v1pb.Plan_Spec, len(specs))
	for i := range specs {
		v1Specs[i] = convertToPlanSpec(projectID, specs[i])
	}
	return v1Specs
}

func convertToPlanSpec(projectID string, spec *storepb.PlanConfig_Spec) *v1pb.Plan_Spec {
	v1Spec := &v1pb.Plan_Spec{
		Id: spec.Id,
	}

	switch v := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		v1Spec.Config = convertToPlanSpecCreateDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		v1Spec.Config = convertToPlanSpecChangeDatabaseConfig(projectID, v)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		v1Spec.Config = convertToPlanSpecExportDataConfig(projectID, v)
	}

	return v1Spec
}

func convertToPlanSpecCreateDatabaseConfig(config *storepb.PlanConfig_Spec_CreateDatabaseConfig) *v1pb.Plan_Spec_CreateDatabaseConfig {
	c := config.CreateDatabaseConfig
	return &v1pb.Plan_Spec_CreateDatabaseConfig{
		CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
			Target:       c.Target,
			Database:     c.Database,
			Table:        c.Table,
			CharacterSet: c.CharacterSet,
			Collation:    c.Collation,
			Cluster:      c.Cluster,
			Owner:        c.Owner,
			Environment:  c.Environment,
		},
	}
}

func convertToPlanSpecChangeDatabaseConfig(projectID string, config *storepb.PlanConfig_Spec_ChangeDatabaseConfig) *v1pb.Plan_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig

	// Only format sheet if SheetSha256 is not empty (for non-release tasks)
	var sheet string
	if c.SheetSha256 != "" {
		sheet = common.FormatSheet(projectID, c.SheetSha256)
	}

	return &v1pb.Plan_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
			Targets:           c.Targets,
			Sheet:             sheet,
			Release:           c.Release,
			GhostFlags:        c.GhostFlags,
			EnablePriorBackup: c.EnablePriorBackup,
			EnableGhost:       c.EnableGhost,
		},
	}
}

func convertToPlanSpecExportDataConfig(projectID string, config *storepb.PlanConfig_Spec_ExportDataConfig) *v1pb.Plan_Spec_ExportDataConfig {
	c := config.ExportDataConfig
	return &v1pb.Plan_Spec_ExportDataConfig{
		ExportDataConfig: &v1pb.Plan_ExportDataConfig{
			Targets:  c.Targets,
			Sheet:    common.FormatSheet(projectID, c.SheetSha256),
			Format:   convertExportFormat(c.Format),
			Password: c.Password,
		},
	}
}

func convertPlanSpecs(specs []*v1pb.Plan_Spec) []*storepb.PlanConfig_Spec {
	storeSpecs := make([]*storepb.PlanConfig_Spec, len(specs))
	for i := range specs {
		storeSpecs[i] = convertPlanSpec(specs[i])
	}
	return storeSpecs
}

func convertPlanSpec(spec *v1pb.Plan_Spec) *storepb.PlanConfig_Spec {
	storeSpec := &storepb.PlanConfig_Spec{
		Id: spec.Id,
	}

	switch v := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		storeSpec.Config = convertPlanSpecCreateDatabaseConfig(v)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		storeSpec.Config = convertPlanSpecChangeDatabaseConfig(v)
	case *v1pb.Plan_Spec_ExportDataConfig:
		storeSpec.Config = convertPlanSpecExportDataConfig(v)
	}
	return storeSpec
}

func convertPlanSpecCreateDatabaseConfig(config *v1pb.Plan_Spec_CreateDatabaseConfig) *storepb.PlanConfig_Spec_CreateDatabaseConfig {
	c := config.CreateDatabaseConfig
	return &storepb.PlanConfig_Spec_CreateDatabaseConfig{
		CreateDatabaseConfig: convertPlanConfigCreateDatabaseConfig(c),
	}
}

func convertPlanConfigCreateDatabaseConfig(c *v1pb.Plan_CreateDatabaseConfig) *storepb.PlanConfig_CreateDatabaseConfig {
	return &storepb.PlanConfig_CreateDatabaseConfig{
		Target:       c.Target,
		Database:     c.Database,
		Table:        c.Table,
		CharacterSet: c.CharacterSet,
		Collation:    c.Collation,
		Cluster:      c.Cluster,
		Owner:        c.Owner,
		Environment:  c.Environment,
	}
}

func convertPlanSpecChangeDatabaseConfig(config *v1pb.Plan_Spec_ChangeDatabaseConfig) *storepb.PlanConfig_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig

	// Sheet can be empty when using Release-based workflow (SQL comes from release files).
	// Plans can use either Sheet-based or Release-based approach, but not both.
	var sheetSha256 string
	if c.Sheet != "" {
		_, sha256, err := common.GetProjectResourceIDSheetSha256(c.Sheet)
		if err != nil {
			return nil
		}
		sheetSha256 = sha256
	}
	return &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
			Targets:           c.Targets,
			SheetSha256:       sheetSha256,
			Release:           c.Release,
			GhostFlags:        c.GhostFlags,
			EnablePriorBackup: c.EnablePriorBackup,
			EnableGhost:       c.EnableGhost,
		},
	}
}

func convertPlanSpecExportDataConfig(config *v1pb.Plan_Spec_ExportDataConfig) *storepb.PlanConfig_Spec_ExportDataConfig {
	c := config.ExportDataConfig
	// Sheet can be empty if not yet attached to the export data config.
	var sheetSha256 string
	if c.Sheet != "" {
		_, sha256, err := common.GetProjectResourceIDSheetSha256(c.Sheet)
		if err != nil {
			return nil
		}
		sheetSha256 = sha256
	}
	return &storepb.PlanConfig_Spec_ExportDataConfig{
		ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{
			Targets:     c.Targets,
			SheetSha256: sheetSha256,
			Format:      convertToExportFormat(c.Format),
			Password:    c.Password,
		},
	}
}

func convertToPlanCheckRunStatus(status store.PlanCheckRunStatus) v1pb.PlanCheckRun_Status {
	switch status {
	case store.PlanCheckRunStatusCanceled:
		return v1pb.PlanCheckRun_CANCELED
	case store.PlanCheckRunStatusDone:
		return v1pb.PlanCheckRun_DONE
	case store.PlanCheckRunStatusFailed:
		return v1pb.PlanCheckRun_FAILED
	case store.PlanCheckRunStatusRunning:
		return v1pb.PlanCheckRun_RUNNING
	default:
		return v1pb.PlanCheckRun_STATUS_UNSPECIFIED
	}
}

func convertToPlanCheckRunResults(results []*storepb.PlanCheckRunResult_Result) []*v1pb.PlanCheckRun_Result {
	var resultsV1 []*v1pb.PlanCheckRun_Result
	for _, result := range results {
		resultsV1 = append(resultsV1, convertToPlanCheckRunResult(result))
	}
	return resultsV1
}

func convertToPlanCheckRunResult(result *storepb.PlanCheckRunResult_Result) *v1pb.PlanCheckRun_Result {
	resultV1 := &v1pb.PlanCheckRun_Result{
		Status:  convertToPlanCheckRunResultStatus(result.Status),
		Title:   result.Title,
		Content: result.Content,
		Code:    result.Code,
		Target:  result.Target,
		Type:    convertToV1ResultType(result.Type),
		Report:  nil,
	}
	switch report := result.Report.(type) {
	case *storepb.PlanCheckRunResult_Result_SqlSummaryReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlSummaryReport_{
			SqlSummaryReport: &v1pb.PlanCheckRun_Result_SqlSummaryReport{
				StatementTypes: report.SqlSummaryReport.StatementTypes,
				AffectedRows:   report.SqlSummaryReport.AffectedRows,
			},
		}
	case *storepb.PlanCheckRunResult_Result_SqlReviewReport_:
		resultV1.Report = &v1pb.PlanCheckRun_Result_SqlReviewReport_{
			SqlReviewReport: &v1pb.PlanCheckRun_Result_SqlReviewReport{
				StartPosition: convertToPosition(report.SqlReviewReport.StartPosition),
				EndPosition:   convertToPosition(report.SqlReviewReport.EndPosition),
			},
		}
	}
	return resultV1
}

func convertToV1ResultType(t storepb.PlanCheckType) v1pb.PlanCheckRun_Result_Type {
	switch t {
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE:
		return v1pb.PlanCheckRun_Result_STATEMENT_ADVISE
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT:
		return v1pb.PlanCheckRun_Result_STATEMENT_SUMMARY_REPORT
	case storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC:
		return v1pb.PlanCheckRun_Result_GHOST_SYNC
	default:
		return v1pb.PlanCheckRun_Result_TYPE_UNSPECIFIED
	}
}

func convertToPlanCheckRunResultStatus(status storepb.Advice_Status) v1pb.Advice_Level {
	switch status {
	case storepb.Advice_STATUS_UNSPECIFIED:
		return v1pb.Advice_ADVICE_LEVEL_UNSPECIFIED
	case storepb.Advice_SUCCESS:
		return v1pb.Advice_SUCCESS
	case storepb.Advice_WARNING:
		return v1pb.Advice_WARNING
	case storepb.Advice_ERROR:
		return v1pb.Advice_ERROR
	default:
		return v1pb.Advice_ADVICE_LEVEL_UNSPECIFIED
	}
}
