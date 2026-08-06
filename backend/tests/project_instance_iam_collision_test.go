package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestProjectInstanceIAM(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	ownerToken := ctl.authInterceptor.token
	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)

	otherProjectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("instance-other-project"),
		Project:   &v1pb.Project{Title: "Other project", AllowSelfApproval: true},
	}))
	a.NoError(err)
	otherProject := otherProjectResp.Msg

	instanceAID := generateRandomString("project-instance-a")
	instanceB := bot35CreateProjectInstance(ctx, t, ctl, otherProject.Name, generateRandomString("project-instance-b"), "Project B instance", pgContainer.adminDataSource(), true)

	projectOwnerToken := bot35CreateProjectUser(ctx, t, ctl, ctl.project.Name, "roles/projectOwner", "project-owner")
	ctl.authInterceptor.token = projectOwnerToken
	instanceA := bot35CreateProjectInstance(ctx, t, ctl, ctl.project.Name, instanceAID, "Project A instance", pgContainer.adminDataSource(), true)

	listA, err := ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{
		Parent: new(ctl.project.Name),
	}))
	a.NoError(err)
	a.Len(listA.Msg.Instances, 1)
	a.Equal(instanceA.Name, listA.Msg.Instances[0].Name)

	_, err = ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{Parent: new(otherProject.Name)}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:     new(otherProject.Name),
		InstanceId: generateRandomString("other-project-instance"),
		Instance: &v1pb.Instance{
			Title:       "Must not create in another project",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	gotA, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceA.Name}))
	a.NoError(err)
	a.Equal(instanceA.Name, gotA.Msg.Name)

	_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceB.Name}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	updatedA, err := ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
		Instance:   &v1pb.Instance{Name: instanceA.Name, Title: "Project A instance updated"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	a.NoError(err)
	a.Equal("Project A instance updated", updatedA.Msg.Title)

	_, err = ctl.instanceServiceClient.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{
		Instance:   &v1pb.Instance{Name: instanceB.Name, Title: "must not update"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
	}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instanceA.Name}))
	a.NoError(err)
	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instanceB.Name}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceA.Name}))
	a.NoError(err)
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceB.Name}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	undeletedA, err := ctl.instanceServiceClient.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceA.Name}))
	a.NoError(err)
	a.Equal(instanceA.Name, undeletedA.Msg.Name)
	_, err = ctl.instanceServiceClient.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceB.Name}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	// A role containing only bb.instances.get grants that single operation on
	// its project, rather than inheriting project-owner permissions.
	ctl.authInterceptor.token = ownerToken
	roleID := generateRandomString("instance-reader")
	_, err = ctl.roleServiceClient.CreateRole(ctx, connect.NewRequest(&v1pb.CreateRoleRequest{
		RoleId: roleID,
		Role: &v1pb.Role{
			Title:       "Project instance reader",
			Permissions: []string{"bb.instances.get"},
		},
	}))
	a.NoError(err)
	readerToken := bot35CreateProjectUser(ctx, t, ctl, ctl.project.Name, fmt.Sprintf("roles/%s", roleID), "project-instance-reader")
	ctl.authInterceptor.token = readerToken
	_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceA.Name}))
	a.NoError(err)
	_, err = ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{Parent: new(ctl.project.Name)}))
	a.Error(err)
	a.Equal(connect.CodePermissionDenied, connect.CodeOf(err))

	// Workspace-wide database roles retain authority over project instances.
	ctl.authInterceptor.token = ownerToken
	for _, role := range []string{"roles/workspaceAdmin", "roles/workspaceDBA"} {
		ctl.authInterceptor.token = ownerToken
		workspaceToken := bot35CreateWorkspaceUser(ctx, t, ctl, role, "project-instance-workspace")
		ctl.authInterceptor.token = workspaceToken
		_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instanceA.Name}))
		a.NoError(err, "%s gets project instance", role)
		workspaceList, err := ctl.instanceServiceClient.ListInstances(ctx, connect.NewRequest(&v1pb.ListInstancesRequest{Parent: new(ctl.project.Name)}))
		a.NoError(err, "%s lists project instances", role)
		a.Len(workspaceList.Msg.Instances, 1)
	}
}

