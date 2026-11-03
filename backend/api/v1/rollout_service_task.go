package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

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
						ProjectIDs: []string{projectID},
					})
					if err != nil {
						return nil, errors.Wrapf(err, "failed to get database group")
					}
					if databaseGroup == nil {
						return nil, errors.Errorf("database group %q not found", config.Targets[0])
					}

					allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{Workspace: common.GetWorkspaceIDFromContext(ctx), ProjectID: &projectID})
					if err != nil {
						return nil, errors.Wrapf(err, "failed to list databases for project %q", projectID)
					}

					matchedDatabases, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to get matched databases in database group %q", databaseGroupID)
					}

					var databases []string
					for _, db := range matchedDatabases {
						databases = append(databases, db.ResourceName())
					}
					config.Targets = databases
				}
			}
		}
		result = append(result, clonedSpec)
	}
	return result, nil
}

func getTaskCreatesFromSpec(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, projectID string) ([]*store.TaskMessage, error) {
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		return getTaskCreatesFromCreateDatabaseConfig(ctx, s, spec, config.CreateDatabaseConfig, projectID)
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getTaskCreatesFromChangeDatabaseConfig(ctx, s, spec, config.ChangeDatabaseConfig, projectID)
	}

	return nil, errors.Errorf("invalid spec config type %T", spec.Config)
}

