package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
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

func newUnconvertedDataType(name []string, text string) *ast.UnconvertedDataType {
	tp := &ast.UnconvertedDataType{Name: name}
	tp.SetText(text)
	return tp
}

func newExpression(expression ast.ExpressionNode, text string) ast.ExpressionNode {
	expression.SetText(text)
	return expression
}

func TestPGConvertCreateTableStmt(t *testing.T) {
	tests := []testData{
		{
			stmt: `
				CREATE TABLE tech_book(
					a int,
					b int,
					c text COLLATE public."de_DE"
				)
				PARTITION BY RANGE (a)
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
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "b",
							Type:       &ast.Integer{Size: 4},
						},
						{
							ColumnName: "c",
							Type:       &ast.Text{},
							Collation: &ast.CollationNameDef{
								Schema: "public",
								Name:   "de_DE",
							},
						},
					},
					PartitionDef: &ast.UnconvertedStmt{},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `CREATE TABLE tech_book(
					a int,
					b int,
					c text COLLATE public."de_DE"
				)
				PARTITION BY RANGE (a)`,
					LastLine: 7,
				},
			},
			columnLine: [][]int{
				{3, 4, 5},
			},
		},
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
							Type:       newUnconvertedDataType([]string{"user defined data type"}, `"user defined data type"`),
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
										MatchType:  ast.ForeignMatchTypeSimple,
										OnUpdate:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
										OnDelete:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
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
								MatchType:  ast.ForeignMatchTypeSimple,
								OnUpdate:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
								OnDelete:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
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
			stmt: "ALTER TABLE techbook ADD COLUMN IF NOT EXISTS a int",
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
							IfNotExists: true,
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
					Text:     "ALTER TABLE techbook ADD COLUMN IF NOT EXISTS a int",
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
			stmt: "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_id ON tech_book ((id+1) DESC, name)",
			want: []ast.Node{
				&ast.CreateIndexStmt{
					IfNotExists:  true,
					Concurrently: true,
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
					Text:     "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_id ON tech_book ((id+1) DESC, name)",
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
					IfExists: false,
					Behavior: ast.DropBehaviorRestrict,
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
		{
			stmt: "DROP INDEX xschema.idx_id, idx_x restrict",
			want: []ast.Node{
				&ast.DropIndexStmt{
					IfExists: false,
					Behavior: ast.DropBehaviorRestrict,
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
					Text:     "DROP INDEX xschema.idx_id, idx_x restrict",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "DROP INDEX IF EXISTS xschema.idx_id, idx_x cascade",
			want: []ast.Node{
				&ast.DropIndexStmt{
					IfExists: true,
					Behavior: ast.DropBehaviorCascade,
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
					Text:     "DROP INDEX IF EXISTS xschema.idx_id, idx_x cascade",
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
							IfExists:       false,
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
		{
			stmt: "ALTER TABLE tech_book DROP CONSTRAINT IF EXISTS uk_tech_a",
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
							IfExists:       true,
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book DROP CONSTRAINT IF EXISTS uk_tech_a",
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
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id) MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION",
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
									MatchType:  ast.ForeignMatchTypeSimple,
									OnUpdate:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
									OnDelete:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id) MATCH SIMPLE ON UPDATE NO ACTION ON DELETE NO ACTION",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id) MATCH FULL ON UPDATE CASCADE ON DELETE SET DEFAULT",
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
									MatchType:  ast.ForeignMatchTypeFull,
									OnUpdate:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeCascade},
									OnDelete:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeSetDefault},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id) MATCH FULL ON UPDATE CASCADE ON DELETE SET DEFAULT",
					LastLine: 1,
				},
			},
		},
		{
			stmt: "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id) MATCH SIMPLE ON UPDATE RESTRICT ON DELETE SET NULL",
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
									MatchType:  ast.ForeignMatchTypeSimple,
									OnUpdate:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeRestrict},
									OnDelete:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeSetNull},
								},
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book ADD CONSTRAINT fk_tech_book_id FOREIGN KEY (id) REFERENCES people(id) MATCH SIMPLE ON UPDATE RESTRICT ON DELETE SET NULL",
					LastLine: 1,
				},
			},
		},
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
									MatchType:  ast.ForeignMatchTypeSimple,
									OnUpdate:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
									OnDelete:   &ast.ReferentialActionDef{Type: ast.ReferentialActionTypeNoAction},
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
		{
			stmt: "ALTER TABLE ONLY circles ADD CONSTRAINT circles_c_excl EXCLUDE USING gist (c WITH &&, d WITH &&) WHERE (a > 0);",
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "circles",
					},
					AlterItemList: []ast.Node{
						&ast.AddConstraintStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "circles",
							},
							Constraint: &ast.ConstraintDef{
								Type:         ast.ConstraintTypeExclusion,
								Name:         "circles_c_excl",
								Exclusions:   "c WITH &&, d WITH &&",
								AccessMethod: ast.IndexMethodTypeGiST,
								WhereClause:  "a > 0",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE ONLY circles ADD CONSTRAINT circles_c_excl EXCLUDE USING gist (c WITH &&, d WITH &&) WHERE (a > 0);",
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
			stmt: "ALTER TABLE tech_book DROP COLUMN IF EXISTS a CASCADE",
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
							IfExists:   true,
							Behavior:   ast.DropBehaviorCascade,
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     "ALTER TABLE tech_book DROP COLUMN IF EXISTS a CASCADE",
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
			stmt: "SELECT public.t.a, t.*, t1.* FROM (SELECT * FROM t) t, t1 ORDER BY t.a, random()",
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
					OrderByClause: []*ast.ByItemDef{
						{
							Expression: newExpression(&ast.ColumnNameDef{
								Table:      &ast.TableDef{Name: "t"},
								ColumnName: "a",
							}, "t.a"),
						},
						{
							Expression: newExpression(&ast.UnconvertedExpressionDef{}, "random()"),
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
					Text:     "SELECT public.t.a, t.*, t1.* FROM (SELECT * FROM t) t, t1 ORDER BY t.a, random()",
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
			stmt: `ALTER TABLE tech_book ALTER COLUMN a TYPE TEXT COLLATE "en_EN"`,
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
							Type:       &ast.Text{},
							Collation: &ast.CollationNameDef{
								Name: "en_EN",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `ALTER TABLE tech_book ALTER COLUMN a TYPE TEXT COLLATE "en_EN"`,
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
			stmt: "INSERT INTO tech_book(a, b) VALUES (1, 'a'), (2, 'b')",
			want: []ast.Node{
				&ast.InsertStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					ColumnList: []string{
						"a",
						"b",
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
					Text:     "INSERT INTO tech_book(a, b) VALUES (1, 'a'), (2, 'b')",
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

func TestCreateSequence(t *testing.T) {
	var one int32 = 1
	tests := []testData{
		{
			stmt: `CREATE SEQUENCE public.tbl_seq_id_seq
				AS integer
				INCREMENT BY 1
				START WITH 1
				MINVALUE 1
				MAXVALUE 1
				CACHE 1
				CYCLE
				OWNED BY public.tbl.id;`,
			want: []ast.Node{
				&ast.CreateSequenceStmt{
					IfNotExists: false,
					SequenceDef: ast.SequenceDef{
						SequenceName: &ast.SequenceNameDef{
							Schema: "public",
							Name:   "tbl_seq_id_seq",
						},
						SequenceDataType: &ast.Integer{
							Size: 4,
						},
						IncrementBy: &one,
						StartWith:   &one,
						MinValue:    &one,
						MaxValue:    &one,
						Cache:       &one,
						Cycle:       true,
						OwnedBy: &ast.ColumnNameDef{
							Table: &ast.TableDef{
								Type:   ast.TableTypeUnknown,
								Name:   "tbl",
								Schema: "public",
							},
							ColumnName: "id",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `CREATE SEQUENCE public.tbl_seq_id_seq
				AS integer
				INCREMENT BY 1
				START WITH 1
				MINVALUE 1
				MAXVALUE 1
				CACHE 1
				CYCLE
				OWNED BY public.tbl.id;`,
					LastLine: 9,
				},
			},
		},
		{
			stmt: `CREATE SEQUENCE public.tbl_seq_id_seq
				AS bigint
				INCREMENT BY 1
				START WITH 1
				NO MINVALUE
				NO MAXVALUE
				CACHE 1;`,
			want: []ast.Node{
				&ast.CreateSequenceStmt{
					IfNotExists: false,
					SequenceDef: ast.SequenceDef{
						SequenceName: &ast.SequenceNameDef{
							Schema: "public",
							Name:   "tbl_seq_id_seq",
						},
						SequenceDataType: &ast.Integer{
							Size: 8,
						},
						IncrementBy: &one,
						StartWith:   &one,
						Cache:       &one,
						Cycle:       false,
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `CREATE SEQUENCE public.tbl_seq_id_seq
				AS bigint
				INCREMENT BY 1
				START WITH 1
				NO MINVALUE
				NO MAXVALUE
				CACHE 1;`,
					LastLine: 7,
				},
			},
		},
		{
			stmt: `CREATE SEQUENCE IF NOT EXISTS public.tbl_seq_id_seq;`,
			want: []ast.Node{
				&ast.CreateSequenceStmt{
					IfNotExists: true,
					SequenceDef: ast.SequenceDef{
						SequenceName: &ast.SequenceNameDef{
							Schema: "public",
							Name:   "tbl_seq_id_seq",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `CREATE SEQUENCE IF NOT EXISTS public.tbl_seq_id_seq;`,
					LastLine: 1,
				},
			},
		},
	}
	runTests(t, tests)
}

func TestDropSequence(t *testing.T) {
	tests := []testData{
		{
			stmt: `DROP SEQUENCE public.tbl_seq_id_seq;`,
			want: []ast.Node{
				&ast.DropSequenceStmt{
					IfExists: false,
					Behavior: ast.DropBehaviorRestrict,
					SequenceNameList: []*ast.SequenceNameDef{
						{
							Schema: "public",
							Name:   "tbl_seq_id_seq",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `DROP SEQUENCE public.tbl_seq_id_seq;`,
					LastLine: 1,
				},
			},
		},
		{
			stmt: `DROP SEQUENCE IF EXISTS tbl_seq1, public.tbl_seq2 CASCADE;`,
			want: []ast.Node{
				&ast.DropSequenceStmt{
					IfExists: true,
					Behavior: ast.DropBehaviorCascade,
					SequenceNameList: []*ast.SequenceNameDef{
						{
							Name: "tbl_seq1",
						},
						{
							Schema: "public",
							Name:   "tbl_seq2",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `DROP SEQUENCE IF EXISTS tbl_seq1, public.tbl_seq2 CASCADE;`,
					LastLine: 1,
				},
			},
		},
	}
	runTests(t, tests)
}

func TestAlterSequence(t *testing.T) {
	one := int32(1)
	boolFalse := false
	boolTrue := true
	tests := []testData{
		{
			stmt: `
        ALTER SEQUENCE IF EXISTS public.tbl_seq_id_seq
          AS bigint
          INCREMENT BY 1
          NO MINVALUE
          NO MAXVALUE
          START WITH 1
          RESTART WITH 1
          CACHE 1
          NO CYCLE
          OWNED BY NONE;`,
			want: []ast.Node{
				&ast.AlterSequenceStmt{
					IfExists: true,
					Name: &ast.SequenceNameDef{
						Schema: "public",
						Name:   "tbl_seq_id_seq",
					},
					Type: &ast.Integer{
						Size: 8,
					},
					IncrementBy: &one,
					NoMinValue:  true,
					NoMaxValue:  true,
					StartWith:   &one,
					RestartWith: &one,
					Cache:       &one,
					Cycle:       &boolFalse,
					OwnedByNone: true,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `ALTER SEQUENCE IF EXISTS public.tbl_seq_id_seq
          AS bigint
          INCREMENT BY 1
          NO MINVALUE
          NO MAXVALUE
          START WITH 1
          RESTART WITH 1
          CACHE 1
          NO CYCLE
          OWNED BY NONE;`,
					LastLine: 11,
				},
			},
		},
		{
			stmt: `
        ALTER SEQUENCE IF EXISTS public.tbl_seq_id_seq
          MINVALUE 1
          MAXVALUE 1
          CYCLE
          OWNED BY public.tbl.id;`,
			want: []ast.Node{
				&ast.AlterSequenceStmt{
					IfExists: true,
					Name: &ast.SequenceNameDef{
						Schema: "public",
						Name:   "tbl_seq_id_seq",
					},
					MinValue: &one,
					MaxValue: &one,
					Cycle:    &boolTrue,
					OwnedBy: &ast.ColumnNameDef{
						Table: &ast.TableDef{
							Type:   ast.TableTypeUnknown,
							Name:   "tbl",
							Schema: "public",
						},
						ColumnName: "id",
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `ALTER SEQUENCE IF EXISTS public.tbl_seq_id_seq
          MINVALUE 1
          MAXVALUE 1
          CYCLE
          OWNED BY public.tbl.id;`,
					LastLine: 6,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCreateExtension(t *testing.T) {
	tests := []testData{
		{
			stmt: `CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public`,
			want: []ast.Node{
				&ast.CreateExtensionStmt{
					Schema:      "public",
					Name:        "pg_trgm",
					IfNotExists: true,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestDropExtension(t *testing.T) {
	tests := []testData{
		{
			stmt: `DROP EXTENSION IF EXISTS pg_trgm, hstore`,
			want: []ast.Node{
				&ast.DropExtensionStmt{
					IfExists: true,
					NameList: []string{
						"pg_trgm",
						"hstore",
					},
					Behavior: ast.DropBehaviorRestrict,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `DROP EXTENSION IF EXISTS pg_trgm, hstore`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCreateFunction(t *testing.T) {
	tests := []testData{
		{
			stmt: `Create function get_car_Price("Price_from" int, Price_to int)  
			returns int  
			language plpgsql  
			as  
			$$  
			Declare  
			 Car_count integer;  
			Begin  
			   select count(*)   
			   into Car_count  
			   from Car  
			   where Car_price between "Price_from" and Price_to;  
			   return Car_count;  
			End;  
			$$;`,
			want: []ast.Node{
				&ast.CreateFunctionStmt{
					Function: &ast.FunctionDef{
						Schema: "",
						Name:   "get_car_price",
						ParameterList: []*ast.FunctionParameterDef{
							{
								Name: "Price_from",
								Type: &ast.Integer{Size: 4},
								Mode: ast.FunctionParameterModeDefault,
							},
							{
								Name: "price_to",
								Type: &ast.Integer{Size: 4},
								Mode: ast.FunctionParameterModeDefault,
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `Create function get_car_Price("Price_from" int, Price_to int)  
			returns int  
			language plpgsql  
			as  
			$$  
			Declare  
			 Car_count integer;  
			Begin  
			   select count(*)   
			   into Car_count  
			   from Car  
			   where Car_price between "Price_from" and Price_to;  
			   return Car_count;  
			End;  
			$$;`,
					LastLine: 15,
				},
			},
		},
		{
			stmt: `CREATE FUNCTION public.trigger_update_updated_ts() RETURNS trigger
			LANGUAGE plpgsql
			AS $$
		BEGIN
		  NEW.updated_ts = extract(epoch from now());
		  RETURN NEW;
		END;
		$$;`,
			want: []ast.Node{
				&ast.CreateFunctionStmt{
					Function: &ast.FunctionDef{
						Schema: "public",
						Name:   "trigger_update_updated_ts",
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text: `CREATE FUNCTION public.trigger_update_updated_ts() RETURNS trigger
			LANGUAGE plpgsql
			AS $$
		BEGIN
		  NEW.updated_ts = extract(epoch from now());
		  RETURN NEW;
		END;
		$$;`,
					LastLine: 8,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestDropFunction(t *testing.T) {
	tests := []testData{
		{
			// pg_query_go will skip the OUT parameters and only parse the parameter data type.
			stmt: `DROP FUNCTION IF EXISTS public.func1(INOUT "Price_from" int, IN price_to int, OUT out_item int), func2()`,
			want: []ast.Node{
				&ast.DropFunctionStmt{
					IfExists: true,
					FunctionList: []*ast.FunctionDef{
						{
							Schema: "public",
							Name:   "func1",
							ParameterList: []*ast.FunctionParameterDef{
								{
									Type: &ast.Integer{Size: 4},
								},
								{
									Type: &ast.Integer{Size: 4},
								},
							},
						},
						{
							Schema: "",
							Name:   "func2",
						},
					},
					Behavior: ast.DropBehaviorRestrict,
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `DROP FUNCTION IF EXISTS public.func1(INOUT "Price_from" int, IN price_to int, OUT out_item int), func2()`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCreateTrigger(t *testing.T) {
	tests := []testData{
		{
			stmt: `CREATE TRIGGER update_principal_updated_ts BEFORE UPDATE ON public.principal FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();`,
			want: []ast.Node{
				&ast.CreateTriggerStmt{
					Trigger: &ast.TriggerDef{
						Name: "update_principal_updated_ts",
						Table: &ast.TableDef{
							Type:   ast.TableTypeUnknown,
							Schema: "public",
							Name:   "principal",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `CREATE TRIGGER update_principal_updated_ts BEFORE UPDATE ON public.principal FOR EACH ROW EXECUTE FUNCTION public.trigger_update_updated_ts();`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestDropTrigger(t *testing.T) {
	tests := []testData{
		{
			stmt: `DROP TRIGGER update_ts ON public.principal`,
			want: []ast.Node{
				&ast.DropTriggerStmt{
					IfExists: false,
					Behavior: ast.DropBehaviorRestrict,
					Trigger: &ast.TriggerDef{
						Name: "update_ts",
						Table: &ast.TableDef{
							Type:   ast.TableTypeUnknown,
							Schema: "public",
							Name:   "principal",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `DROP TRIGGER update_ts ON public.principal`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCreateType(t *testing.T) {
	tests := []testData{
		{
			stmt: `CREATE TYPE public.bug_status AS ENUM ('new', 'open', 'closed');`,
			want: []ast.Node{
				&ast.CreateTypeStmt{
					Type: &ast.EnumTypeDef{
						Name: &ast.TypeNameDef{
							Schema: "public",
							Name:   "bug_status",
						},
						LabelList: []string{"new", "open", "closed"},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `CREATE TYPE public.bug_status AS ENUM ('new', 'open', 'closed');`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestDropType(t *testing.T) {
	tests := []testData{
		{
			stmt: `DROP TYPE public.bug_status, tp1`,
			want: []ast.Node{
				&ast.DropTypeStmt{
					IfExists: false,
					Behavior: ast.DropBehaviorRestrict,
					TypeNameList: []*ast.TypeNameDef{
						{
							Schema: "public",
							Name:   "bug_status",
						},
						{
							Schema: "",
							Name:   "tp1",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `DROP TYPE public.bug_status, tp1`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestAlterType(t *testing.T) {
	tests := []testData{
		{
			stmt: `ALTER TYPE public.bug_status ADD VALUE 'a' BEFORE 'b'`,
			want: []ast.Node{
				&ast.AlterTypeStmt{
					Type: &ast.TypeNameDef{
						Schema: "public",
						Name:   "bug_status",
					},
					AlterItemList: []ast.Node{
						&ast.AddEnumLabelStmt{
							EnumType: &ast.TypeNameDef{
								Schema: "public",
								Name:   "bug_status",
							},
							NewLabel:      "a",
							Position:      ast.PositionTypeBefore,
							NeighborLabel: "b",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{

					Text:     `ALTER TYPE public.bug_status ADD VALUE 'a' BEFORE 'b'`,
					LastLine: 1,
				},
			},
		},
		{
			stmt: `ALTER TYPE public.bug_status ADD VALUE 'a' AFTER 'b'`,
			want: []ast.Node{
				&ast.AlterTypeStmt{
					Type: &ast.TypeNameDef{
						Schema: "public",
						Name:   "bug_status",
					},
					AlterItemList: []ast.Node{
						&ast.AddEnumLabelStmt{
							EnumType: &ast.TypeNameDef{
								Schema: "public",
								Name:   "bug_status",
							},
							NewLabel:      "a",
							Position:      ast.PositionTypeAfter,
							NeighborLabel: "b",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{

					Text:     `ALTER TYPE public.bug_status ADD VALUE 'a' AFTER 'b'`,
					LastLine: 1,
				},
			},
		},
		{
			stmt: `ALTER TYPE public.bug_status ADD VALUE 'a'`,
			want: []ast.Node{
				&ast.AlterTypeStmt{
					Type: &ast.TypeNameDef{
						Schema: "public",
						Name:   "bug_status",
					},
					AlterItemList: []ast.Node{
						&ast.AddEnumLabelStmt{
							EnumType: &ast.TypeNameDef{
								Schema: "public",
								Name:   "bug_status",
							},
							NewLabel:      "a",
							Position:      ast.PositionTypeEnd,
							NeighborLabel: "",
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{

					Text:     `ALTER TYPE public.bug_status ADD VALUE 'a'`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestAttachPartition(t *testing.T) {
	tests := []testData{
		{
			stmt: `ALTER TABLE tech_book ATTACH PARTITION p1 DEFAULT`,
			want: []ast.Node{
				&ast.AlterTableStmt{
					Table: &ast.TableDef{
						Type: ast.TableTypeBaseTable,
						Name: "tech_book",
					},
					AlterItemList: []ast.Node{
						&ast.AttachPartitionStmt{
							Table: &ast.TableDef{
								Type: ast.TableTypeBaseTable,
								Name: "tech_book",
							},
						},
					},
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `ALTER TABLE tech_book ATTACH PARTITION p1 DEFAULT`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestCommit(t *testing.T) {
	tests := []testData{
		{
			stmt: `COMMIT;`,
			want: []ast.Node{
				&ast.CommitStmt{},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `COMMIT;`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}

func TestRenameSchema(t *testing.T) {
	tests := []testData{
		{
			stmt: `ALTER SCHEMA s1 RENAME TO s2`,
			want: []ast.Node{
				&ast.RenameSchemaStmt{
					Schema:  "s1",
					NewName: "s2",
				},
			},
			statementList: []parser.SingleSQL{
				{
					Text:     `ALTER SCHEMA s1 RENAME TO s2`,
					LastLine: 1,
				},
			},
		},
	}

	runTests(t, tests)
}
