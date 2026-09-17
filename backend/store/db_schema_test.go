package store_test

import (
	"context"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestUpsertDBSchemaKeepsCatalog(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace, project) VALUES
			('instance-a', 'default', 'project-a'),
			('instance-b', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES
			('instance-a', 'db', 'project-a'),
			('instance-b', 'db', 'project-a');
	`)
	a.NoError(err)

	catalogs := map[string]*storepb.DatabaseConfig{}
	for _, instanceID := range []string{"instance-a", "instance-b"} {
		a.NoError(s.UpsertDBSchema(ctx, instanceID, "db", &metadatapb.DatabaseSchemaMetadata{Name: "before"}, nil, time.Now(), 1))
		catalogs[instanceID] = &storepb.DatabaseConfig{Schemas: []*storepb.SchemaCatalog{{
			Name: "public",
			Tables: []*storepb.TableCatalog{{
				Name:    "employee",
				Columns: []*storepb.ColumnCatalog{{Name: "secret", SemanticType: instanceID}},
			}},
		}}}
		a.NoError(s.UpdateDBSchema(ctx, instanceID, "db", &store.UpdateDBSchemaMessage{Config: catalogs[instanceID]}))
	}

	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "after"}, nil, time.Now(), 2))

	for instanceID, wantMetadata := range map[string]string{"instance-a": "after", "instance-b": "before"} {
		schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: instanceID, DatabaseName: "db"})
		a.NoError(err)
		a.NotNil(schema, instanceID)
		a.Equal(wantMetadata, schema.GetProto().GetName(), instanceID)
		a.True(proto.Equal(catalogs[instanceID], schema.GetConfig()), instanceID)
	}
}

func TestUpsertDBSchemaKeepsTheLaterRead(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace, project) VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db', 'project-a');
	`)
	a.NoError(err)

	later := time.Now()
	earlier := later.Add(-time.Minute)
	datashare := func(v bool) func(*storepb.DatabaseMetadata) {
		return func(md *storepb.DatabaseMetadata) { md.Datashare = v }
	}
	// The schema fence is the token (2 then 1), not the wall clock these calls
	// also carry: "read-later" is named for when it happened, not for winning.
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "read-later"}, nil, later, 2, datashare(true)))
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "read-earlier"}, nil, earlier, 1, datashare(false)))

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.NotNil(schema)
	a.Equal("read-later", schema.GetProto().GetName())

	database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: new("instance-a"), DatabaseName: new("db")})
	a.NoError(err)
	a.NotNil(database)
	a.True(database.Metadata.GetDatashare())

	tie := time.Now()
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "tie-first"}, nil, tie, 3))
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "tie-second"}, nil, tie, 3))

	schema, err = s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.Equal("tie-first", schema.GetProto().GetName())
}

func TestUpsertDBSchemaOutlastsAClockThatRanBackward(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace, project) VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db', 'project-a');
	`)
	a.NoError(err)

	// A failover to a replica whose clock lags, or a step correction, can move
	// the metadata database's own wall clock backward: the second sync here
	// reads that clock afterward and gets an earlier syncedAt than the first
	// sync did, despite reading the target database after it. The token, from
	// a sequence rather than the clock, still orders them correctly.
	first, err := s.Now(ctx)
	a.NoError(err)
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "before-regression"}, nil, first, 1))

	clockRanBackward := first.Add(-time.Hour)
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "after-regression"}, nil, clockRanBackward, 2))

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.NotNil(schema)
	a.Equal("after-regression", schema.GetProto().GetName())
}

func TestUpsertDBSchemaLetsSuccessClearATiedFailure(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace, project) VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db', 'project-a');
	`)
	a.NoError(err)

	// A failed attempt lands first, at the same wall-clock time a successful
	// read is about to record: this guard is by syncedAt, not by the schema's
	// own token, so this can still win the race to record the database's own
	// metadata.
	tie := time.Now()
	_, err = s.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "instance-a",
		DatabaseName: "db",
		MetadataUpdates: []func(*storepb.DatabaseMetadata){
			func(md *storepb.DatabaseMetadata) {
				md.LastSyncTime = timestamppb.New(tie)
				md.SyncStatus = storepb.SyncStatus_SYNC_STATUS_FAILED
				md.SyncError = "prior failure"
			},
		},
	})
	a.NoError(err)

	// The success still claims db_schema, since nothing has stored there yet,
	// and its metadata update must not lose to the tied failure: this is the
	// guard syncer.go builds for a successful sync.
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "synced"}, nil, tie, 1,
		func(md *storepb.DatabaseMetadata) {
			if md.GetLastSyncTime().AsTime().After(tie) {
				return
			}
			md.LastSyncTime = timestamppb.New(tie)
			md.SyncStatus = storepb.SyncStatus_SYNC_STATUS_OK
			md.SyncError = ""
		},
	))

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.NotNil(schema)
	a.Equal("synced", schema.GetProto().GetName())

	database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: new("instance-a"), DatabaseName: new("db")})
	a.NoError(err)
	a.NotNil(database)
	a.Equal(storepb.SyncStatus_SYNC_STATUS_OK, database.Metadata.GetSyncStatus())
	a.Empty(database.Metadata.GetSyncError())
}

func TestUpsertDBSchemaReplacesARowFromBeforeTheColumn(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	db, s, _ := testcontainer.NewMetadataDB(t)
	// A row an upgraded installation carries: synced_at and sync_token are the
	// column defaults.
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace, project) VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'db', 'project-a');
		INSERT INTO db_schema (instance, db_name, metadata) VALUES ('instance-a', 'db', '{"name":"before-upgrade"}');
	`)
	a.NoError(err)

	syncedAt, err := s.Now(ctx)
	a.NoError(err)
	syncToken, err := s.NextSyncToken(ctx)
	a.NoError(err)
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "synced"}, nil, syncedAt, syncToken))

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.NotNil(schema)
	a.Equal("synced", schema.GetProto().GetName())
}
