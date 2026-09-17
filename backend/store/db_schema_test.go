package store_test

import (
	"context"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

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
		a.NoError(s.UpsertDBSchema(ctx, instanceID, "db", &metadatapb.DatabaseSchemaMetadata{Name: "before"}, nil))
		catalogs[instanceID] = &storepb.DatabaseConfig{Schemas: []*storepb.SchemaCatalog{{
			Name: "public",
			Tables: []*storepb.TableCatalog{{
				Name:    "employee",
				Columns: []*storepb.ColumnCatalog{{Name: "secret", SemanticType: instanceID}},
			}},
		}}}
		a.NoError(s.UpdateDBSchema(ctx, instanceID, "db", &store.UpdateDBSchemaMessage{Config: catalogs[instanceID]}))
	}

	a.NoError(s.UpsertDBSchema(ctx, "instance-a", "db", &metadatapb.DatabaseSchemaMetadata{Name: "after"}, nil))

	for instanceID, wantMetadata := range map[string]string{"instance-a": "after", "instance-b": "before"} {
		schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: "default", InstanceID: instanceID, DatabaseName: "db"})
		a.NoError(err)
		a.NotNil(schema, instanceID)
		a.Equal(wantMetadata, schema.GetProto().GetName(), instanceID)
		a.True(proto.Equal(catalogs[instanceID], schema.GetConfig()), instanceID)
	}
}
