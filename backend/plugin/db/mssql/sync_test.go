package mssql

import (
	"context"
	"testing"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type stubSyncDriver struct {
	schema *metadatapb.DatabaseSchemaMetadata
	err    error
}

func (*stubSyncDriver) Ping(context.Context) error                              { return nil }
func (*stubSyncDriver) Close() error                                            { return nil }
func (*stubSyncDriver) SyncInstance(context.Context) (*odriver.Instance, error) { return nil, nil }
func (d *stubSyncDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.schema, d.err
}

func TestSyncDBSchemaAdapterSeam(t *testing.T) {
	want := &metadatapb.DatabaseSchemaMetadata{
		Name:          "appdb",
		DefaultSchema: "dbo",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "dbo",
			Tables: []*metadatapb.TableMetadata{{Name: "orders"}},
		}},
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubSyncDriver{schema: want}}}

	got, err := driver.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.Equal(t, "dbo", got.DefaultSchema)
	require.Equal(t, "orders", requireTable(t, got, "dbo", "orders").Name)
}
