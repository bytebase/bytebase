package v1

import (
	"cmp"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type maskingLevelEvaluator struct {
	maskingRules            []*storepb.MaskingRulePolicy_MaskingRule
	dataClassificationIDMap map[string]*storepb.DataClassificationSetting_DataClassificationConfig
	semanticTypesMap        map[string]*storepb.SemanticTypeSetting_SemanticType
	maskingAlgorithms       map[string]*storepb.MaskingAlgorithmSetting_Algorithm
}

func newEmptyMaskingLevelEvaluator() *maskingLevelEvaluator {
	return &maskingLevelEvaluator{
		dataClassificationIDMap: make(map[string]*storepb.DataClassificationSetting_DataClassificationConfig),
		semanticTypesMap:        make(map[string]*storepb.SemanticTypeSetting_SemanticType),
		maskingAlgorithms:       make(map[string]*storepb.MaskingAlgorithmSetting_Algorithm),
	}
}

func (m *maskingLevelEvaluator) withMaskingRulePolicy(maskingRulePolicy *storepb.MaskingRulePolicy) *maskingLevelEvaluator {
	if maskingRulePolicy == nil {
		return m
	}

	m.maskingRules = make([]*storepb.MaskingRulePolicy_MaskingRule, 0, len(maskingRulePolicy.Rules))
	m.maskingRules = append(m.maskingRules, maskingRulePolicy.Rules...)

	return m
}

func (m *maskingLevelEvaluator) withDataClassificationSetting(dataClassification *storepb.DataClassificationSetting) *maskingLevelEvaluator {
	if dataClassification == nil {
		return m
	}
	for _, dataClassificationConfig := range dataClassification.Configs {
		m.dataClassificationIDMap[dataClassificationConfig.Id] = dataClassificationConfig
	}
	return m
}

func (m *maskingLevelEvaluator) withSemanticTypeSetting(semanticTypeSetting *storepb.SemanticTypeSetting) *maskingLevelEvaluator {
	if semanticTypeSetting == nil {
		return m
	}
	for _, semanticType := range semanticTypeSetting.Types {
		m.semanticTypesMap[semanticType.Id] = semanticType
	}
	return m
}

func (m *maskingLevelEvaluator) withMaskingAlgorithmSetting(maskingAlgorithmSetting *storepb.MaskingAlgorithmSetting) *maskingLevelEvaluator {
	if maskingAlgorithmSetting == nil {
		return m
	}
	for _, maskingAlgorithm := range maskingAlgorithmSetting.Algorithms {
		m.maskingAlgorithms[maskingAlgorithm.Id] = maskingAlgorithm
	}
	return m
}

func (m *maskingLevelEvaluator) getDataClassificationConfig(classificationID string) *storepb.DataClassificationSetting_DataClassificationConfig {
	return m.dataClassificationIDMap[classificationID]
}

func (m *maskingLevelEvaluator) evaluateMaskingAlgorithmOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName, columnSemanticTypeID, columnClassification string, databaseProjectDataClassificationID string, maskingPolicyMap map[maskingPolicyKey]*storepb.MaskData, filteredMaskingExceptions []*storepb.MaskingExceptionPolicy_MaskingException) (*storepb.MaskingAlgorithmSetting_Algorithm, storepb.MaskingLevel, error) {
	maskingLevel, err := m.evaluateMaskingLevelOfColumn(databaseMessage, schemaName, tableName, columnName, columnClassification, databaseProjectDataClassificationID, maskingPolicyMap, filteredMaskingExceptions)
	if err != nil {
		return nil, storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to evaluate masking level of column")
	}
	if maskingLevel == storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED || maskingLevel == storepb.MaskingLevel_NONE {
		return nil, maskingLevel, nil
	}
	key := maskingPolicyKey{
		schema: schemaName,
		table:  tableName,
		column: columnName,
	}
	if maskingData, ok := maskingPolicyMap[key]; ok {
		algorithmID := ""
		switch maskingLevel {
		case storepb.MaskingLevel_PARTIAL:
			algorithmID = maskingData.PartialMaskingAlgorithmId
		case storepb.MaskingLevel_FULL:
			algorithmID = maskingData.FullMaskingAlgorithmId
		}
		if algorithmID != "" {
			if v, ok := m.maskingAlgorithms[algorithmID]; ok {
				return v, maskingLevel, nil
			}
			// If we cannot find the algorithm, we will return the masking level and a error message to the caller.
			return nil, maskingLevel, errors.Errorf("failed to find the masking algorithm %q", algorithmID)
		}
	}

	semanticType, ok := m.semanticTypesMap[columnSemanticTypeID]
	if !ok {
		return nil, maskingLevel, errors.Errorf("failed to find the semantic type %q", columnSemanticTypeID)
	}
	algorithmID := ""
	switch maskingLevel {
	case storepb.MaskingLevel_PARTIAL:
		algorithmID = semanticType.PartialMaskAlgorithmId
	case storepb.MaskingLevel_FULL:
		algorithmID = semanticType.FullMaskAlgorithmId
	}
	if algorithmID != "" {
		if v, ok := m.maskingAlgorithms[algorithmID]; ok {
			return v, maskingLevel, nil
		}
		// If we cannot find the algorithm, we will return the masking level and a error message to the caller.
		return nil, maskingLevel, errors.Errorf("failed to find the masking algorithm %q", algorithmID)
	}

	return nil, maskingLevel, nil
}

