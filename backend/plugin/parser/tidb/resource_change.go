package tidb

import (
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_TIDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbSchema *model.DBSchema, asts any, statement string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]tidbast.StmtNode)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}

	changedResources := model.NewChangedResources(dbSchema)
	dmlCount := 0
	insertCount := 0
	var sampleDMLs []string
	for _, node := range nodes {
		err := getResourceChanges(database, node, statement, changedResources)
		if err != nil {
			return nil, err
		}

		switch node := node.(type) {
		case *tidbast.InsertStmt:
			if len(node.Lists) > 0 {
				insertCount += len(node.Lists)
				continue
			}

			dmlCount++
			if len(sampleDMLs) < common.MaximumLintExplainSize {
				sampleDMLs = append(sampleDMLs, trimStatement(node.Text()))
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

func getResourceChanges(database string, node tidbast.StmtNode, statement string, changedResources *model.ChangedResources) error {
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
				Name:   table,
				Ranges: []*storepb.Range{getRange(statement, node)},
			},
			false,
		)
	case *tidbast.DropTableStmt:
		if node.IsView {
			for _, name := range node.Tables {
				d, view := name.Schema.O, name.Name.O
				if d == "" {
					d = database
				}
				changedResources.AddView(
					d,
					"",
					&storepb.ChangedResourceView{
						Name:   view,
						Ranges: []*storepb.Range{getRange(statement, node)},
					},
				)
			}
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
					Name:   table,
					Ranges: []*storepb.Range{getRange(statement, node)},
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
				Name:   table,
				Ranges: []*storepb.Range{getRange(statement, node)},
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
						Name:   table,
						Ranges: []*storepb.Range{getRange(statement, node)},
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
						Name:   table,
						Ranges: []*storepb.Range{getRange(statement, node)},
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
				Name:   table,
				Ranges: []*storepb.Range{getRange(statement, node)},
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
				Name:   table,
				Ranges: []*storepb.Range{getRange(statement, node)},
			},
			false,
		)
	case *tidbast.CreateViewStmt:
		d, view := node.ViewName.Schema.O, node.ViewName.Name.O
		if d == "" {
			d = database
		}
		changedResources.AddView(
			d,
			"",
			&storepb.ChangedResourceView{
				Name:   view,
				Ranges: []*storepb.Range{getRange(statement, node)},
			},
		)
	default:
	}
	return nil
}

func getRange(statement string, node tidbast.StmtNode) *storepb.Range {
	r := base.NewRange(statement, trimStatement(node.OriginalText()))
	// TiDB node text does not including the trailing semicolon.
	r.End++
	return r
}

func trimStatement(statement string) string {
	return strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
}
