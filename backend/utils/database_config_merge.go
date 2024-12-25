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
		result.Schemas = append(result.Schemas, v)
	}
	sort.Slice(result.Schemas, func(i, j int) bool {
		return result.Schemas[i].Name < result.Schemas[j].Name
	})
	return result
}

func mergeSchemaConfig(target, baseline, current *storepb.SchemaCatalog) *storepb.SchemaCatalog {
	// Avoid nil values.
	if target == nil {
		target = &storepb.SchemaCatalog{}
	}
	if baseline == nil {
		baseline = &storepb.SchemaCatalog{}
	}
	if current == nil {
		current = &storepb.SchemaCatalog{}
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

	result := &storepb.SchemaCatalog{Name: current.Name}
	for _, v := range currentMap {
		result.Tables = append(result.Tables, v)
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
	sort.Slice(result.Tables, func(i, j int) bool {
		return result.Tables[i].Name < result.Tables[j].Name
	})
	return result
}

func mergeFunctionConfig(target, baseline, current *storepb.FunctionConfig) *storepb.FunctionConfig {
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.GetUpdater(), target.GetUpdateTime(), target.GetSourceBranch(), baseline.GetUpdater(), current.GetUpdater(), current.GetUpdateTime(), current.GetSourceBranch())
	return &storepb.FunctionConfig{
		Name:         current.Name,
		Updater:      lastUpdater,
		UpdateTime:   lastUpdateTime,
		SourceBranch: sourceBranch,
	}
}

func mergeProcedureConfig(target, baseline, current *storepb.ProcedureConfig) *storepb.ProcedureConfig {
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.GetUpdater(), target.GetUpdateTime(), target.GetSourceBranch(), baseline.GetUpdater(), current.GetUpdater(), current.GetUpdateTime(), current.GetSourceBranch())
	return &storepb.ProcedureConfig{
		Name:         current.Name,
		Updater:      lastUpdater,
		UpdateTime:   lastUpdateTime,
		SourceBranch: sourceBranch,
	}
}

func mergeViewConfig(target, baseline, current *storepb.ViewConfig) *storepb.ViewConfig {
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.GetUpdater(), target.GetUpdateTime(), target.GetSourceBranch(), baseline.GetUpdater(), current.GetUpdater(), current.GetUpdateTime(), current.GetSourceBranch())
	return &storepb.ViewConfig{
		Name:         current.Name,
		Updater:      lastUpdater,
		UpdateTime:   lastUpdateTime,
		SourceBranch: sourceBranch,
	}
}

func mergeTableConfig(target, baseline, current *storepb.TableCatalog) *storepb.TableCatalog {
	// Avoid nil values.
	if target == nil {
		target = &storepb.TableCatalog{}
	}
	if baseline == nil {
		baseline = &storepb.TableCatalog{}
	}
	if current == nil {
		current = &storepb.TableCatalog{}
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

	result := &storepb.TableCatalog{Name: current.Name, ClassificationId: current.ClassificationId}
	lastUpdater, lastUpdateTime, sourceBranch := getLastUpdaterAndSourceBranch(target.GetUpdater(), target.GetUpdateTime(), target.GetSourceBranch(), baseline.GetUpdater(), current.GetUpdater(), current.GetUpdateTime(), current.GetSourceBranch())
	result.Updater = lastUpdater
	result.UpdateTime = lastUpdateTime
	result.SourceBranch = sourceBranch
	for _, v := range currentMap {
		result.Columns = append(result.Columns, v)
	}
	sort.Slice(result.Columns, func(i, j int) bool {
		return result.Columns[i].Name < result.Columns[j].Name
	})
	return result
}

func mergeColumnConfig(target, baseline, current *storepb.ColumnCatalog) *storepb.ColumnCatalog {
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
	if target.MaskingLevel != baseline.MaskingLevel {
		current.MaskingLevel = target.MaskingLevel
	}
	if target.FullMaskingAlgorithmId != baseline.FullMaskingAlgorithmId {
		current.FullMaskingAlgorithmId = target.FullMaskingAlgorithmId
	}
	if target.PartialMaskingAlgorithmId != baseline.PartialMaskingAlgorithmId {
		current.PartialMaskingAlgorithmId = target.PartialMaskingAlgorithmId
	}

	if !reflect.DeepEqual(target.Labels, baseline.Labels) {
		current.Labels = target.Labels
	}

	return current
}

func buildSchemaMap(config *storepb.DatabaseConfig) map[string]*storepb.SchemaCatalog {
	m := make(map[string]*storepb.SchemaCatalog)
	for _, v := range config.Schemas {
		m[v.Name] = v
	}
	return m
}

func buildTableMap(config *storepb.SchemaCatalog) map[string]*storepb.TableCatalog {
	m := make(map[string]*storepb.TableCatalog)
	for _, v := range config.Tables {
		m[v.Name] = v
	}
	return m
}

func buildProcedureMap(config *storepb.SchemaCatalog) map[string]*storepb.ProcedureConfig {
	m := make(map[string]*storepb.ProcedureConfig)
	for _, v := range config.ProcedureConfigs {
		m[v.Name] = v
	}
	return m
}

func buildFunctionMap(config *storepb.SchemaCatalog) map[string]*storepb.FunctionConfig {
	m := make(map[string]*storepb.FunctionConfig)
	for _, v := range config.FunctionConfigs {
		m[v.Name] = v
	}
	return m
}

func buildViewMap(config *storepb.SchemaCatalog) map[string]*storepb.ViewConfig {
	m := make(map[string]*storepb.ViewConfig)
	for _, v := range config.ViewConfigs {
		m[v.Name] = v
	}
	return m
}

func buildColumnMap(config *storepb.TableCatalog) map[string]*storepb.ColumnCatalog {
	m := make(map[string]*storepb.ColumnCatalog)
	for _, v := range config.Columns {
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
