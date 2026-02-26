package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestNormalizeCatalogSchemaNames(t *testing.T) {
	tests := []struct {
		name     string
		config   *storepb.DatabaseConfig
		metadata *storepb.DatabaseSchemaMetadata
		want     *storepb.DatabaseConfig
	}{
		{
			name: "no empty schemas - passthrough",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
		},
		{
			name: "empty schema resolved to correct name",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "PII"}}}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "PII"}}}}},
				},
			},
		},
		{
			name: "merge with existing named schema - column level",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "id", SemanticType: "ID"}}}}},
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "PII"}}}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "id", SemanticType: "ID"}, {Name: "email", SemanticType: "PII"}}}}},
				},
			},
		},
		{
			name: "cassandra passthrough - metadata has empty schema",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "", Tables: []*storepb.TableMetadata{{Name: "users"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
		},
		{
			name: "orphan empty schema preserved as-is",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users"}}},
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "nonexistent"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users"}}},
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "nonexistent"}}},
				},
			},
		},
		{
			name: "nil metadata - passthrough",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
			metadata: nil,
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
		},
		{
			name: "unambiguous table resolved from non-default schema",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "logs"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
					{Name: "audit", Tables: []*storepb.TableMetadata{{Name: "logs"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "audit", Tables: []*storepb.TableCatalog{{Name: "logs"}}},
				},
			},
		},
		{
			name: "ambiguous table falls back to search_path default schema",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "logs"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "logs"}}},
					{Name: "audit", Tables: []*storepb.TableMetadata{{Name: "logs"}}},
				},
				SearchPath: "\"$user\", public",
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "logs"}}},
				},
			},
		},
		{
			name: "ambiguous table with no search_path preserved as-is",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "logs"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "logs"}}},
					{Name: "audit", Tables: []*storepb.TableMetadata{{Name: "logs"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "logs"}}},
				},
			},
		},
		{
			name: "multiple empty schemas resolve to different real schemas",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "PII"}}}}},
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "logs", Columns: []*storepb.ColumnCatalog{{Name: "ip", SemanticType: "PII"}}}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
					{Name: "audit", Tables: []*storepb.TableMetadata{{Name: "logs"}}},
				},
			},
			want: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "public", Tables: []*storepb.TableCatalog{{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "PII"}}}}},
					{Name: "audit", Tables: []*storepb.TableCatalog{{Name: "logs", Columns: []*storepb.ColumnCatalog{{Name: "ip", SemanticType: "PII"}}}}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeCatalogSchemaNames(tc.config, tc.metadata)
			require.Equal(t, len(tc.want.Schemas), len(got.Schemas), "schema count mismatch")
			for i, wantSchema := range tc.want.Schemas {
				gotSchema := got.Schemas[i]
				require.Equal(t, wantSchema.Name, gotSchema.Name, "schema name mismatch at index %d", i)
				require.Equal(t, len(wantSchema.Tables), len(gotSchema.Tables), "table count mismatch for schema %q", wantSchema.Name)
				for j, wantTable := range wantSchema.Tables {
					gotTable := gotSchema.Tables[j]
					require.Equal(t, wantTable.Name, gotTable.Name, "table name mismatch")
					require.Equal(t, len(wantTable.Columns), len(gotTable.Columns), "column count mismatch for table %q", wantTable.Name)
					for k, wantCol := range wantTable.Columns {
						gotCol := gotTable.Columns[k]
						require.Equal(t, wantCol.Name, gotCol.Name, "column name mismatch")
						require.Equal(t, wantCol.SemanticType, gotCol.SemanticType, "semantic type mismatch for column %q", wantCol.Name)
					}
				}
			}
		})
	}
}

