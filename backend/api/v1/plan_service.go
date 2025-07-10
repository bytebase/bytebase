package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
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
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan, error: %v", err))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan not found for id: %d", planID))
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

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
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
	if err := s.buildPlanFindWithFilter(ctx, find, req.Filter); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to build plan find with filter, error: %v", err))
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
	projects, err := stores.ListProjectV2(ctx, &store.FindProjectMessage{})
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
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("principal ID not found"))
	}
	projectID, err := common.GetProjectID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}

	// Validate plan specs
	if err := validateSpecs(req.Plan.Specs); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to validate plan specs, error: %v", err))
	}

	planMessage := &store.PlanMessage{
		ProjectID:   projectID,
		PipelineUID: nil,
		Name:        req.Plan.Title,
		Description: req.Plan.Description,
		Config:      convertPlan(req.Plan),
	}
	deployment, err := getPlanDeployment(ctx, s.store, planMessage.Config.GetSpecs(), project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan deployment snapshot, error: %v", err))
	}
	planMessage.Config.Deployment = deployment

	if _, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.dbFactory, planMessage.Name, planMessage.Config.GetSpecs(), deployment, project); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get pipeline from the plan, please check you request, error: %v", err))
	}
	plan, err := s.store.CreatePlan(ctx, planMessage, principalID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan, error: %v", err))
	}

	// Don't create plan checks if the plan comes from releases.
	// Plan check results don't match release checks.
	if !planHasRelease(req.Plan) {
		planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, plan)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for plan, error: %v", err))
		}
		if err := s.store.CreatePlanCheckRuns(ctx, plan, planCheckRuns...); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan check runs, error: %v", err))
		}
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
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	projectID, planID, err := common.GetProjectIDPlanID(req.Plan.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
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
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan %q not found", req.Plan.Name))
	}

	if storePlanConfigHasRelease(oldPlan.Config) && slices.Contains(req.UpdateMask.Paths, "specs") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("disallowed to update the plan steps because the plan is created from a release"))
	}

	ok, err = func() (bool, error) {
		if oldPlan.CreatorUID == user.ID {
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

	// Update the deployment in the end because the specs might change.
	paths := slices.Clone(req.UpdateMask.Paths)
	slices.SortFunc(paths, func(a, b string) int {
		if a == "deployment" {
			return 1
		}
		if b == "deployment" {
			return -1
		}
		return strings.Compare(a, b)
	})
	for _, path := range paths {
		switch path {
		case "title":
			title := req.Plan.Title
			planUpdate.Name = &title
		case "description":
			description := req.Plan.Description
			planUpdate.Description = &description
		case "deployment":
			specs := oldPlan.Config.GetSpecs()
			if planUpdate.Specs != nil {
				specs = *planUpdate.Specs
			}
			deployment, err := getPlanDeployment(ctx, s.store, specs, project)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan deployment snapshot, error: %v", err))
			}
			planUpdate.Deployment = &deployment
		case "specs":
			// Use specs directly for internal storage
			allSpecs := convertPlanSpecs(req.GetPlan().GetSpecs())
			planUpdate.Specs = &allSpecs

			if _, err := GetPipelineCreate(ctx,
				s.store,
				s.sheetManager,
				s.dbFactory,
				oldPlan.Name,
				allSpecs,
				oldPlan.Config.GetDeployment(),
				project); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get pipeline from the plan, please check you request, error: %v", err))
			}

			// Compare specs directly
			oldSpecs := convertToPlanSpecs(oldPlan.Config.Specs)

			removed, added, updated := diffSpecsDirectly(oldSpecs, req.Plan.Specs)
			if len(removed) > 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot remove specs from plan"))
			}
			if len(added) > 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot add specs to plan"))
			}
			if len(updated) == 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("no specs updated"))
			}

			oldSpecsByID := make(map[string]*v1pb.Plan_Spec)
			for _, spec := range oldSpecs {
				oldSpecsByID[spec.Id] = spec
			}

			updatedByID := make(map[string]*v1pb.Plan_Spec)
			for _, spec := range updated {
				updatedByID[spec.Id] = spec
			}

			// Handle task updates for specs
			tasksMap := map[int]*store.TaskMessage{}
			var taskPatchList []*store.TaskPatch
			var issueCommentCreates []*store.IssueCommentMessage

			issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PlanUID: &oldPlan.UID})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue: %v", err))
			}

			if oldPlan.PipelineUID != nil {
				tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: oldPlan.PipelineUID})
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list tasks: %v", err))
				}
				tasksBySpecID := make(map[string][]*store.TaskMessage)
				for _, task := range tasks {
					specID := task.Payload.GetSpecId()
					tasksBySpecID[specID] = append(tasksBySpecID[specID], task)
				}
				for _, task := range tasks {
					doUpdate := false
					taskPatch := &store.TaskPatch{
						ID:        task.ID,
						UpdaterID: user.ID,
					}
					specID := task.Payload.GetSpecId()
					spec, ok := updatedByID[specID]
					if !ok {
						continue
					}

					newTaskType, err := getTaskTypeFromSpec(spec)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get task type from spec, err: %v", err))
					}
					if newTaskType != task.Type {
						taskTypes := []storepb.Task_Type{
							storepb.Task_DATABASE_SCHEMA_UPDATE,
							storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST,
						}
						if !slices.Contains(taskTypes, newTaskType) || !slices.Contains(taskTypes, task.Type) {
							return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("task types in %v are allowed to updated, and they are allowed to be changed to %v", taskTypes, taskTypes))
						}
						taskPatch.Type = &newTaskType
						doUpdate = true
					}

					// Flags for gh-ost.
					if err := func() error {
						switch newTaskType {
						case storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST:
						default:
							return nil
						}

						newFlags := spec.GetChangeDatabaseConfig().GetGhostFlags()
						if _, err := ghost.GetUserFlags(newFlags); err != nil {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid ghost flags %q, error %v", newFlags, err))
						}
						oldFlags := task.Payload.GetFlags()
						if cmp.Equal(oldFlags, newFlags) {
							return nil
						}
						taskPatch.Flags = &newFlags
						doUpdate = true
						return nil
					}(); err != nil {
						return nil, err
					}

					// Prior Backup
					func() {
						if newTaskType != storepb.Task_DATABASE_DATA_UPDATE {
							return
						}
						config, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
						if !ok {
							return
						}

						// Check if backup setting has changed.
						planEnableBackup := config.ChangeDatabaseConfig.GetEnablePriorBackup()
						taskEnableBackup := task.Payload.GetEnablePriorBackup()
						if planEnableBackup != taskEnableBackup {
							taskPatch.EnablePriorBackup = &planEnableBackup
							doUpdate = true
						}
					}()

					// Sheet
					if err := func() error {
						switch newTaskType {
						case storepb.Task_DATABASE_SCHEMA_UPDATE, storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST, storepb.Task_DATABASE_DATA_UPDATE, storepb.Task_DATABASE_EXPORT:
							var oldSheetName string
							if newTaskType == storepb.Task_DATABASE_EXPORT {
								config, ok := spec.Config.(*v1pb.Plan_Spec_ExportDataConfig)
								if !ok {
									return nil
								}
								oldSheetName = config.ExportDataConfig.Sheet
							} else {
								config, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
								if !ok {
									return nil
								}
								oldSheetName = config.ChangeDatabaseConfig.Sheet
							}
							_, sheetUID, err := common.GetProjectResourceIDSheetUID(oldSheetName)
							if err != nil {
								return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get sheet id from %q, error: %v", oldSheetName, err))
							}
							if int(task.Payload.GetSheetId()) == sheetUID {
								return nil
							}

							sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
								UID: &sheetUID,
							})
							if err != nil {
								return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get sheet %q: %v", oldSheetName, err))
							}
							if sheet == nil {
								return connect.NewError(connect.CodeNotFound, errors.Errorf("sheet %q not found", oldSheetName))
							}
							doUpdate = true
							taskPatch.SheetID = &sheet.UID

							if issue != nil {
								oldSheet := common.FormatSheet(issue.Project.ResourceID, int(task.Payload.GetSheetId()))
								newSheet := common.FormatSheet(issue.Project.ResourceID, sheet.UID)
								issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
									IssueUID: issue.UID,
									Payload: &storepb.IssueCommentPayload{
										Event: &storepb.IssueCommentPayload_TaskUpdate_{
											TaskUpdate: &storepb.IssueCommentPayload_TaskUpdate{
												Tasks:     []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.Environment, task.ID)},
												FromSheet: &oldSheet,
												ToSheet:   &newSheet,
											},
										},
									},
								})
							}
						}
						return nil
					}(); err != nil {
						return nil, err
					}

					// ExportDataConfig
					func() {
						if newTaskType != storepb.Task_DATABASE_EXPORT {
							return
						}
						config, ok := spec.Config.(*v1pb.Plan_Spec_ExportDataConfig)
						if !ok {
							return
						}
						if config.ExportDataConfig.Format != convertExportFormat(task.Payload.GetFormat()) {
							format := convertToExportFormat(config.ExportDataConfig.Format)
							taskPatch.ExportFormat = &format
							doUpdate = true
						}
						if (config.ExportDataConfig.Password == nil && task.Payload.GetPassword() != "") || (config.ExportDataConfig.Password != nil && *config.ExportDataConfig.Password != task.Payload.GetPassword()) {
							taskPatch.ExportPassword = config.ExportDataConfig.Password
							doUpdate = true
						}
					}()

					if !doUpdate {
						continue
					}
					tasksMap[task.ID] = task
					taskPatchList = append(taskPatchList, taskPatch)
				}

				if len(taskPatchList) != 0 {
					issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{
						PipelineID: oldPlan.PipelineUID,
					})
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get issue: %v", err))
					}
					if issue != nil {
						// Do not allow to update task if issue is done or canceled.
						if issue.Status == storepb.Issue_DONE || issue.Status == storepb.Issue_CANCELED {
							return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("cannot update task because issue %q is %s", issue.Title, issue.Status))
						}
					}
				}
			}

			for _, taskPatch := range taskPatchList {
				if taskPatch.SheetID != nil {
					task := tasksMap[taskPatch.ID]
					if task.LatestTaskRunStatus == storepb.TaskRun_PENDING || task.LatestTaskRunStatus == storepb.TaskRun_RUNNING || task.LatestTaskRunStatus == storepb.TaskRun_SKIPPED || task.LatestTaskRunStatus == storepb.TaskRun_DONE {
						return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("cannot update plan because task %v is %s", task.ID, task.LatestTaskRunStatus))
					}
				}
			}

			var doUpdateSheet bool
			for _, taskPatch := range taskPatchList {
				// If backup setting has been updated, we need to rerun the plan check runs.
				if taskPatch.EnablePriorBackup != nil {
					planCheckRunsTrigger = true
				}
				if taskPatch.SheetID != nil {
					doUpdateSheet = true
				}
			}

			// For the plan without pipeline, we need to check if the sheet is updated in related specs.
			if oldPlan.PipelineUID == nil {
				for _, specPatch := range updated {
					oldSpec := oldSpecsByID[specPatch.Id]
					if oldSpec.GetChangeDatabaseConfig() != nil && specPatch.GetChangeDatabaseConfig() != nil {
						oldConfig, newConfig := oldSpec.GetChangeDatabaseConfig(), specPatch.GetChangeDatabaseConfig()
						if oldConfig.Sheet != newConfig.Sheet {
							doUpdateSheet = true
							break
						}
					}
				}
			}

			// If sheet is updated, we need to rerun the plan check runs.
			if doUpdateSheet {
				planCheckRunsTrigger = true
			}

			// Check project setting for modify statement.
			if len(taskPatchList) > 0 && doUpdateSheet && !project.Setting.AllowModifyStatement {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("modify statement is not allowed for project %s", project.Title))
			}

			for _, taskPatch := range taskPatchList {
				task := tasksMap[taskPatch.ID]
				if _, err := s.store.UpdateTaskV2(ctx, taskPatch); err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update task %v: %v", task.ID, err))
				}
			}

			for _, issueCommentCreate := range issueCommentCreates {
				if _, err := s.store.CreateIssueComment(ctx, issueCommentCreate, user.ID); err != nil {
					slog.Warn("failed to create issue comments", "issueUID", issue.UID, log.BBError(err))
				}
			}

			if issue != nil && doUpdateSheet {
				if err := func() error {
					issue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{
						PayloadUpsert: &storepb.Issue{
							Approval: &storepb.IssuePayloadApproval{
								ApprovalFindingDone: false,
							},
						},
					})
					if err != nil {
						return errors.Errorf("failed to update issue: %v", err)
					}
					s.stateCfg.ApprovalFinding.Store(issue.UID, issue)
					return nil
				}(); err != nil {
					slog.Error("failed to update issue to refind approval", log.BBError(err))
				}
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update_mask path %q", path))
		}
	}

	if err := s.store.UpdatePlan(ctx, planUpdate); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update plan %q: %v", req.Plan.Name, err))
	}

	updatedPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &oldPlan.UID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get updated plan %q: %v", req.Plan.Name, err))
	}
	if updatedPlan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("updated plan %q not found", req.Plan.Name))
	}

	if planCheckRunsTrigger {
		planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, updatedPlan)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for plan, error: %v", err))
		}
		if err := s.store.CreatePlanCheckRuns(ctx, updatedPlan, planCheckRuns...); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create plan check runs, error: %v", err))
		}
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
	converted, err := convertToPlanCheckRuns(ctx, s.store, projectID, planUID, planCheckRuns)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert plan check runs, error: %v", err))
	}

	return connect.NewResponse(&v1pb.ListPlanCheckRunsResponse{
		PlanCheckRuns: converted,
		NextPageToken: "",
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
					v1ResultStatus := v1pb.PlanCheckRun_Result_Status_value[resultStatusStr]
					if v1ResultStatus == 0 && resultStatusStr != "STATUS_UNSPECIFIED" {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid result_status value: %s", resultStatusStr))
					}
					storeResultStatus := convertToStoreResultStatus(v1pb.PlanCheckRun_Result_Status(v1ResultStatus))
					if find.ResultStatus == nil {
						find.ResultStatus = &[]storepb.PlanCheckRunResult_Result_Status{}
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
						find.ResultStatus = &[]storepb.PlanCheckRunResult_Result_Status{}
					}
					for _, raw := range rawList {
						resultStatusStr, ok := raw.(string)
						if !ok {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("result_status value must be a string"))
						}
						// Convert v1pb result status to store result status
						v1ResultStatus := v1pb.PlanCheckRun_Result_Status_value[resultStatusStr]
						if v1ResultStatus == 0 && resultStatusStr != "STATUS_UNSPECIFIED" {
							return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid result_status value: %s", resultStatusStr))
						}
						storeResultStatus := convertToStoreResultStatus(v1pb.PlanCheckRun_Result_Status(v1ResultStatus))
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

// convertToStoreResultStatus converts v1pb.PlanCheckRun_Result_Status to storepb.PlanCheckRunResult_Result_Status.
func convertToStoreResultStatus(status v1pb.PlanCheckRun_Result_Status) storepb.PlanCheckRunResult_Result_Status {
	switch status {
	case v1pb.PlanCheckRun_Result_ERROR:
		return storepb.PlanCheckRunResult_Result_ERROR
	case v1pb.PlanCheckRun_Result_WARNING:
		return storepb.PlanCheckRunResult_Result_WARNING
	case v1pb.PlanCheckRun_Result_SUCCESS:
		return storepb.PlanCheckRunResult_Result_SUCCESS
	default:
		return storepb.PlanCheckRunResult_Result_STATUS_UNSPECIFIED
	}
}

// RunPlanChecks runs plan checks for a plan.
func (s *PlanService) RunPlanChecks(ctx context.Context, request *connect.Request[v1pb.RunPlanChecksRequest]) (*connect.Response[v1pb.RunPlanChecksResponse], error) {
	req := request.Msg
	projectID, planID, err := common.GetProjectIDPlanID(req.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project, error: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project not found for id: %v", projectID))
	}
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan, error: %v", err))
	}
	if plan == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("plan not found"))
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	if req.SpecId != nil {
		var foundSpec *storepb.PlanConfig_Spec
		for _, spec := range plan.Config.GetSpecs() {
			if spec.Id == *req.SpecId {
				foundSpec = spec
				break
			}
		}
		if foundSpec == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("spec with id %q not found in plan", *req.SpecId))
		}
		planCheckRuns, err = getPlanCheckRunsFromSpec(ctx, s.store, plan, foundSpec)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for spec, error: %v", err))
		}
	} else {
		// If spec ID is not provided, run plan check runs for all specs in the plan.
		planCheckRuns, err = getPlanCheckRunsFromPlan(ctx, s.store, plan)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get plan check runs for plan, error: %v", err))
		}
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

	projectID, _, err := common.GetProjectIDPlanID(req.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
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

func (s *PlanService) buildPlanFindWithFilter(ctx context.Context, planFind *store.FindPlanMessage, filter string) error {
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

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable string, value any) (string, error) {
		switch variable {
		case "title":
			positionalArgs = append(positionalArgs, value)
			return fmt.Sprintf("plan.name = $%d", len(positionalArgs)), nil
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported variable %q", variable))
		}
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				switch variable {
				case "creator":
					user, err := s.getUserByIdentifier(ctx, value.(string))
					if err != nil {
						return "", connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user %v with error %v", value, err.Error()))
					}
					positionalArgs = append(positionalArgs, user.ID)
					return fmt.Sprintf("plan.creator_id = $%d", len(positionalArgs)), nil
				case "has_pipeline":
					hasPipeline, ok := value.(bool)
					if !ok {
						return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`"has_pipeline" should be bool`))
					}
					if !hasPipeline {
						return "plan.pipeline_id IS NULL", nil
					}
					return "plan.pipeline_id IS NOT NULL", nil
				case "has_issue":
					hasIssue, ok := value.(bool)
					if !ok {
						return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`"has_issue" should be bool`))
					}
					if !hasIssue {
						return "issue.id IS NULL", nil
					}
					return "issue.id IS NOT NULL", nil
				default:
					return parseToSQL(variable, value)
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
				positionalArgs = append(positionalArgs, t)
				if functionName == celoperators.GreaterEquals {
					return fmt.Sprintf("plan.created_at >= $%d", len(positionalArgs)), nil
				}
				return fmt.Sprintf("plan.created_at <= $%d", len(positionalArgs)), nil
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}
				strValue = strings.ToLower(strValue)

				switch variable {
				case "title":
					return "LOWER(plan.name) LIKE '%" + strValue + "%'", nil
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "title" supports %q operator, but found %q`, celoverloads.Matches, variable))
				}
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported function %v", functionName))
			}
		default:
			return "", errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return err
	}

	if where != "" {
		planFind.Filter = &store.ListResourceFilter{
			Where: "(" + where + ")",
			Args:  positionalArgs,
		}
	}

	return nil
}

func (s *PlanService) getUserByIdentifier(ctx context.Context, identifier string) (*store.UserMessage, error) {
	email := strings.TrimPrefix(identifier, "users/")
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid empty creator identifier"))
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf(`failed to find user "%s" with error: %v`, email, err))
	}
	if user == nil {
		return nil, errors.Errorf("cannot found user %s", email)
	}
	return user, nil
}

// diffSpecs check if there are any specs removed, added or updated in the new plan.
// Only updating sheet is taken into account.
func diffSpecsDirectly(oldSpecs []*v1pb.Plan_Spec, newSpecs []*v1pb.Plan_Spec) ([]*v1pb.Plan_Spec, []*v1pb.Plan_Spec, []*v1pb.Plan_Spec) {
	oldSpecsMap := make(map[string]*v1pb.Plan_Spec)
	newSpecsMap := make(map[string]*v1pb.Plan_Spec)
	var removed, added, updated []*v1pb.Plan_Spec

	for _, spec := range oldSpecs {
		oldSpecsMap[spec.Id] = spec
	}
	for _, spec := range newSpecs {
		newSpecsMap[spec.Id] = spec
	}

	for _, spec := range oldSpecs {
		if _, ok := newSpecsMap[spec.Id]; !ok {
			removed = append(removed, spec)
		}
	}

	for _, spec := range newSpecs {
		if oldSpec, ok := oldSpecsMap[spec.Id]; !ok {
			added = append(added, spec)
		} else if !cmp.Equal(oldSpec, spec, protocmp.Transform()) {
			updated = append(updated, spec)
		}
	}

	return removed, added, updated
}

func validateSpecs(specs []*v1pb.Plan_Spec) error {
	if len(specs) == 0 {
		return errors.Errorf("the plan has zero spec")
	}
	configTypeCount := map[string]int{}
	seenID := map[string]bool{}

	for _, spec := range specs {
		id := spec.GetId()
		if id == "" {
			return errors.Errorf("spec id cannot be empty")
		}
		if seenID[id] {
			return errors.Errorf("found duplicate spec id %v", id)
		}
		seenID[id] = true

		switch config := spec.Config.(type) {
		case *v1pb.Plan_Spec_CreateDatabaseConfig:
			configTypeCount["create_database"]++
		case *v1pb.Plan_Spec_ChangeDatabaseConfig:
			configTypeCount["change_database"]++
			var databaseTarget, databaseGroupTarget int
			for _, target := range config.ChangeDatabaseConfig.Targets {
				if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
					databaseTarget++
				} else if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
					databaseGroupTarget++
				} else {
					return errors.Errorf("invalid target %v", target)
				}
			}
			// Disallow mixing database and database group targets in the same spec.
			if databaseTarget > 0 && databaseGroupTarget > 0 {
				return errors.Errorf("found databaseTarget and databaseGroupTarget, expect only one kind")
			}
		case *v1pb.Plan_Spec_ExportDataConfig:
			configTypeCount["export_data"]++
		default:
			return errors.Errorf("invalid spec type")
		}
	}
	if len(configTypeCount) > 1 {
		return errors.Errorf("plan contains multiple types of spec configurations (%v), but each plan must contain only one type", len(configTypeCount))
	}
	return nil
}

func getPlanSpecDatabaseGroups(specs []*storepb.PlanConfig_Spec) []string {
	var databaseGroups []string
	for _, spec := range specs {
		if _, ok := spec.Config.(*storepb.PlanConfig_Spec_ChangeDatabaseConfig); !ok {
			continue
		}
		for _, target := range spec.GetChangeDatabaseConfig().GetTargets() {
			if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
				databaseGroups = append(databaseGroups, target)
			}
		}
	}
	return databaseGroups
}

// getAllEnvironmentIDs returns all environment IDs from the store.
func getAllEnvironmentIDs(ctx context.Context, s *store.Store) ([]string, error) {
	environments, err := s.GetEnvironmentSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list environments")
	}
	var environmentIDs []string
	for _, e := range environments.GetEnvironments() {
		environmentIDs = append(environmentIDs, e.Id)
	}
	return environmentIDs, nil
}

func getPlanDeployment(ctx context.Context, s *store.Store, specs []*storepb.PlanConfig_Spec, project *store.ProjectMessage) (*storepb.PlanConfig_Deployment, error) {
	snapshot := &storepb.PlanConfig_Deployment{}

	environmentIDs, err := getAllEnvironmentIDs(ctx, s)
	if err != nil {
		return nil, err
	}
	snapshot.Environments = environmentIDs

	databaseGroups := getPlanSpecDatabaseGroups(specs)

	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
	}

	for _, name := range databaseGroups {
		projectID, id, err := common.GetProjectIDDatabaseGroupID(name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database group id")
		}
		if projectID != project.ResourceID {
			return nil, errors.Errorf("%s does not belong to project %s", name, project.ResourceID)
		}
		databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
			ResourceID: &id,
			ProjectID:  &project.ResourceID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database group")
		}

		matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get matched and unmatched databases in database group %q", id)
		}

		var databases []string
		for _, db := range matchedDatabases {
			databases = append(databases, common.FormatDatabase(db.InstanceID, db.DatabaseName))
		}

		snapshot.DatabaseGroupMappings = append(snapshot.DatabaseGroupMappings, &storepb.PlanConfig_Deployment_DatabaseGroupMapping{
			DatabaseGroup: name,
			Databases:     databases,
		})
	}

	return snapshot, nil
}

func planHasRelease(plan *v1pb.Plan) bool {
	for _, spec := range plan.GetSpecs() {
		if c, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig); ok {
			if c.ChangeDatabaseConfig.Release != "" {
				return true
			}
		}
	}
	return false
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

func getTaskTypeFromSpec(spec *v1pb.Plan_Spec) (storepb.Task_Type, error) {
	switch s := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		return storepb.Task_DATABASE_CREATE, nil
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		switch s.ChangeDatabaseConfig.Type {
		case v1pb.Plan_ChangeDatabaseConfig_DATA:
			return storepb.Task_DATABASE_DATA_UPDATE, nil
		case v1pb.Plan_ChangeDatabaseConfig_MIGRATE:
			return storepb.Task_DATABASE_SCHEMA_UPDATE, nil
		case v1pb.Plan_ChangeDatabaseConfig_MIGRATE_GHOST:
			return storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST, nil
		}
	case *v1pb.Plan_Spec_ExportDataConfig:
		return storepb.Task_DATABASE_EXPORT, nil
	}
	return storepb.Task_TASK_TYPE_UNSPECIFIED, errors.Errorf("unknown spec config type")
}
