package schemasync

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestConvertDBSchema(t *testing.T) {
	defaultString := "Jay"

	dbSchema := &db.Schema{
		Name:         "db1",
		CharacterSet: "charset1",
		Collation:    "collation1",
		TableList: []db.Table{
			{
				Name:          "public.hello",
				ShortName:     "hello",
				Schema:        "public",
				CreatedTs:     20210113,
				UpdatedTs:     20221214,
				Type:          "BASE TABLE",
				Engine:        "btree",
				Collation:     "collation-hello",
				RowCount:      123,
				DataSize:      4542,
				IndexSize:     2113,
				DataFree:      123,
				CreateOptions: "create option hello",
				Comment:       "comment hello",
				ColumnList: []db.Column{
					{
						Name:         "name",
						Position:     1,
						Default:      &defaultString,
						Nullable:     false,
						Type:         "varchar(10)",
						CharacterSet: "charset1",
						Collation:    "collation1",
						Comment:      "col1",
					},
					{
						Name:         "id",
						Position:     0,
						Default:      nil,
						Nullable:     false,
						Type:         "int",
						CharacterSet: "charset1",
						Collation:    "collation1",
						Comment:      "col1",
					},
					{
						Name:         "age",
						Position:     2,
						Default:      nil,
						Nullable:     false,
						Type:         "int",
						CharacterSet: "charset1",
						Collation:    "collation1",
						Comment:      "col1",
					},
				},
				IndexList: []db.Index{
					{
						Name: "id-primary",
						// This could refer to a column or an expression.
						Expression: "id",
						Position:   0,
						Type:       "index type",
						Unique:     true,
						Primary:    true,
						Visible:    true,
						Comment:    "comment index",
					},
					{
						Name: "name-age-unique",
						// This could refer to a column or an expression.
						Expression: "age",
						Position:   1,
						Type:       "index type2",
						Unique:     true,
						Primary:    false,
						Visible:    true,
						Comment:    "comment index2",
					},
					{
						Name: "name-age-unique",
						// This could refer to a column or an expression.
						Expression: "name",
						Position:   0,
						Type:       "index type2",
						Unique:     true,
						Primary:    false,
						Visible:    true,
						Comment:    "comment index2",
					},
				},
			},
			{
				Name:          "hello",
				ShortName:     "hello",
				Schema:        "",
				CreatedTs:     20210113,
				UpdatedTs:     20221214,
				Type:          "BASE TABLE",
				Engine:        "btree",
				Collation:     "collation-hello",
				RowCount:      123,
				DataSize:      4542,
				IndexSize:     2113,
				DataFree:      123,
				CreateOptions: "create option hello",
				Comment:       "comment hello",
				ColumnList:    nil,
				IndexList:     nil,
			},
			{
				Name:          "public.world",
				ShortName:     "world",
				Schema:        "public",
				CreatedTs:     20210114,
				UpdatedTs:     20221215,
				Type:          "BASE TABLE",
				Engine:        "btree",
				Collation:     "collation-world",
				RowCount:      123,
				DataSize:      4542,
				IndexSize:     2113,
				DataFree:      123,
				CreateOptions: "create option world",
				Comment:       "comment world",
				ColumnList:    nil,
				IndexList:     nil,
			},
		},
		ViewList: []db.View{
			{
				Name:       "public.nice_view",
				ShortName:  "nice_view",
				Schema:     "public",
				CreatedTs:  2313,
				UpdatedTs:  21124,
				Definition: "select 1",
				Comment:    "comment view",
			},
		},
		ExtensionList: []db.Extension{
			{
				Name:        "hstore",
				Version:     "v2021",
				Schema:      "public",
				Description: "awesome hstore",
			},
			{
				Name:        "atom",
				Version:     "v2022",
				Schema:      "public",
				Description: "awesome sphere",
			},
		},
	}
	want := &storepb.DatabaseMetadata{
		Name:         "db1",
		CharacterSet: "charset1",
		Collation:    "collation1",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name:          "hello",
						Engine:        "btree",
						Collation:     "collation-hello",
						RowCount:      123,
						DataSize:      4542,
						IndexSize:     2113,
						DataFree:      123,
						CreateOptions: "create option hello",
						Comment:       "comment hello",
					},
				},
			},
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name:          "hello",
						Engine:        "btree",
						Collation:     "collation-hello",
						RowCount:      123,
						DataSize:      4542,
						IndexSize:     2113,
						DataFree:      123,
						CreateOptions: "create option hello",
						Comment:       "comment hello",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:         "id",
								Position:     0,
								HasDefault:   false,
								Default:      "",
								Nullable:     false,
								Type:         "int",
								CharacterSet: "charset1",
								Collation:    "collation1",
								Comment:      "col1",
							},
							{
								Name:         "name",
								Position:     1,
								HasDefault:   true,
								Default:      "Jay",
								Nullable:     false,
								Type:         "varchar(10)",
								CharacterSet: "charset1",
								Collation:    "collation1",
								Comment:      "col1",
							},
							{
								Name:         "age",
								Position:     2,
								HasDefault:   false,
								Default:      "",
								Nullable:     false,
								Type:         "int",
								CharacterSet: "charset1",
								Collation:    "collation1",
								Comment:      "col1",
							},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name: "id-primary",
								// This could refer to a column or an expression.
								Expressions: []string{"id"},
								Type:        "index type",
								Unique:      true,
								Primary:     true,
								Visible:     true,
								Comment:     "comment index",
							},
							{
								Name: "name-age-unique",
								// This could refer to a column or an expression.
								Expressions: []string{"name", "age"},
								Type:        "index type2",
								Unique:      true,
								Primary:     false,
								Visible:     true,
								Comment:     "comment index2",
							},
						},
					},
					{
						Name:          "world",
						Engine:        "btree",
						Collation:     "collation-world",
						RowCount:      123,
						DataSize:      4542,
						IndexSize:     2113,
						DataFree:      123,
						CreateOptions: "create option world",
						Comment:       "comment world",
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "nice_view",
						Definition: "select 1",
						Comment:    "comment view",
					},
				},
			},
		},
		Extensions: []*storepb.ExtensionMetadata{
			{
				Name:        "atom",
				Version:     "v2022",
				Schema:      "public",
				Description: "awesome sphere",
			},
			{
				Name:        "hstore",
				Version:     "v2021",
				Schema:      "public",
				Description: "awesome hstore",
			},
		},
	}
	got := convertDBSchema(dbSchema)
	assert.Equal(t, want, got)
}

