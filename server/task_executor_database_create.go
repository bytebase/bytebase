package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
)

// NewDatabaseCreateTaskExecutor creates a database create task executor.
func NewDatabaseCreateTaskExecutor() TaskExecutor {
	return &DatabaseCreateTaskExecutor{}
}

// DatabaseCreateTaskExecutor is the database create task executor.
type DatabaseCreateTaskExecutor struct {
	completed int32
}

// IsCompleted tells the scheduler if the task execution has completed.
func (exec *DatabaseCreateTaskExecutor) IsCompleted() bool {
	return atomic.LoadInt32(&exec.completed) == 1
}

// GetProgress returns the task progress.
func (*DatabaseCreateTaskExecutor) GetProgress() api.Progress {
	return api.Progress{}
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer atomic.StoreInt32(&exec.completed, 1)
	payload := &api.TaskDatabaseCreatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid create database payload")
	}

	statement := strings.TrimSpace(payload.Statement)
	if statement == "" {
		return true, nil, errors.Errorf("empty create database statement")
	}

	instance := task.Instance
	driver, err := server.dbFactory.GetAdminDatabaseDriver(ctx, task.Instance, "" /* databaseName */)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	project, err := server.store.GetProjectByID(ctx, payload.ProjectID)
	if err != nil {
		return true, nil, errors.Errorf("failed to find project with ID %d", payload.ProjectID)
	}
	if project == nil {
		return true, nil, errors.Errorf("project not found with ID %d", payload.ProjectID)
	}

	var schemaVersion string
	// We will use schema from existing tenant databases for creating a database in a tenant mode project if possible.
	if project.TenantMode == api.TenantModeTenant {
		baseDatabaseName, err := api.GetBaseDatabaseName(payload.DatabaseName, project.DBNameTemplate, payload.Labels)
		if err != nil {
			return true, nil, errors.Wrapf(err, "api.GetBaseDatabaseName(%q, %q, %q) failed", payload.DatabaseName, project.DBNameTemplate, payload.Labels)
		}
		sv, schema, err := exec.getSchemaFromPeerTenantDatabase(ctx, server.store, server.dbFactory, instance, project, project.ID, baseDatabaseName)
		if err != nil {
			return true, nil, err
		}
		schemaVersion = sv
		connectionStmt, err := getConnectionStatement(instance.Engine, payload.DatabaseName)
		if err != nil {
			return true, nil, err
		}
		if !strings.Contains(payload.Statement, connectionStmt) {
			statement = fmt.Sprintf("%s\n%s\n%s", statement, connectionStmt, schema)
		}
	}
	if schemaVersion == "" {
		schemaVersion = common.DefaultMigrationVersion()
	}

	log.Debug("Start creating database...",
		zap.String("instance", instance.Name),
		zap.String("database", payload.DatabaseName),
		zap.String("schemaVersion", schemaVersion),
		zap.String("statement", statement),
	)

	// Create a migrate migration history upon creating the database.
	// TODO(d): support semantic versioning.
	mi := &db.MigrationInfo{
		ReleaseVersion: server.profile.Version,
		Version:        schemaVersion,
		Namespace:      payload.DatabaseName,
		Database:       payload.DatabaseName,
		Environment:    instance.Environment.Name,
		Source:         db.UI,
		Type:           db.Migrate,
		Description:    "Create database",
		CreateDatabase: true,
		Force:          true,
	}
	creator, err := server.store.GetPrincipalByID(ctx, task.CreatorID)
	if err != nil {
		// If somehow we unable to find the principal, we just emit the error since it's not
		// critical enough to fail the entire operation.
		log.Error("Failed to fetch creator for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		mi.Creator = creator.Name
	}
	issue, err := server.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		// If somehow we unable to find the issue, we just emit the error since it's not
		// critical enough to fail the entire operation.
		log.Error("Failed to fetch containing issue for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	}
	if issue == nil {
		err := errors.Errorf("failed to fetch containing issue for composing the migration info, issue not found with pipeline ID %v", task.PipelineID)
		log.Error(err.Error(),
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		mi.IssueID = strconv.Itoa(issue.ID)
	}

	migrationID, _, err := driver.ExecuteMigration(ctx, mi, statement)
	if err != nil {
		return true, nil, err
	}

	// If the database creation statement executed successfully,
	// then we will create a database entry immediately
	// instead of waiting for the next schema sync cycle to sync over this newly created database.
	// This is for 2 reasons:
	// 1. Assign the proper project to the newly created database. Otherwise, the periodic schema
	// sync will place the synced db into the default project.
	// 2. Allow user to see the created database right away.
	database, err := server.store.GetDatabase(ctx, &api.DatabaseFind{InstanceID: &task.InstanceID, Name: &payload.DatabaseName})
	if err != nil {
		return true, nil, err
	}
	if database == nil {
		databaseCreate := &api.DatabaseCreate{
			CreatorID:            api.SystemBotID,
			ProjectID:            payload.ProjectID,
			InstanceID:           task.InstanceID,
			EnvironmentID:        instance.EnvironmentID,
			Name:                 payload.DatabaseName,
			CharacterSet:         payload.CharacterSet,
			Collation:            payload.Collation,
			LastSuccessfulSyncTs: time.Now().Unix(),
			Labels:               &payload.Labels,
			SchemaVersion:        schemaVersion,
		}
		createdDatabase, err := server.store.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			return true, nil, err
		}
		database = createdDatabase
	} else {
		// The database didn't exist before the current run so there was a race condition between sync schema and migration execution.
		// We need to update the project ID from the default project to the target project.
		updatedDatabase, err := server.store.PatchDatabase(ctx, &api.DatabasePatch{ID: database.ID, UpdaterID: api.SystemBotID, ProjectID: &payload.ProjectID})
		if err != nil {
			return true, nil, err
		}
		database = updatedDatabase
	}
	// Set database labels, except bb.environment is immutable and must match instance environment.
	if err := server.setDatabaseLabels(ctx, payload.Labels, database, project, database.CreatorID, false); err != nil {
		return true, nil, errors.Errorf("failed to record database labels after creating database %v", database.ID)
	}

	// After the task related database entry created successfully,
	// we need to update task's database_id and statement with the newly created database immediately.
	// Here is the main reason:
	// The task database_id represents its related database entry both for creating and patching,
	// so we should sync its value right here when the related database entry created.
	// The new statement should include the schema from peer tenant database.
	payload.Statement = statement
	bytes, err := json.Marshal(payload)
	if err != nil {
		return true, nil, errors.Wrap(err, "Failed to construct updated task payload")
	}
	payloadStr := string(bytes)
	taskDatabaseIDPatch := &api.TaskPatch{
		ID:         task.ID,
		UpdaterID:  api.SystemBotID,
		DatabaseID: &database.ID,
		Payload:    &payloadStr,
	}
	if _, err = server.store.PatchTask(ctx, taskDatabaseIDPatch); err != nil {
		return true, nil, err
	}

	return true, &api.TaskRunResultPayload{
		Detail:      fmt.Sprintf("Created database %q", payload.DatabaseName),
		MigrationID: migrationID,
		Version:     mi.Version,
	}, nil
}

func getCreateDatabaseStatement(dbType db.Type, createDatabaseContext api.CreateDatabaseContext, databaseName, adminDatasourceUser string) (string, error) {
	var stmt string
	switch dbType {
	case db.MySQL, db.TiDB:
		return fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", databaseName, createDatabaseContext.CharacterSet, createDatabaseContext.Collation), nil
	case db.Postgres:
		// On Cloud RDS, the data source role isn't the actual superuser with sudo privilege.
		// We need to grant the database owner role to the data source admin so that Bytebase can have permission for the database using the data source admin.
		if adminDatasourceUser != "" && createDatabaseContext.Owner != adminDatasourceUser {
			stmt = fmt.Sprintf("GRANT \"%s\" TO \"%s\";\n", createDatabaseContext.Owner, adminDatasourceUser)
		}
		if createDatabaseContext.Collation == "" {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q;", stmt, databaseName, createDatabaseContext.CharacterSet)
		} else {
			stmt = fmt.Sprintf("%sCREATE DATABASE \"%s\" ENCODING %q LC_COLLATE %q;", stmt, databaseName, createDatabaseContext.CharacterSet, createDatabaseContext.Collation)
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
		return fmt.Sprintf("%s\nALTER DATABASE \"%s\" OWNER TO %s;", stmt, databaseName, createDatabaseContext.Owner), nil
	case db.ClickHouse:
		clusterPart := ""
		if createDatabaseContext.Cluster != "" {
			clusterPart = fmt.Sprintf(" ON CLUSTER `%s`", createDatabaseContext.Cluster)
		}
		return fmt.Sprintf("CREATE DATABASE `%s`%s;", databaseName, clusterPart), nil
	case db.Snowflake:
		return fmt.Sprintf("CREATE DATABASE %s;", databaseName), nil
	case db.SQLite:
		// This is a fake CREATE DATABASE and USE statement since a single SQLite file represents a database. Engine driver will recognize it and establish a connection to create the sqlite file representing the database.
		return fmt.Sprintf("CREATE DATABASE '%s';", databaseName), nil
	}
	return "", errors.Errorf("unsupported database type %s", dbType)
}

func getConnectionStatement(dbType db.Type, databaseName string) (string, error) {
	switch dbType {
	case db.MySQL, db.TiDB:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case db.Postgres:
		return fmt.Sprintf("\\connect \"%s\";\n", databaseName), nil
	case db.ClickHouse:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	case db.Snowflake:
		return fmt.Sprintf("USE DATABASE %s;\n", databaseName), nil
	case db.SQLite:
		return fmt.Sprintf("USE `%s`;\n", databaseName), nil
	}

	return "", errors.Errorf("unsupported database type %s", dbType)
}

// getSchemaFromPeerTenantDatabase gets the schema version and schema from a peer tenant database.
// It's used for creating a database in a tenant mode project.
// When a peer tenant database doesn't exist, we will return an error if there are databases in the project with the same name.
// Otherwise, we will create a blank database without schema.
func (*DatabaseCreateTaskExecutor) getSchemaFromPeerTenantDatabase(ctx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, instance *api.Instance, project *api.Project, projectID int, baseDatabaseName string) (string, string, error) {
	// Find all databases in the project.
	dbList, err := store.FindDatabase(ctx, &api.DatabaseFind{
		ProjectID: &projectID,
	})
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch databases in project ID: %v", projectID)).SetInternal(err)
	}

	deployConfig, err := store.GetDeploymentConfigByProjectID(ctx, projectID)
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch deployment config for project ID: %v", projectID)).SetInternal(err)
	}
	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(deployConfig.Payload)
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, "Failed to get deployment schedule").SetInternal(err)
	}
	matrix, err := getDatabaseMatrixFromDeploymentSchedule(deploySchedule, baseDatabaseName, project.DBNameTemplate, dbList)
	if err != nil {
		return "", "", echo.NewHTTPError(http.StatusInternalServerError, "Failed to create deployment pipeline").SetInternal(err)
	}
	similarDB := getPeerTenantDatabase(matrix, instance.EnvironmentID)

	// When there is no existing tenant, we will look at all existing databases in the tenant mode project.
	// If there are existing databases with the same name, we will disallow the database creation.
	// Otherwise, we will create a blank new database.
	if similarDB == nil {
		// Ignore the database name conflict if the template is empty.
		if project.DBNameTemplate == "" {
			return "", "", nil
		}

		found := false
		for _, db := range dbList {
			var labelList []*api.DatabaseLabel
			if err := json.Unmarshal([]byte(db.Labels), &labelList); err != nil {
				return "", "", errors.Wrapf(err, "failed to unmarshal labels for database ID %v name %q", db.ID, db.Name)
			}
			labelMap := map[string]string{}
			for _, label := range labelList {
				labelMap[label.Key] = label.Value
			}
			dbName, err := formatDatabaseName(baseDatabaseName, project.DBNameTemplate, labelMap)
			if err != nil {
				return "", "", errors.Wrapf(err, "failed to format database name formatDatabaseName(%q, %q, %+v)", baseDatabaseName, project.DBNameTemplate, labelMap)
			}
			if db.Name == dbName {
				found = true
				break
			}
		}
		if found {
			err := errors.Errorf("conflicting database name, project has existing base database named %q, but it's not from the selected peer tenants", baseDatabaseName)
			return "", "", echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		return "", "", nil
	}

	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, similarDB.Instance, similarDB.Name)
	if err != nil {
		return "", "", err
	}
	defer driver.Close(ctx)
	schemaVersion, err := getLatestSchemaVersion(ctx, driver, similarDB.Name)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get migration history for database %q", similarDB.Name)
	}

	var schemaBuf bytes.Buffer
	if _, err := driver.Dump(ctx, similarDB.Name, &schemaBuf, true /* schemaOnly */); err != nil {
		return "", "", err
	}
	return schemaVersion, schemaBuf.String(), nil
}

func getPeerTenantDatabase(databaseMatrix [][]*api.Database, environmentID int) *api.Database {
	var similarDB *api.Database
	// We try to use an existing tenant with the same environment, if possible.
	for _, databaseList := range databaseMatrix {
		for _, db := range databaseList {
			if db.Instance.EnvironmentID == environmentID {
				similarDB = db
				break
			}
		}
		if similarDB != nil {
			break
		}
	}
	if similarDB == nil {
		for _, stage := range databaseMatrix {
			if len(stage) > 0 {
				similarDB = stage[0]
				break
			}
		}
	}

	return similarDB
}
