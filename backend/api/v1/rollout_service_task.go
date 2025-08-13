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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func applyDatabaseGroupSpecTransformations(specs []*storepb.PlanConfig_Spec, deployment *storepb.PlanConfig_Deployment) []*storepb.PlanConfig_Spec {
	var result []*storepb.PlanConfig_Spec
	for _, spec := range specs {
		// Clone the spec to avoid modifying the original
		clonedSpec := proto.CloneOf(spec)

		if config := clonedSpec.GetChangeDatabaseConfig(); config != nil {
			// transform database group.
			if len(config.Targets) == 1 {
				if _, _, err := common.GetProjectIDDatabaseGroupID(config.Targets[0]); err == nil {
					for _, s := range deployment.GetDatabaseGroupMappings() {
						if s.DatabaseGroup == config.Targets[0] {
							config.Targets = s.Databases
							break
						}
					}
				}
			}
		}
		result = append(result, clonedSpec)
	}
	return result
}

func getTaskCreatesFromSpec(ctx context.Context, s *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, spec *storepb.PlanConfig_Spec, project *store.ProjectMessage) ([]*store.TaskMessage, error) {
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s, sheetManager, dbFactory, spec, config.CreateDatabaseConfig, project)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s, spec, config.ChangeDatabaseConfig)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		return getTaskCreatesFromExportDataConfig(ctx, s, spec, config.ExportDataConfig)
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
	if instance.Metadata.GetEngine() == storepb.Engine_ORACLE {
		return nil, errors.Errorf("creating Oracle database is not supported")
	}

	dbEnvironmentID := ""
	if c.Environment != "" {
		dbEnvironmentID = strings.TrimPrefix(c.Environment, common.EnvironmentNamePrefix)
	}
	// Fallback to instance.EnvironmentID if user-set environment is not present.
	var effectiveEnvironmentID string
	if dbEnvironmentID != "" {
		effectiveEnvironmentID = dbEnvironmentID
	} else if instance.EnvironmentID != nil && *instance.EnvironmentID != "" {
		effectiveEnvironmentID = *instance.EnvironmentID
	} else {
		return nil, errors.Errorf("no environment specified for instance %v", instance.ResourceID)
	}
	environment, err := s.GetEnvironmentByID(ctx, effectiveEnvironmentID)
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
			driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
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
		default:
			// Other engines use the original database name
		}

		statement, err := getCreateDatabaseStatement(instance.Metadata.GetEngine(), c, databaseName, adminDataSource.GetUsername())
		if err != nil {
			return nil, err
		}
		sheet, err := sheetManager.CreateSheet(ctx, &store.SheetMessage{
			CreatorID: common.SystemBotID,
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
			InstanceID:   instance.ResourceID,
			DatabaseName: &databaseName,
			Environment:  effectiveEnvironmentID,
			Type:         storepb.Task_DATABASE_CREATE,
			Payload: &storepb.Task{
				SpecId:        spec.Id,
				CharacterSet:  c.CharacterSet,
				TableName:     c.Table,
				Collation:     c.Collation,
				EnvironmentId: dbEnvironmentID,
				DatabaseName:  databaseName,
				SheetId:       int32(sheet.UID),
			},
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

func getTaskCreatesFromChangeDatabaseConfig(
	ctx context.Context,
	s *store.Store,
	spec *storepb.PlanConfig_Spec,
	c *storepb.PlanConfig_ChangeDatabaseConfig,
) ([]*store.TaskMessage, error) {
	databases, err := getDatabaseMessagesByTargets(ctx, s, c.Targets)
	if err != nil {
		return nil, err
	}

	// If a release is specified, we need to expand it into individual tasks for each release file
	if c.Release != "" {
		return getTaskCreatesFromChangeDatabaseConfigWithRelease(ctx, s, spec, c, databases)
	}

	// Possible targets: list of instances/{instance}/databases/{database}.
	var tasks []*store.TaskMessage
	for _, database := range databases {
		v, err := getTaskCreatesFromChangeDatabaseConfigDatabaseTarget(spec, c, database)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, v...)
	}
	return tasks, nil
}

func getDatabaseMessagesByTargets(ctx context.Context, s *store.Store, targets []string) ([]*store.DatabaseMessage, error) {
	databases := []*store.DatabaseMessage{}

	for _, target := range targets {
		if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			databaseGroup, err := getDatabaseGroupByName(ctx, s, target, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database group %q", target)
			}
			for _, matched := range databaseGroup.MatchedDatabases {
				database, err := getDatabaseMessage(ctx, s, matched.Name)
				if err != nil {
					return nil, err
				}
				if database == nil || database.Deleted {
					return nil, errors.Errorf("database %q not found", target)
				}
				databases = append(databases, database)
			}
		} else if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
			database, err := getDatabaseMessage(ctx, s, target)
			if err != nil {
				return nil, err
			}
			if database == nil || database.Deleted {
				return nil, errors.Errorf("database %q not found", target)
			}
			databases = append(databases, database)
		} else {
			return nil, errors.Errorf("invalid target %q", target)
		}
	}
	return databases, nil
}