func TestEqualDatabaseMetadata(t *testing.T) {
	tests := []struct {
		x    *storepb.DatabaseMetadata
		y    *storepb.DatabaseMetadata
		want bool
	}{
		{
			x:    &storepb.DatabaseMetadata{},
			y:    &storepb.DatabaseMetadata{},
			want: true,
		},
		{
			x:    nil,
			y:    &storepb.DatabaseMetadata{},
			want: false,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
			},
			y: &storepb.DatabaseMetadata{
				Name: "world",
			},
			want: false,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "students",
							},
						},
					},
				},
			},
			y: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "students",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "teachers",
							},
						},
					},
				},
			},
			y: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name: "students",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			x: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name:      "students",
								RowCount:  321,
								DataSize:  321,
								IndexSize: 321,
								DataFree:  321,
							},
						},
					},
				},
			},
			y: &storepb.DatabaseMetadata{
				Name: "hello",
				Schemas: []*storepb.SchemaMetadata{
					{
						Name: "public",
						Tables: []*storepb.TableMetadata{
							{
								Name:      "students",
								RowCount:  123,
								DataSize:  123,
								IndexSize: 123,
								DataFree:  123,
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, test := range tests {
		got := equalDatabaseMetadata(test.x, test.y)
		assert.Equal(t, test.want, got)
	}
}
