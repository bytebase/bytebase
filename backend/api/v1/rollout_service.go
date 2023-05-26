package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/taskcheck"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RolloutService represents a service for managing rollout.
type RolloutService struct {
	v1pb.UnimplementedRolloutServiceServer
	store              *store.Store
	licenseService     enterpriseAPI.LicenseService
	dbFactory          *dbfactory.DBFactory
	taskScheduler      *taskrun.Scheduler
	taskCheckScheduler *taskcheck.Scheduler
	stateCfg           *state.State
	activityManager    *activity.Manager
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, taskScheduler *taskrun.Scheduler, taskCheckScheduler *taskcheck.Scheduler, stateCfg *state.State, activityManager *activity.Manager) *RolloutService {
	return &RolloutService{
		store:              store,
		licenseService:     licenseService,
		dbFactory:          dbFactory,
		taskScheduler:      taskScheduler,
		taskCheckScheduler: taskCheckScheduler,
		stateCfg:           stateCfg,
		activityManager:    activityManager,
	}
}

// GetPlan gets a plan.
func (s *RolloutService) GetPlan(ctx context.Context, request *v1pb.GetPlanRequest) (*v1pb.Plan, error) {
	planID, err := getPlanID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	plan, err := s.store.GetPlan(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan, error: %v", err)
	}
	if plan == nil {
		return nil, status.Errorf(codes.NotFound, "plan not found for id: %d", planID)
	}
	return convertToPlan(plan), nil
}

// CreatePlan creates a new plan.
func (s *RolloutService) CreatePlan(ctx context.Context, request *v1pb.CreatePlanRequest) (*v1pb.Plan, error) {
	creatorID := ctx.Value(common.PrincipalIDContextKey).(int)
	projectID, err := getProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
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

	pipelineCreate, err := s.getPipelineCreate(ctx, request.Plan.Steps, project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get pipeline create, error: %v", err)
	}
	if len(pipelineCreate.StageList) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no database matched for deployment")
	}
	firstEnvironmentID := pipelineCreate.StageList[0].EnvironmentID

	issueCreateMessage := &store.IssueMessage{
		Project:     project,
		Title:       request.Plan.Title,
		Type:        api.IssueDatabaseGeneral,
		Description: request.Plan.Description,
		Assignee:    nil,
	}

	// Find an assignee.
	assignee, err := s.taskScheduler.GetDefaultAssignee(ctx, firstEnvironmentID, issueCreateMessage.Project.UID, issueCreateMessage.Type)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find a default assignee, error: %v", err)
	}
	issueCreateMessage.Assignee = assignee

	issueCreatePayload := &storepb.IssuePayload{
		Approval: &storepb.IssuePayloadApproval{
			ApprovalFindingDone: false,
		},
	}
	if !s.licenseService.IsFeatureEnabled(api.FeatureCustomApproval) {
		issueCreatePayload.Approval.ApprovalFindingDone = true
	}

	issueCreatePayloadBytes, err := protojson.Marshal(issueCreatePayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal issue payload, error: %v", err)
	}
	issueCreateMessage.Payload = string(issueCreatePayloadBytes)

	pipeline, err := s.createPipeline(ctx, creatorID, pipelineCreate)
	if err != nil {
		return nil, err
	}
	issueCreateMessage.PipelineUID = &pipeline.ID
	issue, err := s.store.CreateIssueV2(ctx, issueCreateMessage, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create issue, error: %v", err)
	}
	composedIssue, err := s.store.GetIssueByID(ctx, issue.UID)
	if err != nil {
		return nil, err
	}

	if err := s.taskCheckScheduler.SchedulePipelineTaskCheck(ctx, pipeline.ID); err != nil {
		return nil, errors.Wrapf(err, "failed to schedule task check after creating the issue: %v", issue.Title)
	}

	s.stateCfg.ApprovalFinding.Store(issue.UID, issue)

	createActivityPayload := api.ActivityIssueCreatePayload{
		IssueName: issue.Title,
	}

	bytes, err := json.Marshal(createActivityPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: issue.UID,
		Type:        api.ActivityIssueCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: issue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create ActivityIssueCreate activity after creating the issue: %v", issue.Title)
	}

	if len(composedIssue.Pipeline.StageList) > 0 {
		stage := composedIssue.Pipeline.StageList[0]
		createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
			StageID:               stage.ID,
			StageStatusUpdateType: api.StageStatusUpdateTypeBegin,
			IssueName:             issue.Title,
			StageName:             stage.Name,
		}
		bytes, err := json.Marshal(createActivityPayload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create ActivityPipelineStageStatusUpdate activity after creating the issue: %v", issue.Title)
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: *issue.PipelineUID,
			Type:        api.ActivityPipelineStageStatusUpdate,
			Level:       api.ActivityInfo,
			Payload:     string(bytes),
		}
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to create ActivityPipelineStageStatusUpdate activity after creating the issue: %v", issue.Title)
		}
	}

	planMessage := &store.PlanMessage{
		ProjectID:   projectID,
		PipelineUID: nil,
		Name:        request.Plan.Title,
		Description: request.Plan.Description,
		Config: &storepb.PlanConfig{
			Steps: convertPlanSteps(request.Plan.Steps),
		},
	}
	planMessage.PipelineUID = &pipeline.ID

	plan, err := s.store.CreatePlan(ctx, planMessage, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan, error: %v", err)
	}
	return convertToPlan(plan), nil
}

