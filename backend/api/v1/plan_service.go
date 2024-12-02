package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// PlanService represents a service for managing plan.
type PlanService struct {
	v1pb.UnimplementedPlanServiceServer
	store          *store.Store
	sheetManager   *sheet.Manager
	licenseService enterprise.LicenseService
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewPlanService returns a plan service instance.
func NewPlanService(store *store.Store, sheetManager *sheet.Manager, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile *config.Profile, iamManager *iam.Manager) *PlanService {
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
func (s *PlanService) GetPlan(ctx context.Context, request *v1pb.GetPlanRequest) (*v1pb.Plan, error) {
	projectID, planID, err := common.GetProjectIDPlanID(request.Name)
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
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}
	convertedPlan, err := convertToPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to plan, error: %v", err)
	}
	return convertedPlan, nil
}

// ListPlans lists plans.
func (s *PlanService) ListPlans(ctx context.Context, request *v1pb.ListPlansRequest) (*v1pb.ListPlansResponse, error) {
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

	find := &store.FindPlanMessage{
		Limit:     &limitPlusOne,
		Offset:    &offset.offset,
		ProjectID: &projectID,
	}
	plans, err := s.store.ListPlans(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list plans, error: %v", err)
	}

	var nextPageToken string
	// has more pages
	if len(plans) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		plans = plans[:offset.limit]
	}

	convertedPlans, err := convertToPlans(ctx, s.store, plans)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to plans, error: %v", err)
	}

	return &v1pb.ListPlansResponse{
		Plans:         convertedPlans,
		NextPageToken: nextPageToken,
	}, nil
}

// SearchPlans searches plans.
func (s *PlanService) SearchPlans(ctx context.Context, request *v1pb.SearchPlansRequest) (*v1pb.SearchPlansResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	projectIDsFilter, err := getProjectIDsSearchFilter(ctx, user, iam.PermissionPlansGet, s.iamManager, s.store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get projectIDs, error: %v", err)
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

	find := &store.FindPlanMessage{
		Limit:      &limitPlusOne,
		Offset:     &offset.offset,
		ProjectIDs: projectIDsFilter,
	}
	if projectID != "-" {
		find.ProjectID = &projectID
	}
	if err := s.buildPlanFindWithFilter(ctx, find, request.Filter); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to build plan find with filter, error: %v", err)
	}

	plans, err := s.store.ListPlans(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list plans, error: %v", err)
	}

	var nextPageToken string
	// has more pages
	if len(plans) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		plans = plans[:offset.limit]
	}

	convertedPlans, err := convertToPlans(ctx, s.store, plans)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to plans, error: %v", err)
	}

	return &v1pb.SearchPlansResponse{
		Plans:         convertedPlans,
		NextPageToken: nextPageToken,
	}, nil
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
func (s *PlanService) CreatePlan(ctx context.Context, request *v1pb.CreatePlanRequest) (*v1pb.Plan, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
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
	if err := validateSteps(request.Plan.Steps); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to validate plan steps, error: %v", err)
	}

	planMessage := &store.PlanMessage{
		ProjectID:   projectID,
		PipelineUID: nil,
		Name:        request.Plan.Title,
		Description: request.Plan.Description,
		Config:      convertPlan(request.Plan),
	}
	if request.GetPlan().VcsSource != nil {
		planMessage.Config.VcsSource = &storepb.PlanConfig_VCSSource{
			VcsConnector:   request.GetPlan().GetVcsSource().GetVcsConnector(),
			PullRequestUrl: request.GetPlan().GetVcsSource().GetPullRequestUrl(),
			VcsType:        storepb.VCSType(request.GetPlan().GetVcsSource().VcsType),
		}
	}

	serializeTasks := request.GetPlan().GetVcsSource() != nil

	if _, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.licenseService, s.dbFactory, planMessage.Config.GetSteps(), project, serializeTasks); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline from the plan, please check you request, error: %v", err)
	}
	plan, err := s.store.CreatePlan(ctx, planMessage, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan, error: %v", err)
	}

	planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
	}
	if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
	}

	// Tickle plan check scheduler.
	s.stateCfg.PlanCheckTickleChan <- 0

	convertedPlan, err := convertToPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to plan, error: %v", err)
	}
	return convertedPlan, nil
}

