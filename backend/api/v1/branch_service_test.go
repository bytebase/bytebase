package v1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestEqualTable(t *testing.T) {
	tests := []struct {
		s    *storepb.TableMetadata
		t    *storepb.TableMetadata
		want bool
	}{
		{
			s:    &storepb.TableMetadata{Comment: "a"},
			t:    &storepb.TableMetadata{Comment: "b"},
			want: false,
		},
		{
			s: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a"},
				},
			},
			t: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "b"},
				},
			},
			want: false,
		},
		{
			s: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a"},
				},
			},
			t: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a"},
					{Name: "b"},
				},
			},
			want: false,
		},
		{
			s: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int"},
					{Name: "b", Type: "int"},
				},
			},
			t: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int"},
					{Name: "b", Type: "varchar"},
				},
			},
			want: false,
		},
		{
			s: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int"},
				},
			},
			t: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int", Comment: "hello?"},
				},
			},
			want: false,
		},
		{
			s: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int", DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "abc"}},
				},
			},
			t: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int", DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "abc"}},
				},
			},
			want: true,
		},
		{
			s: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int", DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "abc"}},
				},
			},
			t: &storepb.TableMetadata{
				Columns: []*storepb.ColumnMetadata{
					{Name: "a", Type: "int", DefaultValue: &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: "cba"}},
				},
			},
			want: false,
		},
	}

	for i, tc := range tests {
		got := equalTable(tc.s, tc.t)
		require.Equal(t, tc.want, got, i)
	}
}

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
	require.Equal(t, "", diffTarget)
	diffSource := cmp.Diff(wantTrimmedSource, gotSource, protocmp.Transform())
	require.Equal(t, "", diffSource)
}
