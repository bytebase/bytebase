package cassandra

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestIntegration_Cassandra(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "cassandra:4.1",
		ExposedPorts: []string{"9042/tcp"},
		WaitingFor:   wait.ForLog("Startup complete").WithStartupTimeout(180 * time.Second),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	host, err := c.Host(ctx)
	require.NoError(t, err)
	port, err := c.MappedPort(ctx, "9042/tcp")
	require.NoError(t, err)

	// Cassandra default credentials
	username := "cassandra"
	password := "cassandra"

	config := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     host,
			Port:     port.Port(),
			Username: username,
		},
		Password: password,
		ConnectionContext: db.ConnectionContext{
			DatabaseName: "system", // Connect to system keyspace initially
		},
	}

	driver := newDriver()
	connectedDriver, err := driver.Open(ctx, storepb.Engine_CASSANDRA, config)
	require.NoError(t, err)
	defer connectedDriver.Close(ctx)

	t.Run("Ping", func(t *testing.T) {
		err := connectedDriver.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("SyncDBSchema", func(t *testing.T) {
		// sync.go uses the driver, let's cast it to *Driver to call SyncDBSchema
		// because SyncDBSchema is a method on *Driver, not the db.Driver interface (which it implements partially?)
		// Wait, SyncDBSchema IS NOT part of db.Driver interface?
		// Let's check backend/plugin/db/driver.go if possible. 
		// Actually, in sync.go it is defined as func (d *Driver) SyncDBSchema.
		// If it's part of the interface, we can cast or call it.
		// Let's check if *Driver implements it.
		// Looking at imports in cassandra.go: "github.com/bytebase/bytebase/backend/plugin/db"
		
		d, ok := connectedDriver.(*Driver)
		require.True(t, ok)

		meta, err := d.SyncDBSchema(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, meta.Schemas)
		// system keyspace should have tables
	})
	
	t.Run("Execute and Query", func(t *testing.T) {
		// Create a test keyspace
		keyspaceName := "bytebase_test"
		_, err := connectedDriver.Execute(ctx, fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };", keyspaceName), db.ExecuteOptions{})
		require.NoError(t, err)

		// Create table
		_, err = connectedDriver.Execute(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.users (id UUID PRIMARY KEY, name text, age int);", keyspaceName), db.ExecuteOptions{})
		require.NoError(t, err)
		
		// Insert data
		_, err = connectedDriver.Execute(ctx, fmt.Sprintf("INSERT INTO %s.users (id, name, age) VALUES (uuid(), 'Alice', 30);", keyspaceName), db.ExecuteOptions{})
		require.NoError(t, err)

		// Query data
		// We need to switch context or just query with fully qualified name?
		// The driver doesn't seem to support switching database easily without Re-Open, 
		// but QueryConn takes a query.
		results, err := connectedDriver.QueryConn(ctx, nil, fmt.Sprintf("SELECT * FROM %s.users;", keyspaceName), db.QueryContext{Limit: 100})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Len(t, results[0].Rows, 1)
		
		// Verify types
		// Expected: id (uuid), name (text), age (int)
		// Let's check column names and types
		t.Logf("Columns: %v", results[0].ColumnNames)
		t.Logf("Types: %v", results[0].ColumnTypeNames)
		
		assert.Contains(t, results[0].ColumnNames, "name")
		assert.Contains(t, results[0].ColumnTypeNames, "varchar") // text is mapped to varchar in typeToString
	})
}
