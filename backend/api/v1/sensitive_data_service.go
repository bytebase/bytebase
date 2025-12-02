package v1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type SensitiveDataService struct {
	store *store.Store
}

func NewSensitiveDataService(store *store.Store) *SensitiveDataService {
	return &SensitiveDataService{
		store: store,
	}
}

type SensitiveColumnResult struct {
	Column          parserbase.ColumnResource
	Classification  string
	Level           string
	LevelTitle      string
	Sensitive       bool
}

type SensitiveDetectionResult struct {
	SensitiveColumns []SensitiveColumnResult
	HighestLevel     string
}

// DetectSensitiveColumns detects sensitive columns in the given query spans
func (s *SensitiveDataService) DetectSensitiveColumns(ctx context.Context, spans []*parserbase.QuerySpan, instance *store.InstanceMessage) (*SensitiveDetectionResult, error) {
	classificationSetting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find classification setting")
	}

	// Build level map for quick lookup
	levelMap := make(map[string]*storepb.DataClassificationSetting_Level)
	for _, config := range classificationSetting.Configs {
		for _, level := range config.Levels {
			levelMap[level.Id] = level
		}
	}

	result := &SensitiveDetectionResult{
		SensitiveColumns: []SensitiveColumnResult{},
		HighestLevel:     "",
	}

	// Level priority: high > medium > low
	levelPriority := map[string]int{
		"high":   3,
		"medium": 2,
		"low":    1,
	}

	for _, span := range spans {
		for _, spanResult := range span.Results {
			for column := range spanResult.SourceColumns {
				sensitiveColumn, err := s.detectSingleColumn(ctx, instance, column, classificationSetting, levelMap)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to detect sensitive column: %s", column.String())
				}

				if sensitiveColumn.Sensitive {
					result.SensitiveColumns = append(result.SensitiveColumns, sensitiveColumn)

					// Update highest level
					currentPriority := levelPriority[result.HighestLevel]
					detectedPriority := levelPriority[sensitiveColumn.Level]
					if detectedPriority > currentPriority {
						result.HighestLevel = sensitiveColumn.Level
					}
				}
			}
		}
	}

	return result, nil
}

func (s *SensitiveDataService) detectSingleColumn(
	ctx context.Context,
	instance *store.InstanceMessage,
	column parserbase.ColumnResource,
	classificationSetting *storepb.DataClassificationSetting,
	levelMap map[string]*storepb.DataClassificationSetting_Level,
) (SensitiveColumnResult, error) {
	result := SensitiveColumnResult{
		Column: column,
		Sensitive: false,
	}

	// Get database and column metadata
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &column.Database,
	})
	if err != nil {
		return result, errors.Wrapf(err, "failed to find database: %q", column.Database)
	}
	if database == nil {
		return result, nil
	}

	// Get column catalog and metadata
	meta, config, err := getColumnForColumnResource(ctx, s.store, instance.ResourceID, &column)
	if err != nil {
		return result, errors.Wrapf(err, "failed to get column metadata: %q", column.String())
	}
	if meta == nil || config == nil {
		return result, nil
	}

	// Check column classification
	if config.Classification != "" {
		// Find the classification config for this project
		var classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig
		for _, c := range classificationSetting.Configs {
			if c.Id == database.DataClassificationConfigID {
				classificationConfig = c
				break
			}
		}

		if classificationConfig != nil {
			class, ok := classificationConfig.Classification[config.Classification]
			if ok && class.LevelId != nil {
				levelID := *class.LevelId
				level, ok := levelMap[levelID]
				if ok {
					result.Classification = class.Title
					result.Level = levelID
					result.LevelTitle = level.Title
					result.Sensitive = true
				}
			}
		}
	}

	// If no column-level classification, check by column name pattern matching
	if !result.Sensitive {
		result = s.matchByPattern(column.Column, levelMap)
		result.Column = column
	}

	return result, nil
}

func (s *SensitiveDataService) matchByPattern(columnName string, levelMap map[string]*storepb.DataClassificationSetting_Level) SensitiveColumnResult {
	result := SensitiveColumnResult{
		Sensitive: false,
	}

	// Common sensitive column name patterns
	sensitivePatterns := map[string]string{
		// High sensitivity
		"password":     "high",
		"passwd":       "high",
		"secret":       "high",
		"token":        "high",
		"api_key":      "high",
		"apikey":       "high",
		"access_key":   "high",
		"private_key":  "high",
		// Medium sensitivity
		"phone":        "medium",
		"mobile":       "medium",
		"tel":          "medium",
		"email":        "medium",
		"email_address":"medium",
		"address":      "medium",
		"addr":         "medium",
		"name":         "medium",
		"username":     "medium",
		"user_name":    "medium",
		"id_card":      "medium",
		"idcard":       "medium",
		"ssn":          "medium",
		// Low sensitivity
		"created_at":   "low",
		"updated_at":   "low",
		"deleted_at":   "low",
	}

	for pattern, level := range sensitivePatterns {
		if containsIgnoreCase(columnName, pattern) {
			result.Level = level
			if levelInfo, ok := levelMap[level]; ok {
				result.LevelTitle = levelInfo.Title
			}
			result.Sensitive = true
			result.Classification = fmt.Sprintf("Pattern match: %s", pattern)
			break
		}
	}

	return result
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive match for common patterns
	sLower := toLower(s)
	substrLower := toLower(substr)
	return contains(sLower, substrLower)
}

func toLower(s string) string {
	// Simple ASCII-only to lower
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(s)] != substr && indexOf(s, substr) != -1
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Reuse the getColumnForColumnResource function from query_result_masker.go
func getColumnForColumnResource(ctx context.Context, store *store.Store, instanceID string, sourceColumn *parserbase.ColumnResource) (*storepb.ColumnMetadata, *storepb.ColumnCatalog, error) {
	if sourceColumn == nil {
		return nil, nil, nil
	}
	database, err := store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return nil, nil, nil
	}
	dbMetadata, err := store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database schema: %q", sourceColumn.Database)
	}
	if dbMetadata == nil {
		return nil, nil, nil
	}

	var columnMetadata *storepb.ColumnMetadata
	schema := dbMetadata.GetSchemaMetadata(sourceColumn.Schema)
	if schema == nil {
		return nil, nil, nil
	}
	table := schema.GetTable(sourceColumn.Table)
	if table == nil {
		return nil, nil, nil
	}
	column := table.GetColumn(sourceColumn.Column)
	if column == nil {
		return nil, nil, nil
	}
	columnMetadata = column.GetProto()

	columnConfig := column.GetCatalog()
	return columnMetadata, columnConfig, nil
}
