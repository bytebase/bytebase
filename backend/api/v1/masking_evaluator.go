package v1

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type maskingLevelEvaluator struct {
	maskingRules            []*storepb.MaskingRulePolicy_MaskingRule
	dataClassificationIDMap map[string]*storepb.DataClassificationSetting_DataClassificationConfig
	semanticTypesMap        map[string]*storepb.SemanticTypeSetting_SemanticType
}

// MaskingEvaluation contains the result of masking evaluation including the reason.
type MaskingEvaluation struct {
	SemanticTypeID      string
	SemanticTypeTitle   string
	SemanticTypeIcon    string
	MaskingRuleID       string
	Algorithm           string
	Context             string
	ClassificationLevel string
}

func newEmptyMaskingLevelEvaluator() *maskingLevelEvaluator {
	return &maskingLevelEvaluator{
		dataClassificationIDMap: make(map[string]*storepb.DataClassificationSetting_DataClassificationConfig),
		semanticTypesMap:        make(map[string]*storepb.SemanticTypeSetting_SemanticType),
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

func (m *maskingLevelEvaluator) getDataClassificationConfig(classificationID string) *storepb.DataClassificationSetting_DataClassificationConfig {
	return m.dataClassificationIDMap[classificationID]
}

// nolint
func (m *maskingLevelEvaluator) evaluateSemanticTypeOfColumn(
	databaseMessage *store.DatabaseMessage,
	schemaName, tableName, columnName,
	databaseProjectDataClassificationID string,
	columnConfig *storepb.ColumnCatalog,
	filteredMaskingExemptions []*storepb.MaskingExemptionPolicy_Exemption,
) (*MaskingEvaluation, error) {
	eval, err := m.evaluateGlobalMaskingLevelOfColumn(databaseMessage, schemaName, tableName, columnName, databaseProjectDataClassificationID, columnConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to evaluate masking level of column")
	}

	if eval == nil || eval.SemanticTypeID == "" {
		// Check column-level semantic type
		semanticTypeID := columnConfig.GetSemanticType()
		if semanticTypeID != "" {
			semanticType, ok := m.semanticTypesMap[semanticTypeID]
			if ok {
				context := ""
				if schemaName != "" {
					context = fmt.Sprintf("Column-level semantic type: %s.%s.%s.%s.%s", databaseMessage.InstanceID, databaseMessage.DatabaseName, schemaName, tableName, columnName)
				} else {
					context = fmt.Sprintf("Column-level semantic type: %s.%s.%s.%s", databaseMessage.InstanceID, databaseMessage.DatabaseName, tableName, columnName)
				}
				algorithmName := getAlgorithmNameFromSemanticType(semanticType)
				eval = &MaskingEvaluation{
					SemanticTypeID:    semanticTypeID,
					SemanticTypeTitle: semanticType.Title,
					SemanticTypeIcon:  semanticType.Icon,
					Algorithm:         algorithmName,
					Context:           context,
				}
			}
		}
	}

	if eval != nil && eval.SemanticTypeID != "" {
		pass, err := evaluateExemptionOfColumn(databaseMessage, schemaName, tableName, columnName, filteredMaskingExemptions)
		if err != nil {
			return nil, err
		}
		if pass {
			return nil, nil
		}
	}
	return eval, nil
}

func (m *maskingLevelEvaluator) evaluateGlobalMaskingLevelOfColumn(
	databaseMessage *store.DatabaseMessage,
	schemaName, tableName, columnName string,
	databaseProjectDataClassificationID string,
	columnConfig *storepb.ColumnCatalog,
) (*MaskingEvaluation, error) {
	dataClassificationConfig := m.getDataClassificationConfig(databaseProjectDataClassificationID)
	// If the column has DEFAULT masking level in maskingPolicy or not set yet,
	// we will eval the maskingRulePolicy to get the maskingLevel.
	classificationLevel := getClassificationLevelOfColumn(columnConfig.GetClassification(), dataClassificationConfig)
	for _, maskingRule := range m.maskingRules {
		maskingRuleAttributes := map[string]any{
			common.CELAttributeResourceEnvironmentID:       "",
			common.CELAttributeResourceProjectID:           databaseMessage.ProjectID,
			common.CELAttributeResourceInstanceID:          databaseMessage.InstanceID,
			common.CELAttributeResourceDatabaseName:        databaseMessage.DatabaseName,
			common.CELAttributeResourceSchemaName:          schemaName,
			common.CELAttributeResourceTableName:           tableName,
			common.CELAttributeResourceColumnName:          columnName,
			common.CELAttributeResourceClassificationLevel: classificationLevel,
		}
		if databaseMessage.EffectiveEnvironmentID != nil {
			maskingRuleAttributes[common.CELAttributeResourceEnvironmentID] = *databaseMessage.EffectiveEnvironmentID
		}
		pass, err := evaluateMaskingRulePolicyCondition(maskingRule.Condition.Expression, maskingRuleAttributes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate masking rule policy condition")
		}
		if pass {
			semanticTypeID := maskingRule.GetSemanticType()
			semanticType, ok := m.semanticTypesMap[semanticTypeID]
			if !ok {
				ruleTitle := ""
				if maskingRule.Condition != nil && maskingRule.Condition.Title != "" {
					ruleTitle = maskingRule.Condition.Title
				}
				return &MaskingEvaluation{
					SemanticTypeID:      semanticTypeID,
					MaskingRuleID:       maskingRule.Id,
					Context:             fmt.Sprintf("Global masking rule: %s", ruleTitle),
					ClassificationLevel: classificationLevel,
				}, nil
			}
			ruleTitle := ""
			if maskingRule.Condition != nil && maskingRule.Condition.Title != "" {
				ruleTitle = maskingRule.Condition.Title
			}
			// Get algorithm name from semantic type
			algorithmName := getAlgorithmNameFromSemanticType(semanticType)
			return &MaskingEvaluation{
				SemanticTypeID:      semanticTypeID,
				SemanticTypeTitle:   semanticType.Title,
				SemanticTypeIcon:    semanticType.Icon,
				MaskingRuleID:       maskingRule.Id,
				Algorithm:           algorithmName,
				Context:             fmt.Sprintf("Global masking rule: %s", ruleTitle),
				ClassificationLevel: classificationLevel,
			}, nil
		}
	}
	return nil, nil
}

func evaluateExemptionOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName string, filteredMaskingExemptions []*storepb.MaskingExemptionPolicy_Exemption) (bool, error) {
	for _, filteredMaskingExemption := range filteredMaskingExemptions {
		maskingExemptionAttributes := map[string]any{
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
		pass, err := evaluateMaskingExemptionPolicyCondition(filteredMaskingExemption.Condition, maskingExemptionAttributes)
		if err != nil {
			return false, errors.Wrapf(err, "failed to evaluate masking exemption policy condition")
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

func evaluateMaskingExemptionPolicyCondition(expression *expr.Expr, attributes map[string]any) (bool, error) {
	// nil expression means allow to access all databases
	if expression == nil || expression.Expression == "" {
		return true, nil
	}
	maskingExemptionPolicyEnv, err := cel.NewEnv(
		cel.Variable("resource", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL environment for masking exemption policy")
	}
	ast, issues := maskingExemptionPolicyEnv.Compile(expression.Expression)
	if issues != nil && issues.Err() != nil {
		return false, errors.Wrapf(issues.Err(), "failed to get the ast of CEL program for masking exemption policy")
	}
	prg, err := maskingExemptionPolicyEnv.Program(ast)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create CEL program for masking exemption policy")
	}
	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, errors.Wrapf(err, "failed to eval CEL program for masking exemption policy")
	}
	val, err := out.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return false, errors.Wrap(err, "expect bool result for masking exemption policy")
	}
	boolVar, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "expect bool result for masking exemption policy")
	}
	return boolVar, nil
}

func getAlgorithmNameFromSemanticType(semanticType *storepb.SemanticTypeSetting_SemanticType) string {
	if semanticType == nil || semanticType.Algorithm == nil {
		return ""
	}

	switch semanticType.Algorithm.Mask.(type) {
	case *storepb.Algorithm_FullMask_:
		return "Full mask"
	case *storepb.Algorithm_RangeMask_:
		return "Partial mask"
	case *storepb.Algorithm_Md5Mask:
		return "Hash (MD5)"
	case *storepb.Algorithm_InnerOuterMask_:
		return "Inner/Outer mask"
	default:
		return "Unknown"
	}
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
	val, err := out.ConvertToNative(reflect.TypeFor[bool]())
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
	val, err := out.ConvertToNative(reflect.TypeFor[bool]())
	if err != nil {
		return false, errors.Wrap(err, "expect bool result")
	}
	boolVal, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "failed to convert to bool")
	}
	return boolVal, nil
}
