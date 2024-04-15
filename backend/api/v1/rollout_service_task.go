package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"log/slog"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getPipelineCreateFromDatabaseGroupTarget(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, project *store.ProjectMessage) (*store.PipelineMessage, error) {
	switch c.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
	default:
		return nil, errors.Errorf("unsupported change database config type %q for database group target", c.Type)
	}

	projectID, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(c.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project id and database group id from target %q", c.Target)
	}
	if projectID != project.ResourceID {
		return nil, errors.Errorf("project id %q in target %q does not match project id %q in plan config", projectID, c.Target, project.ResourceID)
	}
	databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ProjectUID: &project.UID, ResourceID: &databaseGroupID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %q", databaseGroupID)
	}
	if databaseGroup == nil {
		return nil, errors.Errorf("database group %q not found", databaseGroupID)
	}
	schemaGroups, err := s.ListSchemaGroups(ctx, &store.FindSchemaGroupMessage{DatabaseGroupUID: &databaseGroup.UID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list schema groups for database group %q", databaseGroupID)
	}
	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
	}

	matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get matched and unmatched databases in database group %q", databaseGroupID)
	}
	if len(matchedDatabases) == 0 {
		return nil, errors.Errorf("no matched databases found in database group %q", databaseGroupID)
	}

	environments, err := s.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list environments")
	}

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
	}
	sheetStatement, err := s.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %q", sheetUID)
	}

	taskCreatesMatrixByEnv := map[string][][]*store.TaskMessage{}
	taskIndexDAGsMatrixByEnv := map[string][][]store.TaskIndexDAG{}

	for _, db := range matchedDatabases {
		dbSchema, err := s.GetDBSchema(ctx, db.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get db schema %q", db.UID)
		}
		instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &db.InstanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", db.InstanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance %q not found", db.InstanceID)
		}

		schemaGroupsMatchedTables := map[string][]string{}
		for _, schemaGroup := range schemaGroups {
			matches, _, err := utils.GetMatchedAndUnmatchedTablesInSchemaGroup(ctx, dbSchema, schemaGroup)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get matched and unmatched tables in schema group %q", schemaGroup.ResourceID)
			}
			schemaGroupsMatchedTables[schemaGroup.ResourceID] = matches
		}

		parserEngineType, err := utils.ConvertDatabaseToParserEngineType(instance.Engine)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert database engine %q to parser engine type", instance.Engine)
		}

		statements, schemaGroupNames, err := utils.GetStatementsAndSchemaGroupsFromSchemaGroups(sheetStatement, parserEngineType, c.Target, schemaGroups, schemaGroupsMatchedTables)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get statements from schema groups")
		}

		taskCreates, err := getTaskCreatesFromChangeDatabaseConfigDatabaseGroupStatements(db, instance, spec, c, statements, schemaGroupNames)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get task creates from change database config database group statements")
		}
		var taskIndexDAGs []store.TaskIndexDAG
		for i := 1; i < len(taskCreates); i++ {
			taskIndexDAGs = append(taskIndexDAGs, store.TaskIndexDAG{
				FromIndex: i - 1,
				ToIndex:   i,
			})
		}

		if len(taskCreates) == 0 {
			continue
		}

		taskCreatesMatrixByEnv[db.EffectiveEnvironmentID] = append(taskCreatesMatrixByEnv[db.EffectiveEnvironmentID], taskCreates)
		taskIndexDAGsMatrixByEnv[db.EffectiveEnvironmentID] = append(taskIndexDAGsMatrixByEnv[db.EffectiveEnvironmentID], taskIndexDAGs)
	}

	pipelineCreate := &store.PipelineMessage{
		Name: "Rollout Pipeline",
	}

	for _, env := range environments {
		taskCreatesMatrix := taskCreatesMatrixByEnv[env.ResourceID]
		taskIndexDAGsMatrix := taskIndexDAGsMatrixByEnv[env.ResourceID]
		if len(taskCreatesMatrix) == 0 {
			continue
		}
		creates, dags, err := utils.MergeTaskCreateLists(taskCreatesMatrix, taskIndexDAGsMatrix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to merge task create lists")
		}

		stageCreate := &store.StageMessage{
			EnvironmentID:    env.UID,
			Name:             fmt.Sprintf("%s Stage", env.Title),
			TaskList:         creates,
			TaskIndexDAGList: dags,
		}
		pipelineCreate.Stages = append(pipelineCreate.Stages, stageCreate)
	}

	j, _ := json.Marshal(pipelineCreate)
	slog.Debug("gjiowejgoiej", "j", string(j))

	if len(pipelineCreate.Stages) == 0 {
		return nil, errors.Errorf("get no tasks from the database group target")
	}

	return pipelineCreate, nil
}

