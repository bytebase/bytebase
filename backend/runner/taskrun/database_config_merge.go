package taskrun

import (
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type updateDatabaseConfigAction string

const (
	updateDatabaseConfigActionUpdate updateDatabaseConfigAction = "UPDATE"
)

type databaseConfigNode struct {
	schemaChildren []*schemaConfigNode
}

type schemaConfigNode struct {
	action        updateDatabaseConfigAction
	name          string
	tableChildren []*tableConfigNode
}

type tableConfigNode struct {
	action updateDatabaseConfigAction
	name   string

	columnChildren []*columnConfigNode
}

type columnConfigNode struct {
	action updateDatabaseConfigAction
	name   string

	// semanticTypeAttributeChildren is the semantic type attribute config after the migration, only be used when action is updated.
	semanticTypeAttributeChildren *columnConfigSemanticTypeAttributeNode
}

type columnConfigSemanticTypeAttributeNode struct {
	action updateDatabaseConfigAction

	to string
}

// mergeDatabaseConfig computes the migration from databaseConfig and baselineDatabaseConfig, and applies the migration to appliedTarget, returns the updated databaseConfig.
func mergeDatabaseConfig(databaseConfig, baselineDatabaseConfig, appliedTarget *storepb.DatabaseConfig) *storepb.DatabaseConfig {
	// To avoid determining the databaseConfig, baselineDatabaseConfig and appliedTarget are nil or not, we will replace the nil with empty config at the beginning.
	if databaseConfig == nil {
		databaseConfig = &storepb.DatabaseConfig{}
	}
	if baselineDatabaseConfig == nil {
		baselineDatabaseConfig = &storepb.DatabaseConfig{}
	}
	if appliedTarget == nil {
		appliedTarget = &storepb.DatabaseConfig{}
	}

	diff := &databaseConfigNode{}

	schemaConfigInBaselineMap := make(map[string]*storepb.SchemaConfig)
	for _, schemaConfig := range baselineDatabaseConfig.SchemaConfigs {
		schemaConfigInBaselineMap[schemaConfig.Name] = schemaConfig
	}

	// Computing the diff between databaseConfig and baselineDatabaseConfig, we only use `UPDATE` action.
	for _, schemaConfigWanted := range databaseConfig.SchemaConfigs {
		schemaNode := &schemaConfigNode{
			action: updateDatabaseConfigActionUpdate,
			name:   schemaConfigWanted.Name,
		}
		schemaConfigInBaseline, ok := schemaConfigInBaselineMap[schemaConfigWanted.Name]
		if !ok {
			for _, tableConfigWanted := range schemaConfigWanted.TableConfigs {
				tableNode := &tableConfigNode{
					action: updateDatabaseConfigActionUpdate,
					name:   tableConfigWanted.Name,
				}
				for _, columnConfigWanted := range tableConfigWanted.ColumnConfigs {
					columnNode := buildColumnConfigUpdateNodeFromColumnConfig(columnConfigWanted)
					tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
				}
				schemaNode.tableChildren = append(schemaNode.tableChildren, tableNode)
			}
			diff.schemaChildren = append(diff.schemaChildren, schemaNode)
			continue
		}

		tableConfigInBaselineMap := make(map[string]*storepb.TableConfig)
		for _, tableConfig := range schemaConfigInBaseline.TableConfigs {
			tableConfigInBaselineMap[tableConfig.Name] = tableConfig
		}

		for _, tableConfigWanted := range schemaConfigWanted.TableConfigs {
			tableNode := &tableConfigNode{
				action: updateDatabaseConfigActionUpdate,
				name:   tableConfigWanted.Name,
			}
			tableConfigInBaseline, ok := tableConfigInBaselineMap[tableConfigWanted.Name]
			if !ok {
				for _, columnConfigWanted := range tableConfigWanted.ColumnConfigs {
					columnNode := buildColumnConfigUpdateNodeFromColumnConfig(columnConfigWanted)
					tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
				}
				schemaNode.tableChildren = append(schemaNode.tableChildren, tableNode)
				continue
			}

			columnConfigInBaselineMap := make(map[string]*storepb.ColumnConfig)
			for _, columnConfig := range tableConfigInBaseline.ColumnConfigs {
				columnConfigInBaselineMap[columnConfig.Name] = columnConfig
			}

			for _, columnConfigWanted := range tableConfigWanted.ColumnConfigs {
				columnNode := &columnConfigNode{
					action: updateDatabaseConfigActionUpdate,
					name:   columnConfigWanted.Name,
				}

				columnConfigInBaseline, ok := columnConfigInBaselineMap[columnConfigWanted.Name]
				if !ok {
					columnNode.semanticTypeAttributeChildren = &columnConfigSemanticTypeAttributeNode{
						action: updateDatabaseConfigActionUpdate,
						to:     columnConfigWanted.SemanticTypeId,
					}
					tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
					continue
				}

				// Compare the attribute
				hasAttributesUpdate := false
				if columnConfigWanted.SemanticTypeId != columnConfigInBaseline.SemanticTypeId {
					hasAttributesUpdate = true
				}

				if hasAttributesUpdate {
					columnNode.semanticTypeAttributeChildren = &columnConfigSemanticTypeAttributeNode{
						action: updateDatabaseConfigActionUpdate,
						to:     columnConfigWanted.SemanticTypeId,
					}
				}

				// Append to table node if there is any update.
				if columnNode.semanticTypeAttributeChildren != nil {
					tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
				}
				delete(columnConfigInBaselineMap, columnConfigWanted.Name)
			}
			for _, deletedColumnConfig := range columnConfigInBaselineMap {
				columnNode := getEmptyColumnConfigNode(deletedColumnConfig.Name)
				tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
				schemaNode.tableChildren = append(schemaNode.tableChildren, tableNode)
				diff.schemaChildren = append(diff.schemaChildren, schemaNode)
			}

			// Append to schema node if there is any update.
			if len(tableNode.columnChildren) > 0 {
				schemaNode.tableChildren = append(schemaNode.tableChildren, tableNode)
			}
			delete(tableConfigInBaselineMap, tableConfigWanted.Name)
		}

		for _, deletedTableConfig := range tableConfigInBaselineMap {
			tableNode := &tableConfigNode{
				action: updateDatabaseConfigActionUpdate,
				name:   deletedTableConfig.Name,
			}
			for _, columnConfigInBaseline := range deletedTableConfig.ColumnConfigs {
				columnNode := getEmptyColumnConfigNode(columnConfigInBaseline.Name)
				tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
			}
			schemaNode.tableChildren = append(schemaNode.tableChildren, tableNode)
		}
		if len(schemaNode.tableChildren) > 0 {
			diff.schemaChildren = append(diff.schemaChildren, schemaNode)
		}
		delete(schemaConfigInBaselineMap, schemaConfigWanted.Name)
	}
	for _, deletedSchemaConfig := range schemaConfigInBaselineMap {
		schemaNode := &schemaConfigNode{
			action: updateDatabaseConfigActionUpdate,
			name:   deletedSchemaConfig.Name,
		}
		for _, tableInBaseline := range deletedSchemaConfig.TableConfigs {
			tableNode := &tableConfigNode{
				action: updateDatabaseConfigActionUpdate,
				name:   tableInBaseline.Name,
			}
			for _, columnInBaseline := range tableInBaseline.ColumnConfigs {
				columnNode := getEmptyColumnConfigNode(columnInBaseline.Name)
				tableNode.columnChildren = append(tableNode.columnChildren, columnNode)
			}
			schemaNode.tableChildren = append(schemaNode.tableChildren, tableNode)
		}
		diff.schemaChildren = append(diff.schemaChildren, schemaNode)
	}

	// Applying the diff to appliedTarget.
	result := proto.Clone(appliedTarget).(*storepb.DatabaseConfig)

	// The value of the schemaConfigInTarget is the index of the schemaConfig in appliedTarget.
	schemaConfigInTarget := make(map[string]int)
	for idx, schemaNode := range result.SchemaConfigs {
		schemaConfigInTarget[schemaNode.Name] = idx
	}

	for _, schemaNodeInDiff := range diff.schemaChildren {
		schemaInTargetIdx, ok := schemaConfigInTarget[schemaNodeInDiff.name]
		if !ok {
			schemaNode := &storepb.SchemaConfig{
				Name: schemaNodeInDiff.name,
			}
			for _, tableNodeInDiff := range schemaNodeInDiff.tableChildren {
				tableNode := &storepb.TableConfig{
					Name: tableNodeInDiff.name,
				}
				for _, columnInTarget := range tableNodeInDiff.columnChildren {
					columnNode := &storepb.ColumnConfig{
						Name:           columnInTarget.name,
						SemanticTypeId: columnInTarget.semanticTypeAttributeChildren.to,
					}
					tableNode.ColumnConfigs = append(tableNode.ColumnConfigs, columnNode)
				}
				schemaNode.TableConfigs = append(schemaNode.TableConfigs, tableNode)
			}
			result.SchemaConfigs = append(result.SchemaConfigs, schemaNode)
			continue
		}

		tableConfigInTarget := make(map[string]int)
		for idx, tableNode := range result.SchemaConfigs[schemaInTargetIdx].TableConfigs {
			tableConfigInTarget[tableNode.Name] = idx
		}

		for _, tableNodeInDiff := range schemaNodeInDiff.tableChildren {
			tableInTargetIdx, ok := tableConfigInTarget[tableNodeInDiff.name]
			if !ok {
				tableNode := &storepb.TableConfig{
					Name: tableNodeInDiff.name,
				}

				for _, columnNodeInDiff := range tableNodeInDiff.columnChildren {
					columnNode := &storepb.ColumnConfig{
						Name:           columnNodeInDiff.name,
						SemanticTypeId: columnNodeInDiff.semanticTypeAttributeChildren.to,
					}
					tableNode.ColumnConfigs = append(tableNode.ColumnConfigs, columnNode)
				}
				result.SchemaConfigs[schemaInTargetIdx].TableConfigs = append(result.SchemaConfigs[schemaInTargetIdx].TableConfigs, tableNode)
				continue
			}

			columnConfigInTarget := make(map[string]int)
			for idx, columnNode := range result.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs {
				columnConfigInTarget[columnNode.Name] = idx
			}

			for _, columnNodeInDiff := range tableNodeInDiff.columnChildren {
				columnInTargetIdx, ok := columnConfigInTarget[columnNodeInDiff.name]
				if !ok {
					columnNode := &storepb.ColumnConfig{
						Name:           columnNodeInDiff.name,
						SemanticTypeId: columnNodeInDiff.semanticTypeAttributeChildren.to,
					}
					result.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs = append(result.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs, columnNode)
					continue
				}

				if columnNodeInDiff.semanticTypeAttributeChildren != nil {
					result.SchemaConfigs[schemaInTargetIdx].TableConfigs[tableInTargetIdx].ColumnConfigs[columnInTargetIdx].SemanticTypeId = columnNodeInDiff.semanticTypeAttributeChildren.to
				}
			}
		}
	}

	return result
}

// buildColumnConfigUpdateNodeFromColumnConfig builds the column config update node from the column config, copy all the attributes.
func buildColumnConfigUpdateNodeFromColumnConfig(columnConfig *storepb.ColumnConfig) *columnConfigNode {
	return &columnConfigNode{
		action: updateDatabaseConfigActionUpdate,
		name:   columnConfig.Name,
		semanticTypeAttributeChildren: &columnConfigSemanticTypeAttributeNode{
			action: updateDatabaseConfigActionUpdate,
			to:     columnConfig.SemanticTypeId,
		},
	}
}

// getEmptyColumnConfigNode returns an empty column config node, whose fields are filled with meaningless value.
func getEmptyColumnConfigNode(name string) *columnConfigNode {
	return &columnConfigNode{
		action: updateDatabaseConfigActionUpdate,
		name:   name,
		semanticTypeAttributeChildren: &columnConfigSemanticTypeAttributeNode{
			action: updateDatabaseConfigActionUpdate,
			to:     "",
		},
	}
}
