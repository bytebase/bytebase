package mongodb

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestDump pins what the adapter hands the driver: the metadata Bytebase synced,
// rendered as a mongosh script. It holds the fields SyncDBSchema records for the
// script to read (a collection's create options, an index's options and a view's
// definition); recreating a database from the script is the driver's, covered by
// omni's mongo/driver tests.
func TestDump(t *testing.T) {
	// Open does not reach the server, and a dump reads no connection, so this
	// test needs none.
	opened, err := (&Driver{}).Open(context.Background(), storepb.Engine_MONGODB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Host: "localhost", Port: "27017"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(context.Background()) })

	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{
				{
					Name:          "users",
					CreateOptions: `{"capped":true,"size":{"$numberInt":"4096"}}`,
					Indexes: []*metadatapb.IndexMetadata{
						{Name: "_id_", Expressions: []string{`{"_id":1}`}, Definition: `{"v":{"$numberInt":"2"}}`},
						{Name: "name_1", Unique: true, Expressions: []string{`{"name":1}`}, Definition: `{"v":{"$numberInt":"2"}}`},
						{Name: "created_at_1", Expressions: []string{`{"created_at":1}`}, Definition: `{"expireAfterSeconds":{"$numberInt":"3600"},"v":{"$numberInt":"2"}}`},
					},
				},
			},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v_adults",
				Definition: `{"viewOn":"users","pipeline":[{"$match":{"age":{"$gte":{"$numberInt":"18"}}}}]}`,
			}},
		}},
	}

	var out strings.Builder
	require.NoError(t, opened.Dump(context.Background(), &out, metadata))
	script := out.String()
	require.Contains(t, script, `// MongoDB schema dump of database "app"`)
	require.Contains(t, script, `db.runCommand({"create": "users", "capped": true, "size": 4096});`)
	require.Contains(t, script, `"unique": true, "v": 2`)
	require.Contains(t, script, `"expireAfterSeconds": 3600`)
	require.Contains(t, script, `db.runCommand({"create": "v_adults", "viewOn": "users", "pipeline": [{"$match": {"age": {"$gte": 18}}}]});`)
}