func getTaskCreatesFromCreateDatabaseConfig(ctx context.Context, s *store.Store, spec *storepb.PlanConfig_Spec, c *storepb.PlanConfig_CreateDatabaseConfig, projectID string) ([]*store.TaskMessage, error) {
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
		sheets, err := s.CreateSheets(ctx, projectID, &store.SheetMessage{
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
	projectID string,
) ([]*store.TaskMessage, error) {
	databases, err := getDatabaseMessagesByTargets(ctx, s, c.Targets)
	if err != nil {
		return nil, err
	}

	// Validate ghost directive at task creation time (parsed at execution
	// time). c.SheetSha256 is loop-invariant: fetch it once, not once per
	// target database.
	if c.SheetSha256 != "" {
		sheetContent, err := getSheetContentBySha256(ctx, s, projectID, c.SheetSha256)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet content")
		}

		if ghost.IsGhostEnabled(sheetContent) {
			ghostFlags, err := ghost.ParseGhostDirective(sheetContent)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse ghost directive")
			}
			if _, err := ghost.GetUserFlags(ghostFlags); err != nil {
				return nil, errors.Wrapf(err, "invalid ghost flags %q", ghostFlags)
			}
		}
	}

	var versionedRelease *store.ReleaseMessage
	if c.Release != "" {
		versionedRelease, err = getVersionedRelease(ctx, s, c.Release)
		if err != nil {
			return nil, err
		}
	}

	var releaseStates map[string]*storepb.Task_ReleaseState
	if versionedRelease != nil {
		releaseStates, err = computeReleaseStates(ctx, s, versionedRelease, databases)
		if err != nil {
			return nil, err
		}
	}

	var tasks []*store.TaskMessage
	for _, database := range databases {
		if database.EffectiveEnvironmentID == nil {
			continue
		}
		env := *database.EffectiveEnvironmentID

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
			if versionedRelease != nil {
				payload.ReleaseState = releaseStates[releaseStateKey(database)]
			}
		} else {
			payload.Source = &storepb.Task_SheetSha256{
				SheetSha256: c.SheetSha256,
			}
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

	if len(tasks) == 0 {
		return nil, errors.Errorf("cannot create tasks for spec %s: all target databases are missing an environment", spec.Id)
	}

	return tasks, nil
}

// getVersionedRelease returns the release, or nil when it is declarative:
// a declarative run computes its diff when it runs, so it has no state to store.
func getVersionedRelease(ctx context.Context, s *store.Store, releaseName string) (*store.ReleaseMessage, error) {
	projectID, releaseID, err := common.GetProjectReleaseID(releaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse release name %q", releaseName)
	}
	release, err := s.GetRelease(ctx, &store.FindReleaseMessage{ProjectID: &projectID, ReleaseID: &releaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get release %q", releaseName)
	}
	if release == nil {
		return nil, errors.Errorf("release %q not found", releaseName)
	}
	if release.Payload.GetType() != storepb.SchemaChangeType_VERSIONED {
		return nil, nil
	}
	return release, nil
}

// revisionLister is the one store read the release state needs.
type revisionLister interface {
	ListRevisions(ctx context.Context, find *store.FindRevisionMessage) ([]*store.RevisionMessage, error)
}

func releaseStateKey(database *store.DatabaseMessage) string {
	return database.InstanceID + "/" + database.DatabaseName
}

// computeReleaseStates applies the executor's rule (runVersionedRelease) to
// every target: a version is applied when the database has a live VERSIONED
// revision with it. One query per instance, bounded by the release's file
// count and the instance's targets, so a large database group stays one round
// trip per instance. The states are keyed by releaseStateKey.
func computeReleaseStates(ctx context.Context, lister revisionLister, release *store.ReleaseMessage, databases []*store.DatabaseMessage) (map[string]*storepb.Task_ReleaseState, error) {
	versions := make([]string, 0, len(release.Payload.GetFiles()))
	for _, file := range release.Payload.GetFiles() {
		versions = append(versions, file.Version)
	}

	var instances []string
	namesOf := map[string][]string{}
	seen := map[string]bool{}
	for _, database := range databases {
		key := releaseStateKey(database)
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, ok := namesOf[database.InstanceID]; !ok {
			instances = append(instances, database.InstanceID)
		}
		namesOf[database.InstanceID] = append(namesOf[database.InstanceID], database.DatabaseName)
	}

	versioned := storepb.SchemaChangeType_VERSIONED
	revisionsOf := map[string][]*store.RevisionMessage{}
	for _, instanceID := range instances {
		names := namesOf[instanceID]
		revisions, err := lister.ListRevisions(ctx, &store.FindRevisionMessage{
			InstanceID:    instanceID,
			DatabaseNames: &names,
			Type:          &versioned,
			Versions:      &versions,
			ShowDeleted:   true,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list revisions on instance %q", instanceID)
		}
		for _, revision := range revisions {
			key := revision.InstanceID + "/" + revision.DatabaseName
			revisionsOf[key] = append(revisionsOf[key], revision)
		}
	}

	computeTime := timestamppb.Now()
	states := map[string]*storepb.Task_ReleaseState{}
	for key := range seen {
		states[key] = releaseStateOf(versions, revisionsOf[key], computeTime)
	}
	return states, nil
}

func releaseStateOf(versions []string, revisions []*store.RevisionMessage, computeTime *timestamppb.Timestamp) *storepb.Task_ReleaseState {
	applied := map[string]bool{}
	lastDeleted := map[string]*store.RevisionMessage{}
	for _, revision := range revisions {
		if revision.DeletedAt == nil {
			applied[revision.Version] = true
			continue
		}
		if prev, ok := lastDeleted[revision.Version]; !ok || revision.DeletedAt.After(*prev.DeletedAt) {
			lastDeleted[revision.Version] = revision
		}
	}

	state := &storepb.Task_ReleaseState{ComputeTime: computeTime}
	for _, version := range versions {
		if applied[version] {
			state.AppliedCount++
			continue
		}
		pending := &storepb.Task_ReleaseState_PendingVersion{Version: version}
		if deleted, ok := lastDeleted[version]; ok {
			pending.RevisionDeleteTime = timestamppb.New(*deleted.DeletedAt)
			if deleted.Deleter != nil {
				pending.RevisionDeleter = *deleted.Deleter
			}
		}
		state.PendingVersions = append(state.PendingVersions, pending)
	}
	return state
}

func getDatabaseMessagesByTargets(ctx context.Context, s *store.Store, targets []string) ([]*store.DatabaseMessage, error) {
	databases := []*store.DatabaseMessage{}

	for _, target := range targets {
		if targetProjectID, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			databaseGroup, err := getDatabaseGroupByName(ctx, s, target, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database group %q", target)
			}
			for _, matched := range databaseGroup.MatchedDatabases {
				_, instanceID, databaseName, err := common.GetDatabaseResourceName(matched.Name)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse %q", matched.Name)
				}
				database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
					Workspace:    common.GetWorkspaceIDFromContext(ctx),
					InstanceID:   &instanceID,
					DatabaseName: &databaseName,
				})
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database")
				}
				if database == nil {
					return nil, errors.Errorf("database %q not found", matched.Name)
				}
				if database.ProjectID != targetProjectID {
					return nil, errors.Errorf("database %q does not belong to project %q", matched.Name, targetProjectID)
				}
				if database.ResourceName() != matched.Name {
					return nil, errors.Errorf("database target %q is not canonical for its instance", matched.Name)
				}
				databases = append(databases, database)
			}
		} else if targetProjectID, instanceID, databaseName, err := common.GetDatabaseResourceName(target); err == nil {
			database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{
				Workspace:    common.GetWorkspaceIDFromContext(ctx),
				InstanceID:   &instanceID,
				DatabaseName: &databaseName,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database")
			}
			if database == nil {
				return nil, errors.Errorf("database %q not found", target)
			}
			if targetProjectID != nil && database.ProjectID != *targetProjectID {
				return nil, errors.Errorf("database %q does not belong to project %q", target, *targetProjectID)
			}
			if database.ResourceName() != target {
				return nil, errors.Errorf("database target %q is not canonical for its instance", target)
			}
			databases = append(databases, database)
		} else {
			return nil, errors.Errorf("invalid target %q", target)
		}
	}
	if err := validateRolloutDatabaseInstances(ctx, s, databases); err != nil {
		return nil, err
	}
	return databases, nil
}

