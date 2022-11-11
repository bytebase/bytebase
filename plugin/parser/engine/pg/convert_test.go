package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
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
		res, err := p.Parse(parser.ParseContext{}, test.stmt)
		require.NoError(t, err)
		for i := range test.want {
			test.want[i].SetText(test.statementList[i].Text)
			test.want[i].SetLastLine(test.statementList[i].LastLine)

			switch n := test.want[i].(type) {
			case *ast.CreateTableStmt:
				for j, col := range n.ColumnList {
					col.SetLastLine(test.columnLine[i][j])
					for _, inlineCons := range col.ConstraintList {
						inlineCons.SetLastLine(col.LastLine())
					}
				}
				for j, cons := range n.ConstraintList {
					cons.SetLastLine(test.constraintLine[i][j])
				}
			case *ast.AlterTableStmt:
				for _, item := range n.AlterItemList {
					item.SetLastLine(n.LastLine())
				}
			}
		}
		require.Equal(t, test.want, res, test.stmt)
	}
}

func TestPGConvertCreateTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: `
				CREATE TABLE tech_book(
					a char(20),
					b character(30),
					c varchar(330),
					d character varying(400),
					e text
				)
			`,
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: false,
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "a",
							Type:       &ast.Character{Size: 20},
						},
						{
							ColumnName: "b",
							Type:       &ast.Character{Size: 30},
						},
						{
							ColumnName: "c",
							Type:       &ast.CharacterVarying{Size: 330},
						},
						{
							ColumnName: "d",
							Type:       &ast.CharacterVarying{Size: 400},
						},
						{
							ColumnName: "e",
							Type:       &ast.Text{},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `CREATE TABLE tech_book(
					a char(20),
					b character(30),
					c varchar(330),
					d character varying(400),
					e text
				)`,
					LastLine: 8,
				},
			},
			columnLine: [][]int{
				{3, 4, 5, 6, 7},
			},
		},
		{
			stmt: `CREATE TABLE tech_book(
				a smallint,
				b integer,
				c bigint,
				d decimal(10, 2),
				e numeric(4),
				f real,
				g double precision,
				h smallserial,
				i serial,
				j bigserial,
				k int8,
				l serial8,
				m float8,
				n int,
				o int4,
				p float4,
				q int2,
				r serial2,
				s serial4,
				t decimal,
				u "user defined data type")`,
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: false,
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "a",
							Type:       &ast.Integer{Size: 2},
						},
						{
							ColumnName: "b",
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "c",
							Type:       &ast.Integer{Size: 8},
						},
						{
							ColumnName: "d",
							Type:       &ast.Decimal{Precision: 10, Scale: 2},
						},
						{
							ColumnName: "e",
							Type:       &ast.Decimal{Precision: 4, Scale: 0},
						},
						{
							ColumnName: "f",
							Type:       &ast.Float{Size: 4},
						},
						{
							ColumnName: "g",
							Type:       &ast.Float{Size: 8},
						},
						{
							ColumnName: "h",
							Type:       &ast.Serial{Size: 2},
						},
						{
							ColumnName: "i",
							Type:       &ast.Serial{Size: 4},
						},
						{
							ColumnName: "j",
							Type:       &ast.Serial{Size: 8},
						},
						{
							ColumnName: "k",
							Type:       &ast.Integer{Size: 8},
						},
						{
							ColumnName: "l",
							Type:       &ast.Serial{Size: 8},
						},
						{
							ColumnName: "m",
							Type:       &ast.Float{Size: 8},
						},
						{
							ColumnName: "n",
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "o",
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "p",
							Type:       &ast.Float{Size: 4},
						},
						{
							ColumnName: "q",
							Type:       &ast.Integer{Size: 2},
						},
						{
							ColumnName: "r",
							Type:       &ast.Serial{Size: 2},
						},
						{
							ColumnName: "s",
							Type:       &ast.Serial{Size: 4},
						},
						{
							ColumnName: "t",
							Type:       &ast.Decimal{Precision: 0, Scale: 0},
						},
						{
							ColumnName: "u",
							Type:       &ast.UnconvertedDataType{Name: []string{"user defined data type"}},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `CREATE TABLE tech_book(
				a smallint,
				b integer,
				c bigint,
				d decimal(10, 2),
				e numeric(4),
				f real,
				g double precision,
				h smallserial,
				i serial,
				j bigserial,
				k int8,
				l serial8,
				m float8,
				n int,
				o int4,
				p float4,
				q int2,
				r serial2,
				s serial4,
				t decimal,
				u "user defined data type")`,
					LastLine: 22,
				},
			},
			columnLine: [][]int{
				{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22},
			},
		},
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
							Type:       &ast.Integer{Size: 4},
							ConstraintList: []*ast.ConstraintDef{
								{
									Type:    ast.ConstraintTypeNotNull,
									KeyList: []string{"a"},
								},
							},
						},
						{
							ColumnName: "b",
							Type:       &ast.Integer{Size: 4},
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
					Text:     "CREATE TABLE \"techBook\" (a int NOT NULL, b int CONSTRAINT b_not_null NOT NULL)",
					LastLine: 1,
				},
			},
			columnLine: [][]int{
				{1, 1},
			},
		},
		{
			stmt: "CREATE TABLE IF NOT EXISTS techBook (\"A\" int, b int DEFAULT 1+2+3-4+5)",
			want: []ast.Node{
				&ast.CreateTableStmt{
					IfNotExists: true,
					Name: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "techbook",
					},
					ColumnList: []*ast.ColumnDef{
						{
							ColumnName: "A",
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "b",
							Type:       &ast.Integer{Size: 4},
							ConstraintList: []*ast.ConstraintDef{
								{
									Type:       ast.ConstraintTypeDefault,
									KeyList:    []string{"b"},
									Expression: expressionWithText(&ast.UnconvertedExpressionDef{}, "(((1 + 2) + 3) - 4) + 5"),
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE TABLE IF NOT EXISTS techBook (\"A\" int, b int DEFAULT 1+2+3-4+5)",
					LastLine: 1,
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
							Type:       &ast.Integer{Size: 4},
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
					Text:     "CREATE TABLE tech_book(a INT CONSTRAINT t_pk_a PRIMARY KEY)",
					LastLine: 1,
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
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "b",
							Type:       &ast.Integer{Size: 4},
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
					Text:     "CREATE TABLE tech_book(a INT, b int CONSTRAINT uk_b UNIQUE, CONSTRAINT t_pk_a PRIMARY KEY(a))",
					LastLine: 1,
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
							Type:       &ast.Integer{Size: 4},
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
					Text:     "CREATE TABLE tech_book(a INT CONSTRAINT fk_a REFERENCES people(id), CONSTRAINT fk_a_people_b FOREIGN KEY (a) REFERENCES people(b))",
					LastLine: 1,
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
								{
									ColumnName: "a",
									Type:       &ast.Integer{Size: 4},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE techbook ADD COLUMN a int",
					LastLine: 1,
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
									Type:       &ast.Integer{Size: 4},
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
					Text:     "ALTER TABLE techbook ADD COLUMN a int CONSTRAINT uk_techbook_a UNIQUE",
					LastLine: 1,
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
					Text:     "ALTER TABLE techbook RENAME TO \"techBook\"",
					LastLine: 1,
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
					Text:     "ALTER VIEW techbook RENAME TO \"techBook\"",
					LastLine: 1,
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
					Text:     "ALTER TABLE techbook RENAME abc TO \"ABC\"",
					LastLine: 1,
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
					Text:     "ALTER VIEW techbook RENAME abc TO \"ABC\"",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book RENAME CONSTRAINT uk_tech_a to \"UK_TECH_A\"",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGCreateIndexStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "CREATE INDEX IF NOT EXISTS idx_id ON tech_book ((id+1) DESC, name)",
			want: []ast.Node{
				&ast.CreateIndexStmt{
					IfNotExists: true,
					Index: &ast.IndexDef{
						Name:   "idx_id",
						Table:  &ast.TableDef{Name: "tech_book"},
						Unique: false,
						KeyList: []*ast.IndexKeyDef{
							{
								Type:      ast.IndexKeyTypeExpression,
								Key:       "id + 1",
								SortOrder: ast.SortOrderTypeDescending,
								NullOrder: ast.NullOrderTypeDefault,
							},
							{
								Type:      ast.IndexKeyTypeColumn,
								Key:       "name",
								SortOrder: ast.NullOrderTypeDefault,
								NullOrder: ast.NullOrderTypeDefault,
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE INDEX IF NOT EXISTS idx_id ON tech_book ((id+1) DESC, name)",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "CREATE INDEX idx_id ON tech_book (id ASC NULLS FIRST)",
			want: []ast.Node{
				&ast.CreateIndexStmt{
					Index: &ast.IndexDef{
						Name:   "idx_id",
						Table:  &ast.TableDef{Name: "tech_book"},
						Unique: false,
						KeyList: []*ast.IndexKeyDef{
							{
								Type:      ast.IndexKeyTypeColumn,
								Key:       "id",
								SortOrder: ast.SortOrderTypeAscending,
								NullOrder: ast.NullOrderTypeFirst,
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE INDEX idx_id ON tech_book (id ASC NULLS FIRST)",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "CREATE UNIQUE INDEX idx_id ON tech_book (id NULLS LAST)",
			want: []ast.Node{
				&ast.CreateIndexStmt{
					Index: &ast.IndexDef{
						Name:   "idx_id",
						Table:  &ast.TableDef{Name: "tech_book"},
						Unique: true,
						KeyList: []*ast.IndexKeyDef{
							{
								Type:      ast.IndexKeyTypeColumn,
								Key:       "id",
								SortOrder: ast.NullOrderTypeDefault,
								NullOrder: ast.NullOrderTypeLast,
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE UNIQUE INDEX idx_id ON tech_book (id NULLS LAST)",
					LastLine: 1,
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
					Text:     "DROP INDEX xschema.idx_id, idx_x",
					LastLine: 1,
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
					Text:     "ALTER INDEX xschema.idx_id RENAME TO \"IDX_ID\"",
					LastLine: 1,
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
					Text:     "ALTER INDEX idx_id RENAME TO \"IDX_ID\"",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book DROP CONSTRAINT uk_tech_a",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func expressionWithText(expression ast.ExpressionNode, text string) ast.ExpressionNode {
	expression.SetText(text)
	return expression
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
								Type:           ast.ConstraintTypeCheck,
								Name:           "check_a_bigger_than_b",
								SkipValidation: true,
								Expression:     expressionWithText(&ast.UnconvertedExpressionDef{}, "a > b"),
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT check_a_bigger_than_b CHECK (a > b) NOT VALID",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE (id)",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE ONLY s.person ADD CONSTRAINT person_email_email1_key UNIQUE (email) INCLUDE (email) USING INDEX TABLESPACE demo_table_space;",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type:   ast.TableTypeBaseTable,
						Schema: "s",
						Name:   "person",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type:   ast.TableTypeBaseTable,
								Schema: "s",
								Name:   "person",
							},
							Constraint: &ast.ConstraintDef{
								Type:            ast.ConstraintTypeUnique,
								Name:            "person_email_email1_key",
								KeyList:         []string{"email"},
								Including:       []string{"email"},
								IndexTableSpace: "demo_table_space",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE ONLY s.person ADD CONSTRAINT person_email_email1_key UNIQUE (email) INCLUDE (email) USING INDEX TABLESPACE demo_table_space;",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY (id)",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id)",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT uk_tech_book_id UNIQUE USING INDEX uk_id",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT pk_tech_book_id PRIMARY KEY USING INDEX pk_id",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book DROP COLUMN a",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestPGDropTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "DROP TABLE IF EXISTS tech_book, xschema.user CASCADE",
			want: []ast.Node{
				&ast.DropTableStmt{
					IfExists: true,
					Behavior: ast.DropBehaviorCascade,
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
					Text:     "DROP TABLE IF EXISTS tech_book, xschema.user CASCADE",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "DROP VIEW tech_book, xschema.user RESTRICT",
			want: []ast.Node{
				&ast.DropTableStmt{
					Behavior: ast.DropBehaviorRestrict,
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
					Text:     "DROP VIEW tech_book, xschema.user RESTRICT",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ALTER COLUMN id SET NOT NULL",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book ALTER COLUMN id DROP NOT NULL",
					LastLine: 1,
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
					Text:     "SELECT public.t.a, t.*, t1.* FROM (SELECT * FROM t) t, t1",
					LastLine: 1,
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
					Text:     "SELECT public.t.a, t.*, * FROM t",
					LastLine: 1,
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
					LastLine: 13,
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
					Text:     "DROP DATABASE test",
					LastLine: 1,
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
					Text:     "DROP DATABASE IF EXISTS test",
					LastLine: 1,
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
					Text:     "UPDATE tech_book SET a = 1 FROM (SELECT * FROM t) t WHERE a > 1",
					LastLine: 1,
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
					Text:     "UPDATE tech_book SET a = 1",
					LastLine: 1,
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
					Text:     "UPDATE tech_book SET a = 1 WHERE a > 1",
					LastLine: 1,
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
					Text:     "DELETE FROM tech_book",
					LastLine: 1,
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
					Text:     "DELETE FROM tech_book WHERE a > 1",
					LastLine: 1,
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
					Text:     "ALTER TABLE tech_book SET SCHEMA new_schema",
					LastLine: 1,
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
					Text:     "ALTER VIEW tech_book SET SCHEMA new_schema",
					LastLine: 1,
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
					Text:     "EXPLAIN SELECT * FROM tech_book",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestAlterColumnType(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book ALTER COLUMN a TYPE int",
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
							Type: &ast.Integer{
								Size: 4,
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book ALTER COLUMN a TYPE int",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestInsertStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "INSERT INTO tech_book VALUES (1, 'a'), (2, 'b')",
			want: []ast.Node{
				&ast.InsertStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					ValueList: [][]ast.ExpressionNode{
						{
							&ast.UnconvertedExpressionDef{},
							&ast.StringDef{
								Value: "a",
							},
						},
						{
							&ast.UnconvertedExpressionDef{},
							&ast.StringDef{
								Value: "b",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "INSERT INTO tech_book VALUES (1, 'a'), (2, 'b')",
					LastLine: 1,
				},
			},
		},
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
					Text:     "INSERT INTO tech_book SELECT * FROM book WHERE type='tech'",
					LastLine: 1,
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
					ValueList: [][]ast.ExpressionNode{
						{
							&ast.UnconvertedExpressionDef{},
							&ast.UnconvertedExpressionDef{},
							&ast.UnconvertedExpressionDef{},
							&ast.UnconvertedExpressionDef{},
							&ast.UnconvertedExpressionDef{},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "INSERT INTO tech_book VALUES(1, 2, 3, 4, 5)",
					LastLine: 1,
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
					Text:     "COPY tech_book FROM '/file/path/in/file/system'",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestUnconvertStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "SHOW TABLES",
			want: []ast.Node{&ast.UnconvertedStmt{}},
			statementList: []parser.SingleSQL{
				{
					Text:     "SHOW TABLES",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCommentStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: "COMMENT ON TABLE tech_book IS 'This is a comment.'",
			want: []ast.Node{&ast.CommentStmt{
				Comment: "This is a comment.",
			}},
			statementList: []parser.SingleSQL{
				{
					Text:     "COMMENT ON TABLE tech_book IS 'This is a comment.'",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCreateDatabase(t *testing.T) {
	tests := []testData{
		{
			stmt: "CREATE DATABASE db1 ENCODING = 'UTF8'",
			want: []ast.Node{&ast.CreateDatabaseStmt{
				Name: "db1",
				OptionList: []*ast.DatabaseOptionDef{
					{
						Type:  ast.DatabaseOptionEncoding,
						Value: "UTF8",
					},
				},
			}},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE DATABASE db1 ENCODING = 'UTF8'",
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCreateSchema(t *testing.T) {
	tests := []testData{
		{
			stmt: "CREATE SCHEMA myschema",
			want: []ast.Node{&ast.CreateSchemaStmt{
				Name:        "myschema",
				IfNotExists: false,
			}},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE SCHEMA myschema",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "CREATE SCHEMA myschema AUTHORIZATION joe;",
			want: []ast.Node{&ast.CreateSchemaStmt{
				Name:        "myschema",
				IfNotExists: false,
				RoleSpec:    &ast.RoleSpec{Type: ast.RoleSpecTypeUser, Value: "joe"},
			}},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE SCHEMA myschema AUTHORIZATION joe;",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "CREATE SCHEMA IF NOT EXISTS myschema AUTHORIZATION joe;",
			want: []ast.Node{&ast.CreateSchemaStmt{
				Name:        "myschema",
				IfNotExists: true,
				RoleSpec:    &ast.RoleSpec{Type: ast.RoleSpecTypeUser, Value: "joe"},
			}},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE SCHEMA IF NOT EXISTS myschema AUTHORIZATION joe;",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "CREATE SCHEMA myschema CREATE TABLE tbl (id INT)",
			want: []ast.Node{&ast.CreateSchemaStmt{
				Name:        "myschema",
				IfNotExists: false,
				SchemaElementList: []ast.Node{
					&ast.CreateTableStmt{
						IfNotExists: false,
						Name: &ast.TableDef{
							Type: ast.TableTypeBaseTable,
							Name: "tbl",
						},
						ColumnList: []*ast.ColumnDef{
							{
								ColumnName: "id",
								Type:       &ast.Integer{Size: 4},
							},
						},
					},
				},
			}},
			statementList: []parser.SingleSQL{
				{
					Text:     "CREATE SCHEMA myschema CREATE TABLE tbl (id INT)",
					LastLine: 1,
				},
			},
		},
	}
	runTests(t, tests)
}

func TestDropSchema(t *testing.T) {
	tests := []testData{
		{
			stmt: "DROP SCHEMA s1",
			want: []ast.Node{
				&ast.DropSchemaStmt{
					IfExists:   false,
					SchemaList: []string{"s1"},
					Behavior:   ast.DropBehaviorRestrict,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "DROP SCHEMA s1",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "DROP SCHEMA s1, s2 CASCADE",
			want: []ast.Node{
				&ast.DropSchemaStmt{
					IfExists:   false,
					SchemaList: []string{"s1", "s2"},
					Behavior:   ast.DropBehaviorCascade,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "DROP SCHEMA s1, s2 CASCADE",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "DROP SCHEMA IF EXISTS s1, s2 RESTRICT",
			want: []ast.Node{
				&ast.DropSchemaStmt{
					IfExists:   true,
					SchemaList: []string{"s1", "s2"},
					Behavior:   ast.DropBehaviorRestrict,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "DROP SCHEMA IF EXISTS s1, s2 RESTRICT",
					LastLine: 1,
				},
			},
		},
	}
	runTests(t, tests)
}

func TestAlterColumnDefault(t *testing.T) {
	tests := []testData{
		{
			stmt: "ALTER TABLE tech_book ALTER COLUMN a DROP DEFAULT",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.DropDefaultStmt{
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
					Text:     "ALTER TABLE tech_book ALTER COLUMN a DROP DEFAULT",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ALTER COLUMN a SET DEFAULT 1+2+3",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.SetDefaultStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
							ColumnName: "a",
							Expression: expressionWithText(&ast.UnconvertedExpressionDef{}, "(1 + 2) + 3"),
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book ALTER COLUMN a SET DEFAULT 1+2+3",
					LastLine: 1,
				},
			},
		},
	}
	runTests(t, tests)
}