func validateSteps(_ []*v1pb.Plan_Step) error {
	// FIXME: impl this func
	// targets should be unique
	// if deploymentConfig is used, only one spec is allowed.
	return nil
}

func (s *RolloutService) getPipelineCreate(ctx context.Context, steps []*v1pb.Plan_Step, project *store.ProjectMessage) (*api.PipelineCreate, error) {
	// FIXME: handle deploymentConfig
	pipelineCreate := &api.PipelineCreate{}
	for _, step := range steps {
		stageCreate := api.StageCreate{}

		specEnvironmentIDs := map[string]bool{}
		var stageEnvironmentID string
		registerEnvironmentID := func(environmentID string) error {
			if stageEnvironmentID == "" {
				stageEnvironmentID = environmentID
				return nil
			}
			if stageEnvironmentID != environmentID {
				return errors.Errorf("expect only one environment in a stage, got %s and %s", stageEnvironmentID, environmentID)
			}
			return nil
		}

		for _, spec := range step.Specs {
			taskCreates, taskIndexDAGCreates, err := s.getTaskCreatesFromSpec(ctx, spec, project, registerEnvironmentID)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get task creates from spec")
			}

			stageCreate.TaskList = append(stageCreate.TaskList, taskCreates...)
			offset := len(stageCreate.TaskList)
			for i := range taskIndexDAGCreates {
				taskIndexDAGCreates[i].FromIndex += offset
				taskIndexDAGCreates[i].ToIndex += offset
			}
			stageCreate.TaskIndexDAGList = append(stageCreate.TaskIndexDAGList, taskIndexDAGCreates...)
		}

		if len(specEnvironmentIDs) != 1 {
			return nil, errors.Errorf("expect 1 environment in a step, got %d", len(specEnvironmentIDs))
		}

		var environmentID string
		for k := range specEnvironmentIDs {
			environmentID = k
		}

		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get environment")
		}
		stageCreate.EnvironmentID = environment.UID

		pipelineCreate.StageList = append(pipelineCreate.StageList, stageCreate)
	}
	return pipelineCreate, nil
}

func (s *RolloutService) getTaskCreatesFromSpec(ctx context.Context, spec *v1pb.Plan_Spec, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	if !s.licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) {
		if spec.EarliestAllowedTime != nil && !spec.EarliestAllowedTime.AsTime().IsZero() {
			return nil, nil, errors.Errorf(api.FeatureTaskScheduleTime.AccessErrorMessage())
		}
	}

	switch config := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s.store, s.licenseService, s.dbFactory, spec, config.CreateDatabaseConfig, project, registerEnvironmentID)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s.store, spec, config.ChangeDatabaseConfig, project, registerEnvironmentID)
	case *v1pb.Plan_Spec_RestoreDatabaseConfig:
		return getTaskCreatesFromRestoreDatabaseConfig(ctx, s.store, s.licenseService, s.dbFactory, spec, config.RestoreDatabaseConfig, project, registerEnvironmentID)
	}

	return nil, nil, errors.Errorf("invalid spec config type %T", spec.Config)
}

func getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, spec *v1pb.Plan_Spec, c *v1pb.Plan_CreateDatabaseConfig, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	if c.Database == "" {
		return nil, nil, errors.Errorf("database name is required")
	}
	instanceID, err := getInstanceID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance id from %q", c.Target)
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, nil, err
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance ID not found %v", instanceID)
	}
	if instance.Engine == db.Oracle {
		return nil, nil, errors.Errorf("creating Oracle database is not supported")
	}
	environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, nil, err
	}
	if environment == nil {
		return nil, nil, errors.Errorf("environment ID not found %v", instance.EnvironmentID)
	}

	if err := registerEnvironmentID(environment.ResourceID); err != nil {
		return nil, nil, err
	}

	if instance.Engine == db.MongoDB && c.Table == "" {
		return nil, nil, errors.Errorf("collection name is required for MongoDB")
	}

	taskCreates, err := func() ([]api.TaskCreate, error) {
		if err := checkCharacterSetCollationOwner(instance.Engine, c.CharacterSet, c.Collation, c.Owner); err != nil {
			return nil, err
		}
		if c.Database == "" {
			return nil, errors.Errorf("database name is required")

		}
		if instance.Engine == db.Snowflake {
			// Snowflake needs to use upper case of DatabaseName.
			c.Database = strings.ToUpper(c.Database)
		}
		if instance.Engine == db.MongoDB && c.Table == "" {
			return nil, common.Errorf(common.Invalid, "Failed to create issue, collection name missing for MongoDB")
		}
		// Validate the labels. Labels are set upon task completion.
		labelsJSON, err := convertDatabaseLabels(c.Labels)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid database label %q", c.Labels)
		}

		// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
		if project.TenantMode == api.TenantModeTenant {
			if !licenseService.IsFeatureEnabled(api.FeatureMultiTenancy) {
				return nil, errors.Errorf(api.FeatureMultiTenancy.AccessErrorMessage())
			}
		}

		// Get admin data source username.
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		if adminDataSource == nil {
			return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
		}
		databaseName := c.Database
		switch instance.Engine {
		case db.Snowflake:
			// Snowflake needs to use upper case of DatabaseName.
			databaseName = strings.ToUpper(databaseName)
		case db.MySQL, db.MariaDB, db.OceanBase:
			// For MySQL, we need to use different case of DatabaseName depends on the variable `lower_case_table_names`.
			// https://dev.mysql.com/doc/refman/8.0/en/identifier-case-sensitivity.html
			// And also, meet an error in here is not a big deal, we will just use the original DatabaseName.
			driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
			if err != nil {
				log.Warn("failed to get admin database driver for instance %q, please check the connection for admin data source", zap.Error(err), zap.String("instance", instance.Title))
				break
			}
			defer driver.Close(ctx)
			var lowerCaseTableNames int
			var unused any
			db := driver.GetDB()
			if err := db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'lower_case_table_names'").Scan(&unused, &lowerCaseTableNames); err != nil {
				log.Warn("failed to get lower_case_table_names for instance %q", zap.Error(err), zap.String("instance", instance.Title))
				break
			}
			if lowerCaseTableNames == 1 {
				databaseName = strings.ToLower(databaseName)
			}
		}

		statement, err := getCreateDatabaseStatement(instance.Engine, c, databaseName, adminDataSource.Username)
		if err != nil {
			return nil, err
		}
		sheet, err := s.CreateSheet(ctx, &api.SheetCreate{
			CreatorID:  api.SystemBotID,
			ProjectID:  project.UID,
			Name:       fmt.Sprintf("Sheet for creating database %v", databaseName),
			Statement:  statement,
			Visibility: api.ProjectSheet,
			Source:     api.SheetFromBytebaseArtifact,
			Type:       api.SheetForSQL,
			Payload:    "{}",
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation sheet")
		}

		payload := api.TaskDatabaseCreatePayload{
			SpecID:       spec.Id,
			ProjectID:    project.UID,
			CharacterSet: c.CharacterSet,
			TableName:    c.Table,
			Collation:    c.Collation,
			Labels:       labelsJSON,
			DatabaseName: databaseName,
			SheetID:      sheet.ID,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation task, unable to marshal payload")
		}

		return []api.TaskCreate{
			{
				InstanceID:        instance.UID,
				Name:              fmt.Sprintf("Create database %v", payload.DatabaseName),
				Status:            api.TaskPendingApproval,
				Type:              api.TaskDatabaseCreate,
				DatabaseName:      payload.DatabaseName,
				Payload:           string(bytes),
				EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			},
		}, nil
	}()

	if err != nil {
		return nil, nil, err
	}

	return taskCreates, nil, nil
}

