package trino

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type fakeSyncDumpDriver struct {
	instance     *omnidriver.Instance
	schema       *metadatapb.DatabaseSchemaMetadata
	syncDBErr    error
	dumpOut      string
	dumpErr      error
	dumpedSchema *metadatapb.DatabaseSchemaMetadata
}

func (*fakeSyncDumpDriver) Ping(context.Context) error { return nil }
func (*fakeSyncDumpDriver) Close() error               { return nil }

func (f *fakeSyncDumpDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeSyncDumpDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	if f.syncDBErr != nil {
		return nil, f.syncDBErr
	}
	return f.schema, nil
}

func (f *fakeSyncDumpDriver) Dump(_ context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	f.dumpedSchema = metadata
	if f.dumpErr != nil {
		return f.dumpErr
	}
	_, err := io.WriteString(out, f.dumpOut)
	return err
}

// TestSyncInstanceAndDatabaseDelegateToOmniDriver pins how the Trino adapter
// forwards SyncInstance, SyncDBSchema, and Dump to the omni driver.
func TestSyncInstanceAndDatabaseDelegateToOmniDriver(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantSchema := &metadatapb.DatabaseSchemaMetadata{
		Name: "memory",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "default",
			Tables: []*metadatapb.TableMetadata{{
				Name: "items",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "bigint"},
					{Name: "name", Type: "varchar(10)"},
				},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v_items",
				Definition: "SELECT id FROM memory.default.items",
			}},
		}},
	}
	fake := &fakeSyncDumpDriver{
		instance: &omnidriver.Instance{
			Version:   "482",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "memory"}, {Name: "system"}},
		},
		schema:  wantSchema,
		dumpOut: "CREATE SCHEMA IF NOT EXISTS \"default\";\n",
	}
	driver := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:               fake,
			WithInstanceMetadata: true,
		},
	}

	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "482", instance.Version)
	require.NotNil(t, instance.Metadata)
	require.Len(t, instance.Databases, 2)
	require.Equal(t, "memory", instance.Databases[0].Name)

	gotSchema, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(wantSchema, gotSchema, protocmp.Transform()))

	var dump strings.Builder
	require.NoError(t, driver.Dump(ctx, &dump, gotSchema))
	require.Equal(t, fake.dumpOut, dump.String())
	require.Empty(t, cmp.Diff(wantSchema, fake.dumpedSchema, protocmp.Transform()))

	fake.syncDBErr = errors.New("catalog unavailable")
	_, err = driver.SyncDBSchema(ctx)
	require.ErrorIs(t, err, fake.syncDBErr)

	fake.dumpErr = errors.New("dump failed")
	err = driver.Dump(ctx, &dump, gotSchema)
	require.ErrorIs(t, err, fake.dumpErr)
}
