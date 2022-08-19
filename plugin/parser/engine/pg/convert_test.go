package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/stretchr/testify/require"
)

type testData struct {
	stmt           string
	want           []ast.Node
	statementList  []parser.SingleSQL
	columnLine     [][]int
	constraintLine [][]int
}

func runTests(t *testing.T, tests []testData) {
	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		for i := range test.want {
			test.want[i].SetText(test.statementList[i].Text)
			test.want[i].SetLine(test.statementList[i].Line)

			switch n := test.want[i].(type) {
			case *ast.CreateTableStmt:
				for j, col := range n.ColumnList {
					col.SetLine(test.columnLine[i][j])
					for _, inlineCons := range col.ConstraintList {
						inlineCons.SetLine(col.Line())
					}
				}
				for j, cons := range n.ConstraintList {
					cons.SetLine(test.constraintLine[i][j])
				}
			case *ast.AlterTableStmt:
				for _, item := range n.AlterItemList {
					item.SetLine(n.Line())
				}
			}
		}
		require.Equal(t, test.want, res, test.stmt)
	}
}

func TestPGConvertCreateTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "CREATE TABLE \"techBook\" (a int NOT NULL, b int CONSTRAINT b_not_null NOT NULL)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: false,
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "techBook",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "a",
							ConstraintList: []*ast.ConstraintDef{
								{
									Type:    ast.ConstraintTypeNotNull,
									KeyList: []string{"a"},
								},
							},
						},
						{
							ColumnName: "b",
							ConstraintList: []*ast.ConstraintDef{
								{
									Type:    ast.ConstraintTypeNotNull,
									Name:    "b_not_null",
									KeyList: []string{"b"},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE TABLE \"techBook\" (a int NOT NULL, b int CONSTRAINT b_not_null NOT NULL)",
					Line: 1,
				},
			},
			columnLine: [][]int{
				{1, 1},
			},
		},
		{
			stmt: "CREATE TABLE IF NOT EXISTS techBook (\"A\" int, b int)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: true,
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "techbook",
					},
					ColumnList: []*ast.ColumnDef{
						{ColumnName: "A"},
						{ColumnName: "b"},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE TABLE IF NOT EXISTS techBook (\"A\" int, b int)",
					Line: 1,
				},
			},
			columnLine: [][]int{
				{1, 1},
			},
		},
		{
			stmt: "CREATE TABLE tech_book(a INT CONSTRAINT t_pk_a PRIMARY KEY)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
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
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE TABLE tech_book(a INT CONSTRAINT t_pk_a PRIMARY KEY)",
					Line: 1,
				},
			},
			columnLine: [][]int{
				{1},
			},
		},
		{
			stmt: "CREATE TABLE tech_book(a INT, b int CONSTRAINT uk_b UNIQUE, CONSTRAINT t_pk_a PRIMARY KEY(a))",
			want: []ast.Node{
				&ast.CreateTableStmt{
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
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
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE TABLE tech_book(a INT, b int CONSTRAINT uk_b UNIQUE, CONSTRAINT t_pk_a PRIMARY KEY(a))",
					Line: 1,
				},
			},
			columnLine: [][]int{
				{1, 1},
			},
			constraintLine: [][]int{
				{1},
			},
		},
		{
			stmt: "CREATE TABLE tech_book(a INT CONSTRAINT fk_a REFERENCES people(id), CONSTRAINT fk_a_people_b FOREIGN KEY (a) REFERENCES people(b))",
			want: []ast.Node{
				&ast.CreateTableStmt{
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
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
										Table: &ast.TableDef{
											Type: ast.TableTypeBaseTable,
											Name: "people",
										},
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
								Table: &ast.TableDef{
									Type: ast.TableTypeBaseTable,
									Name: "people",
								},
								ColumnList: []string{"b"},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE TABLE tech_book(a INT CONSTRAINT fk_a REFERENCES people(id), CONSTRAINT fk_a_people_b FOREIGN KEY (a) REFERENCES people(b))",
					Line: 1,
				},
			},
			columnLine: [][]int{
				{1},
			},
			constraintLine: [][]int{
				{1},
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
						Type: ast.TableTypeBaseTable,
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.AddColumnListStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "techbook",
							},
							ColumnList: []*ast.ColumnDef{
								{ColumnName: "a"},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE techbook ADD COLUMN a int",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE techbook ADD COLUMN a int CONSTRAINT uk_techbook_a UNIQUE",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.AddColumnListStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
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
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE techbook ADD COLUMN a int CONSTRAINT uk_techbook_a UNIQUE",
					Line: 1,
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
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.RenameTableStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "techbook",
							},
							NewName: "techBook",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE techbook RENAME TO \"techBook\"",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER VIEW techbook RENAME TO \"techBook\"",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeView,
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.RenameTableStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeView,
								Name: "techbook",
							},
							NewName: "techBook",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER VIEW techbook RENAME TO \"techBook\"",
					Line: 1,
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
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.RenameColumnStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "techbook",
							},
							ColumnName: "abc",
							NewName:    "ABC",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE techbook RENAME abc TO \"ABC\"",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER VIEW techbook RENAME abc TO \"ABC\"",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeView,
						Name: "techbook",
					},
					AlterItemList: []ast.Node{
						&ast.RenameColumnStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeView,
								Name: "techbook",
							},
							ColumnName: "abc",
							NewName:    "ABC",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER VIEW techbook RENAME abc TO \"ABC\"",
					Line: 1,
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
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.RenameConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ConstraintName: "uk_tech_a",
							NewName:        "UK_TECH_A",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book RENAME CONSTRAINT uk_tech_a to \"UK_TECH_A\"",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGCreateIndexStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "CREATE INDEX idx_id ON tech_book (id)",
			want: []ast.Node{
				&ast.CreateIndexStmt{
					Index: &ast.IndexDef{
						Name:   "idx_id",
						Table:  &ast.TableDef{Name: "tech_book"},
						Unique: false,
						KeyList: []*ast.IndexKeyDef{
							{
								Type: ast.IndexKeyTypeColumn,
								Key:  "id",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE INDEX idx_id ON tech_book (id)",
					Line: 1,
				},
			},
		},
		{
			stmt: "CREATE UNIQUE INDEX idx_id ON tech_book (id)",
			want: []ast.Node{
				&ast.CreateIndexStmt{
					Index: &ast.IndexDef{
						Name:   "idx_id",
						Table:  &ast.TableDef{Name: "tech_book"},
						Unique: true,
						KeyList: []*ast.IndexKeyDef{
							{
								Type: ast.IndexKeyTypeColumn,
								Key:  "id",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "CREATE UNIQUE INDEX idx_id ON tech_book (id)",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGDropIndexStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "DROP INDEX xschema.idx_id, idx_x",
			want: []ast.Node{
				&ast.DropIndexStmt{
					IndexList: []*ast.IndexDef{
						{
							Table: &ast.TableDef{Schema: "xschema"},
							Name:  "idx_id",
						},
						{Name: "idx_x"},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DROP INDEX xschema.idx_id, idx_x",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGAlterIndexStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER INDEX xschema.idx_id RENAME TO \"IDX_ID\"",
			want: []ast.Node{
				&ast.RenameIndexStmt{
					Table:     &ast.TableDef{Schema: "xschema"},
					IndexName: "idx_id",
					NewName:   "IDX_ID",
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER INDEX xschema.idx_id RENAME TO \"IDX_ID\"",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER INDEX idx_id RENAME TO \"IDX_ID\"",
			want: []ast.Node{
				&ast.RenameIndexStmt{
					Table:     &ast.TableDef{Schema: ""},
					IndexName: "idx_id",
					NewName:   "IDX_ID",
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER INDEX idx_id RENAME TO \"IDX_ID\"",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGDropConstraintStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book DROP CONSTRAINT uk_tech_a",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.DropConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ConstraintName: "uk_tech_a",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book DROP CONSTRAINT uk_tech_a",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGAddConstraintStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT check_a_bigger_than_b CHECK (a > b) NOT VALID",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							Constraint: &ast.ConstraintDef{
								Type:            ast.ConstraintTypeCheck,
								Name:            "check_a_bigger_than_b",
								SkipValidation:  true,
								CheckExpression: &ast.UnconvertedExpressionDef{},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ADD CONSTRAINT check_a_bigger_than_b CHECK (a > b) NOT VALID",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE (id)",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							Constraint: &ast.ConstraintDef{
								Type:    ast.ConstraintTypeUnique,
								Name:    "uk_tech_book_id",
								KeyList: []string{"id"},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE (id)",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY (id)",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							Constraint: &ast.ConstraintDef{
								Type:    ast.ConstraintTypePrimary,
								Name:    "pk_tech_book_id",
								KeyList: []string{"id"},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY (id)",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id)",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							Constraint: &ast.ConstraintDef{
								Type:    ast.ConstraintTypeForeign,
								Name:    "fk_tech_book_id",
								KeyList: []string{"id"},
								Foreign: &ast.ForeignDef{
									Table: &ast.TableDef{
										Type: ast.TableTypeBaseTable,
										Name: "people",
									},
									ColumnList: []string{"id"},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id)",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE USING INDEX uk_id",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							Constraint: &ast.ConstraintDef{
								Type:      ast.ConstraintTypeUniqueUsingIndex,
								Name:      "uk_tech_book_id",
								IndexName: "uk_id",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE USING INDEX uk_id",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY USING INDEX pk_id",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							Constraint: &ast.ConstraintDef{
								Type:      ast.ConstraintTypePrimaryUsingIndex,
								Name:      "pk_tech_book_id",
								IndexName: "pk_id",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY USING INDEX pk_id",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGDropColumnStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book DROP COLUMN a",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.DropColumnStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ColumnName: "a",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book DROP COLUMN a",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGDropTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "DROP TABLE tech_book, xschema.user",
			want: []ast.Node{
				&ast.DropTableStmt{
					TableList: []*ast.TableDef{
						{
							Type: ast.TableTypeBaseTable,
							Name: "tech_book",
						},
						{
							Type:   ast.TableTypeBaseTable,
							Schema: "xschema",
							Name:   "user",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DROP TABLE tech_book, xschema.user",
					Line: 1,
				},
			},
		},
		{
			stmt: "DROP VIEW tech_book, xschema.user",
			want: []ast.Node{
				&ast.DropTableStmt{
					TableList: []*ast.TableDef{
						{
							Type: ast.TableTypeView,
							Name: "tech_book",
						},
						{
							Type:   ast.TableTypeView,
							Schema: "xschema",
							Name:   "user",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DROP VIEW tech_book, xschema.user",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGNotNullStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book ALTER COLUMN id SET NOT NULL",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.SetNotNullStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ColumnName: "id",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ALTER COLUMN id SET NOT NULL",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ALTER COLUMN id DROP NOT NULL",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.DropNotNullStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ColumnName: "id",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ALTER COLUMN id DROP NOT NULL",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGSelectStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "SELECT public.t.a, t.*, t1.* FROM (SELECT * FROM t) t, t1",
			want: []ast.Node{
				&ast.SelectStmt{
					SetOperation: ast.SetOperationTypeNone,
					FieldList: []ast.ExpressionNode{
						&ast.ColumnNameDef{
							Table: &ast.TableDef{
								Schema: "public",
								Name:   "t",
							},
							ColumnName: "a",
						},
						&ast.ColumnNameDef{
							Table:      &ast.TableDef{Name: "t"},
							ColumnName: "*",
						},
						&ast.ColumnNameDef{
							Table:      &ast.TableDef{Name: "t1"},
							ColumnName: "*",
						},
					},
					SubqueryList: []*ast.SubqueryDef{
						{
							Select: &ast.SelectStmt{
								SetOperation: ast.SetOperationTypeNone,
								FieldList: []ast.ExpressionNode{
									&ast.ColumnNameDef{
										Table:      &ast.TableDef{},
										ColumnName: "*",
									},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "SELECT public.t.a, t.*, t1.* FROM (SELECT * FROM t) t, t1",
					Line: 1,
				},
			},
		},
		{
			stmt: "SELECT public.t.a, t.*, * FROM t",
			want: []ast.Node{
				&ast.SelectStmt{
					SetOperation: ast.SetOperationTypeNone,
					FieldList: []ast.ExpressionNode{
						&ast.ColumnNameDef{
							Table: &ast.TableDef{
								Schema: "public",
								Name:   "t",
							},
							ColumnName: "a",
						},
						&ast.ColumnNameDef{
							Table:      &ast.TableDef{Name: "t"},
							ColumnName: "*",
						},
						&ast.ColumnNameDef{
							Table:      &ast.TableDef{},
							ColumnName: "*",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "SELECT public.t.a, t.*, * FROM t",
					Line: 1,
				},
			},
		},
		{
			stmt: `
				SELECT
					public.t.a, b, lower(a), b>a
				FROM
					t
				WHERE
					a > 0
					AND (c not LIKE 'xyz' or true)
					AND b LIKE '%csdbc'
					AND a in
						(SELECT * FROM t1 WHERE x LIKE b)
				UNION
				SELECT * FROM t`,
			want: []ast.Node{
				&ast.SelectStmt{
					SetOperation: ast.SetOperationTypeUnion,
					LQuery: &ast.SelectStmt{
						SetOperation: ast.SetOperationTypeNone,
						FieldList: []ast.ExpressionNode{
							&ast.ColumnNameDef{
								Table: &ast.TableDef{
									Schema: "public",
									Name:   "t",
								},
								ColumnName: "a",
							},
							&ast.ColumnNameDef{
								Table:      &ast.TableDef{},
								ColumnName: "b",
							},
							&ast.UnconvertedExpressionDef{},
							&ast.UnconvertedExpressionDef{},
						},
						WhereClause: &ast.UnconvertedExpressionDef{},
						PatternLikeList: []*ast.PatternLikeDef{
							{
								Not: true,
								Expression: &ast.ColumnNameDef{
									Table:      &ast.TableDef{},
									ColumnName: "c",
								},
								Pattern: &ast.StringDef{Value: "xyz"},
							},
							{
								Expression: &ast.ColumnNameDef{
									Table:      &ast.TableDef{},
									ColumnName: "b",
								},
								Pattern: &ast.StringDef{Value: "%csdbc"},
							},
						},
						SubqueryList: []*ast.SubqueryDef{
							{
								Select: &ast.SelectStmt{
									SetOperation: ast.SetOperationTypeNone,
									FieldList: []ast.ExpressionNode{
										&ast.ColumnNameDef{
											Table:      &ast.TableDef{},
											ColumnName: "*",
										},
									},
									WhereClause: &ast.PatternLikeDef{
										Expression: &ast.ColumnNameDef{
											Table:      &ast.TableDef{},
											ColumnName: "x",
										},
										Pattern: &ast.ColumnNameDef{
											Table:      &ast.TableDef{},
											ColumnName: "b",
										},
									},
									PatternLikeList: []*ast.PatternLikeDef{
										{
											Expression: &ast.ColumnNameDef{
												Table:      &ast.TableDef{},
												ColumnName: "x",
											},
											Pattern: &ast.ColumnNameDef{
												Table:      &ast.TableDef{},
												ColumnName: "b",
											},
										},
									},
								},
							},
						},
					},
					RQuery: &ast.SelectStmt{
						SetOperation: ast.SetOperationTypeNone,
						FieldList: []ast.ExpressionNode{
							&ast.ColumnNameDef{
								Table:      &ast.TableDef{},
								ColumnName: "*",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `SELECT
					public.t.a, b, lower(a), b>a
				FROM
					t
				WHERE
					a > 0
					AND (c not LIKE 'xyz' or true)
					AND b LIKE '%csdbc'
					AND a in
						(SELECT * FROM t1 WHERE x LIKE b)
				UNION
				SELECT * FROM t`,
					Line: 2,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGDropDatabaseStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "DROP DATABASE test",
			want: []ast.Node{
				&ast.DropDatabaseStmt{
					DatabaseName: "test",
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DROP DATABASE test",
					Line: 1,
				},
			},
		},
		{
			stmt: "DROP DATABASE IF EXISTS test",
			want: []ast.Node{
				&ast.DropDatabaseStmt{
					DatabaseName: "test",
					IfExists:     true,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DROP DATABASE IF EXISTS test",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestUpdateStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "UPDATE tech_book SET a = 1 FROM (SELECT * FROM t) t WHERE a > 1",
			want: []ast.Node{
				&ast.UpdateStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					WhereClause: &ast.UnconvertedExpressionDef{},
					SubqueryList: []*ast.SubqueryDef{
						{
							Select: &ast.SelectStmt{
								SetOperation: ast.SetOperationTypeNone,
								FieldList: []ast.ExpressionNode{
									&ast.ColumnNameDef{
										Table:      &ast.TableDef{},
										ColumnName: "*",
									},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "UPDATE tech_book SET a = 1 FROM (SELECT * FROM t) t WHERE a > 1",
					Line: 1,
				},
			},
		},
		{
			stmt: "UPDATE tech_book SET a = 1",
			want: []ast.Node{
				&ast.UpdateStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "UPDATE tech_book SET a = 1",
					Line: 1,
				},
			},
		},
		{
			stmt: "UPDATE tech_book SET a = 1 WHERE a > 1",
			want: []ast.Node{
				&ast.UpdateStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					WhereClause: &ast.UnconvertedExpressionDef{},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "UPDATE tech_book SET a = 1 WHERE a > 1",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}
func TestDeleteStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "DELETE FROM tech_book",
			want: []ast.Node{
				&ast.DeleteStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DELETE FROM tech_book",
					Line: 1,
				},
			},
		},
		{
			stmt: "DELETE FROM tech_book WHERE a > 1",
			want: []ast.Node{
				&ast.DeleteStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					WhereClause: &ast.UnconvertedExpressionDef{},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "DELETE FROM tech_book WHERE a > 1",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestSetSchemaStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book SET SCHEMA new_schema",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.SetSchemaStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							NewSchema: "new_schema",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book SET SCHEMA new_schema",
					Line: 1,
				},
			},
		},
		{
			stmt: "ALTER VIEW tech_book SET SCHEMA new_schema",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeView,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.SetSchemaStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeView,
								Name: "tech_book",
							},
							NewSchema: "new_schema",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER VIEW tech_book SET SCHEMA new_schema",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestExplainStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "EXPLAIN SELECT * FROM tech_book",
			want: []ast.Node{
				&ast.ExplainStmt{
					Statement: &ast.SelectStmt{
						SetOperation: ast.SetOperationTypeNone,
						FieldList: []ast.ExpressionNode{
							&ast.ColumnNameDef{
								Table:      &ast.TableDef{},
								ColumnName: "*",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "EXPLAIN SELECT * FROM tech_book",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestAlterColumnType(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book ALTER COLUMN a TYPE string",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AlterColumnTypeStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ColumnName: "a",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "ALTER TABLE tech_book ALTER COLUMN a TYPE string",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestInsertStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "INSERT INTO tech_book SELECT * FROM book WHERE type='tech'",
			want: []ast.Node{
				&ast.InsertStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					Select: &ast.SelectStmt{
						FieldList: []ast.ExpressionNode{
							&ast.ColumnNameDef{
								Table:      &ast.TableDef{},
								ColumnName: "*",
							},
						},
						WhereClause: &ast.UnconvertedExpressionDef{},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "INSERT INTO tech_book SELECT * FROM book WHERE type='tech'",
					Line: 1,
				},
			},
		},
		{
			stmt: "INSERT INTO tech_book VALUES(1, 2, 3, 4, 5)",
			want: []ast.Node{
				&ast.InsertStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "INSERT INTO tech_book VALUES(1, 2, 3, 4, 5)",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCopyStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "COPY tech_book FROM '/file/path/in/file/system'",
			want: []ast.Node{
				&ast.CopyStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					FilePath: "/file/path/in/file/system",
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: "COPY tech_book FROM '/file/path/in/file/system'",
					Line: 1,
				},
			},
		},
	}

	runTests(t, tests)
}
