package tidb

import (
	"context"
	"errors"
	"io"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type fakeSyncDriver struct {
	instance     *omnidriver.Instance
	schema       *metadatapb.DatabaseSchemaMetadata
	syncDBErr    error
	dumpSQL      string
	dumpErr      error
	dumpedSchema *metadatapb.DatabaseSchemaMetadata
}

func (*fakeSyncDriver) Ping(context.Context) error { return nil }
func (*fakeSyncDriver) Close() error               { return nil }

func (f *fakeSyncDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeSyncDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	if f.syncDBErr != nil {
		return nil, f.syncDBErr
	}
	return f.schema, nil
}

func (f *fakeSyncDriver) Dump(_ context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	f.dumpedSchema = metadata
	if f.dumpErr != nil {
		return f.dumpErr
	}
	_, err := io.WriteString(out, f.dumpSQL)
	return err
}

// TestSyncDBSchemaPassesTheDriversMetadataThrough proves the adapter returns
// omni's synced database schema unchanged.
func TestSyncDBSchemaPassesTheDriversMetadataThrough(t *testing.T) {
	t.Parallel()

	want := &metadatapb.DatabaseSchemaMetadata{
		Name: "shop",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name: "orders",
				Columns: []*metadatapb.ColumnMetadata{
					{Name: "id", Type: "int"},
					{Name: "note", Type: "varchar(64)"},
				},
			}},
		}},
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeSyncDriver{schema: want}}}

	got, err := driver.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, got, protocmp.Transform()))
}

// TestSyncDBSchemaReportsAMissingDatabase pins the adapter's translation of
// omni's ErrNotFound into Bytebase's common.NotFound error.
func TestSyncDBSchemaReportsAMissingDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := &fakeSyncDriver{syncDBErr: omnidriver.ErrNotFound}
	driver := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:           fake,
			NotFoundResource: "database",
			DatabaseName:     "shop_missing",
		},
	}

	_, err := driver.SyncDBSchema(ctx)
	require.Error(t, err)
	require.Equal(t, common.NotFound, common.ErrorCode(err))
	require.Contains(t, err.Error(), `database "shop_missing" not found`)

	otherErr := errors.New("connection refused")
	fake.syncDBErr = otherErr
	_, err = driver.SyncDBSchema(ctx)
	require.ErrorIs(t, err, otherErr)
}

// TestSyncInstanceMapsTheDriversInstance pins how the adapter maps omni's
// instance version, databases, roles, and lower_case_table_names onto
// Bytebase's InstanceMetadata.
func TestSyncInstanceMapsTheDriversInstance(t *testing.T) {
	t.Parallel()

	grantAttr := "GRANT SELECT ON `shop`.* TO 'bb_sync_probe'@'%'"
	contract := &omnidriver.Instance{
		Version:             "8.0.11-TiDB-v8.5.5",
		LowerCaseTableNames: 2,
		Databases: []*metadatapb.DatabaseSchemaMetadata{
			{Name: "shop", CharacterSet: "utf8mb4", Collation: "utf8mb4_bin"},
		},
		Roles: []*omnidriver.Role{
			{Name: "'bb_sync_probe'@'%'", Attribute: &grantAttr},
		},
	}
	driver := &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:               &fakeSyncDriver{instance: contract},
			WithInstanceMetadata: true,
		},
	}

	instance, err := driver.SyncInstance(context.Background())
	require.NoError(t, err)
	require.Equal(t, contract.Version, instance.Version)
	require.NotNil(t, instance.Metadata)
	require.Equal(t, int32(2), instance.Metadata.GetMysqlLowerCaseTableNames())
	require.Empty(t, cmp.Diff(contract.Databases, instance.Databases, protocmp.Transform()))
	require.Empty(t, cmp.Diff([]*storepb.InstanceRole{{
		Name:      "'bb_sync_probe'@'%'",
		Attribute: &grantAttr,
	}}, instance.Metadata.GetRoles(), protocmp.Transform()))
}
