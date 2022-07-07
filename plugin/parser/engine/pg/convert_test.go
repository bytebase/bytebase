//go:build !release
// +build !release

package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
)

type testData struct {
	stmt string
	want []ast.Node
}

func TestPGConvertCreateTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "CREATE TABLE \"techBook\" (a int, b int)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: false,
					Name: &ast.TableDef{
						Name: "techBook",
					},
					ColumnList: []*ast.ColumnDef{
						{ColumnName: "a"},
						{ColumnName: "b"},
					},
				},
			},
		},
		{
			stmt: "CREATE TABLE IF NOT EXISTS techBook (\"A\" int, b int)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: true,
					Name: &ast.TableDef{
						Name: "techbook",
					},
					ColumnList: []*ast.ColumnDef{
						{ColumnName: "A"},
						{ColumnName: "b"},
					},
				},
			},
		},
		{
			stmt: "CREATE TABLE tech_book(a INT CONSTRAINT t_pk_a PRIMARY KEY)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					Name: &ast.TableDef{
						Name: "tech_book",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "a",
							ConstraintList: []*ast.ConstraintDef{
								{
									Name:    "t_pk_a",
									Type:    ast.ConstraintTypePrimary,
									KeyList: []string{"a"},
								},
							},
						},
					},
				},
			},
		},
		{
			stmt: "CREATE TABLE tech_book(a INT, b int CONSTRAINT uk_b UNIQUE, CONSTRAINT t_pk_a PRIMARY KEY(a))",
			want: []ast.Node{
				&ast.CreateTableStmt{
					Name: &ast.TableDef{
						Name: "tech_book",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "a",
						},
						{
							ColumnName: "b",
							ConstraintList: []*ast.ConstraintDef{
								{
									Name:    "uk_b",
									Type:    ast.ConstraintTypeUnique,
									KeyList: []string{"b"},
								},
							},
						},
					},
					ConstraintList: []*ast.ConstraintDef{
						{
							Name:    "t_pk_a",
							Type:    ast.ConstraintTypePrimary,
							KeyList: []string{"a"},
						},
					},
				},
			},
		},
		{
			stmt: "CREATE TABLE tech_book(a INT CONSTRAINT fk_a REFERENCES people(id), CONSTRAINT fk_a_people_b FOREIGN KEY (a) REFERENCES people(b))",
			want: []ast.Node{
				&ast.CreateTableStmt{
					Name: &ast.TableDef{
						Name: "tech_book",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "a",
							ConstraintList: []*ast.ConstraintDef{
								{
									Name:    "fk_a",
									Type:    ast.ConstraintTypeForeign,
									KeyList: []string{"a"},
									Foreign: &ast.ForeignDef{
										Table:      &ast.TableDef{Name: "people"},
										ColumnList: []string{"id"},
									},
								},
							},
						},
					},
					ConstraintList: []*ast.ConstraintDef{
						{
							Name:    "fk_a_people_b",
							Type:    ast.ConstraintTypeForeign,
							KeyList: []string{"a"},
							Foreign: &ast.ForeignDef{
								Table:      &ast.TableDef{Name: "people"},
								ColumnList: []string{"b"},
							},
						},
					},
				},
			},
		},
	}

	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		require.Nilf(t, pretty.Diff(test.want, res), "stmt: %s", test.stmt)
	}
}

func TestPGAddColumnStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE techbook ADD COLUMN a int",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.AddColumnListStmt{
							Table: &ast.TableDef{
								Name: "techbook",
							},
							ColumnList: []*ast.ColumnDef{
								{ColumnName: "a"},
							},
						},
					},
				},
			},
		},
	}

	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		require.Nilf(t, pretty.Diff(test.want, res), "stmt: %s", test.stmt)
	}
}

func TestPGRenameTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE techbook RENAME TO \"techBook\"",
			want: []ast.Node{
				&ast.RenameTableStmt{
					Table: &ast.TableDef{
						Name: "techbook",
					},
					NewName: "techBook",
				},
			},
		},
	}

	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		require.Nilf(t, pretty.Diff(test.want, res), "stmt: %s", test.stmt)
	}
}

func TestPGRenameColumnStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE techbook RENAME abc TO \"ABC\"",
			want: []ast.Node{
				&ast.RenameColumnStmt{
					Table: &ast.TableDef{
						Name: "techbook",
					},
					ColumnName: "abc",
					NewName:    "ABC",
				},
			},
		},
	}

	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		require.Nilf(t, pretty.Diff(test.want, res), "stmt: %s", test.stmt)
	}
}
