package pg

import (
	"github.com/pkg/errors"

	pgquery "github.com/pganalyze/pg_query_go/v5"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
}

func extractChangedResources(database string, schema string, dbSchema *model.DBSchema, asts any, statement string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	changedResources := model.NewChangedResources(dbSchema)
	dmlCount := 0
	insertCount := 0
	var sampleDMLs []string
	for _, node := range nodes {
		// schema is "public" by default.
		err := getResourceChanges(database, schema, node, statement, changedResources)
		if err != nil {
			return nil, err
		}

		switch node := node.(type) {
		case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
			if node, ok := node.(*ast.InsertStmt); ok && len(node.ValueList) > 0 {
				insertCount += len(node.ValueList)
				continue
			}
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, node.Text())
			}
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         dmlCount,
		SampleDMLS:       sampleDMLs,
		InsertCount:      insertCount,
	}, nil
}

func getResourceChanges(database, schema string, node ast.Node, statement string, changedResources *model.ChangedResources) error {
	switch node := node.(type) {
	case *ast.CreateTableStmt:
		if node.Name.Type == ast.TableTypeBaseTable {
			d, s, table := node.Name.Database, node.Name.Schema, node.Name.Name
			if d == "" {
				d = database
			}
			if s == "" {
				s = schema
			}
			changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
				false,
			)
		}
	case *ast.DropTableStmt:
		for _, table := range node.TableList {
			if table.Type == ast.TableTypeView {
				d, s, v := table.Database, table.Schema, table.Name
				if d == "" {
					d = database
				}
				if s == "" {
					s = schema
				}
				changedResources.AddView(
					d,
					s,
					&storepb.ChangedResourceView{
						Name:   v,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
				)
			} else {
				d, s, table := table.Database, table.Schema, table.Name
				if d == "" {
					d = database
				}
				if s == "" {
					s = schema
				}
				changedResources.AddTable(
					d,
					s,
					&storepb.ChangedResourceTable{
						Name:   table,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					true,
				)
			}
		}
	case *ast.AlterTableStmt:
		if node.Table.Type == ast.TableTypeBaseTable {
			d, s, table := node.Table.Database, node.Table.Schema, node.Table.Name
			if d == "" {
				d = database
			}
			if s == "" {
				s = schema
			}
			changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
				true,
			)

			for _, item := range node.AlterItemList {
				if v, ok := item.(*ast.RenameTableStmt); ok {
					d, s, table := node.Table.Database, node.Table.Schema, v.NewName
					if d == "" {
						d = database
					}
					if s == "" {
						s = schema
					}
					changedResources.AddTable(
						d,
						s,
						&storepb.ChangedResourceTable{
							Name:   table,
							Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
						},
						false,
					)
					// Only one rename table statement is allowed in a single alter table statement.
					break
				}
			}
		}
	case *ast.CreateIndexStmt:
		d, s, table := node.Index.Table.Database, node.Index.Table.Schema, node.Index.Table.Name
		if d == "" {
			d = database
		}
		if s == "" {
			s = schema
		}
		changedResources.AddTable(
			d,
			s,
			&storepb.ChangedResourceTable{
				Name:   table,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
			false,
		)
	case *ast.DropIndexStmt:
		for _, index := range node.IndexList {
			d, s, table := index.Table.Database, index.Table.Schema, index.Table.Name
			if d == "" {
				d = database
			}
			if s == "" {
				s = schema
			}
			changedResources.AddTable(
				d,
				s,
				&storepb.ChangedResourceTable{
					Name:   table,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
				false,
			)
		}
	case *ast.CreateViewStmt:
		d, s, view := node.Name.Database, node.Name.Schema, node.Name.Name
		if d == "" {
			d = database
		}
		if s == "" {
			s = schema
		}
		changedResources.AddView(
			d,
			s,
			&storepb.ChangedResourceView{
				Name:   view,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
		)
	case *ast.CreateFunctionStmt:
		s, f := node.Function.Schema, node.Function.Name
		if s == "" {
			s = schema
		}
		changedResources.AddFunction(
			database,
			s,
			&storepb.ChangedResourceFunction{
				Name:   f,
				Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
			},
		)
	case *ast.DropFunctionStmt:
		for _, ref := range node.FunctionList {
			s, f := ref.Schema, ref.Name
			if s == "" {
				s = schema
			}
			changedResources.AddFunction(
				database,
				s,
				&storepb.ChangedResourceFunction{
					Name:   f,
					Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
				},
			)
		}
	case *ast.CommentStmt:
		if len(node.ParseResult.Stmts) != 1 {
			return errors.New("expect to get one node from comment statement result")
		}
		stmt := node.ParseResult.Stmts[0]
		comment, ok := stmt.Stmt.Node.(*pgquery.Node_CommentStmt)
		if !ok {
			return errors.New("expect to get comment node from comment statement result")
		}

		switch comment.CommentStmt.Objtype {
		case pgquery.ObjectType_OBJECT_COLUMN:
			switch n := comment.CommentStmt.Object.Node.(type) {
			case *pgquery.Node_List:
				s, table, _, err := convertColumnName(n)
				if err != nil {
					return err
				}
				if s == "" {
					s = schema
				}
				changedResources.AddTable(
					database,
					s,
					&storepb.ChangedResourceTable{
						Name:   table,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					false,
				)
			default:
				return errors.Errorf("expect to get a list node but got %T", node)
			}
		case pgquery.ObjectType_OBJECT_TABCONSTRAINT:
			switch n := comment.CommentStmt.Object.Node.(type) {
			case *pgquery.Node_List:
				s, table, _, err := convertConstraintName(n)
				if err != nil {
					return err
				}
				if s == "" {
					s = schema
				}
				changedResources.AddTable(
					database,
					s,
					&storepb.ChangedResourceTable{
						Name:   table,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					false,
				)
			default:
				return errors.Errorf("expect to get a list node but got %T", node)
			}
		case pgquery.ObjectType_OBJECT_TABLE:
			switch n := comment.CommentStmt.Object.Node.(type) {
			case *pgquery.Node_List:
				s, table, err := convertTableName(n)
				if err != nil {
					return err
				}
				if s == "" {
					s = schema
				}
				changedResources.AddTable(
					database,
					s,
					&storepb.ChangedResourceTable{
						Name:   table,
						Ranges: []*storepb.Range{base.NewRange(statement, node.Text())},
					},
					false,
				)
			default:
				return errors.Errorf("expect to get a list node but got %T", node)
			}
		}
	}

	return nil
}

func convertTableName(node *pgquery.Node_List) (string, string, error) {
	list, err := convertNodeList(node)
	if err != nil {
		return "", "", err
	}
	switch len(list) {
	case 2:
		return list[0], list[1], nil
	case 1:
		return "", list[0], nil
	default:
		return "", "", errors.Errorf("expect to get 1 or 2 items but got %d", len(list))
	}
}

func convertConstraintName(node *pgquery.Node_List) (string, string, string, error) {
	list, err := convertNodeList(node)
	if err != nil {
		return "", "", "", err
	}
	switch len(list) {
	case 3:
		return list[0], list[1], list[2], nil
	case 2:
		return "", list[0], list[1], nil
	default:
		return "", "", "", errors.Errorf("expect to get 2 or 3 items but got %d", len(list))
	}
}

func convertColumnName(node *pgquery.Node_List) (string, string, string, error) {
	list, err := convertNodeList(node)
	if err != nil {
		return "", "", "", err
	}
	switch len(list) {
	case 3:
		return list[0], list[1], list[2], nil
	case 2:
		return "", list[0], list[1], nil
	default:
		return "", "", "", errors.Errorf("expect to get 2 or 3 items but got %d", len(list))
	}
}

func convertNodeList(node *pgquery.Node_List) ([]string, error) {
	var list []string
	for _, item := range node.List.Items {
		switch s := item.Node.(type) {
		case *pgquery.Node_String_:
			list = append(list, s.String_.Sval)
		default:
			return nil, errors.Errorf("expect to get a string node but got %T", s)
		}
	}
	return list, nil
}
