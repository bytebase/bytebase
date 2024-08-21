package v1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestTrimDatabaseMetadata(t *testing.T) {
	source := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "schema0"},
			{
				Name: "schema1",
				Tables: []*storepb.TableMetadata{
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "t1-c1", Type: "int"}}},
					{Name: "same-table", Columns: []*storepb.ColumnMetadata{{Name: "c1", Type: "int"}}},
				},
			},
			{
				Name: "same-schema",
				Tables: []*storepb.TableMetadata{
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "c1", Type: "int"}}},
				},
			},
		},
	}
	target := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "same-schema",
				Tables: []*storepb.TableMetadata{
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "c1", Type: "int"}}},
				},
			},
			{
				Name: "schema1",
				Tables: []*storepb.TableMetadata{
					{Name: "same-table", Columns: []*storepb.ColumnMetadata{{Name: "c1", Type: "int"}}},
					{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "t2-c1", Type: "int"}}},
				},
			},
			{Name: "schema2"},
		},
	}
	wantTrimmedSource := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{Name: "schema0"},
			{
				Name: "schema1",
				Tables: []*storepb.TableMetadata{
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "t1-c1", Type: "int"}}},
				},
			},
		},
	}
	wantTrimmedTarget := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "schema1",
				Tables: []*storepb.TableMetadata{
					{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "t2-c1", Type: "int"}}},
				},
			},
			{Name: "schema2"},
		},
	}

	gotSource, gotTarget := trimDatabaseMetadata(source, target)
	diffTarget := cmp.Diff(wantTrimmedTarget, gotTarget, protocmp.Transform())
	require.Empty(t, diffTarget)
	diffSource := cmp.Diff(wantTrimmedSource, gotSource, protocmp.Transform())
	require.Empty(t, diffSource)
}
