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
		if !equalFlag || !equalTableDef || !equalColumnList {
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

func equalColumnDef(expected *ast.ColumnDef, actual *ast.ColumnDef) bool {
	return (expected.ColumnName == actual.ColumnName)
}
