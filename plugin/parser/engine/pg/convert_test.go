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

func runTests(t *testing.T, tests []testData) {
	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		require.Nilf(t, pretty.Diff(test.want, res), "stmt: %s", test.stmt)
	}
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

	runTests(t, tests)
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
		{
			stmt: "ALTER TABLE techbook ADD COLUMN a int CONSTRAINT uk_techbook_a UNIQUE",
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
								{
									ColumnName: "a",
									ConstraintList: []*ast.ConstraintDef{
										{
											Type:    ast.ConstraintTypeUnique,
											Name:    "uk_techbook_a",
											KeyList: []string{"a"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
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

	runTests(t, tests)
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

	runTests(t, tests)
}

func TestPGRenameConstraintStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book RENAME CONSTRAINT uk_tech_a to \"UK_TECH_A\"",
			want: []ast.Node{
				&ast.RenameConstraintStmt{
					Table:          &ast.TableDef{Name: "tech_book"},
					ConstraintName: "uk_tech_a",
					NewName:        "UK_TECH_A",
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

func TestPGDropConstraintStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book DROP CONSTRAINT uk_tech_a",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{Name: "tech_book"},
					AlterItemList: []ast.Node{
						&ast.DropConstraintStmt{
							Table:          &ast.TableDef{Name: "tech_book"},
							ConstraintName: "uk_tech_a",
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGAddConstraintStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE (id)",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{Name: "tech_book"},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{Name: "tech_book"},
							Constraint: &ast.ConstraintDef{
								Type:    ast.ConstraintTypeUnique,
								Name:    "uk_tech_book_id",
								KeyList: []string{"id"},
							},
						},
					},
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY (id)",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{Name: "tech_book"},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{Name: "tech_book"},
							Constraint: &ast.ConstraintDef{
								Type:    ast.ConstraintTypePrimary,
								Name:    "pk_tech_book_id",
								KeyList: []string{"id"},
							},
						},
					},
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id)",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{Name: "tech_book"},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{Name: "tech_book"},
							Constraint: &ast.ConstraintDef{
								Type:    ast.ConstraintTypeForeign,
								Name:    "fk_tech_book_id",
								KeyList: []string{"id"},
								Foreign: &ast.ForeignDef{
									Table:      &ast.TableDef{Name: "people"},
									ColumnList: []string{"id"},
								},
							},
						},
					},
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE USING INDEX uk_id",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{Name: "tech_book"},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{Name: "tech_book"},
							Constraint: &ast.ConstraintDef{
								Type:      ast.ConstraintTypeUniqueUsingIndex,
								Name:      "uk_tech_book_id",
								IndexName: "uk_id",
							},
						},
					},
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY USING INDEX pk_id",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{Name: "tech_book"},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{Name: "tech_book"},
							Constraint: &ast.ConstraintDef{
								Type:      ast.ConstraintTypePrimaryUsingIndex,
								Name:      "pk_tech_book_id",
								IndexName: "pk_id",
							},
						},
					},
				},
			},
		},
	}

	runTests(t, tests)
}