// UpdatePlan updates a plan.
func (s *PlanService) UpdatePlan(ctx context.Context, request *v1pb.UpdatePlanRequest) (*v1pb.Plan, error) {
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	if len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must not be empty")
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	projectID, planID, err := common.GetProjectIDPlanID(request.Plan.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project %q, err: %v", projectID, err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	oldPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: &projectID, UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan %q: %v", request.Plan.Name, err)
	}
	if oldPlan == nil {
		return nil, status.Errorf(codes.NotFound, "plan %q not found", request.Plan.Name)
	}

	ok, err = func() (bool, error) {
		if oldPlan.CreatorUID == user.ID {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionPlansUpdate, user, oldPlan.ProjectID)
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to update plan")
	}

	planUpdate := &store.UpdatePlanMessage{
		UID:       oldPlan.UID,
		UpdaterID: user.ID,
	}

	var planCheckRunsTrigger bool
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			title := request.Plan.Title
			planUpdate.Name = &title
		case "description":
			description := request.Plan.Description
			planUpdate.Description = &description
		case "steps":
			planUpdate.Config = &storepb.PlanConfig{
				Steps: convertPlanSteps(request.Plan.Steps),
			}

			serializeTasks := oldPlan.Config.GetVcsSource() != nil

			if _, err := GetPipelineCreate(ctx, s.store, s.sheetManager, s.licenseService, s.dbFactory, convertPlanSteps(request.Plan.GetSteps()), project, serializeTasks); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline from the plan, please check you request, error: %v", err)
			}

			oldSteps := convertToPlanSteps(oldPlan.Config.Steps)
			issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PlanUID: &oldPlan.UID})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get issue: %v", err)
			}

			removed, added, updated := diffSpecs(oldSteps, request.Plan.Steps)
			if len(removed) > 0 {
				return nil, status.Errorf(codes.InvalidArgument, "cannot remove specs from plan")
			}
			if len(added) > 0 {
				return nil, status.Errorf(codes.InvalidArgument, "cannot add specs to plan")
			}
			if len(updated) == 0 {
				return nil, status.Errorf(codes.InvalidArgument, "no specs updated")
			}

			oldSpecsByID := make(map[string]*v1pb.Plan_Spec)
			for _, step := range oldSteps {
				for _, spec := range step.Specs {
					oldSpecsByID[spec.Id] = spec
				}
			}

			updatedByID := make(map[string]*v1pb.Plan_Spec)
			for _, spec := range updated {
				updatedByID[spec.Id] = spec
			}

			tasksMap := map[int]*store.TaskMessage{}
			var taskPatchList []*api.TaskPatch
			var issueCommentCreates []*store.IssueCommentMessage
			var taskDAGRebuildList []struct {
				fromTaskIDs []int
				toTaskID    int
			}

			if oldPlan.PipelineUID != nil {
				tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: oldPlan.PipelineUID})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to list tasks: %v", err)
				}
				tasksBySpecID := make(map[string][]*store.TaskMessage)
				for _, task := range tasks {
					var taskSpecID struct {
						SpecID string `json:"specId"`
					}
					if err := json.Unmarshal([]byte(task.Payload), &taskSpecID); err != nil {
						return nil, status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
					}
					tasksBySpecID[taskSpecID.SpecID] = append(tasksBySpecID[taskSpecID.SpecID], task)
				}
				for _, task := range tasks {
					doUpdate := false
					taskPatch := &api.TaskPatch{
						ID:        task.ID,
						UpdaterID: user.ID,
					}
					var taskSpecID struct {
						SpecID string `json:"specId"`
					}
					if err := json.Unmarshal([]byte(task.Payload), &taskSpecID); err != nil {
						return nil, status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
					}
					spec, ok := updatedByID[taskSpecID.SpecID]
					if !ok {
						continue
					}

					// Flags for gh-ost.
					if err := func() error {
						if task.Type != api.TaskDatabaseSchemaUpdateGhostSync {
							return nil
						}
						payload := &storepb.TaskDatabaseUpdatePayload{}
						if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(task.Payload), payload); err != nil {
							return status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
						}
						newFlags := spec.GetChangeDatabaseConfig().GetGhostFlags()
						if _, err := ghost.GetUserFlags(newFlags); err != nil {
							return status.Errorf(codes.InvalidArgument, "invalid ghost flags %q, error %v", newFlags, err)
						}
						oldFlags := payload.Flags
						if cmp.Equal(oldFlags, newFlags) {
							return nil
						}
						taskPatch.Flags = &newFlags
						doUpdate = true
						return nil
					}(); err != nil {
						return nil, err
					}

					// EarliestAllowedTs
					if spec.EarliestAllowedTime.GetSeconds() != task.EarliestAllowedTs {
						seconds := spec.EarliestAllowedTime.GetSeconds()
						taskPatch.EarliestAllowedTs = &seconds
						doUpdate = true

						var fromEarliestAllowedTime, toEarliestAllowedTime *timestamppb.Timestamp
						if task.EarliestAllowedTs != 0 {
							fromEarliestAllowedTime = timestamppb.New(time.Unix(task.EarliestAllowedTs, 0))
						}
						if seconds != 0 {
							toEarliestAllowedTime = timestamppb.New(time.Unix(seconds, 0))
						}
						issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
							IssueUID: issue.UID,
							Payload: &storepb.IssueCommentPayload{
								Event: &storepb.IssueCommentPayload_TaskUpdate_{
									TaskUpdate: &storepb.IssueCommentPayload_TaskUpdate{
										Tasks:                   []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.StageID, task.ID)},
										FromEarliestAllowedTime: fromEarliestAllowedTime,
										ToEarliestAllowedTime:   toEarliestAllowedTime,
									},
								},
							},
						})
					}

					// PreUpdateBackupDetail
					if err := func() error {
						if task.Type != api.TaskDatabaseDataUpdate {
							return nil
						}
						payload := &storepb.TaskDatabaseUpdatePayload{}
						if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(task.Payload), payload); err != nil {
							return status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
						}
						config, ok := spec.Config.(*v1pb.Plan_Spec_ChangeDatabaseConfig)
						if !ok {
							return nil
						}

						// The target backup database name.
						// Format: instances/{instance}/databases/{database}
						var backupDatabaseName *string
						if config.ChangeDatabaseConfig.PreUpdateBackupDetail == nil {
							if payload.PreUpdateBackupDetail.Database != "" {
								emptyValue := ""
								backupDatabaseName = &emptyValue
							}
						} else {
							if config.ChangeDatabaseConfig.PreUpdateBackupDetail.Database != payload.PreUpdateBackupDetail.Database {
								backupDatabaseName = &config.ChangeDatabaseConfig.PreUpdateBackupDetail.Database
							}
						}
						if backupDatabaseName != nil {
							if *backupDatabaseName != "" {
								// If backup is enabled, we need to check if the backup is available for the source database. AKA, the task's target database.
								sourceDatabaseName := config.ChangeDatabaseConfig.Target
								instanceID, databaseName, err := common.GetInstanceDatabaseID(sourceDatabaseName)
								if err != nil {
									return errors.Wrapf(err, "failed to get instance database id from %q", sourceDatabaseName)
								}
								instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
								if err != nil {
									return errors.Wrapf(err, "failed to get instance %s", instanceID)
								}
								if instance == nil {
									return status.Errorf(codes.NotFound, "instance %q not found", instanceID)
								}
								database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
									InstanceID:          &instanceID,
									DatabaseName:        &databaseName,
									IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
								})
								if err != nil {
									return errors.Wrapf(err, "failed to get database %s", databaseName)
								}
								if database == nil {
									return status.Errorf(codes.NotFound, "database %q not found", databaseName)
								}
								if database.Metadata == nil || !database.Metadata.GetBackupAvailable() {
									return status.Errorf(codes.FailedPrecondition, "backup is not available for database %q", databaseName)
								}
							}

							taskPatch.PreUpdateBackupDetail = &storepb.PreUpdateBackupDetail{
								Database: *backupDatabaseName,
							}
							doUpdate = true
						}
						return nil
					}(); err != nil {
						return nil, err
					}

					// Sheet
					if err := func() error {
						switch task.Type {
						case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync, api.TaskDatabaseDataUpdate, api.TaskDatabaseDataExport:
							var taskPayload struct {
								SheetID int `json:"sheetId"`
							}
							if err := json.Unmarshal([]byte(task.Payload), &taskPayload); err != nil {
								return status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
							}

							var oldSheetName string
							if task.Type == api.TaskDatabaseDataExport {
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
								return status.Errorf(codes.Internal, "failed to get sheet id from %q, error: %v", oldSheetName, err)
							}
							if taskPayload.SheetID == sheetUID {
								return nil
							}

							sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
								UID: &sheetUID,
							})
							if err != nil {
								return status.Errorf(codes.Internal, "failed to get sheet %q: %v", oldSheetName, err)
							}
							if sheet == nil {
								return status.Errorf(codes.NotFound, "sheet %q not found", oldSheetName)
							}
							doUpdate = true
							taskPatch.SheetID = &sheet.UID

							oldSheet := common.FormatSheet(issue.Project.ResourceID, taskPayload.SheetID)
							newSheet := common.FormatSheet(issue.Project.ResourceID, sheet.UID)
							issueCommentCreates = append(issueCommentCreates, &store.IssueCommentMessage{
								IssueUID: issue.UID,
								Payload: &storepb.IssueCommentPayload{
									Event: &storepb.IssueCommentPayload_TaskUpdate_{
										TaskUpdate: &storepb.IssueCommentPayload_TaskUpdate{
											Tasks:     []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.StageID, task.ID)},
											FromSheet: &oldSheet,
											ToSheet:   &newSheet,
										},
									},
								},
							})
						}
						return nil
					}(); err != nil {
						return nil, err
					}

					// ExportDataConfig
					if err := func() error {
						if task.Type != api.TaskDatabaseDataExport {
							return nil
						}
						payload := &storepb.TaskDatabaseDataExportPayload{}
						if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(task.Payload), payload); err != nil {
							return status.Errorf(codes.Internal, "failed to unmarshal task payload: %v", err)
						}
						config, ok := spec.Config.(*v1pb.Plan_Spec_ExportDataConfig)
						if !ok {
							return nil
						}
						if config.ExportDataConfig.Format != convertExportFormat(payload.Format) {
							format := convertToExportFormat(config.ExportDataConfig.Format)
							taskPatch.ExportFormat = &format
							doUpdate = true
						}
						if (config.ExportDataConfig.Password == nil && payload.Password != "") || (config.ExportDataConfig.Password != nil && *config.ExportDataConfig.Password != payload.Password) {
							taskPatch.ExportPassword = config.ExportDataConfig.Password
							doUpdate = true
						}
						return nil
					}(); err != nil {
						return nil, err
					}

					// version
					if err := func() error {
						switch task.Type {
						case api.TaskDatabaseSchemaBaseline, api.TaskDatabaseSchemaUpdate, api.TaskDatabaseSchemaUpdateSDL, api.TaskDatabaseSchemaUpdateGhostSync, api.TaskDatabaseDataUpdate:
						default:
							return nil
						}
						var taskPayload struct {
							SchemaVersion string `json:"schemaVersion"`
						}
						if err := json.Unmarshal([]byte(task.Payload), &taskPayload); err != nil {
							return errors.Wrapf(err, "failed to unmarshal task payload")
						}
						if v := spec.GetChangeDatabaseConfig().GetSchemaVersion(); v != "" && v != taskPayload.SchemaVersion {
							taskPatch.SchemaVersion = &v
							doUpdate = true
						}
						return nil
					}(); err != nil {
						return nil, err
					}

					// task dag
					if err := func() error {
						oldSpec, ok := oldSpecsByID[taskSpecID.SpecID]
						if !ok {
							return nil
						}

						sort.Slice(oldSpec.DependsOnSpecs, func(i, j int) bool {
							return oldSpec.DependsOnSpecs[i] < oldSpec.DependsOnSpecs[j]
						})
						sort.Slice(spec.DependsOnSpecs, func(i, j int) bool {
							return spec.DependsOnSpecs[i] < spec.DependsOnSpecs[j]
						})
						if slices.Equal(oldSpec.GetDependsOnSpecs(), spec.GetDependsOnSpecs()) {
							return nil
						}

						// rebuild the task dag
						var fromTaskIDs []int
						for _, dependsOnSpec := range spec.GetDependsOnSpecs() {
							for _, dependsOnTask := range tasksBySpecID[dependsOnSpec] {
								fromTaskIDs = append(fromTaskIDs, dependsOnTask.ID)
							}
						}
						taskDAGRebuildList = append(taskDAGRebuildList, struct {
							fromTaskIDs []int
							toTaskID    int
						}{
							fromTaskIDs: fromTaskIDs,
							toTaskID:    task.ID,
						})
						return nil
					}(); err != nil {
						return nil, err
					}

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
						return nil, status.Errorf(codes.Internal, "failed to get issue: %v", err)
					}
					if issue != nil {
						// Do not allow to update task if issue is done or canceled.
						if issue.Status == api.IssueDone || issue.Status == api.IssueCanceled {
							return nil, status.Errorf(codes.FailedPrecondition, "cannot update task because issue %q is %s", issue.Title, issue.Status)
						}
					}
				}
			}

			for _, taskPatch := range taskPatchList {
				if taskPatch.SheetID != nil || taskPatch.EarliestAllowedTs != nil {
					task := tasksMap[taskPatch.ID]
					if task.LatestTaskRunStatus == api.TaskRunPending || task.LatestTaskRunStatus == api.TaskRunRunning || task.LatestTaskRunStatus == api.TaskRunSkipped || task.LatestTaskRunStatus == api.TaskRunDone {
						return nil, status.Errorf(codes.FailedPrecondition, "cannot update plan because task %q is %s", task.Name, task.LatestTaskRunStatus)
					}
				}
			}

			var doUpdateSheet bool
			for _, taskPatch := range taskPatchList {
				// If pre-backup detail has been updated, we need to rerun the plan check runs.
				if taskPatch.PreUpdateBackupDetail != nil {
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
				return nil, status.Errorf(codes.FailedPrecondition, "modify statement is not allowed for project %s", project.Title)
			}

			for _, taskPatch := range taskPatchList {
				task := tasksMap[taskPatch.ID]
				if _, err := s.store.UpdateTaskV2(ctx, taskPatch); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to update task %q: %v", task.Name, err)
				}
			}

			for _, taskDAGRebuild := range taskDAGRebuildList {
				if err := s.store.RebuildTaskDAG(ctx, taskDAGRebuild.fromTaskIDs, taskDAGRebuild.toTaskID); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to rebuild task dag: %v", err)
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
						PayloadUpsert: &storepb.IssuePayload{
							Approval: &storepb.IssuePayloadApproval{
								ApprovalFindingDone: false,
							},
						},
					}, api.SystemBotID)
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
			return nil, status.Errorf(codes.InvalidArgument, "invalid update_mask path %q", path)
		}
	}

	if err := s.store.UpdatePlan(ctx, planUpdate); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update plan %q: %v", request.Plan.Name, err)
	}

	updatedPlan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &oldPlan.UID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated plan %q: %v", request.Plan.Name, err)
	}
	if updatedPlan == nil {
		return nil, status.Errorf(codes.NotFound, "updated plan %q not found", request.Plan.Name)
	}

	if planCheckRunsTrigger {
		planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, updatedPlan)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
		}
		if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
		}
	}

	convertedPlan, err := convertToPlan(ctx, s.store, updatedPlan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to plan, error: %v", err)
	}
	return convertedPlan, nil
}