func getTaskCreatesFromChangeDatabaseConfig(ctx context.Context, s *store.Store, spec *v1pb.Plan_Spec, c *v1pb.Plan_ChangeDatabaseConfig, _ *store.ProjectMessage, registerEnvironmentID func(string) error) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	// possible target:
	// 1. instances/{instance}/databases/{database}
	instanceID, databaseName, err := getInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance database id from target %q", c.Target)
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance %q not found", instanceID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database %q not found", databaseName)
	}

	if err := registerEnvironmentID(database.EnvironmentID); err != nil {
		return nil, nil, err
	}

	switch c.Type {
	case v1pb.Plan_ChangeDatabaseConfig_BASELINE:
		payload := api.TaskDatabaseSchemaBaselinePayload{
			SpecID:        spec.Id,
			SchemaVersion: c.SchemaVersion,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal task database schema baseline payload")
		}
		payloadString := string(bytes)
		taskCreate := api.TaskCreate{
			Name:              fmt.Sprintf("Establish baseline for database %q", database),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaBaseline,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []api.TaskCreate{taskCreate}, nil, nil

	case v1pb.Plan_ChangeDatabaseConfig_MIGRATE:
		_, sheetIDStr, err := getProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		sheet, err := s.GetSheetV2(ctx, &api.SheetFind{ID: &sheetID}, api.SystemBotID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet %q", sheetID)
		}
		if sheet == nil {
			return nil, nil, errors.Errorf("sheet %q not found", sheetID)
		}
		payload := api.TaskDatabaseSchemaUpdatePayload{
			SpecID:        spec.Id,
			SheetID:       sheetID,
			SchemaVersion: c.SchemaVersion,
			VCSPushEvent:  nil,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal task database schema update payload")
		}
		payloadString := string(bytes)
		taskCreate := api.TaskCreate{
			Name:              fmt.Sprintf("DDL(schema) for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdate,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []api.TaskCreate{taskCreate}, nil, nil

	case v1pb.Plan_ChangeDatabaseConfig_MIGRATE_SDL:
		_, sheetIDStr, err := getProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		sheet, err := s.GetSheetV2(ctx, &api.SheetFind{ID: &sheetID}, api.SystemBotID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet %q", sheetID)
		}
		if sheet == nil {
			return nil, nil, errors.Errorf("sheet %q not found", sheetID)
		}
		payload := api.TaskDatabaseSchemaUpdateSDLPayload{
			SheetID:       sheetID,
			SchemaVersion: c.SchemaVersion,
			VCSPushEvent:  nil,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update SDL payload")
		}
		payloadString := string(bytes)
		taskCreate := api.TaskCreate{
			Name:              fmt.Sprintf("SDL for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdateSDL,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []api.TaskCreate{taskCreate}, nil, nil

	case v1pb.Plan_ChangeDatabaseConfig_MIGRATE_GHOST:
		_, sheetIDStr, err := getProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		sheet, err := s.GetSheetV2(ctx, &api.SheetFind{ID: &sheetID}, api.SystemBotID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet %q", sheetID)
		}
		if sheet == nil {
			return nil, nil, errors.Errorf("sheet %q not found", sheetID)
		}
		var taskCreateList []api.TaskCreate
		// task "sync"
		payloadSync := api.TaskDatabaseSchemaUpdateGhostSyncPayload{
			SpecID:        spec.Id,
			SheetID:       sheetID,
			SchemaVersion: c.SchemaVersion,
			VCSPushEvent:  nil,
		}
		bytesSync, err := json.Marshal(payloadSync)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update gh-ost sync payload")
		}
		taskCreateList = append(taskCreateList, api.TaskCreate{
			Name:              fmt.Sprintf("Update schema gh-ost sync for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdateGhostSync,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           string(bytesSync),
		})

		// task "cutover"
		payloadCutover := api.TaskDatabaseSchemaUpdateGhostCutoverPayload{
			SpecID: spec.Id,
		}
		bytesCutover, err := json.Marshal(payloadCutover)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update ghost cutover payload")
		}
		taskCreateList = append(taskCreateList, api.TaskCreate{
			Name:              fmt.Sprintf("Update schema gh-ost cutover for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseSchemaUpdateGhostCutover,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           string(bytesCutover),
		})

		// The below list means that taskCreateList[0] blocks taskCreateList[1].
		// In other words, task "sync" blocks task "cutover".
		taskIndexDAGList := []api.TaskIndexDAG{
			{FromIndex: 0, ToIndex: 1},
		}
		return taskCreateList, taskIndexDAGList, nil

	case v1pb.Plan_ChangeDatabaseConfig_DATA:
		_, sheetIDStr, err := getProjectResourceIDSheetID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		sheetID, err := strconv.Atoi(sheetIDStr)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert sheet id %q to int", sheetIDStr)
		}
		sheet, err := s.GetSheetV2(ctx, &api.SheetFind{ID: &sheetID}, api.SystemBotID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet %q", sheetID)
		}
		if sheet == nil {
			return nil, nil, errors.Errorf("sheet %q not found", sheetID)
		}
		payload := api.TaskDatabaseDataUpdatePayload{
			SheetID:           sheetID,
			SchemaVersion:     c.SchemaVersion,
			VCSPushEvent:      nil,
			RollbackEnabled:   c.RollbackEnabled,
			RollbackSQLStatus: api.RollbackSQLStatusPending,
		}
		if c.RollbackDetail != nil {
			reviewID, err := getReviewID(c.RollbackDetail.RollbackFromReview)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get review id from review %q", c.RollbackDetail.RollbackFromReview)
			}
			payload.RollbackFromIssueID = reviewID
			taskID, err := getTaskID(c.RollbackDetail.RollbackFromTask)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get task id from task %q", c.RollbackDetail.RollbackFromTask)
			}
			payload.RollbackFromTaskID = taskID
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to marshal database data update payload")
		}
		payloadString := string(bytes)
		taskCreate := api.TaskCreate{
			Name:              fmt.Sprintf("DML(data) for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Status:            api.TaskPendingApproval,
			Type:              api.TaskDatabaseDataUpdate,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []api.TaskCreate{taskCreate}, nil, nil
	default:
		return nil, nil, errors.Errorf("unsupported change database config type %q", c.Type)
	}
}

func getTaskCreatesFromRestoreDatabaseConfig(ctx context.Context, s *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory, spec *v1pb.Plan_Spec, c *v1pb.Plan_RestoreDatabaseConfig, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	if c.Source == nil {
		return nil, nil, errors.Errorf("missing source in restore database config")
	}
	instanceID, databaseName, err := getInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance and database id from target %q", c.Target)
	}
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance %q not found", instanceID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database %q not found", databaseName)
	}
	if database.ProjectID != project.ResourceID {
		return nil, nil, errors.Errorf("database %q is not in project %q", databaseName, project.ResourceID)
	}

	if err := registerEnvironmentID(database.EnvironmentID); err != nil {
		return nil, nil, err
	}

	var taskCreates []api.TaskCreate

	if c.CreateDatabaseConfig != nil {
		restorePayload := api.TaskDatabasePITRRestorePayload{
			ProjectID: project.UID,
		}
		// restore to a new database
		targetInstanceID, err := getInstanceID(c.CreateDatabaseConfig.Target)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get instance id from %q", c.CreateDatabaseConfig.Target)
		}
		targetInstance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &targetInstanceID})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find the instance with ID %q", targetInstanceID)
		}

		// task 1: create the database
		createDatabaseTasks, _, err := getTaskCreatesFromCreateDatabaseConfig(ctx, s, licenseService, dbFactory, spec, c.CreateDatabaseConfig, project, registerEnvironmentID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create the database create task list")
		}
		if len(createDatabaseTasks) != 1 {
			return nil, nil, errors.Errorf("expected 1 task to create the database, got %d", len(createDatabaseTasks))
		}
		taskCreates = append(taskCreates, createDatabaseTasks[0])

		// task 2: restore the database
		switch source := c.Source.(type) {
		case *v1pb.Plan_RestoreDatabaseConfig_Backup:
			backupInstanceID, backupDatabaseName, backupName, err := getInstanceDatabaseIDBackupName(source.Backup)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to parse backup name %q", source.Backup)
			}
			backupDatabase, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &backupInstanceID,
				DatabaseName: &backupDatabaseName,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get database %q", backupDatabaseName)
			}
			if backupDatabase == nil {
				return nil, nil, errors.Errorf("failed to find database %q where backup %q is created", backupDatabaseName, source.Backup)
			}
			backup, err := s.GetBackupV2(ctx, &store.FindBackupMessage{
				DatabaseUID: &backupDatabase.UID,
				Name:        &backupName,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get backup %q", backupName)
			}
			if backup == nil {
				return nil, nil, errors.Errorf("failed to find backup %q", backupName)
			}
			restorePayload.BackupID = &backup.UID
		case *v1pb.Plan_RestoreDatabaseConfig_PointInTime:
			ts := source.PointInTime.GetSeconds()
			restorePayload.PointInTimeTs = &ts
		}
		restorePayload.TargetInstanceID = &targetInstance.UID
		restorePayload.DatabaseName = &c.CreateDatabaseConfig.Database

		restorePayloadBytes, err := json.Marshal(restorePayload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR restore task, unable to marshal payload")
		}

		restoreTaskCreate := api.TaskCreate{
			Name:       fmt.Sprintf("Restore to new database %q", *restorePayload.DatabaseName),
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRRestore,
			InstanceID: instance.UID,
			DatabaseID: &database.UID,
			Payload:    string(restorePayloadBytes),
		}
		taskCreates = append(taskCreates, restoreTaskCreate)
	} else {
		// in-place restore

		// task 1: restore
		restorePayload := api.TaskDatabasePITRRestorePayload{
			ProjectID: project.UID,
		}
		switch source := c.Source.(type) {
		case *v1pb.Plan_RestoreDatabaseConfig_Backup:
			backupInstanceID, backupDatabaseName, backupName, err := getInstanceDatabaseIDBackupName(source.Backup)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to parse backup name %q", source.Backup)
			}
			backupDatabase, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &backupInstanceID,
				DatabaseName: &backupDatabaseName,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get database %q", backupDatabaseName)
			}
			if backupDatabase == nil {
				return nil, nil, errors.Errorf("failed to find database %q where backup %q is created", backupDatabaseName, source.Backup)
			}
			backup, err := s.GetBackupV2(ctx, &store.FindBackupMessage{
				DatabaseUID: &backupDatabase.UID,
				Name:        &backupName,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get backup %q", backupName)
			}
			if backup == nil {
				return nil, nil, errors.Errorf("failed to find backup %q", backupName)
			}
			restorePayload.BackupID = &backup.UID
		case *v1pb.Plan_RestoreDatabaseConfig_PointInTime:
			ts := source.PointInTime.GetSeconds()
			restorePayload.PointInTimeTs = &ts
		}
		restorePayloadBytes, err := json.Marshal(restorePayload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR restore task, unable to marshal payload")
		}

		restoreTaskCreate := api.TaskCreate{
			Name:       fmt.Sprintf("Restore to PITR database %q", database.DatabaseName),
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRRestore,
			InstanceID: instance.UID,
			DatabaseID: &database.UID,
			Payload:    string(restorePayloadBytes),
		}

		taskCreates = append(taskCreates, restoreTaskCreate)

		// task 2: cutover
		cutoverPayload := api.TaskDatabasePITRCutoverPayload{}
		cutoverPayloadBytes, err := json.Marshal(cutoverPayload)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create PITR cutover task, unable to marshal payload")
		}
		taskCreates = append(taskCreates, api.TaskCreate{
			Name:       fmt.Sprintf("Swap PITR and the original database %q", database.DatabaseName),
			InstanceID: instance.UID,
			DatabaseID: &database.UID,
			Status:     api.TaskPendingApproval,
			Type:       api.TaskDatabaseRestorePITRCutover,
			Payload:    string(cutoverPayloadBytes),
		})
	}

	// We make sure that we will always return 2 tasks.
	taskIndexDAGs := []api.TaskIndexDAG{
		{
			FromIndex: 0,
			ToIndex:   1,
		},
	}
	return taskCreates, taskIndexDAGs, nil
}

