package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestServiceAccountCreatedAt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)

	email := "deploy@service.bytebase.com"
	created, err := stores.CreateServiceAccount(ctx, &store.CreateServiceAccountMessage{
		Email: email, Name: "deploy", Workspace: "default", ServiceKeyHash: "hash",
	})
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), created.CreatedAt, time.Minute)

	_, err = db.ExecContext(ctx, `UPDATE service_account SET created_at = '2024-01-08 10:00:00+00' WHERE email = $1`, email)
	require.NoError(t, err)
	want := time.Date(2024, 1, 8, 10, 0, 0, 0, time.UTC)

	byEmail, err := stores.GetServiceAccountByEmail(ctx, email)
	require.NoError(t, err)
	require.True(t, want.Equal(byEmail.CreatedAt), byEmail.CreatedAt)

	listed, err := stores.ListServiceAccounts(ctx, &store.FindServiceAccountMessage{Workspace: "default"})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.True(t, want.Equal(listed[0].CreatedAt), listed[0].CreatedAt)

	renamed := "deploy bot"
	updated, err := stores.UpdateServiceAccount(ctx, listed[0], &store.UpdateServiceAccountMessage{Name: &renamed})
	require.NoError(t, err)
	require.True(t, want.Equal(updated.CreatedAt), updated.CreatedAt)
}

func TestWorkloadIdentityCreatedAt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)

	email := "deploy-ci@workload.bytebase.com"
	created, err := stores.CreateWorkloadIdentity(ctx, &store.CreateWorkloadIdentityMessage{
		Email: email, Name: "deploy-ci", Workspace: "default",
		Config: &storepb.WorkloadIdentityConfig{
			ProviderType:     storepb.WorkloadIdentityConfig_GITHUB,
			IssuerUrl:        "https://token.actions.githubusercontent.com",
			AllowedAudiences: []string{"bytebase"},
			SubjectPattern:   "repo:example-org/billing-api:*",
		},
	})
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), created.CreatedAt, time.Minute)

	_, err = db.ExecContext(ctx, `UPDATE workload_identity SET created_at = '2026-06-02 10:00:00+00' WHERE email = $1`, email)
	require.NoError(t, err)
	want := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)

	byEmail, err := stores.GetWorkloadIdentityByEmail(ctx, email)
	require.NoError(t, err)
	require.True(t, want.Equal(byEmail.CreatedAt), byEmail.CreatedAt)

	listed, err := stores.ListWorkloadIdentities(ctx, &store.FindWorkloadIdentityMessage{Workspace: "default"})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.True(t, want.Equal(listed[0].CreatedAt), listed[0].CreatedAt)

	renamed := "deploy ci"
	updated, err := stores.UpdateWorkloadIdentity(ctx, listed[0], &store.UpdateWorkloadIdentityMessage{Name: &renamed})
	require.NoError(t, err)
	require.True(t, want.Equal(updated.CreatedAt), updated.CreatedAt)
}