// ListPlanCheckRuns lists plan check runs for the plan.
func (s *PlanService) ListPlanCheckRuns(ctx context.Context, request *v1pb.ListPlanCheckRunsRequest) (*v1pb.ListPlanCheckRunsResponse, error) {
	projectID, planUID, err := common.GetProjectIDPlanID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		PlanUID:    &planUID,
		LatestOnly: request.LatestOnly,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list plan check runs, error: %v", err)
	}
	converted, err := convertToPlanCheckRuns(ctx, s.store, projectID, planUID, planCheckRuns)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert plan check runs, error: %v", err)
	}

	return &v1pb.ListPlanCheckRunsResponse{
		PlanCheckRuns: converted,
		NextPageToken: "",
	}, nil
}

// RunPlanChecks runs plan checks for a plan.
func (s *PlanService) RunPlanChecks(ctx context.Context, request *v1pb.RunPlanChecksRequest) (*v1pb.RunPlanChecksResponse, error) {
	projectID, planID, err := common.GetProjectIDPlanID(request.Name)
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
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found")
	}

	planCheckRuns, err := getPlanCheckRunsFromPlan(ctx, s.store, plan)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan check runs for plan, error: %v", err)
	}
	if err := s.store.CreatePlanCheckRuns(ctx, planCheckRuns...); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan check runs, error: %v", err)
	}

	// Tickle plan check scheduler.
	s.stateCfg.PlanCheckTickleChan <- 0

	return &v1pb.RunPlanChecksResponse{}, nil
}

