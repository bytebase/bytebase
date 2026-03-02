package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidateCatalogSchemaNames(t *testing.T) {
	tests := []struct {
		name     string
		config   *storepb.DatabaseConfig
		metadata *storepb.DatabaseSchemaMetadata
		wantErr  bool
	}{
		{
			name: "named schemas pass",
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
			wantErr: false,
		},
		{
			name: "empty schema rejected for named-schema engine",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{
					{Name: "", Tables: []*storepb.TableCatalog{{Name: "users"}}},
				},
			},
			metadata: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{Name: "public", Tables: []*storepb.TableMetadata{{Name: "users"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "empty schema allowed when metadata has empty schema",
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
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCatalogSchemaNames(tc.config, tc.metadata)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
