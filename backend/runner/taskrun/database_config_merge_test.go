package taskrun

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
		current     *storepb.DatabaseConfig
		want        *storepb.DatabaseConfig
	}

	testCases := []testCase{
		{
			description: "easy change and the target config is the same as the baseline config",
			target: &storepb.DatabaseConfig{
				Name: "db1",
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
			current: &storepb.DatabaseConfig{
				Name: "db1",
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
			current: &storepb.DatabaseConfig{
				Name: "db1",
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
							// 	ColumnConfigs: []*storepb.ColumnConfig{
							// 		{
							// 			Name:           "a",
							// 			SemanticTypeId: "id_a",
							// 		},
							// 	},
							// },
							// Create a new table.
							{
								Name: "table3",
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
								ColumnConfigs: []*storepb.ColumnConfig{
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
			current: &storepb.DatabaseConfig{
				Name: "db1",
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
								ColumnConfigs: []*storepb.ColumnConfig{
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
				SchemaConfigs: []*storepb.SchemaConfig{
					{
						Name: "schema1",
						TableConfigs: []*storepb.TableConfig{
							{
								Name: "table1",
								ColumnConfigs: []*storepb.ColumnConfig{
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
								ColumnConfigs: []*storepb.ColumnConfig{
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
		got := mergeDatabaseConfig(tc.target, tc.baseline, tc.current)
		equal := proto.Equal(got, tc.want)
		a.True(equal, fmt.Sprintf("%s: \ngot:\t%%+v, \nexpected:\t%%+v", tc.description), got, tc.want)
	}
}