// BatchCancelPlanCheckRuns cancels a list of plan check runs.
func (s *PlanService) BatchCancelPlanCheckRuns(ctx context.Context, request *v1pb.BatchCancelPlanCheckRunsRequest) (*v1pb.BatchCancelPlanCheckRunsResponse, error) {
	if len(request.PlanCheckRuns) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "plan check runs cannot be empty")
	}

	projectID, _, err := common.GetProjectIDPlanID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, error: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user, error: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %v not found", principalID)
	}

	var planCheckRunIDs []int
	for _, planCheckRun := range request.PlanCheckRuns {
		_, _, planCheckRunID, err := common.GetProjectIDPlanIDPlanCheckRunID(planCheckRun)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		planCheckRunIDs = append(planCheckRunIDs, planCheckRunID)
	}

	planCheckRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		UIDs: &planCheckRunIDs,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list plan check runs, error: %v", err)
	}

	// Check if any of the given plan check runs are not running.
	for _, planCheckRun := range planCheckRuns {
		switch planCheckRun.Status {
		case store.PlanCheckRunStatusRunning:
		default:
			return nil, status.Errorf(codes.InvalidArgument, "planCheckRun %v(%v) is not running", planCheckRun.UID, planCheckRun.Type)
		}
	}
	// Cancel the plan check runs.
	for _, planCheckRun := range planCheckRuns {
		if cancelFunc, ok := s.stateCfg.RunningPlanCheckRunsCancelFunc.Load(planCheckRun.UID); ok {
			cancelFunc.(context.CancelFunc)()
		}
	}
	// Update the status of the plan check runs to canceled.
	if err := s.store.BatchCancelPlanCheckRuns(ctx, planCheckRunIDs, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch patch task run status to canceled, error: %v", err)
	}

	return &v1pb.BatchCancelPlanCheckRunsResponse{}, nil
}