func convertToPlan(plan *store.PlanMessage) *v1pb.Plan {
	return &v1pb.Plan{
		Name:        fmt.Sprintf("%s%s/%s%d", projectNamePrefix, plan.ProjectID, planPrefix, plan.UID),
		Uid:         fmt.Sprintf("%d", plan.UID),
		Review:      "",
		Title:       plan.Name,
		Description: plan.Description,
		Steps:       convertToPlanSteps(plan.Config.Steps),
	}
}

func convertToPlanSteps(steps []*storepb.PlanConfig_Step) []*v1pb.Plan_Step {
	v1Steps := make([]*v1pb.Plan_Step, len(steps))
	for i := range steps {
		v1Steps[i] = convertToPlanStep(steps[i])
	}
	return v1Steps
}

func convertToPlanStep(step *storepb.PlanConfig_Step) *v1pb.Plan_Step {
	return &v1pb.Plan_Step{
		Specs: convertToPlanSpecs(step.Specs),
	}
}

func convertToPlanSpecs(specs []*storepb.PlanConfig_Spec) []*v1pb.Plan_Spec {
	v1Specs := make([]*v1pb.Plan_Spec, len(specs))
	for i := range specs {
		v1Specs[i] = convertToPlanSpec(specs[i])
	}
	return v1Specs
}

