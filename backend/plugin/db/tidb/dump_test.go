package tidb

import (
	"bytes"
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestDumpRendersTheSyncedMetadata proves the adapter hands the schema it synced
// to the omni driver's Dumper and propagates dump errors.
func TestDumpRendersTheSyncedMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantMetadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "shop",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:    "orders",
				Comment: "orders",
				Indexes: []*metadatapb.IndexMetadata{{Name: "orders_note"}},
			}},
		}},
	}
	fake := &fakeSyncDriver{
		schema:  wantMetadata,
		dumpSQL: "CREATE TABLE `orders` (`id` bigint, `note` varchar(64), KEY `orders_note` (`note`));\n",
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, driver.Dump(ctx, &out, metadata))
	require.Equal(t, fake.dumpSQL, out.String())
	require.Empty(t, cmp.Diff(wantMetadata, fake.dumpedSchema, protocmp.Transform()))

	fake.dumpErr = errors.New("render failed")
	require.ErrorIs(t, driver.Dump(ctx, &out, metadata), fake.dumpErr)
}
