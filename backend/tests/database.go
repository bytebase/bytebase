package tests

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	if err := ctl.registerDatabase(ctx, instance, databaseName); err != nil {
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

// instanceLocks serializes registerDatabase per instance: parallel subtests
// share instances, and a sync that reads the list before another subtest's
// update marks that subtest's database deleted.
var (
	instanceLocksMu sync.Mutex
	instanceLocks   = map[string]*sync.Mutex{}
)

func instanceLock(name string) *sync.Mutex {
	instanceLocksMu.Lock()
	defer instanceLocksMu.Unlock()
	if instanceLocks[name] == nil {
		instanceLocks[name] = &sync.Mutex{}
	}
	return instanceLocks[name]
}

// registerDatabase names a database in the instance's sync list, which an
// instance on the shared target carries so it never imports another test's
// databases, and syncs the instance. An instance with a server to itself keeps
// no list and only syncs.
func (ctl *controller) registerDatabase(ctx context.Context, instance *v1pb.Instance, databaseName string) error {
	mu := instanceLock(instance.Name)
	mu.Lock()
	defer mu.Unlock()

	if instance.GetSyncDatabases() != nil {
		resp, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
		if err != nil {
			return err
		}
		if databases := resp.Msg.GetSyncDatabases().GetDatabases(); !slices.Contains(databases, databaseName) {
			if _, err := ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
				Instance: &v1pb.Instance{
					Name:          instance.Name,
					SyncDatabases: &v1pb.SyncDatabases{Databases: append(databases, databaseName)},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"sync_databases"}},
			})); err != nil {
				return err
			}
		}
	}
	_, err := ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	return err
}