func convertToPlanSpec(spec *storepb.PlanConfig_Spec) *v1pb.Plan_Spec {
	v1Spec := &v1pb.Plan_Spec{
		EarliestAllowedTime: spec.EarliestAllowedTime,
		Id:                  spec.Id,
	}

	switch v := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		v1Spec.Config = convertToPlanSpecCreateDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		v1Spec.Config = convertToPlanSpecChangeDatabaseConfig(v)
	case *storepb.PlanConfig_Spec_RestoreDatabaseConfig:
		v1Spec.Config = convertToPlanSpecRestoreDatabaseConfig(v)
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
			Backup:       c.Backup,
			Labels:       c.Labels,
		},
	}
}

func convertToPlanCreateDatabaseConfig(c *storepb.PlanConfig_CreateDatabaseConfig) *v1pb.Plan_CreateDatabaseConfig {
	return &v1pb.Plan_CreateDatabaseConfig{
		Target:       c.Target,
		Database:     c.Database,
		Table:        c.Table,
		CharacterSet: c.CharacterSet,
		Collation:    c.Collation,
		Cluster:      c.Cluster,
		Owner:        c.Owner,
		Backup:       c.Backup,
		Labels:       c.Labels,
	}
}

func convertToPlanSpecChangeDatabaseConfig(config *storepb.PlanConfig_Spec_ChangeDatabaseConfig) *v1pb.Plan_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &v1pb.Plan_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
			Target:          c.Target,
			Sheet:           c.Sheet,
			Type:            v1pb.Plan_ChangeDatabaseConfig_Type(c.Type),
			SchemaVersion:   c.SchemaVersion,
			RollbackEnabled: c.RollbackEnabled,
		},
	}
}

func convertToPlanSpecRestoreDatabaseConfig(config *storepb.PlanConfig_Spec_RestoreDatabaseConfig) *v1pb.Plan_Spec_RestoreDatabaseConfig {
	c := config.RestoreDatabaseConfig
	v1Config := &v1pb.Plan_Spec_RestoreDatabaseConfig{
		RestoreDatabaseConfig: &v1pb.Plan_RestoreDatabaseConfig{
			Target: c.Target,
		},
	}
	switch source := c.Source.(type) {
	case *storepb.PlanConfig_RestoreDatabaseConfig_Backup:
		v1Config.RestoreDatabaseConfig.Source = &v1pb.Plan_RestoreDatabaseConfig_Backup{
			Backup: source.Backup,
		}
	case *storepb.PlanConfig_RestoreDatabaseConfig_PointInTime:
		v1Config.RestoreDatabaseConfig.Source = &v1pb.Plan_RestoreDatabaseConfig_PointInTime{
			PointInTime: source.PointInTime,
		}
	}
	// c.CreateDatabaseConfig is defined as optional in proto
	// so we need to test if it's nil
	if c.CreateDatabaseConfig != nil {
		v1Config.RestoreDatabaseConfig.CreateDatabaseConfig = convertToPlanCreateDatabaseConfig(c.CreateDatabaseConfig)
	}
	return v1Config
}

