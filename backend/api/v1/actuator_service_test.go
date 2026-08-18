package v1

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestAuthenticationInfoAndActuatorBoundary(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	require.NoError(t, migrator.MigrateSchema(ctx, container.GetDB()))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	profile := &config.Profile{
		Version:     "9.9.9",
		GitCommit:   "sensitive-commit",
		ExternalURL: "https://bytebase.example.com",
	}
	actuatorService := NewActuatorService(stores, profile, nil, licenseService, nil)
	authService := NewAuthService(stores, "test-secret", licenseService, profile, nil)

	publicResponse, err := authService.GetAuthenticationRestriction(ctx, connect.NewRequest(&v1pb.GetAuthenticationRestrictionRequest{}))
	require.NoError(t, err)

	var populated []string
	publicResponse.Msg.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		populated = append(populated, string(field.Name()))
		return true
	})
	slices.Sort(populated)
	require.Equal(t, []string{"restriction"}, populated)
	require.Empty(t, publicResponse.Msg.Workspace)

	profile.SaaS = true
	saasPublicResponse, err := authService.GetAuthenticationRestriction(ctx, connect.NewRequest(&v1pb.GetAuthenticationRestrictionRequest{}))
	require.NoError(t, err)
	require.True(t, saasPublicResponse.Msg.Restriction.DisallowSignup)
	require.True(t, saasPublicResponse.Msg.Restriction.DisallowPasswordSignin)
	profile.SaaS = false

	const workspaceID = "actuator-auth-boundary"
	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "Actuator auth boundary"},
	}, "admin@example.com")
	require.NoError(t, err)

	publicResponse, err = authService.GetAuthenticationRestriction(ctx, connect.NewRequest(&v1pb.GetAuthenticationRestrictionRequest{}))
	require.NoError(t, err)
	populated = nil
	publicResponse.Msg.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		populated = append(populated, string(field.Name()))
		return true
	})
	slices.Sort(populated)
	require.Equal(t, []string{"restriction", "workspace"}, populated)
	require.Equal(t, common.FormatWorkspace(workspaceID), publicResponse.Msg.Workspace)

	_, err = actuatorService.GetActuatorInfo(ctx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	user, err := stores.CreateUser(ctx, &store.UserMessage{
		Email:        "admin@example.com",
		Name:         "Admin",
		PasswordHash: "unused",
		Profile:      &storepb.UserProfile{},
	})
	require.NoError(t, err)
	authenticatedCtx := context.WithValue(ctx, common.UserContextKey, user)
	authenticatedCtx = context.WithValue(authenticatedCtx, common.WorkspaceIDContextKey, workspaceID)

	privateResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.Equal(t, "9.9.9", privateResponse.Msg.Version)
	require.Equal(t, "sensitive-commit", privateResponse.Msg.GitCommit)
	require.Equal(t, common.FormatWorkspace(workspaceID), privateResponse.Msg.Workspace)
	require.NotEmpty(t, privateResponse.Msg.DefaultProject)
}
