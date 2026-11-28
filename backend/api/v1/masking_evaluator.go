package v1

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

type maskingLevelEvaluator struct {
	maskingRules            []compiledMaskingRule
	usesPrincipalGroups     bool
	principalGroups         []string
	dataClassificationIDMap map[string]*storepb.DataClassificationSetting_DataClassificationConfig
	semanticTypesMap        map[string]*storepb.SemanticTypeSetting_SemanticType
}

type compiledMaskingRule struct {
	rule    *storepb.MaskingRulePolicy_MaskingRule
	program cel.Program
	err     error
}

// MaskingEvaluation contains the result of masking evaluation including the reason.
type MaskingEvaluation struct {
	SemanticTypeID      string
	SemanticTypeTitle   string
	SemanticTypeIcon    string
	MaskingRuleID       string
	Algorithm           string
	Context             string
	ClassificationLevel int32
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

	m.maskingRules = make([]compiledMaskingRule, len(maskingRulePolicy.Rules))
	m.usesPrincipalGroups = false
	if len(m.maskingRules) == 0 {
		return m
	}
	env, envErr := cel.NewEnv(common.MaskingRulePolicyCELAttributes...)
	for i, rule := range maskingRulePolicy.Rules {
		compiled := &m.maskingRules[i]
		compiled.rule = rule
		expression := rule.GetCondition().GetExpression()
		if expression == "" {
			continue
		}
		if envErr != nil {
			compiled.err = envErr
			continue
		}
		ast, issues := env.Compile(expression)
		if compiled.err = issues.Err(); compiled.err != nil {
			continue
		}
		for _, reference := range ast.NativeRep().ReferenceMap() {
			if reference.Name == common.CELAttributePrincipalGroups {
				m.usesPrincipalGroups = true
				break
			}
		}
		compiled.program, compiled.err = env.Program(ast)
	}

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
			semanticType := m.semanticTypesMap[semanticTypeID]
			context := ""
			if schemaName != "" {
				context = fmt.Sprintf("Column-level semantic type: %s.%s.%s.%s.%s", databaseMessage.InstanceID, databaseMessage.DatabaseName, schemaName, tableName, columnName)
			} else {
				context = fmt.Sprintf("Column-level semantic type: %s.%s.%s.%s", databaseMessage.InstanceID, databaseMessage.DatabaseName, tableName, columnName)
			}
			eval = &MaskingEvaluation{
				SemanticTypeID:    semanticTypeID,
				SemanticTypeTitle: semanticType.GetTitle(),
				SemanticTypeIcon:  semanticType.GetIcon(),
				Algorithm:         masker.AlgorithmName(semanticType.GetAlgorithm()),
				Context:           context,
			}
		}
	}

	if eval != nil && eval.SemanticTypeID != "" {
		dataClassificationConfig := m.getDataClassificationConfig(databaseProjectDataClassificationID)
		classificationLevel := getClassificationLevelOfColumn(columnConfig.GetClassification(), dataClassificationConfig)
		pass, err := evaluateExemptionOfColumn(databaseMessage, schemaName, tableName, columnName, classificationLevel, filteredMaskingExemptions)
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
	if len(m.maskingRules) == 0 {
		return nil, nil
	}
	dataClassificationConfig := m.getDataClassificationConfig(databaseProjectDataClassificationID)
	classificationLevel := getClassificationLevelOfColumn(columnConfig.GetClassification(), dataClassificationConfig)
	maskingRuleAttributes := map[string]any{
		common.CELAttributePrincipalGroups:             m.principalGroups,
		common.CELAttributeResourceEnvironmentID:       "",
		common.CELAttributeResourceProjectID:           databaseMessage.ProjectID,
		common.CELAttributeResourceInstanceID:          databaseMessage.InstanceID,
		common.CELAttributeResourceDatabaseName:        databaseMessage.DatabaseName,
		common.CELAttributeResourceSchemaName:          schemaName,
		common.CELAttributeResourceTableName:           tableName,
		common.CELAttributeResourceColumnName:          columnName,
		common.CELAttributeResourceClassificationLevel: int64(classificationLevel),
	}
	if databaseMessage.EffectiveEnvironmentID != nil {
		maskingRuleAttributes[common.CELAttributeResourceEnvironmentID] = *databaseMessage.EffectiveEnvironmentID
	}
	for _, compiled := range m.maskingRules {
		pass, err := compiled.matches(maskingRuleAttributes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to evaluate masking rule policy condition")
		}
		if pass {
			maskingRule := compiled.rule
			semanticTypeID := maskingRule.GetSemanticType()
			semanticType := m.semanticTypesMap[semanticTypeID]

			ruleTitle := ""
			if maskingRule.Condition != nil && maskingRule.Condition.Title != "" {
				ruleTitle = maskingRule.Condition.Title
			}
			if ruleTitle == "" {
				ruleTitle = maskingRule.Id
			}

			return &MaskingEvaluation{
				SemanticTypeID:      semanticTypeID,
				SemanticTypeTitle:   semanticType.GetTitle(),
				SemanticTypeIcon:    semanticType.GetIcon(),
				MaskingRuleID:       maskingRule.Id,
				Algorithm:           masker.AlgorithmName(semanticType.GetAlgorithm()),
				Context:             fmt.Sprintf("Global masking rule: %s", ruleTitle),
				ClassificationLevel: classificationLevel,
			}, nil
		}
	}
	return nil, nil
}

func evaluateExemptionOfColumn(databaseMessage *store.DatabaseMessage, schemaName, tableName, columnName string, classificationLevel int32, filteredMaskingExemptions []*storepb.MaskingExemptionPolicy_Exemption) (bool, error) {
	for _, filteredMaskingExemption := range filteredMaskingExemptions {
		maskingExemptionAttributes := map[string]any{
			"resource": map[string]any{
				"instance_id":          databaseMessage.InstanceID,
				"database_name":        databaseMessage.DatabaseName,
				"schema_name":          schemaName,
				"table_name":           tableName,
				"column_name":          columnName,
				"classification_level": int64(classificationLevel),
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

func getClassificationLevelOfColumn(columnClassificationID string, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) int32 {
	if columnClassificationID == "" || classificationConfig == nil {
		return 0
	}

	classification, ok := classificationConfig.Classification[columnClassificationID]
	if !ok {
		return 0
	}
	if classification.Level == nil {
		return 0
	}

	return *classification.Level
}

func evaluateMaskingExemptionPolicyCondition(expression *expr.Expr, attributes map[string]any) (bool, error) {
	// nil expression means allow to access all databases
	if expression == nil || expression.Expression == "" {
		return true, nil
	}
	maskingExemptionPolicyEnv, err := cel.NewEnv(
		cel.Variable("resource", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.AnyType)),
		ext.Strings(),
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

func (r compiledMaskingRule) matches(attributes map[string]any) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	if r.program == nil {
		return true, nil
	}
	out, _, err := r.program.Eval(attributes)
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
