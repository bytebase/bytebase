package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestUpdateDatabaseConfig(t *testing.T) {
	type testCase struct {
		description string
		target      *storepb.DatabaseConfig
		baseline    *storepb.DatabaseConfig
		head        *storepb.DatabaseConfig
		want        *storepb.DatabaseConfig
	}

	testCases := []testCase{
		{
			description: "easy change and the target config is the same as the baseline config",
			target: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
									{
										Name:           "b",
										SemanticTypeId: "id_b",
										Labels:         map[string]string{"hello": "world"},
									},
								},
							},
						},
					},
				},
			},
			baseline: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
								},
							},
						},
					},
				},
			},
			head: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
								},
							},
						},
					},
				},
			},
			want: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
									{
										Name:           "b",
										SemanticTypeId: "id_b",
										Labels:         map[string]string{"hello": "world"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "if the target config has changed the same column, we should overwrite it",
			target: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_b",
									},
								},
							},
						},
					},
				},
			},
			baseline: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
								},
							},
						},
					},
				},
			},
			head: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_c",
									},
								},
							},
						},
					},
				},
			},
			want: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_b",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "if the target config has changed, we should not overwrite the difference set",
			target: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										// Modify the semantic type id to id_b.
										Name:           "a",
										SemanticTypeId: "id_b",
										Labels:         map[string]string{"hello": "world"},
									},
									{
										// Do not change.
										Name:           "b",
										SemanticTypeId: "id_b",
									},
									{
										// Add a new column.
										Name:           "c",
										SemanticTypeId: "id_c",
									},
								},
							},
							// Delete the table.
							// {
							// 	Name: "table2",
							// 	Columns: []*storepb.ColumnCatalog{
							// 		{
							// 			Name:           "a",
							// 			SemanticTypeId: "id_a",
							// 		},
							// 	},
							// },
							// Create a new table.
							{
								Name: "table3",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
									{
										Name:           "b",
										SemanticTypeId: "id_b",
									},
								},
							},
						},
					},
				},
			},
			baseline: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
										Labels:         map[string]string{"world": "hello"},
									},
									{
										Name:           "b",
										SemanticTypeId: "id_b",
									},
								},
							},
							{
								Name: "table2",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
								},
							},
						},
					},
				},
			},
			head: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_c",
									},
									{
										Name:           "b",
										SemanticTypeId: "id_c",
									},
								},
							},
							{
								Name: "table2",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "b",
										SemanticTypeId: "id_b",
									},
								},
							},
						},
					},
				},
			},
			want: &storepb.DatabaseConfig{
				Name: "db1",
				Schemas: []*storepb.SchemaCatalog{
					{
						Name: "schema1",
						Tables: []*storepb.TableCatalog{
							{
								Name: "table1",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_b",
										Labels:         map[string]string{"hello": "world"},
									},
									{
										Name:           "b",
										SemanticTypeId: "id_c",
									},
									{
										Name:           "c",
										SemanticTypeId: "id_c",
									},
								},
							},
							{
								Name: "table3",
								Columns: []*storepb.ColumnCatalog{
									{
										Name:           "a",
										SemanticTypeId: "id_a",
									},
									{
										Name:           "b",
										SemanticTypeId: "id_b",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := MergeDatabaseConfig(tc.target, tc.baseline, tc.head)
		equal := proto.Equal(got, tc.want)
		a.True(equal, fmt.Sprintf("%s: \ngot:\t%%+v, \nexpected:\t%%+v", tc.description), got, tc.want)
	}
}
