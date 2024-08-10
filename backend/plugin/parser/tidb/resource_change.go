package tidb

import (
	"sort"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_TIDB, extractChangedResources)
}

func extractChangedResources(database string, _ string, asts any, statement string) (*base.ChangeSummary, error) {
	nodes, ok := asts.([]tidbast.StmtNode)
	if !ok {
		return nil, errors.Errorf("invalid ast type %T", asts)
	}
	resourceChangeMap := make(map[string]*base.ResourceChange)
	dmlCount := 0
	insertCount := 0
	var sampleDMLs []string
	for _, node := range nodes {
		err := getResourceChanges(database, node, statement, resourceChangeMap)
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

	var resourceChanges []*base.ResourceChange
	for _, change := range resourceChangeMap {
		resourceChanges = append(resourceChanges, change)
	}
	sort.Slice(resourceChanges, func(i, j int) bool {
		return resourceChanges[i].String() < resourceChanges[j].String()
	})
	return &base.ChangeSummary{
		ResourceChanges: resourceChanges,
		DMLCount:        dmlCount,
		SampleDMLS:      sampleDMLs,
		InsertCount:     insertCount,
	}, nil
}

func getResourceChanges(database string, node tidbast.StmtNode, statement string, resourceChangeMap map[string]*base.ResourceChange) error {
	switch node := node.(type) {
	case *tidbast.CreateTableStmt:
		resource := base.SchemaResource{
			Database: database,
			Table:    node.Table.Name.O,
		}
		if node.Table.Schema.O != "" {
			resource.Database = node.Table.Schema.O
		}
		putResourceChange(resourceChangeMap, &base.ResourceChange{
			Resource: resource,
			Ranges:   []base.Range{getRange(statement, node)},
		})
	case *tidbast.DropTableStmt:
		// TODO(d): deal with DROP VIEW statement.
		if node.IsView {
			return nil
		}
		for _, name := range node.Tables {
			resource := base.SchemaResource{
				Database: database,
				Table:    name.Name.O,
			}
			if name.Schema.O != "" {
				resource.Database = name.Schema.O
			}

			putResourceChange(resourceChangeMap, &base.ResourceChange{
				Resource:    resource,
				Ranges:      []base.Range{getRange(statement, node)},
				AffectTable: true,
			})
		}
	case *tidbast.AlterTableStmt:
		resource := base.SchemaResource{
			Database: database,
			Table:    node.Table.Name.O,
		}
		if node.Table.Schema.O != "" {
			resource.Database = node.Table.Schema.O
		}
		putResourceChange(resourceChangeMap, &base.ResourceChange{
			Resource:    resource,
			Ranges:      []base.Range{getRange(statement, node)},
			AffectTable: true,
		})
	case *tidbast.RenameTableStmt:
		for _, tableToTable := range node.TableToTables {
			{
				resource := base.SchemaResource{
					Database: database,
					Table:    tableToTable.OldTable.Name.O,
				}
				if tableToTable.OldTable.Schema.O != "" {
					resource.Database = tableToTable.OldTable.Schema.O
				}
				putResourceChange(resourceChangeMap, &base.ResourceChange{
					Resource: resource,
					Ranges:   []base.Range{getRange(statement, node)},
				})
			}
			{
				resource := base.SchemaResource{
					Database: database,
					Table:    tableToTable.NewTable.Name.O,
				}
				if tableToTable.NewTable.Schema.O != "" {
					resource.Database = tableToTable.NewTable.Schema.O
				}
				putResourceChange(resourceChangeMap, &base.ResourceChange{
					Resource: resource,
					Ranges:   []base.Range{getRange(statement, node)},
				})
			}
		}
	default:
	}
	return nil
}

func getRange(statement string, node tidbast.StmtNode) base.Range {
	r := base.NewRange(statement, trimStatement(node.OriginalText()))
	// TiDB node text does not including the trailing semicolon.
	r.End++
	return r
}

func putResourceChange(resourceChangeMap map[string]*base.ResourceChange, change *base.ResourceChange) {
	v, ok := resourceChangeMap[change.String()]
	if !ok {
		resourceChangeMap[change.String()] = change
		return
	}

	v.Ranges = append(v.Ranges, change.Ranges...)
	if change.AffectTable {
		v.AffectTable = true
	}
}

func trimStatement(statement string) string {
	return strings.TrimLeft(strings.TrimRight(statement, " \n\t;"), " \n\t")
}
