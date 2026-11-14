package sample

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestMain(m *testing.M) { testcontainer.Main(m) }

func TestCreateMetadataEnforcesInstanceLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
 INSERT INTO workspace(resource_id) VALUES ('quota-workspace');
 INSERT INTO project(resource_id, workspace, name) VALUES ('sample-project', 'quota-workspace', 'Sample');
 `)
	require.NoError(t, err)
	projectID := "sample-project"
	one := 1
	registration := Registration{
		WorkspaceID:       "quota-workspace",
		InstanceProjectID: &projectID,
		InstanceID:        "first-sample",
		AdminDataSource:   &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN},
		Limits:            &store.InstanceLimits{Count: &one},
	}
	_, err = CreateMetadata(ctx, stores, registration)
	require.NoError(t, err)
	registration.InstanceID = "second-sample"
	_, err = CreateMetadata(ctx, stores, registration)
	var quota *store.InstanceLimitError
	require.ErrorAs(t, err, &quota)
	require.False(t, quota.Activated)
	require.Equal(t, 1, quota.Limit)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM instance WHERE workspace = $1`, registration.WorkspaceID).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM instance WHERE resource_id = $1`, registration.InstanceID).Scan(&count))
	require.Zero(t, count)
}