func convertPlanSteps(steps []*v1pb.Plan_Step) []*storepb.PlanConfig_Step {
	storeSteps := make([]*storepb.PlanConfig_Step, len(steps))
	for i := range steps {
		storeSteps[i] = convertPlanStep(steps[i])
	}
	return storeSteps
}

func convertPlanStep(step *v1pb.Plan_Step) *storepb.PlanConfig_Step {
	return &storepb.PlanConfig_Step{
		Specs: convertPlanSpecs(step.Specs),
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
		EarliestAllowedTime: spec.EarliestAllowedTime,
		Id:                  spec.Id,
	}

	switch v := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		storeSpec.Config = convertPlanSpecCreateDatabaseConfig(v)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		storeSpec.Config = convertPlanSpecChangeDatabaseConfig(v)
	case *v1pb.Plan_Spec_RestoreDatabaseConfig:
		storeSpec.Config = convertPlanSpecRestoreDatabaseConfig(v)
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
		Backup:       c.Backup,
		Labels:       c.Labels,
	}
}

func convertPlanSpecChangeDatabaseConfig(config *v1pb.Plan_Spec_ChangeDatabaseConfig) *storepb.PlanConfig_Spec_ChangeDatabaseConfig {
	c := config.ChangeDatabaseConfig
	return &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
		ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
			Target:          c.Target,
			Sheet:           c.Sheet,
			Type:            storepb.PlanConfig_ChangeDatabaseConfig_Type(c.Type),
			SchemaVersion:   c.SchemaVersion,
			RollbackEnabled: c.RollbackEnabled,
		},
	}
}

func convertPlanSpecRestoreDatabaseConfig(config *v1pb.Plan_Spec_RestoreDatabaseConfig) *storepb.PlanConfig_Spec_RestoreDatabaseConfig {
	c := config.RestoreDatabaseConfig
	storeConfig := &storepb.PlanConfig_Spec_RestoreDatabaseConfig{
		RestoreDatabaseConfig: &storepb.PlanConfig_RestoreDatabaseConfig{
			Target: c.Target,
		},
	}
	switch source := c.Source.(type) {
	case *v1pb.Plan_RestoreDatabaseConfig_Backup:
		storeConfig.RestoreDatabaseConfig.Source = &storepb.PlanConfig_RestoreDatabaseConfig_Backup{
			Backup: source.Backup,
		}
	case *v1pb.Plan_RestoreDatabaseConfig_PointInTime:
		storeConfig.RestoreDatabaseConfig.Source = &storepb.PlanConfig_RestoreDatabaseConfig_PointInTime{
			PointInTime: source.PointInTime,
		}
	}
	// c.CreateDatabaseConfig is defined as optional in proto
	// so we need to test if it's nil
	if c.CreateDatabaseConfig != nil {
		storeConfig.RestoreDatabaseConfig.CreateDatabaseConfig = convertPlanConfigCreateDatabaseConfig(c.CreateDatabaseConfig)
	}
	return storeConfig
}

// checkCharacterSetCollationOwner checks if the character set, collation and owner are legal according to the dbType.
func checkCharacterSetCollationOwner(dbType db.Type, characterSet, collation, owner string) error {
	switch dbType {
	case db.Spanner:
		// Spanner does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("Spanner does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Spanner does not support collation, but got %s", collation)
		}
	case db.ClickHouse:
		// ClickHouse does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("ClickHouse does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("ClickHouse does not support collation, but got %s", collation)
		}
	case db.Snowflake:
		if characterSet != "" {
			return errors.Errorf("Snowflake does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Snowflake does not support collation, but got %s", collation)
		}
	case db.Postgres:
		if owner == "" {
			return errors.Errorf("database owner is required for PostgreSQL")
		}
	case db.Redshift:
		if owner == "" {
			return errors.Errorf("database owner is required for Redshift")
		}
	case db.SQLite, db.MongoDB, db.MSSQL:
		// no-op.
	default:
		if characterSet == "" {
			return errors.Errorf("character set missing for %s", string(dbType))
		}
		// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
		// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
		// install it.
		if collation == "" {
			return errors.Errorf("collation missing for %s", string(dbType))
		}
	}
	return nil
}

// convertDatabaseLabels converts the map[string]string labels to []*api.DatabaseLabel JSON string.
func convertDatabaseLabels(labelsMap map[string]string) (string, error) {
	if len(labelsMap) == 0 {
		return "", nil
	}
	// For scalability, each database can have up to four labels for now.
	if len(labelsMap) > api.DatabaseLabelSizeMax {
		return "", errors.Errorf("database labels are up to a maximum of %d", api.DatabaseLabelSizeMax)
	}
	var labels []*api.DatabaseLabel
	for k, v := range labelsMap {
		labels = append(labels, &api.DatabaseLabel{
			Key:   k,
			Value: v,
		})
	}
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal labels json")
	}
	return string(labelsJSON), nil
}

