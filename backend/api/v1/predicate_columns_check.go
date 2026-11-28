package v1

import (
	"context"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// sensitivePredicateRefusal returns the message to report for statements whose
// conditions read a column masked for user, or "" if none do. The database
// evaluates conditions on unmasked values, so the rows it returns, their
// count, and the time it takes would reveal them.
func (s *queryResultMasker) sensitivePredicateRefusal(ctx context.Context, spans []*base.QuerySpan, instance *store.InstanceMessage) (string, error) {
	if !isPredicateColumnsCheckEnabled(instance.Metadata.GetEngine()) {
		return "", nil
	}
	sensitiveColumns, err := s.extractSensitivePredicateColumns(ctx, spans, instance)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, errors.New(err.Error()))
	}
	for _, columns := range sensitiveColumns {
		if len(columns) == 0 {
			continue
		}
		names := make([]string, 0, len(columns))
		for _, column := range columns {
			names = append(names, column.String())
		}
		slices.Sort(names)
		return "Using sensitive columns in query conditions is not allowed: " + strings.Join(names, ", "), nil
	}
	return "", nil
}

func (s *queryResultMasker) extractSensitivePredicateColumns(ctx context.Context, spans []*base.QuerySpan, instance *store.InstanceMessage) ([][]base.ColumnResource, error) {
	var result [][]base.ColumnResource

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
			instance,
			span.PredicateColumns,
			data,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sensitive columns for predicate")
		}
		result = append(result, sensitiveColumns)
	}

	return result, nil
}

func (s *queryResultMasker) getSensitiveColumnsForPredicate(
	ctx context.Context,
	instance *store.InstanceMessage,
	predicateColumns base.SourceColumnSet,
	data *maskingDataProvider,
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

		exemptions := s.exemptionsForPrincipal(ctx, data, database.ProjectID)

		isSensitive, err := s.evaluator.isSensitiveColumn(database, column, project.Setting.DataClassificationConfigId, config, exemptions)
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
	switch engine {
	case storepb.Engine_MSSQL, storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		return true
	default:
		return false
	}
}