func transformDeploymentConfigTargetToSteps(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, project *store.ProjectMessage) ([]*storepb.PlanConfig_Step, error) {
	projectID, _, err := common.GetProjectIDDeploymentConfigID(c.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project and deployment id from target %q", c.Target)
	}
	if project.ResourceID != projectID {
		return nil, errors.Errorf("project id %q in target %q does not match project id %q in plan config", projectID, c.Target, project.ResourceID)
	}

	switch c.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_BASELINE:
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
	default:
		return nil, errors.Errorf("unsupported change database config type: %v", c.Type)
	}

	deploymentConfig, err := s.GetDeploymentConfigV2(ctx, project.UID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deployment config")
	}
	apiDeploymentConfig, err := deploymentConfig.ToAPIDeploymentConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deployment config to api deployment config")
	}
	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(apiDeploymentConfig.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate and get deployment schedule")
	}
	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases")
	}
	matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database matrix from deployment schedule")
	}

	var steps []*storepb.PlanConfig_Step
	for i, databases := range matrix {
		if len(databases) == 0 {
			continue
		}

		step := &storepb.PlanConfig_Step{
			Title: deploySchedule.Deployments[i].Name,
		}
		for _, database := range databases {
			s, ok := proto.Clone(spec).(*storepb.PlanConfig_Spec)
			if !ok {
				return nil, errors.Errorf("failed to clone, got %T", s)
			}
			proto.Merge(s, &storepb.PlanConfig_Spec{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Target: common.FormatDatabase(database.InstanceID, database.DatabaseName),
					},
				},
			})
			step.Specs = append(step.Specs, s)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func getTaskCreatesFromSpec(ctx context.Context, s *store.Store, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) != nil {
		if spec.EarliestAllowedTime != nil && !spec.EarliestAllowedTime.AsTime().IsZero() {
			return nil, nil, errors.Errorf(api.FeatureTaskScheduleTime.AccessErrorMessage())
		}
	}

	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s, licenseService, dbFactory, spec, config.CreateDatabaseConfig, project, registerEnvironmentID)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s, spec, config.ChangeDatabaseConfig, project, registerEnvironmentID)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		return getTaskCreatesFromExportDataConfig(ctx, s, spec, config.ExportDataConfig, project, registerEnvironmentID)
	}

	return nil, nil, errors.Errorf("invalid spec config type %T", spec.Config)
}

func getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, s *store.Store, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_CreateDatabaseConfig, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	if c.Database == "" {
		return nil, nil, errors.Errorf("database name is required")
	}

	instance, err := getInstanceMessage(ctx, s, c.Target)
	if err != nil {
		return nil, nil, err
	}
	if instance.Engine == storepb.Engine_ORACLE || instance.Engine == storepb.Engine_OCEANBASE_ORACLE {
		return nil, nil, errors.Errorf("creating Oracle database is not supported")
	}

	dbEnvironmentID := strings.TrimPrefix(c.Environment, common.EnvironmentNamePrefix)
	// Fallback to instance.EnvironmentID if user-set environment is not present.
	environmentID := instance.EnvironmentID
	if dbEnvironmentID != "" {
		environmentID = dbEnvironmentID
	}

	environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
	if err != nil {
		return nil, nil, err
	}
	if environment == nil {
		return nil, nil, errors.Errorf("environment ID not found %v", environmentID)
	}
	if err := registerEnvironmentID(environmentID); err != nil {
		return nil, nil, err
	}

	if instance.Engine == storepb.Engine_MONGODB && c.Table == "" {
		return nil, nil, errors.Errorf("collection name is required for MongoDB")
	}

	taskCreates, err := func() ([]*store.TaskMessage, error) {
		if err := checkCharacterSetCollationOwner(instance.Engine, c.CharacterSet, c.Collation, c.Owner); err != nil {
			return nil, err
		}
		if c.Database == "" {
			return nil, errors.Errorf("database name is required")
		}
		if instance.Engine == storepb.Engine_SNOWFLAKE {
			// Snowflake needs to use upper case of DatabaseName.
			c.Database = strings.ToUpper(c.Database)
		}
		if instance.Engine == storepb.Engine_MONGODB && c.Table == "" {
			return nil, common.Errorf(common.Invalid, "Failed to create issue, collection name missing for MongoDB")
		}
		// Validate the labels. Labels are set upon task completion.
		labelsJSON, err := convertDatabaseLabels(c.Labels)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid database label %q", c.Labels)
		}

		// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
		if project.TenantMode == api.TenantModeTenant {
			if err := licenseService.IsFeatureEnabled(api.FeatureMultiTenancy); err != nil {
				return nil, err
			}
		}

		// Get admin data source username.
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, api.Admin)
		if adminDataSource == nil {
			return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.Title)
		}
		databaseName := c.Database
		switch instance.Engine {
		case storepb.Engine_SNOWFLAKE:
			// Snowflake needs to use upper case of DatabaseName.
			databaseName = strings.ToUpper(databaseName)
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			// For MySQL, we need to use different case of DatabaseName depends on the variable `lower_case_table_names`.
			// https://dev.mysql.com/doc/refman/8.0/en/identifier-case-sensitivity.html
			// And also, meet an error in here is not a big deal, we will just use the original DatabaseName.
			driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
			if err != nil {
				slog.Warn("failed to get admin database driver for instance %q, please check the connection for admin data source", log.BBError(err), slog.String("instance", instance.Title))
				break
			}
			defer driver.Close(ctx)
			var lowerCaseTableNames int
			var unused any
			db := driver.GetDB()
			if err := db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'lower_case_table_names'").Scan(&unused, &lowerCaseTableNames); err != nil {
				slog.Warn("failed to get lower_case_table_names for instance %q", log.BBError(err), slog.String("instance", instance.Title))
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
		sheet, err := s.CreateSheet(ctx, &store.SheetMessage{
			CreatorID:  api.SystemBotID,
			ProjectUID: project.UID,
			Title:      fmt.Sprintf("Sheet for creating database %v", databaseName),
			Statement:  statement,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation sheet")
		}

		payload := api.TaskDatabaseCreatePayload{
			SpecID:        spec.Id,
			ProjectID:     project.UID,
			CharacterSet:  c.CharacterSet,
			TableName:     c.Table,
			Collation:     c.Collation,
			EnvironmentID: dbEnvironmentID,
			Labels:        labelsJSON,
			DatabaseName:  databaseName,
			SheetID:       sheet.UID,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation task, unable to marshal payload")
		}

		return []*store.TaskMessage{
			{
				InstanceID:        instance.UID,
				DatabaseID:        nil,
				Name:              fmt.Sprintf("Create database %v", payload.DatabaseName),
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

func getTaskCreatesFromChangeDatabaseConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, project *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	// possible target:
	// 1. instances/{instance}/databases/{database}
	// 2. projects/{project}/databaseGroups/{databaseGroup}
	// 3. projects/{project}/deploymentConfigs/{deploymentConfig}
	if _, _, err := common.GetInstanceDatabaseID(c.Target); err == nil {
		return getTaskCreatesFromChangeDatabaseConfigDatabaseTarget(ctx, s, spec, c, project, registerEnvironmentID)
	}
	if _, _, err := common.GetProjectIDDatabaseGroupID(c.Target); err == nil {
		return nil, nil, errors.Errorf("unexpected database group target %q", c.Target)
	}
	if _, _, err := common.GetProjectIDDeploymentConfigID(c.Target); err == nil {
		return nil, nil, errors.Errorf("unexpected deployment config target %q", c.Target)
	}

	return nil, nil, errors.Errorf("unknown target %q", c.Target)
}

func getTaskCreatesFromExportDataConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ExportDataConfig, _ *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance and database from target %q", c.Target)
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
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database %q not found", databaseName)
	}

	if err := registerEnvironmentID(database.EffectiveEnvironmentID); err != nil {
		return nil, nil, err
	}

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
	}
	payload := api.TaskDatabaseDataExportPayload{
		SpecID:  spec.Id,
		SheetID: sheetUID,
		Format:  c.Format,
	}
	if c.Password != nil {
		payload.Password = *c.Password
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to marshal task database data export payload")
	}
	payloadString := string(bytes)
	taskCreate := &store.TaskMessage{
		Name:              fmt.Sprintf("Export data from database %q", database.DatabaseName),
		InstanceID:        instance.UID,
		DatabaseID:        &database.UID,
		Type:              api.TaskDatabaseDataExport,
		EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
		Payload:           payloadString,
	}
	return []*store.TaskMessage{taskCreate}, nil, nil
}

func getTaskCreatesFromChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, _ *store.ProjectMessage, registerEnvironmentID func(string) error) ([]*store.TaskMessage, []store.TaskIndexDAG, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance and database from target %q", c.Target)
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
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, nil, errors.Errorf("database %q not found", databaseName)
	}

	if err := registerEnvironmentID(database.EffectiveEnvironmentID); err != nil {
		return nil, nil, err
	}

	switch c.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_BASELINE:
		payload := api.TaskDatabaseSchemaBaselinePayload{
			SpecID:        spec.Id,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal task database schema baseline payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("Establish baseline for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Type:              api.TaskDatabaseSchemaBaseline,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		payload := api.TaskDatabaseSchemaUpdatePayload{
			SpecID:        spec.Id,
			SheetID:       sheetUID,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal task database schema update payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("DDL(schema) for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Type:              api.TaskDatabaseSchemaUpdate,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		payload := api.TaskDatabaseSchemaUpdateSDLPayload{
			SpecID:        spec.Id,
			SheetID:       sheetUID,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update SDL payload")
		}
		payloadString := string(bytes)
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("SDL for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Type:              api.TaskDatabaseSchemaUpdateSDL,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		if _, err := ghost.GetUserFlags(c.GhostFlags); err != nil {
			return nil, nil, errors.Wrapf(err, "invalid ghost flags %q, error: %v", c.GhostFlags, err)
		}
		var taskCreateList []*store.TaskMessage
		// task "sync"
		payloadSync := api.TaskDatabaseSchemaUpdateGhostSyncPayload{
			SpecID:        spec.Id,
			SheetID:       sheetUID,
			SchemaVersion: getOrDefaultSchemaVersion(c.SchemaVersion),
			Flags:         c.GhostFlags,
		}
		bytesSync, err := json.Marshal(payloadSync)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to marshal database schema update gh-ost sync payload")
		}
		taskCreateList = append(taskCreateList, &store.TaskMessage{
			Name:              fmt.Sprintf("Update schema gh-ost sync for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
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
		taskCreateList = append(taskCreateList, &store.TaskMessage{
			Name:              fmt.Sprintf("Update schema gh-ost cutover for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Type:              api.TaskDatabaseSchemaUpdateGhostCutover,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           string(bytesCutover),
		})

		// The below list means that taskCreateList[0] blocks taskCreateList[1].
		// In other words, task "sync" blocks task "cutover".
		taskIndexDAGList := []store.TaskIndexDAG{
			{FromIndex: 0, ToIndex: 1},
		}
		return taskCreateList, taskIndexDAGList, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		preUpdateBackupDetail := api.PreUpdateBackupDetail{}
		if c.GetPreUpdateBackupDetail().GetDatabase() != "" {
			preUpdateBackupDetail.Database = c.GetPreUpdateBackupDetail().GetDatabase()
		}
		payload := api.TaskDatabaseDataUpdatePayload{
			SpecID:                spec.Id,
			SheetID:               sheetUID,
			SchemaVersion:         getOrDefaultSchemaVersion(c.SchemaVersion),
			RollbackEnabled:       c.RollbackEnabled,
			RollbackSQLStatus:     api.RollbackSQLStatusPending,
			PreUpdateBackupDetail: preUpdateBackupDetail,
		}
		if c.RollbackDetail != nil {
			issueID, err := common.GetIssueID(c.RollbackDetail.RollbackFromIssue)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to get issue id from issue %q", c.RollbackDetail.RollbackFromIssue)
			}
			payload.RollbackFromIssueID = issueID
			taskID, err := common.GetTaskID(c.RollbackDetail.RollbackFromTask)
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
		taskCreate := &store.TaskMessage{
			Name:              fmt.Sprintf("DML(data) for database %q", database.DatabaseName),
			InstanceID:        instance.UID,
			DatabaseID:        &database.UID,
			Type:              api.TaskDatabaseDataUpdate,
			EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
			Payload:           payloadString,
		}
		return []*store.TaskMessage{taskCreate}, nil, nil
	default:
		return nil, nil, errors.Errorf("unsupported change database config type %q", c.Type)
	}
}

func getTaskCreatesFromChangeDatabaseConfigDatabaseGroupStatements(db *store.DatabaseMessage, instance *store.InstanceMessage, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, statements []string, schemaGroupNames []string) ([]*store.TaskMessage, error) {
	var creates []*store.TaskMessage
	for idx, statement := range statements {
		schemaVersionSuffix := fmt.Sprintf("-%03d", idx)
		schemaGroupName := schemaGroupNames[idx]
		switch c.Type {
		case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
			payload := api.TaskDatabaseSchemaUpdatePayload{
				SpecID:          spec.Id,
				SheetID:         0,
				SchemaVersion:   getOrDefaultSchemaVersionWithSuffix(c.SchemaVersion, schemaVersionSuffix),
				SchemaGroupName: schemaGroupName,
			}
			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to marshal task database schema update payload")
			}
			payloadString := string(bytes)
			taskCreate := &store.TaskMessage{
				Name:              fmt.Sprintf("DDL(schema) for database %q", db.DatabaseName),
				InstanceID:        instance.UID,
				DatabaseID:        &db.UID,
				Type:              api.TaskDatabaseSchemaUpdate,
				EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
				Payload:           payloadString,
				Statement:         statement,
			}
			creates = append(creates, taskCreate)

		case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
			payload := api.TaskDatabaseDataUpdatePayload{
				SpecID:            spec.Id,
				SheetID:           0,
				SchemaVersion:     getOrDefaultSchemaVersionWithSuffix(c.SchemaVersion, schemaVersionSuffix),
				RollbackEnabled:   c.RollbackEnabled,
				RollbackSQLStatus: api.RollbackSQLStatusPending,
				SchemaGroupName:   schemaGroupName,
			}

			bytes, err := json.Marshal(payload)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to marshal database data update payload")
			}
			payloadString := string(bytes)
			taskCreate := &store.TaskMessage{
				Name:              fmt.Sprintf("DML(data) for database %q", db.DatabaseName),
				InstanceID:        instance.UID,
				DatabaseID:        &db.UID,
				Type:              api.TaskDatabaseDataUpdate,
				EarliestAllowedTs: spec.EarliestAllowedTime.GetSeconds(),
				Payload:           payloadString,
				Statement:         statement,
			}
			creates = append(creates, taskCreate)
		}
	}

	return creates, nil
}

// checkCharacterSetCollationOwner checks if the character set, collation and owner are legal according to the dbType.
func checkCharacterSetCollationOwner(dbType storepb.Engine, characterSet, collation, owner string) error {
	switch dbType {
	case storepb.Engine_SPANNER:
		// Spanner does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("Spanner does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Spanner does not support collation, but got %s", collation)
		}
	case storepb.Engine_CLICKHOUSE:
		// ClickHouse does not support character set and collation at the database level.
		if characterSet != "" {
			return errors.Errorf("ClickHouse does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("ClickHouse does not support collation, but got %s", collation)
		}
	case storepb.Engine_SNOWFLAKE:
		if characterSet != "" {
			return errors.Errorf("Snowflake does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("Snowflake does not support collation, but got %s", collation)
		}
	case storepb.Engine_POSTGRES:
		if owner == "" {
			return errors.Errorf("database owner is required for PostgreSQL")
		}
	case storepb.Engine_REDSHIFT:
		if owner == "" {
			return errors.Errorf("database owner is required for Redshift")
		}
	case storepb.Engine_RISINGWAVE:
		if characterSet != "" {
			return errors.Errorf("RisingWave does not support character set, but got %s", characterSet)
		}
		if collation != "" {
			return errors.Errorf("RisingWave does not support collation, but got %s", collation)
		}
	case storepb.Engine_SQLITE, storepb.Engine_MONGODB, storepb.Engine_MSSQL:
		// no-op.
	default:
		if characterSet == "" {
			return errors.Errorf("character set missing for %s", dbType.String())
		}
		// For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
		// If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
		// install it.
		if collation == "" {
			return errors.Errorf("collation missing for %s", dbType.String())
		}
	}
	return nil
}

func getCreateDatabaseStatement(dbType storepb.Engine, c *storepb.PlanConfig_CreateDatabaseConfig, databaseName, adminDatasourceUser string) (string, error) {
	var stmt string
	switch dbType {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, c.CharacterSet, c.Collation), nil
	case storepb.Engine_MSSQL:
		return fmt.Sprintf(`CREATE DATABASE "%s";`, databaseName), nil
	case storepb.Engine_POSTGRES:
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
	case storepb.Engine_CLICKHOUSE:
		clusterPart := ""
		if c.Cluster != "" {
			clusterPart = fmt.Sprintf(" ON CLUSTER `%s`", c.Cluster)
		}
		return fmt.Sprintf("CREATE DATABASE `%s`%s;", databaseName, clusterPart), nil
	case storepb.Engine_SNOWFLAKE:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case storepb.Engine_SQLITE:
		// This is a fake CREATE DATABASE and USE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		return fmt.Sprintf("CREATE DATABASE '%s';", databaseName), nil
	case storepb.Engine_MONGODB:
		// We just run createCollection in mongosh instead of execute `use <database>` first, because we execute the
		// mongodb statement in mongosh with --file flag, and it doesn't support `use <database>` statement in the file.
		// And we pass the database name to Bytebase engine driver, which will be used to build the connection string.
		return fmt.Sprintf(`db.createCollection("%s");`, c.Table), nil
	case storepb.Engine_SPANNER:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case storepb.Engine_ORACLE:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case storepb.Engine_REDSHIFT:
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
	case storepb.Engine_HIVE:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	}
	return "", errors.Errorf("unsupported database type %s", dbType)
}

func getOrDefaultSchemaVersion(v string) string {
	if v != "" {
		return v
	}
	return common.DefaultMigrationVersion().Version
}

func getOrDefaultSchemaVersionWithSuffix(v string, suffix string) string {
	if v != "" {
		return v + suffix
	}
	return common.DefaultMigrationVersion().Version + suffix
}
