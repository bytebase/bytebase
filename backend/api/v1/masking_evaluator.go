package v1

import (
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

// TODO(d): redefine default algorithm later.
var (
	defaultFullAlgorithm = &storepb.Algorithm{
		Id: "default",
		Mask: &storepb.Algorithm_FullMask_{FullMask: &storepb.Algorithm_FullMask{
			Substitution: "******",
		}},
	}
	defaultPartialAlgorithm = &storepb.Algorithm{
		Id: "default",
		Mask: &storepb.Algorithm_RangeMask_{RangeMask: &storepb.Algorithm_RangeMask{
			Slices: []*storepb.Algorithm_RangeMask_Slice{
				{
					Start:        0,
					End:          4,
					Substitution: "****",
				},
			},
		}},
	}
)

// nolint
func (m *maskingLevelEvaluator) evaluateMaskingAlgorithmOfColumn(
	databaseMessage *store.DatabaseMessage,
	schemaName, tableName, columnName,
	databaseProjectDataClassificationID string,
	columnConfig *storepb.ColumnCatalog,
	filteredMaskingExceptions []*storepb.MaskingExceptionPolicy_MaskingException,
) (*storepb.Algorithm, storepb.MaskingLevel, error) {
	// TODO(d): deprecate globalMaskingLevel.
	globalMaskingLevel, err := m.evaluateGlobalMaskingLevelOfColumn(databaseMessage, schemaName, tableName, columnName, databaseProjectDataClassificationID, columnConfig)
	if err != nil {
		return nil, storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to evaluate masking level of column")
	}

	fullAlgorithmID, partialAlgorithmID := columnConfig.GetFullMaskingAlgorithmId(), columnConfig.GetPartialMaskingAlgorithmId()
	if fullAlgorithmID == "" && partialAlgorithmID == "" && columnConfig.GetSemanticTypeId() != "" {
		semanticType, ok := m.semanticTypesMap[columnConfig.GetSemanticTypeId()]
		if ok {
			if semanticType.GetFullMaskAlgorithmId() != "" {
				fullAlgorithmID = semanticType.GetFullMaskAlgorithmId()
			}
			if semanticType.GetPartialMaskAlgorithmId() != "" {
				partialAlgorithmID = semanticType.GetPartialMaskAlgorithmId()
			}
		}
	}
	var fullAlgorithm, partialAlgorithm *storepb.Algorithm
	if fullAlgorithmID != "" {
		fullAlgorithm = m.maskingAlgorithms[fullAlgorithmID]
	}
	if partialAlgorithmID != "" {
		partialAlgorithm = m.maskingAlgorithms[partialAlgorithmID]
	}
	if fullAlgorithm == nil {
		fullAlgorithm = defaultFullAlgorithm
	}
	if partialAlgorithm == nil {
		partialAlgorithm = defaultPartialAlgorithm
	}

	switch globalMaskingLevel {
	case storepb.MaskingLevel_NONE:
		return nil, storepb.MaskingLevel_NONE, nil
	case storepb.MaskingLevel_PARTIAL, storepb.MaskingLevel_FULL:
		pass, err := evaluateExceptionOfColumn(databaseMessage, schemaName, tableName, columnName, filteredMaskingExceptions)
		if err != nil {
			return nil, storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, err
		}
		if pass {
			return nil, storepb.MaskingLevel_NONE, nil
		}
		if globalMaskingLevel == storepb.MaskingLevel_FULL {
			return fullAlgorithm, globalMaskingLevel, nil
		}
		if globalMaskingLevel == storepb.MaskingLevel_PARTIAL {
			return partialAlgorithm, globalMaskingLevel, nil
		}
	}

	switch columnConfig.GetMaskingLevel() {
	case storepb.MaskingLevel_NONE:
		return nil, storepb.MaskingLevel_NONE, nil
	case storepb.MaskingLevel_FULL:
		return fullAlgorithm, storepb.MaskingLevel_FULL, nil
	case storepb.MaskingLevel_PARTIAL:
		return partialAlgorithm, storepb.MaskingLevel_PARTIAL, nil
	}
	return nil, storepb.MaskingLevel_NONE, nil
}

func (m *maskingLevelEvaluator) evaluateGlobalMaskingLevelOfColumn(
	databaseMessage *store.DatabaseMessage,
	schemaName, tableName, columnName string,
	databaseProjectDataClassificationID string,
	columnConfig *storepb.ColumnCatalog,
) (storepb.MaskingLevel, error) {
	dataClassificationConfig := m.getDataClassificationConfig(databaseProjectDataClassificationID)
	// If the column has DEFAULT masking level in maskingPolicy or not set yet,
	// we will eval the maskingRulePolicy to get the maskingLevel.
	classificationLevel := getClassificationLevelOfColumn(columnConfig.GetClassificationId(), dataClassificationConfig)
	for _, maskingRule := range m.maskingRules {
		maskingRuleAttributes := map[string]any{
			"environment_id":       databaseMessage.EffectiveEnvironmentID,
			"project_id":           databaseMessage.ProjectID,
			"instance_id":          databaseMessage.InstanceID,
			"database_name":        databaseMessage.DatabaseName,
			"schema_name":          schemaName,
			"table_name":           tableName,
			"column_name":          columnName,
			"classification_level": classificationLevel,
		}
		pass, err := evaluateMaskingRulePolicyCondition(maskingRule.Condition.Expression, maskingRuleAttributes)
		if err != nil {
			return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, errors.Wrapf(err, "failed to evaluate masking rule policy condition")
		}
		if pass {
			return maskingRule.MaskingLevel, nil
		}
	}
	return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED, nil
}

func evaluateExceptionOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName string, filteredMaskingExceptions []*storepb.MaskingExceptionPolicy_MaskingException) (bool, error) {
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
		pass, err := evaluateMaskingExceptionPolicyCondition(filteredMaskingException.Condition, maskingExceptionAttributes)
		if err != nil {
			return false, errors.Wrapf(err, "failed to evaluate masking exception policy condition")
		}
		if pass {
			return true, nil
		}
	}
	return false, nil
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