func getCreateDatabaseStatement(dbType db.Type, c *v1pb.Plan_CreateDatabaseConfig, databaseName, adminDatasourceUser string) (string, error) {
	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		return fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, c.CharacterSet, c.Collation), nil
	case db.MSSQL:
		return fmt.Sprintf(`CREATE DATABASE "%s";`, databaseName), nil
	case db.Postgres:
		// On Cloud RDS, the data source role isn't the actual superuser with sudo privilege.
		// We need to grant the database owner role to the data source admin so that Bytebase can have permission for the database using the data source admin.
		if adminDatasourceUser != "" && c.Owner != adminDatasourceUser {
			stmt = fmt.Sprintf("GRANT \"%s\" TO \"%s\";\n", c.Owner, adminDatasourceUser)
		}
		if c.Collation == "" {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q;", stmt, databaseName, c.CharacterSet)
		} else {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q;", stmt, databaseName, c.CharacterSet, c.Collation)
		}
		// Set the database owner.
		// We didn't use CREATE DATABASE WITH OWNER because RDS requires the current role to be a member of the database owner.
		// However, people can still use ALTER DATABASE to change the owner afterwards.
		// Error string below:
		// query: CREATE DATABASE h1 WITH OWNER hello;
		// ERROR:  must be member of role "hello"
		//
		// For tenant project, the schema for the newly created database will belong to the same owner.
		// TODO(d): alter schema "public" owner to the database owner.
		return fmt.Sprintf("%s\nALTER DATABASE \"%s\" OWNER TO \"%s\";", stmt, databaseName, c.Owner), nil
	case db.ClickHouse:
		clusterPart := ""
		if c.Cluster != "" {
			clusterPart = fmt.Sprintf(" ON CLUSTER `%s`", c.Cluster)
		}
		return fmt.Sprintf("CREATE DATABASE `%s`%s;", databaseName, clusterPart), nil
	case db.Snowflake:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.SQLite:
		// This is a fake CREATE DATABASE and USE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		return fmt.Sprintf("CREATE DATABASE '%s';", databaseName), nil
	case db.MongoDB:
		// We just run createCollection in mongosh instead of execute `use <database>` first, because we execute the
		// mongodb statement in mongosh with --file flag, and it doesn't support `use <database>` statement in the file.
		// And we pass the database name to Bytebase engine driver, which will be used to build the connection string.
		return fmt.Sprintf(`db.createCollection("%s");`, c.Table), nil
	case db.Spanner:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.Oracle:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.Redshift:
		options := make(map[string]string)
		if adminDatasourceUser != "" && c.Owner != adminDatasourceUser {
			options["OWNER"] = fmt.Sprintf("%q", c.Owner)
		}
		stmt := fmt.Sprintf("CREATE DATABASE \"%s\"", databaseName)
		if len(options) > 0 {
			list := make([]string, 0, len(options))
			for k, v := range options {
				list = append(list, fmt.Sprintf("%s=%s", k, v))
			}
			stmt = fmt.Sprintf("%s WITH\n\t%s", stmt, strings.Join(list, "\n\t"))
		}
		return fmt.Sprintf("%s;", stmt), nil
	}
	return "", errors.Errorf("unsupported database type %s", dbType)
}

func (s *RolloutService) createPipeline(ctx context.Context, creatorID int, pipelineCreate *api.PipelineCreate) (*store.PipelineMessage, error) {
	pipelineCreated, err := s.store.CreatePipelineV2(ctx, &store.PipelineMessage{Name: pipelineCreate.Name}, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create pipeline for issue")
	}

	var stageCreates []*store.StageMessage
	for _, stage := range pipelineCreate.StageList {
		stageCreates = append(stageCreates, &store.StageMessage{
			Name:          stage.Name,
			EnvironmentID: stage.EnvironmentID,
			PipelineID:    pipelineCreated.ID,
		})
	}
	createdStages, err := s.store.CreateStageV2(ctx, stageCreates, creatorID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create stages for issue")
	}
	if len(createdStages) != len(stageCreates) {
		return nil, errors.Errorf("failed to create stages, expect to have created %d stages, got %d", len(stageCreates), len(createdStages))
	}

	for i, stageCreate := range pipelineCreate.StageList {
		createdStage := createdStages[i]

		var taskCreateList []*api.TaskCreate
		for _, taskCreate := range stageCreate.TaskList {
			c := taskCreate
			c.CreatorID = creatorID
			c.PipelineID = pipelineCreated.ID
			c.StageID = createdStage.ID
			taskCreateList = append(taskCreateList, &c)
		}
		tasks, err := s.store.CreateTasksV2(ctx, taskCreateList...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create tasks for issue")
		}

		// TODO(p0ny): create task dags in batch.
		for _, indexDAG := range stageCreate.TaskIndexDAGList {
			if err := s.store.CreateTaskDAGV2(ctx, &store.TaskDAGMessage{
				FromTaskID: tasks[indexDAG.FromIndex].ID,
				ToTaskID:   tasks[indexDAG.ToIndex].ID,
			}); err != nil {
				return nil, errors.Wrap(err, "failed to create task DAG for issue")
			}
		}
	}

	return pipelineCreated, nil
}
