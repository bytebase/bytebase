package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestUpsertDBSchemaRefusesOlderRead covers the lost update that made a migrated
// database read back its pre-migration schema: the queued syncer reads the live
// database before a migration runs and writes the projection after the migration
// task's own sync has already written it.
func TestUpsertDBSchemaRefusesOlderRead(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default'), ('instance-b', 'default');
	`)
	ctx, s := fixture.ctx, fixture.store

	// The same database name in two instances: db_schema is keyed by (instance, db_name).
	for _, instanceID := range []string{"instance-a", "instance-b"} {
		_, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
			ProjectID:    "default",
			InstanceID:   instanceID,
			DatabaseName: "shared",
			Metadata:     &storepb.DatabaseMetadata{},
		})
		require.NoError(t, err)
	}

	readBefore := time.Now()
	readAfter := readBefore.Add(time.Minute)

	stored, err := s.UpsertDBSchema(ctx, "instance-a", "shared", schemaWithTable("before"), []byte("before"), readBefore)
	require.NoError(t, err)
	require.True(t, stored)

	stored, err = s.UpsertDBSchema(ctx, "instance-b", "shared", schemaWithTable("other"), []byte("other"), readAfter)
	require.NoError(t, err)
	require.True(t, stored)

	// The migration task's own sync.
	stored, err = s.UpsertDBSchema(ctx, "instance-a", "shared", schemaWithTable("after"), []byte("after"), readAfter)
	require.NoError(t, err)
	require.True(t, stored)
	requireStoredSchema(ctx, t, fixture, "instance-a", "shared", "after", readAfter)

	// The queued sync that read before the migration and writes after it.
	stored, err = s.UpsertDBSchema(ctx, "instance-a", "shared", schemaWithTable("before"), []byte("before"), readBefore)
	require.NoError(t, err)
	require.False(t, stored)
	requireStoredSchema(ctx, t, fixture, "instance-a", "shared", "after", readAfter)

	// A second read at the same instant is not older, so it still lands.
	stored, err = s.UpsertDBSchema(ctx, "instance-a", "shared", schemaWithTable("again"), []byte("again"), readAfter)
	require.NoError(t, err)
	require.True(t, stored)
	requireStoredSchema(ctx, t, fixture, "instance-a", "shared", "again", readAfter)

	// The refused write never reached the other instance's row.
	requireStoredSchema(ctx, t, fixture, "instance-b", "shared", "other", readAfter)
}

// TestUpsertDBSchemaStoresOverUnstampedRow covers the rows that predate the stamp:
// an upgraded deployment has synced_at NULL everywhere, so the first sync of each
// database wins however old its read is, and the guard orders every write after it.
// An unstamped row is why no backfill is needed - one write is enough to order the
// pair, whichever of the two lands on the NULL.
func TestUpsertDBSchemaStoresOverUnstampedRow(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	`)
	ctx, s := fixture.ctx, fixture.store

	_, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "default",
		InstanceID:   "instance-a",
		DatabaseName: "db",
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	_, err = fixture.db.ExecContext(ctx,
		"INSERT INTO db_schema (instance, db_name, metadata, raw_dump) VALUES ($1, $2, $3, $4)",
		"instance-a", "db", `{"name":"db"}`, "before")
	require.NoError(t, err)

	// The older of two syncs racing across the upgrade reaches the NULL row first.
	olderRead := time.Now().Add(-time.Hour)
	stored, err := s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("older"), []byte("older"), olderRead)
	require.NoError(t, err)
	require.True(t, stored)
	requireStoredSchema(ctx, t, fixture, "instance-a", "db", "older", olderRead)

	// Its stamp is enough to order the newer sync behind it, and the next older read
	// behind that, so the window the NULL leaves open is one write wide.
	newerRead := olderRead.Add(time.Minute)
	stored, err = s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("newer"), []byte("newer"), newerRead)
	require.NoError(t, err)
	require.True(t, stored)
	requireStoredSchema(ctx, t, fixture, "instance-a", "db", "newer", newerRead)

	stored, err = s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("older"), []byte("older"), olderRead)
	require.NoError(t, err)
	require.False(t, stored)
	requireStoredSchema(ctx, t, fixture, "instance-a", "db", "newer", newerRead)
}