func TestMergeTableCatalogs(t *testing.T) {
	tests := []struct {
		name     string
		base     []*storepb.TableCatalog
		override []*storepb.TableCatalog
		want     []*storepb.TableCatalog
	}{
		{
			name: "no overlap",
			base: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "id"}}},
			},
			override: []*storepb.TableCatalog{
				{Name: "orders", Columns: []*storepb.ColumnCatalog{{Name: "total"}}},
			},
			want: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "id"}}},
				{Name: "orders", Columns: []*storepb.ColumnCatalog{{Name: "total"}}},
			},
		},
		{
			name: "overlapping table merges columns",
			base: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "id", SemanticType: "ID"}}},
			},
			override: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "PII"}}},
			},
			want: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "id", SemanticType: "ID"}, {Name: "email", SemanticType: "PII"}}},
			},
		},
		{
			name: "overlapping column - override wins",
			base: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "OLD"}}},
			},
			override: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "NEW"}}},
			},
			want: []*storepb.TableCatalog{
				{Name: "users", Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "NEW"}}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeTableCatalogs(tc.base, tc.override)
			require.Equal(t, len(tc.want), len(got), "table count mismatch")
			for i, wantTable := range tc.want {
				gotTable := got[i]
				require.Equal(t, wantTable.Name, gotTable.Name)
				require.Equal(t, len(wantTable.Columns), len(gotTable.Columns), "column count for %q", wantTable.Name)
				for j, wantCol := range wantTable.Columns {
					require.Equal(t, wantCol.Name, gotTable.Columns[j].Name)
					require.Equal(t, wantCol.SemanticType, gotTable.Columns[j].SemanticType)
				}
			}
		})
	}
}

func TestMergeColumnCatalogs(t *testing.T) {
	tests := []struct {
		name         string
		dst          *storepb.TableCatalog
		src          *storepb.TableCatalog
		wantColumns  []*storepb.ColumnCatalog
		wantClassify string
	}{
		{
			name: "new columns appended",
			dst:  &storepb.TableCatalog{Name: "t", Columns: []*storepb.ColumnCatalog{{Name: "a", SemanticType: "X"}}},
			src:  &storepb.TableCatalog{Name: "t", Columns: []*storepb.ColumnCatalog{{Name: "b", SemanticType: "Y"}}},
			wantColumns: []*storepb.ColumnCatalog{
				{Name: "a", SemanticType: "X"},
				{Name: "b", SemanticType: "Y"},
			},
		},
		{
			name: "duplicate column - src wins",
			dst:  &storepb.TableCatalog{Name: "t", Columns: []*storepb.ColumnCatalog{{Name: "a", SemanticType: "OLD"}}},
			src:  &storepb.TableCatalog{Name: "t", Columns: []*storepb.ColumnCatalog{{Name: "a", SemanticType: "NEW"}}},
			wantColumns: []*storepb.ColumnCatalog{
				{Name: "a", SemanticType: "NEW"},
			},
		},
		{
			name:         "classification from src overrides dst",
			dst:          &storepb.TableCatalog{Name: "t", Classification: "old-class"},
			src:          &storepb.TableCatalog{Name: "t", Classification: "new-class"},
			wantClassify: "new-class",
		},
		{
			name:         "empty classification from src preserves dst",
			dst:          &storepb.TableCatalog{Name: "t", Classification: "keep"},
			src:          &storepb.TableCatalog{Name: "t", Classification: ""},
			wantClassify: "keep",
		},
		{
			name: "src with no columns preserves dst columns",
			dst:  &storepb.TableCatalog{Name: "t", Columns: []*storepb.ColumnCatalog{{Name: "a", SemanticType: "X"}}},
			src:  &storepb.TableCatalog{Name: "t"},
			wantColumns: []*storepb.ColumnCatalog{
				{Name: "a", SemanticType: "X"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mergeColumnCatalogs(tc.dst, tc.src)
			if tc.wantColumns != nil {
				require.Equal(t, len(tc.wantColumns), len(tc.dst.Columns), "column count")
				for i, want := range tc.wantColumns {
					require.Equal(t, want.Name, tc.dst.Columns[i].Name)
					require.Equal(t, want.SemanticType, tc.dst.Columns[i].SemanticType)
				}
			}
			if tc.wantClassify != "" {
				require.Equal(t, tc.wantClassify, tc.dst.Classification)
			}
		})
	}
}
