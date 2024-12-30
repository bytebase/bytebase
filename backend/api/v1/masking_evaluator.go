package v1

import (
	"log/slog"
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type maskingLevelEvaluator struct {
	maskingRules            []*storepb.MaskingRulePolicy_MaskingRule
	dataClassificationIDMap map[string]*storepb.DataClassificationSetting_DataClassificationConfig
	semanticTypesMap        map[string]*storepb.SemanticTypeSetting_SemanticType
	maskingAlgorithms       map[string]*storepb.Algorithm
}

func newEmptyMaskingLevelEvaluator() *maskingLevelEvaluator {
	return &maskingLevelEvaluator{
		dataClassificationIDMap: make(map[string]*storepb.DataClassificationSetting_DataClassificationConfig),
		semanticTypesMap:        make(map[string]*storepb.SemanticTypeSetting_SemanticType),
		maskingAlgorithms:       make(map[string]*storepb.Algorithm),
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

// nolint
func (m *maskingLevelEvaluator) withDataClassificationSetting(dataClassification *storepb.DataClassificationSetting) *maskingLevelEvaluator {
	if dataClassification == nil {
		return m
	}
	for _, dataClassificationConfig := range dataClassification.Configs {
		m.dataClassificationIDMap[dataClassificationConfig.Id] = dataClassificationConfig
	}
	return m
}

// nolint
func (m *maskingLevelEvaluator) withSemanticTypeSetting(semanticTypeSetting *storepb.SemanticTypeSetting) *maskingLevelEvaluator {
	if semanticTypeSetting == nil {
		return m
	}
	for _, semanticType := range semanticTypeSetting.Types {
		m.semanticTypesMap[semanticType.Id] = semanticType
	}
	return m
}

// nolint
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

type maskingPolicyKey struct {
	schema string
	table  string
	column string
}

type maskData struct {
	Schema                    string
	Table                     string
	Column                    string
	MaskingLevel              storepb.MaskingLevel
	FullMaskingAlgorithmID    string
	PartialMaskingAlgorithmID string
}

// nolint
func (m *maskingLevelEvaluator) evaluateMaskingAlgorithmOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName, columnSemanticTypeID, columnClassification string, databaseProjectDataClassificationID string, maskingPolicyMap map[maskingPolicyKey]*maskData, filteredMaskingExceptions []*storepb.MaskingExceptionPolicy_MaskingException) (*storepb.Algorithm, storepb.MaskingLevel, error) {
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
			algorithmID = maskingData.PartialMaskingAlgorithmID
		case storepb.MaskingLevel_FULL:
			algorithmID = maskingData.FullMaskingAlgorithmID
		}
		if algorithmID != "" {
			if v, ok := m.maskingAlgorithms[algorithmID]; ok {
				return v, maskingLevel, nil
			}
			slog.Warn(
				"failed to find the masking algorithm",
				slog.String("algorithm", algorithmID),
				slog.String("schema", schemaName),
				slog.String("table", tableName),
				slog.String("column", columnName),
			)
			// If we cannot find the algorithm, we just log the warning and treat it is as none masking.
			return nil, storepb.MaskingLevel_NONE, nil
		}
	}

	if columnSemanticTypeID == "" {
		return nil, maskingLevel, nil
	}

	semanticType, ok := m.semanticTypesMap[columnSemanticTypeID]
	if !ok {
		slog.Warn(
			"failed to find the semantic type",
			slog.String("semantic_type", columnSemanticTypeID),
			slog.String("schema", schemaName),
			slog.String("table", tableName),
			slog.String("column", columnName),
		)
		return nil, storepb.MaskingLevel_NONE, nil
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
		slog.Warn(
			"failed to find the masking algorithm",
			slog.String("algorithm", algorithmID),
			slog.String("schema", schemaName),
			slog.String("table", tableName),
			slog.String("column", columnName),
		)
		// If we cannot find the algorithm, we just log the warning and treat it is as none masking.
		return nil, storepb.MaskingLevel_NONE, nil
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
func (m *maskingLevelEvaluator) evaluateMaskingLevelOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName, columnClassification string, databaseProjectDataClassificationID string, maskingPolicyMap map[maskingPolicyKey]*maskData, filteredMaskingExceptions []*storepb.MaskingExceptionPolicy_MaskingException) (storepb.MaskingLevel, error) {
	finalLevel := storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED

	key := maskingPolicyKey{
		schema: schemaName,
		table:  tableName,
		column: columnName,
	}
	maskingData, ok := maskingPolicyMap[key]
	if (!ok) || (maskingData.MaskingLevel == storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED) {
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
				break
			}
		}
	} else {
		finalLevel = maskingData.MaskingLevel
	}

	if finalLevel == storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED || finalLevel == storepb.MaskingLevel_NONE {
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
		hit, err := evaluateMaskingExceptionPolicyCondition(filteredMaskingException.Condition, maskingExceptionAttributes)
		if err != nil {
			return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to evaluate masking exception policy condition")
		}
		if hit {
			return storepb.MaskingLevel_NONE, nil
		}
	}
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

func evaluateMaskingExceptionPolicyCondition(expression *expr.Expr, attributes map[string]any) (bool, error) {
	// nil expression means allow to access all databases
	if expression == nil || expression.Expression == "" {
		return true, nil
	}
	maskingExceptionPolicyEnv, err := cel.NewEnv(
		cel.Variable("resource", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL environment for masking exception policy")
	}
	ast, issues := maskingExceptionPolicyEnv.Compile(expression.Expression)
	if issues != nil && issues.Err() != nil {
		return false, errors.Wrapf(issues.Err(), "failed to get the ast of CEL program for masking exception policy")
	}
	prg, err := maskingExceptionPolicyEnv.Program(ast)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL program for masking exception policy")
	}
	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, errors.Wrapf(err, "failed to eval CEL program for masking exception policy")
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result for masking exception policy")
	}
	boolVar, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "expect bool result for masking exception policy")
	}
	return boolVar, nil
}

func evaluateMaskingRulePolicyCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	maskingRulePolicyEnv, err := cel.NewEnv(common.MaskingRulePolicyCELAttributes...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL environment for masking rule policy")
	}
	ast, issues := maskingRulePolicyEnv.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, errors.Wrapf(issues.Err(), "failed to get the ast of CEL program for masking rule")
	}
	prg, err := maskingRulePolicyEnv.Program(ast)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL program for masking rule")
	}
	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, errors.Wrapf(err, "failed to eval CEL program for masking rule")
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result for masking rule")
	}
	boolVar, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "expect bool result for masking rule")
	}
	return boolVar, nil
}

func evaluateQueryExportPolicyCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	env, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
	if err != nil {
		return false, err
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, err
	}

	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, err
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result")
	}
	boolVal, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "failed to convert to bool")
	}
	return boolVal, nil
}