func getTaskCreatesFromExportDataConfig(
	ctx context.Context,
	s *store.Store,
	spec *storepb.PlanConfig_Spec,
	c *storepb.PlanConfig_ExportDataConfig,
) ([]*store.TaskMessage, error) {
	databases, err := getDatabaseMessagesByTargets(ctx, s, c.Targets)
	if err != nil {
		return nil, err
	}

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
	}
	payload := &storepb.Task{
		SpecId:  spec.Id,
		SheetId: int32(sheetUID),
		Format:  c.Format,
	}
	if c.Password != nil {
		payload.Password = *c.Password
	}

	tasks := []*store.TaskMessage{}
	for _, database := range databases {
		env := ""
		if database.EffectiveEnvironmentID != nil {
			env = *database.EffectiveEnvironmentID
		}
		tasks = append(tasks, &store.TaskMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Environment:  env,
			Type:         storepb.Task_DATABASE_EXPORT,
			Payload:      payload,
		})
	}
	return tasks, nil
}

func getTaskCreatesFromChangeDatabaseConfigDatabaseTarget(
	spec *storepb.PlanConfig_Spec,
	c *storepb.PlanConfig_ChangeDatabaseConfig,
	database *store.DatabaseMessage,
) ([]*store.TaskMessage, error) {
	switch c.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		env := ""
		if database.EffectiveEnvironmentID != nil {
			env = *database.EffectiveEnvironmentID
		}
		taskCreate := &store.TaskMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Environment:  env,
			Type:         storepb.Task_DATABASE_SCHEMA_UPDATE,
			Payload: &storepb.Task{
				SpecId:  spec.Id,
				SheetId: int32(sheetUID),
			},
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
		env := ""
		if database.EffectiveEnvironmentID != nil {
			env = *database.EffectiveEnvironmentID
		}
		taskCreate := &store.TaskMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Environment:  env,
			Type:         storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST,
			Payload: &storepb.Task{
				SpecId:  spec.Id,
				SheetId: int32(sheetUID),
				Flags:   c.GhostFlags,
			},
		}
		return []*store.TaskMessage{taskCreate}, nil

	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(c.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q", c.Sheet)
		}
		env := ""
		if database.EffectiveEnvironmentID != nil {
			env = *database.EffectiveEnvironmentID
		}
		taskCreate := &store.TaskMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Environment:  env,
			Type:         storepb.Task_DATABASE_DATA_UPDATE,
			Payload: &storepb.Task{
				SpecId:            spec.Id,
				SheetId:           int32(sheetUID),
				EnablePriorBackup: c.EnablePriorBackup,
			},
		}
		return []*store.TaskMessage{taskCreate}, nil
	default:
		return nil, errors.Errorf("unsupported change database config type %q", c.Type)
	}
}

