package tests

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// createDatabase gives a test a database on a Postgres instance and registers
// it in the project, as the create-database task leaves it: schema synced, and
// the environment set when one is given. It skips the rollout, which costs a
// plan, an issue, a task and a PIPELINE_COMPLETED webhook per database;
// TestDatabaseEnvironment keeps that path covered, through
// createDatabaseByRollout.
func (ctl *controller) createDatabase(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, environment *v1pb.EnvironmentSetting_Environment, databaseName string, owner string) error {
	pg, err := pgContainerOf(instance)
	if err != nil {
		return err
	}
	if err := ctl.allowLastPlanEditorApproval(ctx, project); err != nil {
		return err
	}
	name := pgx.Identifier{databaseName}.Sanitize()
	if _, err := pg.GetDB().ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		return err
	}
	if owner != "" {
		if _, err := pg.GetDB().ExecContext(ctx, "ALTER DATABASE "+name+" OWNER TO "+pgx.Identifier{owner}.Sanitize()); err != nil {
			return err
		}
	}
	if err := ctl.waitDatabaseRegistered(ctx, instance, databaseName); err != nil {
		return err
	}

	database := &v1pb.Database{
		Name:    fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
		Project: project.Name,
	}
	paths := []string{"project"}
	if environment != nil {
		database.Environment = &environment.Name
		paths = append(paths, "environment")
	}
	if _, err := ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database:   database,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
	})); err != nil {
		return err
	}
	_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{Name: database.Name}))
	return err
}

// createDatabaseByRollout creates a database through the create-database plan,
// issue and rollout, for the engines createDatabase does not reach and the test
// that covers the workflow itself.
func (ctl *controller) createDatabaseByRollout(ctx context.Context, project *v1pb.Project, instance *v1pb.Instance, environment *v1pb.EnvironmentSetting_Environment, databaseName string) error {
	if err := ctl.registerDatabase(ctx, instance, databaseName); err != nil {
		return err
	}
	if err := ctl.allowLastPlanEditorApproval(ctx, project); err != nil {
		return err
	}

	characterSet, collation, owner := "utf8mb4", "utf8mb4_general_ci", ""
	if instance.Engine == v1pb.Engine_POSTGRES {
		characterSet, collation, owner = "UTF8", "en_US.UTF-8", "postgres"
	}
	environmentName := ""
	if environment != nil {
		environmentName = environment.Name
	}

	planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: uuid.NewString(),
					Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
							Target:       instance.Name,
							Database:     databaseName,
							CharacterSet: characterSet,
							Collation:    collation,
							Owner:        owner,
							Environment:  environmentName,
						},
					},
				},
			},
		},
	}))
	if err != nil {
		return err
	}
	issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       fmt.Sprintf("create database %q", databaseName),
			Description: fmt.Sprintf("This creates a database %q.", databaseName),
			Plan:        planResp.Msg.Name,
			Type:        v1pb.Issue_DATABASE_CHANGE,
		},
	}))
	if err != nil {
		return err
	}
	rolloutResp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: planResp.Msg.Name}))
	if err != nil {
		return err
	}

	return ctl.waitRollout(ctx, issueResp.Msg.Name, rolloutResp.Msg.Name)
}

// allowLastPlanEditorApproval opts the project into approving a plan its own
// editor wrote. Provisioning is test setup approved by the same test actor, and
// tests go on to approve their own plans; production projects keep the
// restrictive default.
func (ctl *controller) allowLastPlanEditorApproval(ctx context.Context, project *v1pb.Project) error {
	_, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:                        project.Name,
			AllowLastPlanEditorApproval: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_last_plan_editor_approval"}},
	}))
	return err
}

// allowJustInTimeAccess opts the project into just-in-time access, which
// CreateAccessGrant requires.
func (ctl *controller) allowJustInTimeAccess(ctx context.Context, project *v1pb.Project) error {
	_, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:                  project.Name,
			AllowJustInTimeAccess: true,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_just_in_time_access"}},
	}))
	return err
}

// instanceSyncLists holds, per instance, the sync list registerDatabase has
// written and the lock that serializes those writes. The list is rebuilt from
// this record rather than from a read of the instance: the server serves
// instances from a cache, so a read can miss another subtest's update, and
// writing the list back without that subtest's database makes the next sync
// mark it deleted under the test still using it.
var (
	instanceSyncListsMu sync.Mutex
	instanceSyncLists   = map[string]*instanceSyncList{}
)

type instanceSyncList struct {
	mu        sync.Mutex
	databases []string
}

func instanceSyncListOf(instance *v1pb.Instance) *instanceSyncList {
	instanceSyncListsMu.Lock()
	defer instanceSyncListsMu.Unlock()
	if instanceSyncLists[instance.Name] == nil {
		instanceSyncLists[instance.Name] = &instanceSyncList{databases: slices.Clone(instance.GetSyncDatabases().GetDatabases())}
	}
	return instanceSyncLists[instance.Name]
}

// registerDatabase names a database in the instance's sync list, which an
// instance on the shared target carries so it never imports another test's
// databases, and syncs the instance. An instance with a server to itself keeps
// no list and only syncs. The database need not exist yet: the list may name
// databases the target does not hold, and the sync passes over them.
func (ctl *controller) registerDatabase(ctx context.Context, instance *v1pb.Instance, databaseName string) error {
	syncList := instanceSyncListOf(instance)
	syncList.mu.Lock()
	defer syncList.mu.Unlock()

	if !slices.Contains(syncList.databases, databaseName) {
		syncList.databases = append(syncList.databases, databaseName)
	}
	if instance.GetSyncDatabases() != nil {
		if _, err := ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
			Instance: &v1pb.Instance{
				Name:          instance.Name,
				SyncDatabases: &v1pb.SyncDatabases{Databases: slices.Clone(syncList.databases)},
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"sync_databases"}},
		})); err != nil {
			return err
		}
	}
	_, err := ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	return err
}

// instanceSyncAttempts bounds the syncs waitDatabaseRegistered spends on one
// database.
const instanceSyncAttempts = 3

// waitDatabaseRegistered registers a database the target already holds and
// returns once the instance sync has given it an active resource, which the
// caller needs before it can address the database. One sync can miss it: the
// sync reads the list through the server's instance cache, which a concurrent
// read can refill with a copy taken before this database joined it.
func (ctl *controller) waitDatabaseRegistered(ctx context.Context, instance *v1pb.Instance, databaseName string) error {
	name := fmt.Sprintf("%s/databases/%s", instance.Name, databaseName)
	var last error
	for range instanceSyncAttempts {
		if err := ctl.registerDatabase(ctx, instance, databaseName); err != nil {
			return err
		}
		resp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: name}))
		switch {
		case err != nil:
			last = err
		case resp.Msg.State != v1pb.State_ACTIVE:
			last = errors.Errorf("database %s is %s", name, resp.Msg.State)
		default:
			return nil
		}
	}
	return errors.Wrapf(last, "database %s is not registered after %d instance syncs", name, instanceSyncAttempts)
}
