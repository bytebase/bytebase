package utils

import (
	"reflect"
	"sort"

	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// MergeDatabaseConfig computes the migration from target and baseline, and applies the migration to current databaseConfig.
// Return the merged databaseConfig.
func MergeDatabaseConfig(target, baseline, current *storepb.DatabaseConfig) *storepb.DatabaseConfig {
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
		// Remove from the current since the change is to delete the schema.
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

	targetProcedureMap, baselineProcedureMap, currentProcedureMap := buildProcedureMap(target), buildProcedureMap(baseline), buildProcedureMap(current)
	for procedureName, targetProcedure := range targetProcedureMap {
		currentProcedure, hasCurrent := currentProcedureMap[procedureName]
		baselineProcedure := baselineProcedureMap[procedureName]
		if hasCurrent {
			currentProcedureMap[procedureName] = mergeProcedureConfig(targetProcedure, baselineProcedure, currentProcedure)
		} else {
			currentProcedureMap[procedureName] = targetProcedure
		}
	}

	targetFunctionMap, baselineFunctionMap, currentFunctionMap := buildFunctionMap(target), buildFunctionMap(baseline), buildFunctionMap(current)
	for functionName, targetFunction := range targetFunctionMap {
		currentFunction, hasCurrent := currentFunctionMap[functionName]
		baselineFunction := baselineFunctionMap[functionName]
		if hasCurrent {
			currentFunctionMap[functionName] = mergeFunctionConfig(targetFunction, baselineFunction, currentFunction)
		} else {
			currentFunctionMap[functionName] = targetFunction
		}
	}

	targetViewMap, baselineViewMap, currentViewMap := buildViewMap(target), buildViewMap(baseline), buildViewMap(current)
	for viewName, targetView := range targetViewMap {
		currentView, hasCurrent := currentViewMap[viewName]
		baselineView := baselineViewMap[viewName]
		if hasCurrent {
			currentViewMap[viewName] = mergeViewConfig(targetView, baselineView, currentView)
		} else {
			currentViewMap[viewName] = targetView
		}
	}

	result := &storepb.SchemaConfig{Name: current.Name}
	for _, v := range currentMap {
		result.TableConfigs = append(result.TableConfigs, v)
	}
	for _, v := range currentProcedureMap {
		result.ProcedureConfigs = append(result.ProcedureConfigs, v)
	}
	for _, v := range currentFunctionMap {
		result.FunctionConfigs = append(result.FunctionConfigs, v)
	}
	for _, v := range currentViewMap {
		result.ViewConfigs = append(result.ViewConfigs, v)
	}
	sort.Slice(result.TableConfigs, func(i, j int) bool {
		return result.TableConfigs[i].Name < result.TableConfigs[j].Name
	})
	return result
}

func mergeFunctionConfig(target, baseline, current *storepb.FunctionConfig) *storepb.FunctionConfig {
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.Updater, target.UpdateTime, target.SourceBranch, baseline.Updater, current.Updater, current.UpdateTime, current.SourceBranch)
	return &storepb.FunctionConfig{
		Name:         current.Name,
		Updater:      lastUpdater,
		UpdateTime:   lastUpdateTime,
		SourceBranch: sourceBranch,
	}
}

func mergeProcedureConfig(target, baseline, current *storepb.ProcedureConfig) *storepb.ProcedureConfig {
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.Updater, target.UpdateTime, target.SourceBranch, baseline.Updater, current.Updater, current.UpdateTime, current.SourceBranch)
	return &storepb.ProcedureConfig{
		Name:         current.Name,
		Updater:      lastUpdater,
		UpdateTime:   lastUpdateTime,
		SourceBranch: sourceBranch,
	}
}

func mergeViewConfig(target, baseline, current *storepb.ViewConfig) *storepb.ViewConfig {
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.Updater, target.UpdateTime, target.SourceBranch, baseline.Updater, current.Updater, current.UpdateTime, current.SourceBranch)
	return &storepb.ViewConfig{
		Name:         current.Name,
		Updater:      lastUpdater,
		UpdateTime:   lastUpdateTime,
		SourceBranch: sourceBranch,
	}
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

	result := &storepb.TableConfig{Name: current.Name, ClassificationId: current.ClassificationId}
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.Updater, target.UpdateTime, target.SourceBranch, baseline.Updater, current.Updater, current.UpdateTime, current.SourceBranch)
	result.Updater = lastUpdater
	result.UpdateTime = lastUpdateTime
	result.SourceBranch = sourceBranch
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
		if target == nil {
			return current
		}
		return target
	}
	// Current is never nil.
	// If baseline = A, target = B, current = C, we should set merged value to B.
	// If baseline = A, target = A, current = B, we should set merged value to B since there is no change intentially.
	if target.SemanticTypeId != baseline.SemanticTypeId {
		current.SemanticTypeId = target.SemanticTypeId
	}
	if target.ClassificationId != baseline.ClassificationId {
		current.ClassificationId = target.ClassificationId
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

func buildProcedureMap(config *storepb.SchemaConfig) map[string]*storepb.ProcedureConfig {
	m := make(map[string]*storepb.ProcedureConfig)
	for _, v := range config.ProcedureConfigs {
		m[v.Name] = v
	}
	return m
}

func buildFunctionMap(config *storepb.SchemaConfig) map[string]*storepb.FunctionConfig {
	m := make(map[string]*storepb.FunctionConfig)
	for _, v := range config.FunctionConfigs {
		m[v.Name] = v
	}
	return m
}

func buildViewMap(config *storepb.SchemaConfig) map[string]*storepb.ViewConfig {
	m := make(map[string]*storepb.ViewConfig)
	for _, v := range config.ViewConfigs {
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

func getLastUpdaterAndSourceBranch(target string, targetTime *timestamppb.Timestamp, targetBranch string, baseline string, current string, currentTime *timestamppb.Timestamp, currentBranch string) (string, *timestamppb.Timestamp, string) {
	if target == baseline {
		return current, currentTime, currentBranch
	}
	return target, targetTime, targetBranch
}
