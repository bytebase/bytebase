package taskrun

import (
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type databaseConfigNode struct {
	schemas []*schemaConfigNode
}

type schemaConfigNode struct {
	name   string
	tables []*tableConfigNode
}

type tableConfigNode struct {
	name string

	columns []*columnConfigNode
}

type columnConfigNode struct {
	name string

	// semanticTypeAttributeChildren is the semantic type attribute config after the migration, only be used when action is updated.
	semanticTypeAttributeChildren *columnConfigSemanticTypeAttributeNode
}

type columnConfigSemanticTypeAttributeNode struct {
	to string
}

// mergeDatabaseConfig computes the migration from targetDatabaseConfig and baselineDatabaseConfig, and applies the migration to databaseConfig.
// Return the merged databaseConfig.
func mergeDatabaseConfig(target, baseline, current *storepb.DatabaseConfig) *storepb.DatabaseConfig {
	// Avoid nil values.
	if target == nil {
		target = &storepb.DatabaseConfig{}
	}
	if baseline == nil {
		baseline = &storepb.DatabaseConfig{}
	}
	if current == nil {
		current = &storepb.DatabaseConfig{}
	}
	current = proto.Clone(current).(*storepb.DatabaseConfig)

	diff := &databaseConfigNode{}
	schemaConfigInBaselineMap := make(map[string]*storepb.SchemaConfig)
	for _, schemaConfig := range baseline.SchemaConfigs {
		schemaConfigInBaselineMap[schemaConfig.Name] = schemaConfig
	}

	// Computing the diff between databaseConfig and baselineDatabaseConfig, we only use `UPDATE` action.
	for _, schemaConfigWanted := range target.SchemaConfigs {
		schemaNode := &schemaConfigNode{
			name: schemaConfigWanted.Name,
		}
		schemaConfigInBaseline, ok := schemaConfigInBaselineMap[schemaConfigWanted.Name]
		if !ok {
			for _, tableConfigWanted := range schemaConfigWanted.TableConfigs {
				tableNode := &tableConfigNode{
					name: tableConfigWanted.Name,
				}
				for _, columnConfigWanted := range tableConfigWanted.ColumnConfigs {
					columnNode := buildColumnConfigUpdateNodeFromColumnConfig(columnConfigWanted)
					tableNode.columns = append(tableNode.columns, columnNode)
				}
				schemaNode.tables = append(schemaNode.tables, tableNode)
			}
			diff.schemas = append(diff.schemas, schemaNode)
			continue
		}

		tableConfigInBaselineMap := make(map[string]*storepb.TableConfig)
		for _, tableConfig := range schemaConfigInBaseline.TableConfigs {
			tableConfigInBaselineMap[tableConfig.Name] = tableConfig
		}

		for _, tableConfigWanted := range schemaConfigWanted.TableConfigs {
			tableNode := &tableConfigNode{
				name: tableConfigWanted.Name,
			}
			tableConfigInBaseline, ok := tableConfigInBaselineMap[tableConfigWanted.Name]
			if !ok {
				for _, columnConfigWanted := range tableConfigWanted.ColumnConfigs {
					columnNode := buildColumnConfigUpdateNodeFromColumnConfig(columnConfigWanted)
					tableNode.columns = append(tableNode.columns, columnNode)
				}
				schemaNode.tables = append(schemaNode.tables, tableNode)
				continue
			}

			columnConfigInBaselineMap := make(map[string]*storepb.ColumnConfig)
			for _, columnConfig := range tableConfigInBaseline.ColumnConfigs {
				columnConfigInBaselineMap[columnConfig.Name] = columnConfig
			}

			for _, columnConfigWanted := range tableConfigWanted.ColumnConfigs {
				columnNode := &columnConfigNode{
					name: columnConfigWanted.Name,
				}

				columnConfigInBaseline, ok := columnConfigInBaselineMap[columnConfigWanted.Name]
				if !ok {
					columnNode.semanticTypeAttributeChildren = &columnConfigSemanticTypeAttributeNode{
						to: columnConfigWanted.SemanticTypeId,
					}
					tableNode.columns = append(tableNode.columns, columnNode)
					continue
				}

				// Compare the attribute
				hasAttributesUpdate := false
				if columnConfigWanted.SemanticTypeId != columnConfigInBaseline.SemanticTypeId {
					hasAttributesUpdate = true
				}

				if hasAttributesUpdate {
					columnNode.semanticTypeAttributeChildren = &columnConfigSemanticTypeAttributeNode{
						to: columnConfigWanted.SemanticTypeId,
					}
				}

				// Append to table node if there is any update.
				if columnNode.semanticTypeAttributeChildren != nil {
					tableNode.columns = append(tableNode.columns, columnNode)
				}
				delete(columnConfigInBaselineMap, columnConfigWanted.Name)
			}
			for _, deletedColumnConfig := range columnConfigInBaselineMap {
				columnNode := getEmptyColumnConfigNode(deletedColumnConfig.Name)
				tableNode.columns = append(tableNode.columns, columnNode)
				schemaNode.tables = append(schemaNode.tables, tableNode)
				diff.schemas = append(diff.schemas, schemaNode)
			}

			// Append to schema node if there is any update.
			if len(tableNode.columns) > 0 {
				schemaNode.tables = append(schemaNode.tables, tableNode)
			}
			delete(tableConfigInBaselineMap, tableConfigWanted.Name)
		}

		for _, deletedTableConfig := range tableConfigInBaselineMap {
			tableNode := &tableConfigNode{
				name: deletedTableConfig.Name,
			}
			for _, columnConfigInBaseline := range deletedTableConfig.ColumnConfigs {
				columnNode := getEmptyColumnConfigNode(columnConfigInBaseline.Name)
				tableNode.columns = append(tableNode.columns, columnNode)
			}
			schemaNode.tables = append(schemaNode.tables, tableNode)
		}
		if len(schemaNode.tables) > 0 {
			diff.schemas = append(diff.schemas, schemaNode)
		}
		delete(schemaConfigInBaselineMap, schemaConfigWanted.Name)
	}
	for _, deletedSchemaConfig := range schemaConfigInBaselineMap {
		schemaNode := &schemaConfigNode{
			name: deletedSchemaConfig.Name,
		}
		for _, tableInBaseline := range deletedSchemaConfig.TableConfigs {
			tableNode := &tableConfigNode{
				name: tableInBaseline.Name,
			}
			for _, columnInBaseline := range tableInBaseline.ColumnConfigs {
				columnNode := getEmptyColumnConfigNode(columnInBaseline.Name)
				tableNode.columns = append(tableNode.columns, columnNode)
			}
			schemaNode.tables = append(schemaNode.tables, tableNode)
		}
		diff.schemas = append(diff.schemas, schemaNode)
	}

	// Applying the diff to appliedTarget.
	// The value of the schemaConfigInTarget is the index of the schemaConfig in appliedTarget.
	schemaConfigInTarget := make(map[string]int)
	for idx, schemaNode := range current.SchemaConfigs {
		schemaConfigInTarget[schemaNode.Name] = idx
	}

	for _, schemaNodeInDiff := range diff.schemas {
		schemaInTargetIdx, ok := schemaConfigInTarget[schemaNodeInDiff.name]
		if !ok {
			schemaNode := &storepb.SchemaConfig{
				Name: schemaNodeInDiff.name,
			}
			for _, tableNodeInDiff := range schemaNodeInDiff.tables {
				tableNode := &storepb.TableConfig{
					Name: tableNodeInDiff.name,
				}
				for _, columnInTarget := range tableNodeInDiff.columns {
					columnNode := &storepb.ColumnConfig{
						Name:           columnInTarget.name,
						SemanticTypeId: columnInTarget.semanticTypeAttributeChildren.to,
					}
					tableNode.ColumnConfigs = append(tableNode.ColumnConfigs, columnNode)
				}
				schemaNode.TableConfigs = append(schemaNode.TableConfigs, tableNode)
			}
			current.SchemaConfigs = append(current.SchemaConfigs, schemaNode)
			continue
		}

		tableConfigInTarget := make(map[string]int)
		for idx, tableNode := range current.SchemaConfigs[schemaInTargetIdx].TableConfigs {
			tableConfigInTarget[tableNode.Name] = idx
		}

		for _, tableNodeInDiff := range schemaNodeInDiff.tables {
			tableInTargetIdx, ok := tableConfigInTarget[tableNodeInDiff.name]
			if !ok {
				tableNode := &storepb.TableConfig{
					Name: tableNodeInDiff.name,
				}

				for _, columnNodeInDiff := range tableNodeInDiff.columns {
					columnNode := &storepb.ColumnConfig{
						Name:           columnNodeInDiff.name,
						SemanticTypeId: columnNodeInDiff.semanticTypeAttributeChildren.to,
					}
					tableNode.ColumnConfigs = append(tableNode.ColumnConfigs, columnNode)
				}
				current.SchemaConfigs[schemaInTargetIdx].TableConfigs = append(current.SchemaConfigs[schemaInTargetIdx].TableConfigs, tableNode)
				continue
			}

			columnConfigInTarget := make(map[string]int)
			for idx, columnNode := range current.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs {
				columnConfigInTarget[columnNode.Name] = idx
			}

			for _, columnNodeInDiff := range tableNodeInDiff.columns {
				columnInTargetIdx, ok := columnConfigInTarget[columnNodeInDiff.name]
				if !ok {
					columnNode := &storepb.ColumnConfig{
						Name:           columnNodeInDiff.name,
						SemanticTypeId: columnNodeInDiff.semanticTypeAttributeChildren.to,
					}
					current.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs = append(current.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs, columnNode)
					continue
				}

				if columnNodeInDiff.semanticTypeAttributeChildren != nil {
					current.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs[columnInTargetIdx].SemanticTypeId = columnNodeInDiff.semanticTypeAttributeChildren.to
				}
			}
		}
	}

	return current
}

// buildColumnConfigUpdateNodeFromColumnConfig builds the column config update node from the column config, copy all the attributes.
func buildColumnConfigUpdateNodeFromColumnConfig(columnConfig *storepb.ColumnConfig) *columnConfigNode {
	return &columnConfigNode{
		name: columnConfig.Name,
		semanticTypeAttributeChildren: &columnConfigSemanticTypeAttributeNode{
			to: columnConfig.SemanticTypeId,
		},
	}
}

// getEmptyColumnConfigNode returns an empty column config node, whose fields are filled with meaningless value.
func getEmptyColumnConfigNode(name string) *columnConfigNode {
	return &columnConfigNode{
		name: name,
		semanticTypeAttributeChildren: &columnConfigSemanticTypeAttributeNode{
			to: "",
		},
	}
}
