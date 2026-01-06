package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/ghost"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func applyDatabaseGroupSpecTransformations(ctx context.Context, s *store.Store, specs []*storepb.PlanConfig_Spec, projectID string) ([]*storepb.PlanConfig_Spec, error) {
	var result []*storepb.PlanConfig_Spec
	for _, spec := range specs {
		// Clone the spec to avoid modifying the original
		clonedSpec := proto.CloneOf(spec)

		if config := clonedSpec.GetChangeDatabaseConfig(); config != nil {
			// transform database group.
			if len(config.Targets) == 1 {
				if _, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(config.Targets[0]); err == nil {
					// Re-evaluate database group matches live instead of using deployment snapshot
					databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
						ResourceID: &databaseGroupID,
						ProjectID:  &projectID,
					})
					if err != nil {
						return nil, errors.Wrapf(err, "failed to get database group")
					}
					if databaseGroup == nil {
						return nil, errors.Errorf("database group %q not found", config.Targets[0])
					}

					allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &projectID})
					if err != nil {
						return nil, errors.Wrapf(err, "failed to list databases for project %q", projectID)
					}

					matchedDatabases, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to get matched databases in database group %q", databaseGroupID)
					}

					var databases []string
					for _, db := range matchedDatabases {
						databases = append(databases, common.FormatDatabase(db.InstanceID, db.DatabaseName))
					}
					config.Targets = databases
				}
			}
		}
		result = append(result, clonedSpec)
	}
	return result, nil
}

func getTaskCreatesFromSpec(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec) ([]*store.TaskMessage, error) {
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s, spec, config.CreateDatabaseConfig)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s, spec, config.ChangeDatabaseConfig)
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		return getTaskCreatesFromExportDataConfig(ctx, s, spec, config.ExportDataConfig)
	}

	return nil, errors.Errorf("invalid spec config type %T", spec.Config)
}

func getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_CreateDatabaseConfig) ([]*store.TaskMessage, error) {
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
			if !store.IsObjectCaseSensitive(instance) {
				databaseName = strings.ToLower(databaseName)
			}
		default:
			// Other engines use the original database name
		}

		statement, err := getCreateDatabaseStatement(instance.Metadata.GetEngine(), c, databaseName, adminDataSource.GetUsername())
		if err != nil {
			return nil, err
		}
		sheets, err := s.CreateSheets(ctx, &store.SheetMessage{
			Statement: statement,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database creation sheet")
		}
		sheet := sheets[0]

		v := &store.TaskMessage{
			InstanceID:   instance.ResourceID,
			DatabaseName: &databaseName,
			Environment:  effectiveEnvironmentID,
			Type:         storepb.Task_DATABASE_CREATE,
			Payload: &storepb.Task{
				SpecId: spec.Id,
				Source: &storepb.Task_SheetSha256{
					SheetSha256: sheet.Sha256,
				},
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

	var tasks []*store.TaskMessage
	for _, database := range databases {
		env := ""
		if database.EffectiveEnvironmentID != nil {
			env = *database.EffectiveEnvironmentID
		}

		// All change database tasks use DATABASE_MIGRATE type
		taskType := storepb.Task_DATABASE_MIGRATE

		// Build task payload
		payload := &storepb.Task{
			SpecId:            spec.Id,
			EnablePriorBackup: c.EnablePriorBackup,
		}

		// Set source: either release or sheet
		if c.Release != "" {
			payload.Source = &storepb.Task_Release{
				Release: c.Release,
			}
		} else {
			payload.Source = &storepb.Task_SheetSha256{
				SheetSha256: c.SheetSha256,
			}
		}

		// Add ghost flags if specified
		if c.EnableGhost {
			if _, err := ghost.GetUserFlags(c.GhostFlags); err != nil {
				return nil, errors.Wrapf(err, "invalid ghost flags %q", c.GhostFlags)
			}
			payload.Flags = c.GhostFlags
			payload.EnableGhost = c.EnableGhost
		}

		taskCreate := &store.TaskMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Environment:  env,
			Type:         taskType,
			Payload:      payload,
		}
		tasks = append(tasks, taskCreate)
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
				instanceID, databaseName, err := common.GetInstanceDatabaseID(matched.Name)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse %q", matched.Name)
				}
				database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
					InstanceID:   &instanceID,
					DatabaseName: &databaseName,
				})
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database")
				}
				if database == nil {
					return nil, errors.Errorf("database %q not found", matched.Name)
				}
				databases = append(databases, database)
			}
		} else if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
			instanceID, databaseName, err := common.GetInstanceDatabaseID(target)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse %q", target)
			}
			database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
				InstanceID:   &instanceID,
				DatabaseName: &databaseName,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database")
			}
			if database == nil {
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

	payload := &storepb.Task{
		SpecId: spec.Id,
		Source: &storepb.Task_SheetSha256{
			SheetSha256: c.SheetSha256,
		},
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
	case storepb.Engine_SQLITE, storepb.Engine_MONGODB, storepb.Engine_MSSQL, storepb.Engine_DORIS, storepb.Engine_STARROCKS:
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
	case storepb.Engine_DORIS:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case storepb.Engine_STARROCKS:
		return fmt.Sprintf("CREATE DATABASE `%s`;", databaseName), nil
	default:
		return "", errors.Errorf("unsupported database type %s", dbType)
	}
}