// evaluateMaskingLevelOfColumn evaluates the masking level of the given column.
//
// Args:
//
// - databaseMessage: the database message for the column belongs to.
//
// - databaseName / schemaName / tableName: the database / schema / table name for the column belongs to, schema can be empty likes in MySQL.
//
// - column: the column metadata.
//
// - databaseProjectDataClassificationID: the data classification id of the project the database belongs to.
//
// - maskingPolicyMap: the map of maskingPolicy for the database column belongs to.
//
// - filteredMaskingExceptions: the exceptions should apply for current principal.
func (m *maskingLevelEvaluator) evaluateMaskingLevelOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName, columnClassification string, databaseProjectDataClassificationID string, maskingPolicyMap map[maskingPolicyKey]*storepb.MaskData, filteredMaskingExceptions []*storepb.MaskingExceptionPolicy_MaskingException) (storepb.MaskingLevel, error) {
	finalLevel := storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED

	key := maskingPolicyKey{
		schema: schemaName,
		table:  tableName,
		column: columnName,
	}
	maskingData, ok := maskingPolicyMap[key]
	if (!ok) || (maskingData.MaskingLevel == storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED) {
		slog.Debug("column set DEFAULT masking level in masking policy or masking policy not set yet", slog.String("column", columnName), slog.Any("masking policy", maskingData))
		dataClassificationConfig := m.getDataClassificationConfig(databaseProjectDataClassificationID)
		// If the column has DEFAULT masking level in maskingPolicy or not set yet,
		// we will eval the maskingRulePolicy to get the maskingLevel.
		columnClassificationLevel := getClassificationLevelOfColumn(columnClassification, dataClassificationConfig)
		for _, maskingRule := range m.maskingRules {
			maskingRuleAttributes := map[string]any{
				"environment_id":       databaseMessage.EffectiveEnvironmentID,
				"project_id":           databaseMessage.ProjectID,
				"instance_id":          databaseMessage.InstanceID,
				"database_name":        databaseMessage.DatabaseName,
				"schema_name":          schemaName,
				"table_name":           tableName,
				"column_name":          columnName,
				"classification_level": columnClassificationLevel,
			}
			pass, err := evaluateMaskingRulePolicyCondition(maskingRule.Condition.Expression, maskingRuleAttributes)
			if err != nil {
				return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to evaluate masking rule policy condition")
			}
			if pass {
				finalLevel = maskingRule.MaskingLevel
				slog.Debug("hit masking rule", slog.String("column", columnName), slog.Any("masking rule", maskingRule), slog.Any("masking level", maskingRule.MaskingLevel.String()))
				break
			}
		}
	} else {
		slog.Debug("column set specific masking level in masking policy", slog.String("column", columnName), slog.Any("masking level", maskingData.MaskingLevel.String()))
		finalLevel = maskingData.MaskingLevel
	}

	if finalLevel == storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED || finalLevel == storepb.MaskingLevel_NONE {
		// After looking up the maskingPolicy and maskingRulePolicy, if the maskingLevel is still MASKING_LEVEL_UNSPECIFIED or NONE,
		// return the MASKING_LEVEL_NONE, which means no masking and do not need eval exceptions anymore.
		slog.Debug("After looking up maskingPolicy and maskingRulePolicy, the masking level is UNSPECIFIED or NONE", slog.Any("masking level", finalLevel.String()))
		return storepb.MaskingLevel_NONE, nil
	}

	// If the column has PARTIAL/FULL masking level,
	// try to find the MaskingExceptionPolicy for current principal, return the minimum level of two.
	// If there is no MaskingExceptionPolicy for current principal, return the masking level in maskingPolicy.
	for _, filteredMaskingException := range filteredMaskingExceptions {
		maskingExceptionAttributes := map[string]any{
			"resource": map[string]any{
				"instance_id":   databaseMessage.InstanceID,
				"database_name": databaseMessage.DatabaseName,
				"schema_name":   schemaName,
				"table_name":    tableName,
				"column_name":   columnName,
			},
			"request": map[string]any{
				"time": time.Now(),
			},
		}
		hit, err := evaluateMaskingExceptionPolicyCondition(filteredMaskingException.Condition.Expression, maskingExceptionAttributes)
		if err != nil {
			return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to evaluate masking exception policy condition")
		}
		if !hit {
			continue
		}
		slog.Debug("hit masking exception", slog.String("column", columnName), slog.Any("masking exception", filteredMaskingException), slog.Any("masking level", filteredMaskingException.MaskingLevel.String()))
		// TODO(zp): Expectedly, a column should hit only one exception,
		// but we can take the strictest level here to make the whole program more robust.
		if cmp.Less[storepb.MaskingLevel](filteredMaskingException.MaskingLevel, finalLevel) {
			finalLevel = filteredMaskingException.MaskingLevel
		}
	}
	slog.Debug("final level of column", slog.String("column", columnName), slog.Any("final level", finalLevel.String()))
	return finalLevel, nil
}

func getClassificationLevelOfColumn(columnClassificationID string, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) string {
	if columnClassificationID == "" || classificationConfig == nil {
		return ""
	}

	classification, ok := classificationConfig.Classification[columnClassificationID]
	if !ok {
		return ""
	}
	if classification.LevelId == nil {
		return ""
	}

	return *classification.LevelId
}
