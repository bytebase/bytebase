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

func TestValidateCatalogSemanticTypeIDs(t *testing.T) {
	tests := []struct {
		name    string
		config  *storepb.DatabaseConfig
		setting *storepb.SemanticTypeSetting
		wantErr string
	}{
		{
			name: "empty semantic type allowed",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{{
					Name: "public",
					Tables: []*storepb.TableCatalog{{
						Name:    "users",
						Columns: []*storepb.ColumnCatalog{{Name: "email"}},
					}},
				}},
			},
			setting: &storepb.SemanticTypeSetting{},
		},
		{
			name: "known column semantic type passes",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{{
					Name: "public",
					Tables: []*storepb.TableCatalog{{
						Name:    "users",
						Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "email"}},
					}},
				}},
			},
			setting: &storepb.SemanticTypeSetting{
				Types: []*storepb.SemanticTypeSetting_SemanticType{{Id: "email"}},
			},
		},
		{
			name: "unknown column semantic type rejected",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{{
					Name: "public",
					Tables: []*storepb.TableCatalog{{
						Name:    "users",
						Columns: []*storepb.ColumnCatalog{{Name: "email", SemanticType: "missing"}},
					}},
				}},
			},
			setting: &storepb.SemanticTypeSetting{
				Types: []*storepb.SemanticTypeSetting_SemanticType{{Id: "email"}},
			},
			wantErr: "invalid semantic type for column \"email\": semantic type id \"missing\" not found",
		},
		{
			name: "nested object semantic type rejected",
			config: &storepb.DatabaseConfig{
				Schemas: []*storepb.SchemaCatalog{{
					Name: "public",
					Tables: []*storepb.TableCatalog{{
						Name: "events",
						Columns: []*storepb.ColumnCatalog{{
							Name: "payload",
							ObjectSchema: &storepb.ObjectSchema{
								Type: storepb.ObjectSchema_OBJECT,
								Kind: &storepb.ObjectSchema_StructKind_{
									StructKind: &storepb.ObjectSchema_StructKind{
										Properties: map[string]*storepb.ObjectSchema{
											"email": {Type: storepb.ObjectSchema_STRING, SemanticType: "missing"},
										},
									},
								},
							},
						}},
					}},
				}},
			},
			setting: &storepb.SemanticTypeSetting{
				Types: []*storepb.SemanticTypeSetting_SemanticType{{Id: "email"}},
			},
			wantErr: "invalid object schema for column \"payload\": invalid semantic type for object property \"email\": semantic type id \"missing\" not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCatalogSemanticTypeIDs(tc.config, tc.setting)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
