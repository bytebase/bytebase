package v1

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func (s *QueryResultMasker) ExtractSensitivePredicateColumns(ctx context.Context, spans []*base.QuerySpan, instance *store.InstanceMessage, user *store.UserMessage) ([][]base.ColumnResource, error) {
	var result [][]base.ColumnResource

	classificationSetting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find classification setting")
	}

	maskingRulePolicy, err := s.store.GetMaskingRulePolicy(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find masking rule policy")
	}

	semanticTypesSetting, err := s.store.GetSemanticTypesSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find semantic types setting")
	}

	m := newEmptyMaskingLevelEvaluator().
		withMaskingRulePolicy(maskingRulePolicy).
		withDataClassificationSetting(classificationSetting).
		withSemanticTypeSetting(semanticTypesSetting)

	// Collect all predicate columns from all spans for batch fetching.
	allColumns := make(base.SourceColumnSet)
	for _, span := range spans {
		for col := range span.PredicateColumns {
			allColumns[col] = true
		}
	}

	data, err := newMaskingDataProviderFromColumns(ctx, s.store, instance, allColumns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize masking data provider")
	}

	for _, span := range spans {
		sensitiveColumns, err := s.getSensitiveColumnsForPredicate(
			ctx,
			m,
			instance,
			span.PredicateColumns,
			data,
			user,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sensitive columns for predicate")
		}
		result = append(result, sensitiveColumns)
	}

	return result, nil
}

func (s *QueryResultMasker) getSensitiveColumnsForPredicate(
	ctx context.Context,
	m *maskingLevelEvaluator,
	instance *store.InstanceMessage,
	predicateColumns base.SourceColumnSet,
	data *maskingDataProvider,
	currentPrincipal *store.UserMessage,
) ([]base.ColumnResource, error) {
	if instance != nil && !isPredicateColumnsCheckEnabled(instance.Metadata.GetEngine()) {
		return nil, nil
	}

	var result []base.ColumnResource

	for column := range predicateColumns {
		database := data.getDatabase(column.Database)
		if database == nil {
			continue
		}

		project := data.getProject(database.ProjectID)
		if project == nil {
			continue
		}

		columnMeta, config := data.getColumn(&column)
		if columnMeta == nil {
			continue
		}

		var exemptions []*storepb.MaskingExemptionPolicy_Exemption
		if policy := data.getMaskingExemptionPolicy(database.ProjectID); policy != nil {
			for _, e := range policy.Exemptions {
				for _, member := range e.Members {
					if utils.MemberContainsUser(ctx, s.store, member, currentPrincipal) {
						exemptions = append(exemptions, e)
						break
					}
				}
			}
		}

		isSensitive, err := m.isSensitiveColumn(database, column, project.Setting.DataClassificationConfigId, config, exemptions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check if column is sensitive")
		}
		if isSensitive {
			result = append(result, column)
		}
	}

	return result, nil
}

func (m *maskingLevelEvaluator) isSensitiveColumn(database *store.DatabaseMessage, column base.ColumnResource, classificationConfigID string, columnConfig *storepb.ColumnCatalog, exception []*storepb.MaskingExemptionPolicy_Exemption) (bool, error) {
	evaluation, err := m.evaluateSemanticTypeOfColumn(database, column.Schema, column.Table, column.Column, classificationConfigID, columnConfig, exception)
	if err != nil {
		return false, errors.Wrapf(err, "failed to evaluate semantic type of column")
	}
	return evaluation != nil && evaluation.SemanticTypeID != "", nil
}

func isPredicateColumnsCheckEnabled(engine storepb.Engine) bool {
	return engine == storepb.Engine_MSSQL
}
