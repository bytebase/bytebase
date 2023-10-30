package taskrun

import (
	"reflect"
	"sort"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// mergeDatabaseConfig computes the migration from target and baseline, and applies the migration to current databaseConfig.
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

	targetMap, baselineMap, currentMap := buildSchemaMap(target), buildSchemaMap(baseline), buildSchemaMap(current)
	for schemaName, targetSchema := range targetMap {
		currentSchema, hasCurrent := currentMap[schemaName]
		baselineSchema := baselineMap[schemaName]
		if hasCurrent {
			currentMap[schemaName] = mergeSchemaConfig(targetSchema, baselineSchema, currentSchema)
		} else {
			currentMap[schemaName] = targetSchema
		}
	}
	for schemaName := range baselineMap {
		if _, ok := targetMap[schemaName]; ok {
			// Already checked above.
			continue
		}
		// Remove from the current since the change is to delete the schema..
		delete(currentMap, schemaName)
	}

	result := &storepb.DatabaseConfig{Name: current.Name}
	for _, v := range currentMap {
		result.SchemaConfigs = append(result.SchemaConfigs, v)
	}
	sort.Slice(result.SchemaConfigs, func(i, j int) bool {
		return result.SchemaConfigs[i].Name < result.SchemaConfigs[j].Name
	})
	return result
}

func mergeSchemaConfig(target, baseline, current *storepb.SchemaConfig) *storepb.SchemaConfig {
	// Avoid nil values.
	if target == nil {
		target = &storepb.SchemaConfig{}
	}
	if baseline == nil {
		baseline = &storepb.SchemaConfig{}
	}
	if current == nil {
		current = &storepb.SchemaConfig{}
	}

	targetMap, baselineMap, currentMap := buildTableMap(target), buildTableMap(baseline), buildTableMap(current)
	for tableName, targetTable := range targetMap {
		currentTable, hasCurrent := currentMap[tableName]
		baselineTable := baselineMap[tableName]
		if hasCurrent {
			currentMap[tableName] = mergeTableConfig(targetTable, baselineTable, currentTable)
		} else {
			currentMap[tableName] = targetTable
		}
	}
	for tableName := range baselineMap {
		if _, ok := targetMap[tableName]; ok {
			// Already checked above.
			continue
		}
		// Remove from the current since the change is to delete the schema..
		delete(currentMap, tableName)
	}

	result := &storepb.SchemaConfig{Name: current.Name}
	for _, v := range currentMap {
		result.TableConfigs = append(result.TableConfigs, v)
	}
	sort.Slice(result.TableConfigs, func(i, j int) bool {
		return result.TableConfigs[i].Name < result.TableConfigs[j].Name
	})
	return result
}

func mergeTableConfig(target, baseline, current *storepb.TableConfig) *storepb.TableConfig {
	// Avoid nil values.
	if target == nil {
		target = &storepb.TableConfig{}
	}
	if baseline == nil {
		baseline = &storepb.TableConfig{}
	}
	if current == nil {
		current = &storepb.TableConfig{}
	}

	targetMap, baselineMap, currentMap := buildColumnMap(target), buildColumnMap(baseline), buildColumnMap(current)
	for columnName, targetColumn := range targetMap {
		currentColumn, hasCurrent := currentMap[columnName]
		baselineColumn := baselineMap[columnName]
		if hasCurrent {
			currentMap[columnName] = mergeColumnConfig(targetColumn, baselineColumn, currentColumn)
		} else {
			currentMap[columnName] = targetColumn
		}
	}
	for tableName := range baselineMap {
		if _, ok := targetMap[tableName]; ok {
			// Already checked above.
			continue
		}
		// Remove from the current since the change is to delete the schema..
		delete(currentMap, tableName)
	}

	result := &storepb.TableConfig{Name: current.Name}
	for _, v := range currentMap {
		result.ColumnConfigs = append(result.ColumnConfigs, v)
	}
	sort.Slice(result.ColumnConfigs, func(i, j int) bool {
		return result.ColumnConfigs[i].Name < result.ColumnConfigs[j].Name
	})
	return result
}

func mergeColumnConfig(target, baseline, current *storepb.ColumnConfig) *storepb.ColumnConfig {
	if baseline == nil {
		// Baseline could be nil. When it's nil, we should set the current stale value to target value.
		return target
	}
	// Current is never nil.
	// If baseline = A, target = B, current = C, we should set merged value to B.
	// If baseline = A, target = A, current = B, we should set merged value to B since there is no change intentially.
	if target.SemanticTypeId != baseline.SemanticTypeId {
		current.SemanticTypeId = target.SemanticTypeId
	}

	if !reflect.DeepEqual(target.Labels, baseline.Labels) {
		current.Labels = target.Labels
	}

	return current
}

func buildSchemaMap(config *storepb.DatabaseConfig) map[string]*storepb.SchemaConfig {
	m := make(map[string]*storepb.SchemaConfig)
	for _, v := range config.SchemaConfigs {
		m[v.Name] = v
	}
	return m
}

func buildTableMap(config *storepb.SchemaConfig) map[string]*storepb.TableConfig {
	m := make(map[string]*storepb.TableConfig)
	for _, v := range config.TableConfigs {
		m[v.Name] = v
	}
	return m
}

func buildColumnMap(config *storepb.TableConfig) map[string]*storepb.ColumnConfig {
	m := make(map[string]*storepb.ColumnConfig)
	for _, v := range config.ColumnConfigs {
		m[v.Name] = v
	}
	return m
}