func (s *PlanService) buildPlanFindWithFilter(ctx context.Context, planFind *store.FindPlanMessage, filter string) error {
	filters, err := ParseFilter(filter)
	if err != nil {
		return errors.Wrap(err, "failed to parse filter")
	}
	for _, spec := range filters {
		switch spec.Key {
		case "creator":
			if spec.Operator != ComparatorTypeEqual {
				return errors.New(`only support "=" operation for "creator" filter`)
			}
			user, err := s.getUserByIdentifier(ctx, spec.Value)
			if err != nil {
				return errors.Wrap(err, "failed to get user by identifier")
			}
			planFind.CreatorID = &user.ID
		case "create_time":
			if spec.Operator != ComparatorTypeGreaterEqual && spec.Operator != ComparatorTypeLessEqual {
				return errors.New(`only support ">=" and "<=" operation for "create_time" filter`)
			}
			t, err := time.Parse(time.RFC3339, spec.Value)
			if err != nil {
				return errors.Wrap(err, "failed to parse create_time value")
			}
			ts := t.Unix()
			if spec.Operator == ComparatorTypeGreaterEqual {
				planFind.CreatedTsAfter = &ts
			} else {
				planFind.CreatedTsBefore = &ts
			}
		case "has_pipeline":
			if spec.Operator != ComparatorTypeEqual {
				return errors.New(`only support "=" operation for "has_pipeline" filter`)
			}
			switch spec.Value {
			case "false":
				planFind.NoPipeline = true
			case "true":
			default:
				return errors.Errorf("invalid value %q for has_pipeline", spec.Value)
			}
		case "has_issue":
			if spec.Operator != ComparatorTypeEqual {
				return errors.New(`only support "=" operation for "has_issue" filter`)
			}
			switch spec.Value {
			case "false":
				planFind.NoIssue = true
			case "true":
			default:
				return errors.Errorf("invalid value %q for has_issue", spec.Value)
			}
		}
	}
	return nil
}