func validateRolloutDatabaseInstances(ctx context.Context, s *store.Store, databases []*store.DatabaseMessage) error {
	instanceIDs := make([]string, 0, len(databases))
	seen := make(map[string]bool, len(databases))
	for _, database := range databases {
		if !seen[database.InstanceID] {
			seen[database.InstanceID] = true
			instanceIDs = append(instanceIDs, database.InstanceID)
		}
	}
	if len(instanceIDs) == 0 {
		return nil
	}

	instances, err := s.ListInstances(ctx, &store.FindInstanceMessage{
		Workspace:   common.GetWorkspaceIDFromContext(ctx),
		ResourceIDs: &instanceIDs,
		ShowDeleted: true,
	})
	if err != nil {
		return errors.Wrap(err, "failed to list rollout target instances")
	}
	instanceByID := make(map[string]*store.InstanceMessage, len(instances))
	for _, instance := range instances {
		instanceByID[instance.ResourceID] = instance
	}
	for _, database := range databases {
		instance := instanceByID[database.InstanceID]
		if instance == nil {
			return errors.Errorf("instance %q not found", database.InstanceID)
		}
		if instance.Deleted {
			return errors.Errorf("instance %q has been deleted", database.InstanceID)
		}
	}
	return nil
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
	case storepb.Engine_MONGODB, storepb.Engine_MSSQL, storepb.Engine_DORIS, storepb.Engine_STARROCKS:
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
	case storepb.Engine_MONGODB:
		// MongoDB has no top-level CREATE DATABASE — a database is implicitly created
		// when its first collection is. We pass the database name to the engine driver
		// (which uses it to build the connection URI) and emit createCollection.
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

func getSheetContentBySha256(ctx context.Context, s *store.Store, projectID, sha256 string) (string, error) {
	sheet, err := s.GetSheetForProject(ctx, projectID, sha256, true)
	if err != nil {
		return "", err
	}
	if sheet == nil {
		return "", errors.Errorf("sheet not found for sha256: %s", sha256)
	}
	return sheet.Statement, nil
}
