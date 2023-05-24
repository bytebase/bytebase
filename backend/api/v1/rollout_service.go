package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RolloutService represents a service for managing rollout.
type RolloutService struct {
	v1pb.UnimplementedRolloutServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
	dbFactory      *dbfactory.DBFactory
}

// NewRolloutService returns a rollout service instance.
func NewRolloutService(store *store.Store, licenseService enterpriseAPI.LicenseService, dbFactory *dbfactory.DBFactory) *RolloutService {
	return &RolloutService{
		store:          store,
		licenseService: licenseService,
		dbFactory:      dbFactory,
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
	principalUID := ctx.Value(common.PrincipalIDContextKey).(int)
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
		return nil, status.Errorf(codes.NotFound, "project not found for id: %d", projectID)
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
	plan, err := s.store.CreatePlan(ctx, planMessage, principalUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create plan, error: %v", err)
	}
	return convertToPlan(plan), nil
}

func (s *RolloutService) getPipelineCreate(ctx context.Context, steps []*v1pb.Plan_Step, project *store.ProjectMessage) (*api.PipelineCreate, error) {
	pipelineCreate := &api.PipelineCreate{}
	for _, step := range steps {
		stageCreate := api.StageCreate{}
		for _, spec := range step.Specs {
			taskCreates, taskIndexDAGCreates, err := s.getTaskCreatesFromSpec(ctx, spec, project)
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
		pipelineCreate.StageList = append(pipelineCreate.StageList, stageCreate)
	}
	return pipelineCreate, nil
}

func (s *RolloutService) getTaskCreatesFromSpec(ctx context.Context, spec *v1pb.Plan_Spec, project *store.ProjectMessage) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	useme := spec.Id

	switch config := spec.Config.(type) {
	case *v1pb.Plan_Spec_CreateDatabaseConfig:
		return s.getTaskCreatesFromCreateDatabaseConfig(ctx, spec, config.CreateDatabaseConfig, project)
	case *v1pb.Plan_Spec_ChangeDatabaseConfig:
		return s.getTaskCreatesFromChangeDatabaseConfig(ctx, spec, config.ChangeDatabaseConfig)
	case *v1pb.Plan_Spec_RestoreDatabaseConfig:
		return s.getTaskCreatesFromRestoreDatabaseConfig(ctx, config.RestoreDatabaseConfig)
	}
}

func (s *RolloutService) getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, spec *v1pb.Plan_Spec, c *v1pb.Plan_CreateDatabaseConfig, project *store.ProjectMessage) ([]api.TaskCreate, []api.TaskIndexDAG, error) {
	if c.Database == "" {
		return nil, nil, errors.Errorf("database name is required")
	}
	instanceID, err := getInstanceID(c.Target)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get instance id from %q", c.Target)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, nil, err
	}
	if instance == nil {
		return nil, nil, errors.Errorf("instance ID not found %v", instanceID)
	}
	if instance.Engine == db.Oracle {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "Creating Oracle database is not supported")
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, nil, err
	}
	if environment == nil {
		return nil, nil, errors.Errorf("environment ID not found %v", instance.EnvironmentID)
	}

	if instance.Engine == db.MongoDB && c.Table == "" {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "Failed to create issue, collection name missing for MongoDB")
	}

	taskCreates, err := func() ([]api.TaskCreate, error) {
		if err := checkCharacterSetCollationOwner(instance.Engine, c.CharacterSet, c.Collation, c.Owner); err != nil {
			return nil, err
		}
		if c.Database == "" {
			return nil, common.Errorf(common.Invalid, "Failed to create issue, database name missing")
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
			return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid database label %q, error: %v", c.Labels, err))
		}

		// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
		if project.TenantMode == api.TenantModeTenant {
			if !s.licenseService.IsFeatureEnabled(api.FeatureMultiTenancy) {
				return nil, echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
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
			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
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
		sheet, err := s.store.CreateSheet(ctx, &api.SheetCreate{
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
				InstanceID:   instance.UID,
				Name:         fmt.Sprintf("Create database %v", payload.DatabaseName),
				Status:       api.TaskPendingApproval,
				Type:         api.TaskDatabaseCreate,
				DatabaseName: payload.DatabaseName,
				Payload:      string(bytes),
			},
		}, nil
	}()

	if err != nil {
		return nil, nil, err
	}

	return taskCreates, nil, nil
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
		err := errors.Errorf("database labels are up to a maximum of %d", api.DatabaseLabelSizeMax)
		return "", echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
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
