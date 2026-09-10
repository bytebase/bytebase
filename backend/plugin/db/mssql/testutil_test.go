package mssql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func openMSSQL(ctx context.Context, t *testing.T, host, port, database string) db.Driver {
	t.Helper()
	driverInstance := &Driver{}
	cfg := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "sa",
			Host:     host,
			Port:     port,
			Database: database,
		},
		Password: "Test123!",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
		},
	}
	driver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, cfg)
	require.NoError(t, err)
	return driver
}

// newSyncTestDatabase gives the test a database of its own on the shared
// container so the cases can run in parallel, and returns a driver open on it.
func newSyncTestDatabase(ctx context.Context, t *testing.T, container *testcontainer.Container) *Driver {
	t.Helper()

	name := fmt.Sprintf("sync_%s", strings.ReplaceAll(uuid.New().String(), "-", "_"))
	_, err := container.GetDB().Exec(fmt.Sprintf("CREATE DATABASE [%s]", name))
	require.NoError(t, err)

	opened := openMSSQL(ctx, t, container.GetHost(), container.GetPort(), name)
	t.Cleanup(func() { opened.Close(ctx) })

	driver, ok := opened.(*Driver)
	require.True(t, ok, "expected *Driver")
	return driver
}

// executeBatches runs a setup script one GO-separated batch at a time, which is
// what SQL Server requires for CREATE SCHEMA and CREATE INDEX.
func executeBatches(ctx context.Context, t *testing.T, driver *Driver, script string) {
	t.Helper()

	for _, batch := range splitSQLStatements(script) {
		if strings.TrimSpace(batch) == "" {
			continue
		}
		_, err := driver.Execute(ctx, batch, db.ExecuteOptions{})
		require.NoError(t, err, "setup batch: %s", batch)
	}
}

func requireTable(t *testing.T, metadata *storepb.DatabaseSchemaMetadata, schemaName, tableName string) *storepb.TableMetadata {
	t.Helper()

	for _, schema := range metadata.Schemas {
		if schema.Name != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			if table.Name == tableName {
				return table
			}
		}
	}
	require.FailNowf(t, "table not synced", "%s.%s", schemaName, tableName)
	return nil
}

func splitSQLStatements(script string) []string {
	var statements []string
	var current strings.Builder

	for _, line := range strings.Split(script, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "GO") {
			if current.Len() > 0 {
				statements = append(statements, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}
	return statements
}
