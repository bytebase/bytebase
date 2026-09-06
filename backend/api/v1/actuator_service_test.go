package v1

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	"github.com/bytebase/bytebase/backend/component/sample/saas"
	"github.com/bytebase/bytebase/backend/component/sample/selfhost"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestAuthenticationInfoAndActuatorBoundary(t *testing.T) {
	ctx := context.Background()
	_, stores, pgURL := testcontainer.NewMetadataDB(t)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	profile := &config.Profile{
		Version:     "9.9.9",
		GitCommit:   "sensitive-commit",
		ExternalURL: "https://bytebase.example.com",
	}
	sampleManager, err := saas.NewManager(
		stores,
		pgURL,
		nil,
		sample.ManagerOptions{ReplicaID: "replica-a"},
	)
	require.NoError(t, err)
	actuatorService := NewActuatorService(stores, profile, nil, licenseService, sampleManager)
	authService := NewAuthService(stores, "test-secret", licenseService, profile, nil)

	publicResponse, err := authService.GetAuthenticationInfo(ctx, connect.NewRequest(&v1pb.GetAuthenticationInfoRequest{}))
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
	saasPublicResponse, err := authService.GetAuthenticationInfo(ctx, connect.NewRequest(&v1pb.GetAuthenticationInfoRequest{}))
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

	publicResponse, err = authService.GetAuthenticationInfo(ctx, connect.NewRequest(&v1pb.GetAuthenticationInfoRequest{}))
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
	actuatorService.sampleManager = nil
	nonSaaSResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.False(t, nonSaaSResponse.Msg.Sample.Available)

	selfHostManager := selfhost.NewManager(stores, profile, nil, sample.ManagerOptions{ReplicaID: "replica-a"})
	actuatorService.sampleManager = selfHostManager
	profile.PgURL = pgURL
	externalMetadataResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.False(t, externalMetadataResponse.Msg.Sample.Available)
	profile.PgURL = ""
	embeddedMetadataResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.True(t, embeddedMetadataResponse.Msg.Sample.Available)

	actuatorService.sampleManager = &sampleManagerStub{listInstancesErr: errors.New("failed to decode sample setup")}
	degradedSampleResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.True(t, degradedSampleResponse.Msg.Sample.Available)
	require.Empty(t, degradedSampleResponse.Msg.Sample.Instances)

	profile.SaaS = true
	actuatorService.sampleManager = sampleManager

	availableResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.True(t, availableResponse.Msg.Sample.Available)

	defaultProjectID, err := stores.GetDefaultProjectID(ctx, workspaceID)
	require.NoError(t, err)
	projectID := defaultProjectID
	payload, err := protojson.Marshal(&storepb.SaaSSampleInstanceSetupPayload{
		ProjectId:    projectID,
		InstanceId:   "sample-instance",
		Title:        "Sample Project Instance",
		DatabaseName: "sample-database",
		RoleName:     "sample-role",
	})
	require.NoError(t, err)
	reservation, created, err := stores.ReserveSampleInstanceSetup(ctx, &store.SampleInstanceSetupMessage{
		WorkspaceID: workspaceID,
		ReplicaID:   "replica-a",
		Payload:     payload,
	})
	require.NoError(t, err)
	require.True(t, created)
	expiresAt := time.Now().Add(7 * 24 * time.Hour).Truncate(time.Microsecond)
	activatedAt := expiresAt.Add(-7 * 24 * time.Hour)
	activated, err := stores.ActivateSampleInstanceSetup(ctx, workspaceID, reservation.ReplicaID, []string{projectID}, activatedAt, &expiresAt)
	require.NoError(t, err)
	require.True(t, activated)
	privateResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.Equal(t, "9.9.9", privateResponse.Msg.Version)
	require.Equal(t, "sensitive-commit", privateResponse.Msg.GitCommit)
	require.Equal(t, common.FormatWorkspace(workspaceID), privateResponse.Msg.Workspace)
	require.NotEmpty(t, privateResponse.Msg.DefaultProject)
	require.True(t, privateResponse.Msg.Sample.Available)
	require.Len(t, privateResponse.Msg.Sample.Instances, 1)
	require.Equal(t, common.FormatProjectInstance(projectID, "sample-instance"), privateResponse.Msg.Sample.Instances[0].Instance)
	require.True(t, expiresAt.Equal(privateResponse.Msg.Sample.Instances[0].ExpireTime.AsTime()))

	cleanupNow := expiresAt.Add(time.Second)
	cleanupResult, err := stores.WithLockedSampleInstanceSetupForCleanup(
		ctx,
		cleanupNow,
		cleanupNow.Add(-time.Hour),
		"",
		func(ctx context.Context, tx *store.SampleInstanceSetupTx, _ *store.SampleInstanceSetupMessage) error {
			return tx.MarkDeleted(ctx, cleanupNow)
		},
	)
	require.NoError(t, err)
	require.True(t, cleanupResult.Found)
	deletedResponse, err := actuatorService.GetActuatorInfo(authenticatedCtx, connect.NewRequest(&v1pb.GetActuatorInfoRequest{}))
	require.NoError(t, err)
	require.True(t, deletedResponse.Msg.Sample.Available)
	require.Len(t, deletedResponse.Msg.Sample.Instances, 1)
	require.Equal(t, common.FormatProjectInstance(projectID, "sample-instance"), deletedResponse.Msg.Sample.Instances[0].Instance)
	require.Nil(t, deletedResponse.Msg.Sample.Instances[0].ExpireTime)
}
