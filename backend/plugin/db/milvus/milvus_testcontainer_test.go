package milvus

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestMilvusIntegration_QueryAndSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Milvus integration test in short mode")
	}

	ctx := context.Background()
	host, port := integrationAddress()
	driver := openIntegrationDriver(t, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host: host,
			Port: port,
		},
	})
	requireMilvusReady(t, ctx, driver, host, port)

	collectionName := fmt.Sprintf("bb_it_items_%d", time.Now().UnixNano())
	seedCollection(t, ctx, driver, collectionName)
	t.Cleanup(func() {
		_, _ = driver.callMilvus(ctx, "/v2/vectordb/collections/drop", map[string]any{
			"collectionName": collectionName,
		})
	})

	results, err := driver.QueryConn(ctx, nil, fmt.Sprintf("show collections; select id from %s limit 2;", collectionName), db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, []string{"collection"}, results[0].ColumnNames)
	require.NotEmpty(t, results[0].Rows)
	require.Equal(t, []string{"id"}, results[1].ColumnNames)
	require.Len(t, results[1].Rows, 2)

	describeResult, err := driver.QueryConn(ctx, nil, fmt.Sprintf("describe collection %s;", collectionName), db.QueryContext{})
	require.NoError(t, err)
	require.Len(t, describeResult, 1)
	require.Equal(t, []string{"field", "value"}, describeResult[0].ColumnNames)

	instanceMetadata, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, instanceMetadata.Databases)
	require.NotEmpty(t, instanceMetadata.Databases[0].Schemas)
	require.NotEmpty(t, instanceMetadata.Databases[0].Schemas[0].Tables)

	dbSchema, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, dbSchema.Schemas)
	found := false
	for _, table := range dbSchema.Schemas[0].Tables {
		if table.Name == collectionName {
			found = true
			require.NotEmpty(t, table.Columns)
		}
	}
	require.True(t, found, "expected collection %q in synced schema", collectionName)
}

func TestMilvusIntegration_ACLUserCanPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Milvus integration test in short mode")
	}

	ctx := context.Background()
	host, port := integrationAddress()
	admin := openIntegrationDriver(t, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host: host,
			Port: port,
		},
	})
	requireMilvusReady(t, ctx, admin, host, port)

	userName := fmt.Sprintf("bb_it_user_%d", time.Now().UnixNano())
	password := "BbItPass123!"
	roleName := fmt.Sprintf("bb_it_role_%d", time.Now().UnixNano())

	if err := ensureACLArtifacts(ctx, admin, userName, password, roleName); err != nil {
		if strings.Contains(err.Error(), "404") {
			t.Skipf("acl endpoints are not exposed by current Milvus runtime: %v", err)
		}
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = admin.callMilvus(ctx, "/v2/vectordb/users/revoke_role", map[string]any{
			"userName": userName,
			"roleName": roleName,
		})
		_, _ = admin.callMilvus(ctx, "/v2/vectordb/roles/drop", map[string]any{
			"roleName": roleName,
		})
		_, _ = admin.callMilvus(ctx, "/v2/vectordb/users/drop", map[string]any{
			"userName": userName,
		})
	})

	userDriver := openIntegrationDriver(t, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     host,
			Port:     port,
			Username: userName,
		},
		Password: password,
	})
	require.NoError(t, userDriver.Ping(ctx))
}

func integrationAddress() (string, string) {
	host := strings.TrimSpace(os.Getenv("BYTEBASE_TEST_MILVUS_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("BYTEBASE_TEST_MILVUS_PORT"))
	if port == "" {
		port = "19530"
	}
	return host, port
}

func openIntegrationDriver(t *testing.T, config db.ConnectionConfig) *Driver {
	t.Helper()
	opened, err := newDriver().Open(context.Background(), storepb.Engine_MILVUS, config)
	require.NoError(t, err)
	driver, ok := opened.(*Driver)
	require.True(t, ok)
	return driver
}

func requireMilvusReady(t *testing.T, ctx context.Context, d *Driver, host, port string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = d.Ping(ctx)
		if lastErr == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Skipf("milvus is not reachable at %s:%s: %v", host, port, lastErr)
}

func seedCollection(t *testing.T, ctx context.Context, d *Driver, collectionName string) {
	t.Helper()
	_, err := d.callMilvus(ctx, "/v2/vectordb/collections/create", map[string]any{
		"collectionName": collectionName,
		"dimension":      4,
	})
	require.NoError(t, err)

	_, err = d.callMilvus(ctx, "/v2/vectordb/entities/insert", map[string]any{
		"collectionName": collectionName,
		"data": []map[string]any{
			{"id": 1, "vector": []float64{0.1, 0.2, 0.3, 0.4}},
			{"id": 2, "vector": []float64{0.5, 0.6, 0.7, 0.8}},
		},
	})
	require.NoError(t, err)

	_, err = d.callMilvus(ctx, "/v2/vectordb/collections/load", map[string]any{
		"collectionName": collectionName,
	})
	require.NoError(t, err)
}

func ensureACLArtifacts(ctx context.Context, d *Driver, userName, password, roleName string) error {
	if _, err := d.callMilvus(ctx, "/v2/vectordb/users/create", map[string]any{
		"userName": userName,
		"password": password,
	}); err != nil {
		return err
	}
	if _, err := d.callMilvus(ctx, "/v2/vectordb/roles/create", map[string]any{
		"roleName": roleName,
	}); err != nil {
		return err
	}
	if _, err := d.callMilvus(ctx, "/v2/vectordb/users/grant_role", map[string]any{
		"userName": userName,
		"roleName": roleName,
	}); err != nil {
		return err
	}
	return nil
}