// TestUpsertDBSchemaRefusesOlderReadBlockedOnRowLock is the same ordering with the
// writes actually overlapping: the older read waits on the row lock the newer write
// holds, so it evaluates the guard against a row that changed under it.
func TestUpsertDBSchemaRefusesOlderReadBlockedOnRowLock(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	`)
	ctx, s := fixture.ctx, fixture.store

	_, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "default",
		InstanceID:   "instance-a",
		DatabaseName: "db",
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)

	readBefore := time.Now()
	readAfter := readBefore.Add(time.Minute)

	stored, err := s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("before"), []byte("before"), readBefore)
	require.NoError(t, err)
	require.True(t, stored)

	// Stand in for the migration task's sync: hold the row lock UpsertDBSchema takes
	// first, write the post-migration schema, and only then let the older read in.
	tx, err := fixture.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, "SELECT 1 FROM db_schema WHERE instance = $1 AND db_name = $2 FOR UPDATE", "instance-a", "db")
	require.NoError(t, err)

	type result struct {
		stored bool
		err    error
	}
	olderRead := make(chan result, 1)
	go func() {
		stored, err := s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("before"), []byte("before"), readBefore)
		olderRead <- result{stored: stored, err: err}
	}()

	// The older read must still be waiting on the lock; nothing else orders it.
	select {
	case r := <-olderRead:
		t.Fatalf("older read completed before the row lock was released: stored=%v err=%v", r.stored, r.err)
	case <-time.After(200 * time.Millisecond):
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE db_schema SET metadata = $1, raw_dump = $2, synced_at = $3 WHERE instance = $4 AND db_name = $5",
		`{"name":"db","schemas":[{"name":"public","tables":[{"name":"after"}]}]}`, "after", readAfter, "instance-a", "db")
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	select {
	case r := <-olderRead:
		require.NoError(t, r.err)
		require.False(t, r.stored)
	case <-time.After(10 * time.Second):
		t.Fatal("older read never completed after the row lock was released")
	}
	requireStoredSchema(ctx, t, fixture, "instance-a", "db", "after", readAfter)
}

// TestUpsertDBSchemaKeepsCatalogConfig covers the other direction: a catalog edit
// made while a sync was dumping must survive that sync's write.
func TestUpsertDBSchemaKeepsCatalogConfig(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	`)
	ctx, s := fixture.ctx, fixture.store

	_, err := s.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:    "default",
		InstanceID:   "instance-a",
		DatabaseName: "db",
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)

	readAt := time.Now()
	stored, err := s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("before"), []byte("before"), readAt)
	require.NoError(t, err)
	require.True(t, stored)

	config := &storepb.DatabaseConfig{
		Schemas: []*storepb.SchemaCatalog{{Name: "public", Tables: []*storepb.TableCatalog{{Name: "t", Classification: "1-1"}}}},
	}
	require.NoError(t, s.UpdateDBSchema(ctx, "instance-a", "db", &store.UpdateDBSchemaMessage{Config: config}))

	stored, err = s.UpsertDBSchema(ctx, "instance-a", "db", schemaWithTable("after"), []byte("after"), readAt.Add(time.Minute))
	require.NoError(t, err)
	require.True(t, stored)

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	require.NoError(t, err)
	require.Equal(t, "1-1", schema.GetConfig().GetSchemas()[0].GetTables()[0].GetClassification())
}

func schemaWithTable(table string) *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "public", Tables: []*metadatapb.TableMetadata{{Name: table}}}},
	}
}

// requireStoredSchema asserts the projection a schema read serves, and the stamp
// that decides which read owns it, read past the schema cache.
func requireStoredSchema(ctx context.Context, t *testing.T, fixture *storePostgresFixture, instanceID, databaseName, table string, syncedAt time.Time) {
	t.Helper()
	var rawDump string
	var stored sql.NullTime
	require.NoError(t, fixture.db.QueryRowContext(ctx,
		"SELECT raw_dump, synced_at FROM db_schema WHERE instance = $1 AND db_name = $2",
		instanceID, databaseName).Scan(&rawDump, &stored))
	require.Equal(t, table, rawDump)
	require.True(t, stored.Valid)
	require.WithinDuration(t, syncedAt, stored.Time, time.Microsecond)

	schema, err := fixture.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		Workspace:    "default",
		InstanceID:   instanceID,
		DatabaseName: databaseName,
	})
	require.NoError(t, err)
	require.NotNil(t, schema.GetSchemaMetadata("public").GetTable(table))
	require.Equal(t, table, string(schema.GetRawDump()))
}
