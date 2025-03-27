package v1

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/component/sheet"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func transformDatabaseGroupSpecs(ctx context.Context, s *store.Store, project *store.ProjectMessage, specs []*storepb.PlanConfig_Spec, deployment *storepb.PlanConfig_Deployment) ([]*storepb.PlanConfig_Spec, error) {
	var rspecs []*storepb.PlanConfig_Spec

	for _, spec := range specs {
		if config := spec.GetChangeDatabaseConfig(); config != nil {
			// transform database group.
			if _, _, err := common.GetProjectIDDatabaseGroupID(config.Target); err == nil {
				specsFromDatabaseGroup, err := transformDatabaseGroupTargetToSpecs(ctx, s, spec, config, project, deployment)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to transform databaseGroup target to steps")
				}
				rspecs = append(rspecs, specsFromDatabaseGroup...)
				continue
			}
		}
		rspecs = append(rspecs, spec)
	}

	return rspecs, nil
}

func transformDatabaseGroupTargetToSpecs(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, project *store.ProjectMessage, deployment *storepb.PlanConfig_Deployment) ([]*storepb.PlanConfig_Spec, error) {
	// Use snapshot result if it's present.
	for _, s := range deployment.GetDatabaseGroupMappings() {
		if s.DatabaseGroup == c.Target {
			var specs []*storepb.PlanConfig_Spec
			for _, database := range s.Databases {
				s, ok := proto.Clone(spec).(*storepb.PlanConfig_Spec)
				if !ok {
					return nil, errors.Errorf("failed to clone, got %T", s)
				}
				proto.Merge(s, &storepb.PlanConfig_Spec{
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Target: database,
						},
					},
				})
				specs = append(specs, s)
			}
			return specs, nil
		}
	}

	projectID, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(c.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project and deployment id from target %q", c.Target)
	}
	if project.ResourceID != projectID {
		return nil, errors.Errorf("project id %q in target %q does not match project id %q in plan config", projectID, c.Target, project.ResourceID)
	}

	databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ProjectID: &project.ResourceID, ResourceID: &databaseGroupID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %q", databaseGroupID)
	}
	if databaseGroup == nil {
		return nil, errors.Errorf("database group %q not found", databaseGroupID)
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

	var specs []*storepb.PlanConfig_Spec
	for _, database := range matchedDatabases {
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
		specs = append(specs, s)
	}
	return specs, nil
}

func getTaskCreatesFromSpec(ctx context.Context, s *store.Store, sheetManager *sheet.Manager, licenseService enterprise.LicenseService, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, project *store.ProjectMessage) ([]*store.TaskMessage, error) {
	if licenseService.IsFeatureEnabled(api.FeatureTaskScheduleTime) != nil {
		if spec.EarliestAllowedTime != nil && !spec.EarliestAllowedTime.AsTime().IsZero() {
			return nil, errors.New(api.FeatureTaskScheduleTime.AccessErrorMessage())
		}
	}

	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s, sheetManager, dbFactory, spec, config.CreateDatabaseConfig, project)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s, spec, config.ChangeDatabaseConfig, project)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		return getTaskCreatesFromExportDataConfig(ctx, s, spec, config.ExportDataConfig, project)
	}

	return nil, errors.Errorf("invalid spec config type %T", spec.Config)
}

func getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, s *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_CreateDatabaseConfig, project *store.ProjectMessage) ([]*store.TaskMessage, error) {
	if c.Database == "" {
		return nil, errors.Errorf("database name is required")
	}

	instance, err := getInstanceMessage(ctx, s, c.Target)
	if err != nil {
		return nil, err
	}
	if instance.Metadata.GetEngine() == storepb.Engine_ORACLE || instance.Metadata.GetEngine() == storepb.Engine_OCEANBASE_ORACLE {
		return nil, errors.Errorf("creating Oracle database is not supported")
	}

	dbEnvironmentID := ""
	if c.Environment != "" {
		dbEnvironmentID = strings.TrimPrefix(c.Environment, common.EnvironmentNamePrefix)
	}
	// Fallback to instance.EnvironmentID if user-set environment is not present.
	effectiveEnvironmentID := instance.EnvironmentID
	if dbEnvironmentID != "" {
		effectiveEnvironmentID = dbEnvironmentID
	}
	environment, err := s.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &effectiveEnvironmentID})
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, errors.Errorf("environment ID not found %v", effectiveEnvironmentID)
	}

	if instance.Metadata.GetEngine() == storepb.Engine_MONGODB && c.Table == "" {
		return nil, errors.Errorf("collection name is required for MongoDB")
	}

	taskCreates, err := func() ([]*store.TaskMessage, error) {
		if err := checkCharacterSetCollationOwner(instance.Metadata.GetEngine(), c.CharacterSet, c.Collation, c.Owner); err != nil {
			return nil, err
		}
		if c.Database == "" {
			return nil, errors.Errorf("database name is required")
		}
		if instance.Metadata.GetEngine() == storepb.Engine_SNOWFLAKE {
			// Snowflake needs to use upper case of DatabaseName.
			c.Database = strings.ToUpper(c.Database)
		}
		if instance.Metadata.GetEngine() == storepb.Engine_MONGODB && c.Table == "" {
			return nil, common.Errorf(common.Invalid, "Failed to create issue, collection name missing for MongoDB")
		}

		// Get admin data source username.
		adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
		if adminDataSource == nil {
			return nil, common.Errorf(common.Internal, "admin data source not found for instance %q", instance.ResourceID)
		}
		databaseName := c.Database
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_SNOWFLAKE:
			// Snowflake needs to use upper case of DatabaseName.
			databaseName = strings.ToUpper(databaseName)
		case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			// For MySQL, we need to use different case of DatabaseName depends on the variable `lower_case_table_names`.
			// https://dev.mysql.com/doc/refman/8.0/en/identifier-case-sensitivity.html
			// And also, meet an error in here is not a big deal, we will just use the original DatabaseName.
			driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{
				OperationalComponent: "show-variables",
			})
			if err != nil {
				slog.Warn("failed to get admin database driver for instance %q, please check the connection for admin data source", log.BBError(err), slog.String("instance", instance.ResourceID))
				break
			}
			defer driver.Close(ctx)
			var lowerCaseTableNames int
			var unused any
			db := driver.GetDB()
			if err := db.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'lower_case_table_names'").Scan(&unused, &lowerCaseTableNames); err != nil {
				slog.Warn("failed to get lower_case_table_names for instance %q", log.BBError(err), slog.String("instance", instance.ResourceID))
				break
			}
			if lowerCaseTableNames == 1 {
				databaseName = strings.ToLower(databaseName)
			}
		}

		statement, err := getCreateDatabaseStatement(instance.Metadata.GetEngine(), c, databaseName, adminDataSource.GetUsername())
		if err != nil {
			return nil, err
		}
		sheet, err := sheetManager.CreateSheet(ctx, &store.SheetMessage{
			CreatorID: api.SystemBotID,
			ProjectID: project.ResourceID,
			Title:     fmt.Sprintf("Sheet for creating database %v", databaseName),
			Statement: statement,
			Payload: &storepb.SheetPayload{
				Engine: instance.Metadata.GetEngine(),
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation sheet")
		}

		v := &store.TaskMessage{
			InstanceID:    instance.ResourceID,
			DatabaseName:  &databaseName,
			EnvironmentID: effectiveEnvironmentID,
			Type:          api.TaskDatabaseCreate,
			Payload: &storepb.TaskPayload{
				SpecId:        spec.Id,
				CharacterSet:  c.CharacterSet,
				TableName:     c.Table,
				Collation:     c.Collation,
				EnvironmentId: dbEnvironmentID,
				DatabaseName:  databaseName,
				SheetId:       int32(sheet.UID),
			},
		}
		if spec.EarliestAllowedTime.GetSeconds() > 0 {
			t := spec.EarliestAllowedTime.AsTime()
			v.EarliestAllowedAt = &t
		}
		return []*store.TaskMessage{
			v,
		}, nil
	}()
	if err != nil {
		return nil, err
	}

	return taskCreates, nil
}

func getTaskCreatesFromChangeDatabaseConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, project *store.ProjectMessage) ([]*store.TaskMessage, error) {
	// possible target:
	// 1. instances/{instance}/databases/{database}
	// 2. projects/{project}/databaseGroups/{databaseGroup}
	if _, _, err := common.GetInstanceDatabaseID(c.Target); err == nil {
		return getTaskCreatesFromChangeDatabaseConfigDatabaseTarget(ctx, s, spec, c, project)
	}
	if _, _, err := common.GetProjectIDDatabaseGroupID(c.Target); err == nil {
		return nil, errors.Errorf("unexpected database group target %q", c.Target)
	}

	return nil, errors.Errorf("unknown target %q", c.Target)
}

func getTaskCreatesFromExportDataConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ExportDataConfig, _ *store.ProjectMessage) ([]*store.TaskMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance and database from target %q", c.Target)
	}

	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", instanceID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database %q not found", databaseName)
	}

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
	}
	payload := &storepb.TaskPayload{
		SpecId:  spec.Id,
		SheetId: int32(sheetUID),
		Format:  c.Format,
	}
	if c.Password != nil {
		payload.Password = *c.Password
	}
	taskCreate := &store.TaskMessage{
		InstanceID:    database.InstanceID,
		DatabaseName:  &database.DatabaseName,
		EnvironmentID: database.EffectiveEnvironmentID,
		Type:          api.TaskDatabaseDataExport,
		Payload:       payload,
	}
	if spec.EarliestAllowedTime.GetSeconds() > 0 {
		t := spec.EarliestAllowedTime.AsTime()
		taskCreate.EarliestAllowedAt = &t
	}
	return []*store.TaskMessage{taskCreate}, nil
}

func getTaskCreatesFromChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_ChangeDatabaseConfig, _ *store.ProjectMessage) ([]*store.TaskMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(c.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance and database from target %q", c.Target)
	}

	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", instanceID)
	}
	database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database %q not found", databaseName)
	}

	switch c.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_BASELINE:
		taskCreate := &store.TaskMessage{
			InstanceID:    database.InstanceID,
			DatabaseName:  &database.DatabaseName,
			EnvironmentID: database.EffectiveEnvironmentID,
			Type:          api.TaskDatabaseSchemaBaseline,
			Payload: &storepb.TaskPayload{
				SpecId:        spec.Id,
				SchemaVersion: c.SchemaVersion,
				TaskReleaseSource: &storepb.TaskReleaseSource{
					File: spec.SpecReleaseSource.GetFile(),
				},
			},
		}
		if spec.EarliestAllowedTime.GetSeconds() > 0 {
			t := spec.EarliestAllowedTime.AsTime()
			taskCreate.EarliestAllowedAt = &t
		}
		return []*store.TaskMessage{taskCreate}, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		taskCreate := &store.TaskMessage{
			InstanceID:    database.InstanceID,
			DatabaseName:  &database.DatabaseName,
			EnvironmentID: database.EffectiveEnvironmentID,
			Type:          api.TaskDatabaseSchemaUpdate,
			Payload: &storepb.TaskPayload{
				SpecId:        spec.Id,
				SheetId:       int32(sheetUID),
				SchemaVersion: c.SchemaVersion,
				TaskReleaseSource: &storepb.TaskReleaseSource{
					File: spec.SpecReleaseSource.GetFile(),
				},
			},
		}
		if spec.EarliestAllowedTime.GetSeconds() > 0 {
			t := spec.EarliestAllowedTime.AsTime()
			taskCreate.EarliestAllowedAt = &t
		}
		return []*store.TaskMessage{taskCreate}, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		if _, err := ghost.GetUserFlags(c.GhostFlags); err != nil {
			return nil, errors.Wrapf(err, "invalid ghost flags %q", c.GhostFlags)
		}
		taskCreate := &store.TaskMessage{
			InstanceID:    database.InstanceID,
			DatabaseName:  &database.DatabaseName,
			EnvironmentID: database.EffectiveEnvironmentID,
			Type:          api.TaskDatabaseSchemaUpdateGhost,
			Payload: &storepb.TaskPayload{
				SpecId:        spec.Id,
				SheetId:       int32(sheetUID),
				SchemaVersion: c.SchemaVersion,
				Flags:         c.GhostFlags,
				TaskReleaseSource: &storepb.TaskReleaseSource{
					File: spec.SpecReleaseSource.GetFile(),
				},
			},
		}
		if spec.EarliestAllowedTime.GetSeconds() > 0 {
			t := spec.EarliestAllowedTime.AsTime()
			taskCreate.EarliestAllowedAt = &t
		}
		return []*store.TaskMessage{taskCreate}, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		preUpdateBackupDetail := &storepb.PreUpdateBackupDetail{}
		if c.GetPreUpdateBackupDetail().GetDatabase() != "" {
			preUpdateBackupDetail.Database = c.GetPreUpdateBackupDetail().GetDatabase()
		}
		taskCreate := &store.TaskMessage{
			InstanceID:    database.InstanceID,
			DatabaseName:  &database.DatabaseName,
			EnvironmentID: database.EffectiveEnvironmentID,
			Type:          api.TaskDatabaseDataUpdate,
			Payload: &storepb.TaskPayload{
				SpecId:                spec.Id,
				SheetId:               int32(sheetUID),
				SchemaVersion:         c.SchemaVersion,
				PreUpdateBackupDetail: preUpdateBackupDetail,
				TaskReleaseSource: &storepb.TaskReleaseSource{
					File: spec.SpecReleaseSource.GetFile(),
				},
			},
		}
		if spec.EarliestAllowedTime.GetSeconds() > 0 {
			t := spec.EarliestAllowedTime.AsTime()
			taskCreate.EarliestAllowedAt = &t
		}
		return []*store.TaskMessage{taskCreate}, nil
	default:
		return nil, errors.Errorf("unsupported change database config type %q", c.Type)
	}
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
	case storepb.Engine_COCKROACHDB:
		if owner == "" {
			return errors.Errorf("database owner is required for CockroachDB")
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
	case storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB:
		collationPart := ""
		if c.Collation != "" {
			collationPart = fmt.Sprintf(" LC_COLLATE %q", c.Collation)
		}
		stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q%s;", stmt, databaseName, c.CharacterSet, collationPart)
		// Set the database owner.
		// We didn't use CREATE DATABASE WITH OWNER because RDS requires the current role to be a member of the database owner.
		// However, people can still use ALTER DATABASE to change the owner afterwards.
		// Error string below:
		// query: CREATE DATABASE h1 WITH OWNER hello;
		// ERROR:  must be member of role "hello"
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