func TestProjectInstanceCollisionIsolation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	projectBResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("instance-collision-project"),
		Project:   &v1pb.Project{Title: "Instance collision project", AllowSelfApproval: true},
	}))
	a.NoError(err)
	projectB := projectBResp.Msg

	instanceAID := generateRandomString("project-instance-a")
	instanceA := bot35CreateProjectInstance(ctx, t, ctl, ctl.project.Name, instanceAID, "Collision instance A", pgContainer.adminDataSource(), false)

	// Instance IDs are workspace-global, even though their resource names are
	// project nested. Reusing a local ID in another project is rejected.
	_, err = ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:     new(projectB.Name),
		InstanceId: instanceAID,
		Instance: &v1pb.Instance{
			Title:       "Duplicate instance ID",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			DataSources: []*v1pb.DataSource{pgContainer.adminDataSource()},
		},
	}))
	a.Error(err)
	a.Equal(connect.CodeAlreadyExists, connect.CodeOf(err))

	instanceBID := generateRandomString("project-instance-b")
	instanceB := bot35CreateProjectInstance(ctx, t, ctl, projectB.Name, instanceBID, "Collision instance B", pgContainer.adminDataSource(), false)

	// Both instances discover PostgreSQL's local "postgres" database. That
	// descendant-name collision must remain bound to the instance's project
	// prefix and must not permit a project A request to mutate B.
	const databaseID = "postgres"
	databaseBName := instanceB.Name + "/databases/" + databaseID
	databaseB, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseBName}))
	a.NoError(err)

	// The top-level form cannot resolve a project-owned instance either.
	_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{
		Name: "instances/" + instanceAID,
	}))
	a.Error(err)

	// Pointing project A at B's instance ID is rejected before it can select
	// B's identically named descendant.
	wrongPrefixInstanceName := ctl.project.Name + "/instances/" + instanceBID
	_, err = ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: wrongPrefixInstanceName}))
	a.Error(err)
	wrongPrefixDatabaseName := wrongPrefixInstanceName + "/databases/" + databaseID
	_, err = ctl.databaseServiceClient.UpdateDatabase(ctx, connect.NewRequest(&v1pb.UpdateDatabaseRequest{
		Database:   &v1pb.Database{Name: wrongPrefixDatabaseName, Labels: map[string]string{"bot35": "wrong-project"}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
	}))
	a.Error(err)

	databaseBAfter, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{Name: databaseBName}))
	a.NoError(err)
	a.Equal(databaseB.Msg.Labels, databaseBAfter.Msg.Labels, "project B database snapshot must remain untouched")

	// Keep a positive assertion for the correctly scoped sibling: otherwise a
	// broken setup that never created B's instance/database could still pass.
	a.Equal(projectB.Name+"/instances/", instanceB.Name[:len(projectB.Name+"/instances/")])
	a.Equal(ctl.project.Name+"/instances/", instanceA.Name[:len(ctl.project.Name+"/instances/")])
}

func bot35CreateProjectInstance(
	ctx context.Context,
	t *testing.T,
	ctl *controller,
	projectName, instanceID, title string,
	dataSource *v1pb.DataSource,
	skipDatabaseSync bool,
) *v1pb.Instance {
	t.Helper()
	a := require.New(t)
	instance := &v1pb.Instance{
		Title:       title,
		Engine:      v1pb.Engine_POSTGRES,
		Environment: new("environments/prod"),
		DataSources: []*v1pb.DataSource{dataSource},
	}
	if skipDatabaseSync {
		instance.SyncDatabases = &v1pb.SyncDatabases{}
	}
	resp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		Parent:     new(projectName),
		InstanceId: instanceID,
		Instance:   instance,
	}))
	a.NoError(err)
	return resp.Msg
}

func bot35CreateProjectUser(ctx context.Context, t *testing.T, ctl *controller, projectName, role, prefix string) string {
	t.Helper()
	a := require.New(t)
	email := fmt.Sprintf("%s-%s@example.com", prefix, generateRandomString("user"))
	const password = "1024bytebase"
	user, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Email: email, Password: password, Title: prefix},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, user.Msg.Workspace, fmt.Sprintf("user:%s", email), "roles/workspaceMember")
	a.NoError(err)

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{Resource: projectName}))
	a.NoError(err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{Role: role, Members: []string{fmt.Sprintf("user:%s", email)}})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{Resource: projectName, Policy: policy}))
	a.NoError(err)

	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: password}))
	a.NoError(err)
	return login.Msg.Token
}

func bot35CreateWorkspaceUser(ctx context.Context, t *testing.T, ctl *controller, role, prefix string) string {
	t.Helper()
	a := require.New(t)
	email := fmt.Sprintf("%s-%s@example.com", prefix, generateRandomString("user"))
	const password = "1024bytebase"
	user, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Email: email, Password: password, Title: prefix},
	}))
	a.NoError(err)
	_, err = ctl.addMemberToWorkspaceIAM(ctx, user.Msg.Workspace, fmt.Sprintf("user:%s", email), role)
	a.NoError(err)
	login, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{Email: email, Password: password}))
	a.NoError(err)
	return login.Msg.Token
}