func (s *PlanService) getUserByIdentifier(ctx context.Context, identifier string) (*store.UserMessage, error) {
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

func (s *PlanService) PreviewPlan(ctx context.Context, request *v1pb.PreviewPlanRequest) (*v1pb.PreviewPlanResponse, error) {
	projectID, err := common.GetProjectID(request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get project from %v, err: %v", request.Project, err)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project, err: %v", err)
	}

	_, releaseUID, err := common.GetProjectReleaseUID(request.Release)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get releaseUID from %q, err: %v", request.Release, err)
	}
	release, err := s.store.GetRelease(ctx, releaseUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get release, err: %v", err)
	}
	if release == nil {
		return nil, status.Errorf(codes.NotFound, "release %q not found", request.Release)
	}

	allDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list databases, err: %v", err)
	}
	allDatabasesByName := map[string]*store.DatabaseMessage{}

	for _, db := range allDatabases {
		name := common.FormatDatabase(db.InstanceID, db.DatabaseName)
		allDatabasesByName[name] = db
	}

	var databaseTargets, databaseGroupTargets [][]string
	for _, target := range request.Targets {
		if instance, database, err := common.GetInstanceDatabaseID(target); err == nil {
			databaseTargets = append(databaseTargets, []string{instance, database})
		} else if projectID, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			databaseGroupTargets = append(databaseGroupTargets, []string{projectID, databaseGroupID})
		} else {
			return nil, status.Errorf(codes.InvalidArgument, "unknown target %v", target)
		}
	}

	databasesToDeploy := map[string]bool{}

	for _, databaseTarget := range databaseTargets {
		name := common.FormatDatabase(databaseTarget[0], databaseTarget[1])
		databasesToDeploy[name] = true
	}

	for _, databaseGroupTarget := range databaseGroupTargets {
		projectID, databaseGroupID := databaseGroupTarget[0], databaseGroupTarget[1]
		if projectID != project.ResourceID {
			return nil, status.Errorf(codes.InvalidArgument, "databaseGroup target projectID %q doesn't match the projectID of request.project %q", projectID, request.Project)
		}

		databaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ProjectUID: &project.UID, ResourceID: &databaseGroupID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database group %q", databaseGroupID)
		}
		if databaseGroup == nil {
			return nil, errors.Errorf("database group %q not found", databaseGroupID)
		}
		matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find matched databases")
		}
		for _, db := range matchedDatabases {
			name := common.FormatDatabase(db.InstanceID, db.DatabaseName)
			databasesToDeploy[name] = true
		}
	}

	response := &v1pb.PreviewPlanResponse{}

	var allSpecs []*v1pb.Plan_Spec
	for database := range databasesToDeploy {
		db, ok := allDatabasesByName[database]
		if !ok {
			continue
		}
		specs, ooo, abm, err := s.getSpecsForDatabase(ctx, db, release, request.Release, request.AllowOutOfOrder)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get specs for database, err: %v", err)
		}
		if len(ooo.Files) > 0 {
			response.OutOfOrderFiles = append(response.OutOfOrderFiles, ooo)
		}
		if len(abm.Files) > 0 {
			response.AppliedButModifiedFiles = append(response.AppliedButModifiedFiles, abm)
		}
		allSpecs = append(allSpecs, specs...)
	}

	response.Plan = &v1pb.Plan{
		Title: fmt.Sprintf("Preview plan for release %q", request.Release),
		Steps: []*v1pb.Plan_Step{
			{
				Title: "",
				Specs: allSpecs,
			},
		},
		ReleaseSource: &v1pb.Plan_ReleaseSource{
			Release: request.Release,
		},
	}

	return response, nil
}

