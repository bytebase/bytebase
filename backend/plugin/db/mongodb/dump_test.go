package mongodb

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestCutoverDumpRoundTrip proves the script Dump writes restores a dropped
// database: create a corpus, sync it, dump it, drop it, replay the dump, then
// sync and dump again. The old driver's Dump wrote nothing, so the whole
// behavior is one the cutover adds; what this pins is that the script the
// driver writes is the script the driver can read back.
func TestCutoverDumpRoundTrip(t *testing.T) {
	ctx := context.Background()
	container, database := testcontainer.NewMongoDatabase(t)
	opened, err := (&Driver{}).Open(ctx, storepb.Engine_MONGODB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     container.GetHost(),
			Port:     container.GetPort(),
			Username: container.GetUsername(),
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: database},
		Password:          container.GetPassword(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(ctx) })
	require.NoError(t, opened.Ping(ctx))

	_, err = opened.Execute(ctx, parityCorpus, db.ExecuteOptions{})
	require.NoError(t, err)

	before := syncDB(ctx, t, opened)
	dump := dumpDB(ctx, t, opened, before)
	require.Contains(t, dump, `db.runCommand({"create": "v_adults"`)
	require.Contains(t, dump, `db.runCommand({"createIndexes": "users"`)

	// Dropping the database is the destructive half of the round trip: the
	// driver must notice it is gone before the replay recreates it.
	_, err = opened.Execute(ctx, `db.dropDatabase()`, db.ExecuteOptions{})
	require.NoError(t, err)
	_, err = opened.SyncDBSchema(ctx)
	require.ErrorIs(t, err, omnidriver.ErrNotFound)
	require.ErrorContains(t, err, "does not exist")

	_, err = opened.Execute(ctx, dump, db.ExecuteOptions{})
	require.NoError(t, err)

	after := syncDB(ctx, t, opened)
	require.Equal(t, dump, dumpDB(ctx, t, opened, after), "replaying the dump and dumping again must write the same script")

	// The restored database holds the same collections, options, indexes and
	// views. Its documents are not restored by a schema dump, so the counts and
	// sizes differ.
	if diff := cmp.Diff(before, after, protocmp.Transform(),
		protocmp.IgnoreFields(&metadatapb.TableMetadata{}, "row_count", "data_size", "index_size"),
	); diff != "" {
		t.Errorf("restored database differs from the source (-source +restored):\n%s", diff)
	}
}

func syncDB(ctx context.Context, t *testing.T, opened db.Driver) *metadatapb.DatabaseSchemaMetadata {
	t.Helper()
	schema, err := opened.SyncDBSchema(ctx)
	require.NoError(t, err)
	return schema
}

func dumpDB(ctx context.Context, t *testing.T, opened db.Driver, schema *metadatapb.DatabaseSchemaMetadata) string {
	t.Helper()
	var out strings.Builder
	require.NoError(t, opened.Dump(ctx, &out, schema))
	require.NotEmpty(t, out.String())
	return out.String()
}
