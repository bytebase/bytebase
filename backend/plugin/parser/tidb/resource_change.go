package tidb

import (
	"strings"
	"unicode"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_TIDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, statement string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	dmlCount := 0
	insertCount := 0
	var sampleDMLs []string
	for _, ast := range asts {
		tidbAST, ok := GetTiDBAST(ast)
		if !ok {
			return nil, errors.New("expected TiDB AST")
		}
		node := tidbAST.Node
		err := getResourceChanges(database, node, statement, changedResources)
		if err != nil {
			return nil, err
		}

		switch n := node.(type) {
		case *tidbast.InsertStmt:
			if len(n.Lists) > 0 {
				insertCount += len(n.Lists)
				continue
			}

			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, trimStatement(n.Text()))
			}
		case *tidbast.UpdateStmt, *tidbast.DeleteStmt:
			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, trimStatement(node.Text()))
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

func getResourceChanges(database string, node tidbast.StmtNode, _ string, changedResources *model.ChangedResources) error {
	switch node := node.(type) {
	case *tidbast.CreateTableStmt:
		d, table := node.Table.Schema.O, node.Table.Name.O
		if d == "" {
			d = database
		}
		changedResources.AddTable(
			d,
			"",
			&storepb.ChangedResourceTable{
				Name: table,
			},
			false,
		)
	case *tidbast.DropTableStmt:
		if node.IsView {
			// View tracking removed - not used in risk/approval calculations
			return nil
		}
		for _, name := range node.Tables {
			d, table := name.Schema.O, name.Name.O
			if d == "" {
				d = database
			}
			changedResources.AddTable(
				d,
				"",
				&storepb.ChangedResourceTable{
					Name: table,
				},
				true,
			)
		}
	case *tidbast.AlterTableStmt:
		d, table := node.Table.Schema.O, node.Table.Name.O
		if d == "" {
			d = database
		}
		changedResources.AddTable(
			d,
			"",
			&storepb.ChangedResourceTable{
				Name: table,
			},
			true,
		)
	case *tidbast.RenameTableStmt:
		for _, tableToTable := range node.TableToTables {
			{
				d, table := tableToTable.OldTable.Schema.O, tableToTable.OldTable.Name.O
				if d == "" {
					d = database
				}
				changedResources.AddTable(
					d,
					"",
					&storepb.ChangedResourceTable{
						Name: table,
					},
					false,
				)
			}
			{
				d, table := tableToTable.NewTable.Schema.O, tableToTable.NewTable.Name.O
				if d == "" {
					d = database
				}
				changedResources.AddTable(
					d,
					"",
					&storepb.ChangedResourceTable{
						Name: table,
					},
					false,
				)
			}
		}
	case *tidbast.CreateIndexStmt:
		d, table := node.Table.Schema.O, node.Table.Name.O
		if d == "" {
			d = database
		}
		changedResources.AddTable(
			d,
			"",
			&storepb.ChangedResourceTable{
				Name: table,
			},
			false,
		)
	case *tidbast.DropIndexStmt:
		d, table := node.Table.Schema.O, node.Table.Name.O
		if d == "" {
			d = database
		}
		changedResources.AddTable(
			d,
			"",
			&storepb.ChangedResourceTable{
				Name: table,
			},
			false,
		)
	case *tidbast.CreateViewStmt:
		// View tracking removed - not used in risk/approval calculations
	case *tidbast.InsertStmt:
		tables, err := extractTableRefs(node.Table)
		if err != nil {
			return errors.Wrap(err, "failed to extract table refs")
		}
		for _, table := range tables {
			d := table.database
			if d == "" {
				d = database
			}
			changedResources.AddTable(
				d,
				"",
				&storepb.ChangedResourceTable{
					Name: table.table,
				},
				false,
			)
		}
	case *tidbast.UpdateStmt:
		tables, err := extractTableRefs(node.TableRefs)
		if err != nil {
			return errors.Wrap(err, "failed to extract table refs")
		}
		for _, table := range tables {
			d := table.database
			if d == "" {
				d = database
			}
			changedResources.AddTable(
				d,
				"",
				&storepb.ChangedResourceTable{
					Name: table.table,
				},
				false,
			)
		}
	case *tidbast.DeleteStmt:
		tables, err := extractTableRefs(node.TableRefs)
		if err != nil {
			return errors.Wrap(err, "failed to extract table refs")
		}
		for _, table := range tables {
			d := table.database
			if d == "" {
				d = database
			}
			changedResources.AddTable(
				d,
				"",
				&storepb.ChangedResourceTable{
					Name: table.table,
				},
				false,
			)
		}
		if node.Tables != nil {
			for _, table := range node.Tables.Tables {
				d := table.Schema.O
				if d == "" {
					d = database
				}
				changedResources.AddTable(
					d,
					"",
					&storepb.ChangedResourceTable{
						Name: table.Name.O,
					},
					false,
				)
			}
		}
	default:
	}
	return nil
}

func trimStatement(statement string) string {
	return strings.TrimLeftFunc(strings.TrimRightFunc(statement, utils.IsSpaceOrSemicolon), unicode.IsSpace)
}

type table struct {
	database string
	table    string
}

func extractResultSetNode(n tidbast.ResultSetNode) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	switch n := n.(type) {
	case *tidbast.SelectStmt:
		return nil, nil
	case *tidbast.SubqueryExpr:
		return nil, nil
	case *tidbast.TableSource:
		return extractTableSource(n)
	case *tidbast.TableName:
		return extractTableName(n)
	case *tidbast.Join:
		return extractJoin(n)
	case *tidbast.SetOprStmt:
		return nil, nil
	}
	return nil, nil
}

func extractTableRefs(n *tidbast.TableRefsClause) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	return extractJoin(n.TableRefs)
}

func extractJoin(n *tidbast.Join) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	l, err := extractResultSetNode(n.Left)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract left node in join")
	}
	r, err := extractResultSetNode(n.Right)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract right node in join")
	}
	l = append(l, r...)
	return l, nil
}

func extractTableSource(n *tidbast.TableSource) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	return extractResultSetNode(n.Source)
}

func extractTableName(n *tidbast.TableName) ([]table, error) {
	if n == nil {
		return nil, nil
	}
	return []table{
		{
			table:    n.Name.O,
			database: n.Schema.O,
		},
	}, nil
}
