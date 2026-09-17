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
		a.NoError(s.UpsertDBSchema(ctx, instanceID, "db", &metadatapb.DatabaseSchemaMetadata{Name: "before"}, nil, time.Now()))
		catalogs[instanceID] = &storepb.DatabaseConfig{Schemas: []*storepb.SchemaCatalog{{
			Name: "public",
			Tables: []*storepb.TableCatalog{{
				Name:    "employee",
				Columns: []*storepb.ColumnCatalog{{Name: "secret", SemanticType: instanceID}},
			}},
		}}}
		a.NoError(s.UpdateDBSchema(ctx, instanceID, "db", &store.UpdateDBSchemaMessage{Config: catalogs[instanceID]}))
	}

	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "after"}, nil, time.Now()))

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
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "read-later"}, nil, later, datashare(true)))
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "read-earlier"}, nil, earlier, datashare(false)))

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.NotNil(schema)
	a.Equal("read-later", schema.GetProto().GetName())

	database, err := s.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: new("instance-a"), DatabaseName: new("db")})
	a.NoError(err)
	a.NotNil(database)
	a.True(later.Equal(database.Metadata.GetLastSyncTime().AsTime()))
	a.True(database.Metadata.GetDatashare())

	tie := time.Now()
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "tie-first"}, nil, tie))
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "tie-second"}, nil, tie))

	schema, err = s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.Equal("tie-first", schema.GetProto().GetName())
}

func TestUpsertDBSchemaIgnoresASyncTimeTheDatabaseHasNotReached(t *testing.T) {
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

	// A replica clock that ran ahead wrote this before syncs were ordered.
	_, err = s.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:   "instance-a",
		DatabaseName: "db",
		MetadataUpdates: []func(*storepb.DatabaseMetadata){
			func(md *storepb.DatabaseMetadata) { md.LastSyncTime = timestamppb.New(time.Now().Add(time.Hour)) },
		},
	})
	a.NoError(err)

	syncedAt, err := s.Now(ctx)
	a.NoError(err)
	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "synced"}, nil, syncedAt))

	schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: "instance-a", DatabaseName: "db"})
	a.NoError(err)
	a.NotNil(schema)
	a.Equal("synced", schema.GetProto().GetName())
}