func (s *PlanService) getSpecsForDatabase(ctx context.Context, database *store.DatabaseMessage, release *store.ReleaseMessage, releaseName string, allowOoo bool) ([]*v1pb.Plan_Spec, *v1pb.PreviewPlanResponse_DatabaseFiles, *v1pb.PreviewPlanResponse_DatabaseFiles, error) {
	revisions, err := s.store.ListRevisions(ctx, &store.FindRevisionMessage{DatabaseUID: &database.UID})
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to list revisions")
	}
	return getSpecs(database, revisions, release, releaseName, allowOoo)
}

func getSpecs(database *store.DatabaseMessage, revisions []*store.RevisionMessage, release *store.ReleaseMessage, releaseName string, allowOoo bool) ([]*v1pb.Plan_Spec, *v1pb.PreviewPlanResponse_DatabaseFiles, *v1pb.PreviewPlanResponse_DatabaseFiles, error) {
	var specs []*v1pb.Plan_Spec
	var outOfOrderFiles []string
	var modifiedFiles []string

	var lastVersion string
	revisionByVersion := map[string]*store.RevisionMessage{}

	for _, r := range revisions {
		if lastVersion == "" {
			lastVersion = r.Version
		} else if lastVersion < r.Version {
			lastVersion = r.Version
		}
		revisionByVersion[r.Version] = r
	}

	slices.SortFunc(release.Payload.Files, func(a, b *storepb.ReleasePayload_File) int {
		if a.Version < b.Version {
			return -1
		}
		if a.Version > b.Version {
			return 1
		}
		return 0
	})

	for _, file := range release.Payload.Files {
		r, ok := revisionByVersion[file.Version]
		if ok {
			// applied, so we will not deploy it.
			if r.Payload.SheetSha256 != file.SheetSha256 {
				// It's been modified! warn users.
				modifiedFiles = append(modifiedFiles, common.FormatReleaseFile(releaseName, file.Id))
			}
			continue
		}

		if lastVersion != "" && lastVersion >= file.Version {
			// out of order detected
			outOfOrderFiles = append(outOfOrderFiles, common.FormatReleaseFile(releaseName, file.Id))

			// allowOutOfOrder=false
			// continue so that we don't add it into the plan.
			if !allowOoo {
				continue
			}
		}

		spec := &v1pb.Plan_Spec{
			Id: uuid.NewString(),
			SpecReleaseSource: &v1pb.Plan_SpecReleaseSource{
				File: common.FormatReleaseFile(releaseName, file.Id),
			},
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Type:          v1pb.Plan_ChangeDatabaseConfig_MIGRATE,
					Target:        common.FormatDatabase(database.InstanceID, database.DatabaseName),
					Sheet:         file.Sheet,
					SchemaVersion: file.Version,
				},
			},
		}
		specs = append(specs, spec)
	}

	return specs,
		&v1pb.PreviewPlanResponse_DatabaseFiles{
			Database: common.FormatDatabase(database.InstanceID, database.DatabaseName),
			Files:    outOfOrderFiles,
		},
		&v1pb.PreviewPlanResponse_DatabaseFiles{
			Database: common.FormatDatabase(database.InstanceID, database.DatabaseName),
			Files:    modifiedFiles,
		},
		nil
}