func getTaskCreatesFromChangeDatabaseConfigWithRelease(
	ctx context.Context,
	s *store.Store,
	spec *storepb.PlanConfig_Spec,
	c *storepb.PlanConfig_ChangeDatabaseConfig,
	databases []*store.DatabaseMessage,
) ([]*store.TaskMessage, error) {
	// Parse release name to get project ID and release UID
	_, releaseUID, err := common.GetProjectReleaseUID(c.Release)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse release name %q", c.Release)
	}

	// Fetch the release
	release, err := s.GetRelease(ctx, releaseUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get release %d", releaseUID)
	}
	if release == nil {
		return nil, errors.Errorf("release %d not found", releaseUID)
	}

	// Create tasks for each release file that hasn't been applied
	var taskCreates []*store.TaskMessage
	for _, database := range databases {
		// Get existing revisions for the database to check which files have already been applied
		revisions, err := s.ListRevisions(ctx, &store.FindRevisionMessage{
			InstanceID:   &database.InstanceID,
			DatabaseName: &database.DatabaseName,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list revisions for database %q", database.DatabaseName)
		}

		// Find the max declarative version. Could be nil.
		var maxDeclarativeVersion *model.Version
		// Create a map of applied versions of VERSIONED revisions
		appliedVersions := make(map[string]string) // version -> sha256
		for _, revision := range revisions {
			switch revision.Payload.Type {
			case storepb.RevisionPayload_VERSIONED:
				appliedVersions[revision.Version] = revision.Payload.SheetSha256
			case storepb.RevisionPayload_DECLARATIVE:
				v, err := model.NewVersion(revision.Version)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse revision version %q", revision.Version)
				}
				if maxDeclarativeVersion == nil || maxDeclarativeVersion.LessThan(v) {
					maxDeclarativeVersion = v
				}
			default:
				return nil, errors.Errorf("unexpected revision type %q", revision.Payload.Type)
			}
		}

		for _, file := range release.Payload.Files {
			switch file.Type {
			case storepb.ReleasePayload_File_VERSIONED:
				// Skip if this version has already been applied
				if _, ok := appliedVersions[file.Version]; ok {
					// Skip files that have been applied with the same content
					// If SHA256 differs, it means the file has been modified after being applied. CheckRelease should have warned it.
					continue
				}

				// Parse sheet ID from the file's sheet reference
				_, sheetUID, err := common.GetProjectResourceIDSheetUID(file.Sheet)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q in release file %q", file.Sheet, file.Id)
				}

				// Determine task type based on file change type
				var taskType storepb.Task_Type
				switch file.ChangeType {
				case storepb.ReleasePayload_File_DDL, storepb.ReleasePayload_File_CHANGE_TYPE_UNSPECIFIED:
					taskType = storepb.Task_DATABASE_SCHEMA_UPDATE
				case storepb.ReleasePayload_File_DDL_GHOST:
					taskType = storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST
				case storepb.ReleasePayload_File_DML:
					taskType = storepb.Task_DATABASE_DATA_UPDATE
				default:
					return nil, errors.Errorf("unsupported release file change type %q", file.ChangeType)
				}

				// Create task payload
				payload := &storepb.Task{
					SpecId:        spec.Id,
					SheetId:       int32(sheetUID),
					SchemaVersion: file.Version,
					TaskReleaseSource: &storepb.TaskReleaseSource{
						File: common.FormatReleaseFile(c.Release, file.Id),
					},
				}

				// Add ghost flags if this is a ghost migration
				if taskType == storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST && c.GhostFlags != nil {
					payload.Flags = c.GhostFlags
				}

				if taskType == storepb.Task_DATABASE_DATA_UPDATE {
					payload.EnablePriorBackup = c.EnablePriorBackup
				}

				env := ""
				if database.EffectiveEnvironmentID != nil {
					env = *database.EffectiveEnvironmentID
				}
				taskCreate := &store.TaskMessage{
					InstanceID:   database.InstanceID,
					DatabaseName: &database.DatabaseName,
					Environment:  env,
					Type:         taskType,
					Payload:      payload,
				}
				taskCreates = append(taskCreates, taskCreate)
			case storepb.ReleasePayload_File_DECLARATIVE:
				// error if applied revisions contain a higher version than the declarative file
				v, err := model.NewVersion(file.Version)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse file version %q", file.Version)
				}
				if maxDeclarativeVersion != nil && v.LessThan(maxDeclarativeVersion) {
					return nil, errors.Errorf("cannot run declarative file %q with version %q, because there is a higher versioned declarative revision %q applied", file.Id, file.Version, maxDeclarativeVersion.String())
				}

				// Parse sheet ID from the file's sheet reference
				_, sheetUID, err := common.GetProjectResourceIDSheetUID(file.Sheet)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get sheet id from sheet %q in release file %q", file.Sheet, file.Id)
				}
				payload := &storepb.Task{
					SpecId:        spec.Id,
					SheetId:       int32(sheetUID),
					SchemaVersion: file.Version,
					TaskReleaseSource: &storepb.TaskReleaseSource{
						File: common.FormatReleaseFile(c.Release, file.Id),
					},
				}
				env := ""
				if database.EffectiveEnvironmentID != nil {
					env = *database.EffectiveEnvironmentID
				}
				taskCreate := &store.TaskMessage{
					InstanceID:   database.InstanceID,
					DatabaseName: &database.DatabaseName,
					Environment:  env,
					Type:         storepb.Task_DATABASE_SCHEMA_UPDATE_SDL,
					Payload:      payload,
				}
				taskCreates = append(taskCreates, taskCreate)
			default:
				return nil, errors.Errorf("unsupported release file type %q", file.Type)
			}
		}
	}

	return taskCreates, nil
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
	default:
		return "", errors.Errorf("unsupported database type %s", dbType)
	}
}
