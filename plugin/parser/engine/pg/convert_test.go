package pg

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
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
									Reference: &ast.ReferenceDef{
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
							Reference: &ast.ReferenceDef{
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
		require.True(t, equalCreateTableStmt(test.want, res))
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
		require.True(t, equalAlterTableStmt(test.want, res))
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
		require.True(t, equalRenameTableStmt(test.want, res))
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
					Column: &ast.ColumnDef{
						ColumnName: "abc",
					},
					NewName: "ABC",
				},
			},
		},
	}

	p := &PostgreSQLParser{}

	for _, test := range tests {
		res, err := p.Parse(parser.Context{}, test.stmt)
		require.NoError(t, err)
		require.True(t, equalRenameColumnStmt(test.want, res))
	}
}

func equalRenameColumnStmt(expected []ast.Node, actual []ast.Node) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		exp, ok := expected[i].(*ast.RenameColumnStmt)
		if !ok {
			return false
		}
		act, ok := actual[i].(*ast.RenameColumnStmt)
		if !ok {
			return false
		}

		equalTableDef := equalTableDef(exp.Table, act.Table)
		equalColumnDef := equalColumnDef(exp.Column, act.Column)
		equalNewName := (exp.NewName == act.NewName)
		if !equalTableDef || !equalColumnDef || !equalNewName {
			return false
		}
	}

	return true
}

func equalRenameTableStmt(expected []ast.Node, actual []ast.Node) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		exp, ok := expected[i].(*ast.RenameTableStmt)
		if !ok {
			return false
		}
		act, ok := actual[i].(*ast.RenameTableStmt)
		if !ok {
			return false
		}

		equalTableDef := equalTableDef(exp.Table, act.Table)
		equalNewName := (exp.NewName == act.NewName)
		if !equalTableDef || !equalNewName {
			return false
		}
	}

	return true
}

func equalAlterTableStmt(expected []ast.Node, actual []ast.Node) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		exp, ok := expected[i].(*ast.AlterTableStmt)
		if !ok {
			return false
		}
		act, ok := actual[i].(*ast.AlterTableStmt)
		if !ok {
			return false
		}

		equalTableDef := equalTableDef(exp.Table, act.Table)
		equalAlterAction := equalAlterAction(exp.AlterItemList, act.AlterItemList)

		if !equalTableDef || !equalAlterAction {
			return false
		}
	}

	return true
}

func equalAlterAction(expected []ast.Node, actual []ast.Node) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		if exp, ok := expected[i].(*ast.AddColumnListStmt); ok {
			act, ok := actual[i].(*ast.AddColumnListStmt)
			if !ok {
				return false
			}
			if !equalAddColumnStmt(exp, act) {
				return false
			}
		}
	}

	return true
}

func equalAddColumnStmt(expected *ast.AddColumnListStmt, actual *ast.AddColumnListStmt) bool {
	equalTableDef := equalTableDef(expected.Table, actual.Table)
	equalColumnList := equalColumnList(expected.ColumnList, actual.ColumnList)
	return equalTableDef && equalColumnList
}

func equalCreateTableStmt(expected []ast.Node, actual []ast.Node) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		exp, ok := expected[i].(*ast.CreateTableStmt)
		if !ok {
			return false
		}
		act, ok := actual[i].(*ast.CreateTableStmt)
		if !ok {
			return false
		}

		equalFlag := (exp.IfNotExists == act.IfNotExists)
		equalTableDef := equalTableDef(exp.Name, act.Name)
		equalColumnList := equalColumnList(exp.ColumnList, act.ColumnList)
		equalConstraintList := equalConstraintList(exp.ConstraintList, act.ConstraintList)
		if !equalFlag || !equalTableDef || !equalColumnList || !equalConstraintList {
			return false
		}
	}
	return true
}

func equalTableDef(expected *ast.TableDef, actual *ast.TableDef) bool {
	equalDatabase := (expected.Database == actual.Database)
	equalSchema := (expected.Schema == actual.Schema)
	equalName := (expected.Name == actual.Name)
	return equalDatabase && equalSchema && equalName
}

func equalColumnList(expected []*ast.ColumnDef, actual []*ast.ColumnDef) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		if !equalColumnDef(expected[i], actual[i]) {
			return false
		}
	}

	return true
}

func equalStringList(expected []string, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}

	return true
}

func equalReference(expected *ast.ReferenceDef, actual *ast.ReferenceDef) bool {
	if expected == nil && actual == nil {
		return true
	}

	if expected == nil || actual == nil {
		return false
	}

	equalTable := equalTableDef(expected.Table, actual.Table)
	equalColumnList := equalStringList(expected.ColumnList, actual.ColumnList)

	return equalTable && equalColumnList
}

func equalConstraint(expected *ast.ConstraintDef, actual *ast.ConstraintDef) bool {
	equalType := (expected.Type == actual.Type)
	equalName := (expected.Name == actual.Name)
	equalKey := equalStringList(expected.KeyList, actual.KeyList)
	equalReference := equalReference(expected.Reference, actual.Reference)

	return equalType && equalName && equalKey && equalReference
}

func equalConstraintList(expected []*ast.ConstraintDef, actual []*ast.ConstraintDef) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i := range expected {
		if !equalConstraint(expected[i], actual[i]) {
			return false
		}
	}

	return true
}

func equalColumnDef(expected *ast.ColumnDef, actual *ast.ColumnDef) bool {
	equalName := (expected.ColumnName == actual.ColumnName)
	equalConstraintList := equalConstraintList(expected.ConstraintList, actual.ConstraintList)

	return equalName && equalConstraintList
}
