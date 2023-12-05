package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestIsDiffConflict(t *testing.T) {
	type testCase struct {
		description string
		base        *storepb.DatabaseSchemaMetadata
		head        *storepb.DatabaseSchemaMetadata
		target      *storepb.DatabaseSchemaMetadata
		want        bool
	}

	defaultBase := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "employees",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "text",
							},
						},
					},
				},
			},
		},
	}

	testCases := []testCase{
		{
			description: "create different table with different name should not conflict",
			base:        defaultBase,
			head: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
								},
							},
							{
								Name: "salary",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "employee_id",
										Type: "int",
									},
									{
										Name: "amount",
										Type: "int",
									},
								},
							},
						},
					},
				},
			},
			target: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
								},
							},
							{
								Name: "office",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "address",
										Type: "text",
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}

	a := require.New(t)

	for _, tc := range testCases {
		diffBetweenBaseAndHead, err := diffMetadata(tc.base, tc.head)
		a.NoError(err, tc.description)
		a.NotNil(diffBetweenBaseAndHead, tc.description)
		diffBetweenBaseAndTarget, err := diffMetadata(tc.base, tc.target)
		a.NoError(err, tc.description)
		a.NotNil(diffBetweenBaseAndTarget, tc.description)

		isConflict, _ := diffBetweenBaseAndTarget.isConflictWith(diffBetweenBaseAndHead)
		a.Equal(tc.want, isConflict, tc.description)
	}
}

func TestTryMerge(t *testing.T) {
	type testCase struct {
		description string
		base        *storepb.DatabaseSchemaMetadata
		head        *storepb.DatabaseSchemaMetadata
		target      *storepb.DatabaseSchemaMetadata
		want        *storepb.DatabaseSchemaMetadata
	}

	defaultBase := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "employees",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "text",
							},
						},
					},
				},
			},
		},
	}

	testCases := []testCase{
		{
			description: "create different table with different name should not conflict",
			base:        defaultBase,
			head: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
								},
							},
							{
								Name: "salary",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "employee_id",
										Type: "int",
									},
									{
										Name: "amount",
										Type: "int",
									},
								},
							},
						},
					},
				},
			},
			target: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
								},
							},
							{
								Name: "office",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "address",
										Type: "text",
									},
								},
							},
						},
					},
				},
			},
			want: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
								},
							},
							{
								Name: "office",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "address",
										Type: "text",
									},
								},
							},
							{
								Name: "salary",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "employee_id",
										Type: "int",
									},
									{
										Name: "amount",
										Type: "int",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "add different column in the same table",
			base:        defaultBase,
			head: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
									{
										Name: "salary",
										Type: "int",
									},
								},
							},
						},
					},
				},
			},
			target: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
									{
										Name: "phone",
										Type: "text",
									},
								},
							},
						},
					},
				},
			},
			want: &storepb.DatabaseSchemaMetadata{
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "",
						Tables: []*storepb.TableMetadata{
							{
								Name: "employees",
								Columns: []*storepb.ColumnMetadata{
									{
										Name: "id",
										Type: "int",
									},
									{
										Name: "name",
										Type: "text",
									},
									{
										Name: "phone",
										Type: "text",
									},
									{
										Name: "salary",
										Type: "int",
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
		got, err := tryMerge(tc.base, tc.head, tc.target)
		a.NoError(err, tc.description)
		a.NotNil(got, tc.description)
		equal := proto.Equal(tc.want, got)
		a.True(equal, tc.description)
	}
}
