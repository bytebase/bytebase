package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (s *QueryResultMasker) ExtractSensitivePredicateColumns(ctx context.Context, spans []*base.QuerySpan, instance *store.InstanceMessage, action storepb.MaskingExceptionPolicy_MaskingException_Action) ([][]base.ColumnResource, error) {
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

	maskingExceptionPolicyMap := make(map[string]*storepb.MaskingExceptionPolicy)

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	currentPrincipal, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find current principal")
	}
	if currentPrincipal == nil {
		return nil, status.Errorf(codes.Internal, "current principal not found")
	}

	for _, span := range spans {
		sensitiveColumns, err := s.getSensitiveColumnsForPredicate(
			ctx,
			m,
			instance,
			span.PredicateColumns,
			maskingExceptionPolicyMap,
			action,
			currentPrincipal,
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
	maskingExceptionPolicyMap map[string]*storepb.MaskingExceptionPolicy,
	action storepb.MaskingExceptionPolicy_MaskingException_Action,
	currentPrincipal *store.UserMessage,
) ([]base.ColumnResource, error) {
	if instance != nil && !isPredicateColumnsCheckEnabled(instance.Engine) {
		return nil, nil
	}

	var result []base.ColumnResource

	for column := range predicateColumns {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instance.ResourceID,
			DatabaseName: &column.Database,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", column.Database)
		}
		if database == nil {
			continue
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &database.ProjectID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get project %q", database.ProjectID)
		}
		if project == nil {
			continue
		}

		meta, config, err := s.getColumnForColumnResource(ctx, instance.ResourceID, &column)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database metadata for column resource: %q", column.String())
		}
		// Span and metadata are not the same in real time, so we fall back to none masker.
		if meta == nil {
			return nil, nil
		}

		if _, ok := maskingExceptionPolicyMap[database.ProjectID]; !ok {
			policy, err := s.store.GetMaskingExceptionPolicyByProjectUID(ctx, project.UID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find masking exception policy for project %q", project.ResourceID)
			}
			maskingExceptionPolicyMap[database.ProjectID] = policy
		}
		maskingExceptionPolicy := maskingExceptionPolicyMap[database.ProjectID]

		var maskingExceptionContainsCurrentPrincipal []*storepb.MaskingExceptionPolicy_MaskingException
		if maskingExceptionPolicy != nil {
			for _, maskingException := range maskingExceptionPolicy.MaskingExceptions {
				if maskingException.Action != action {
					continue
				}

				users := utils.GetUsersByMember(ctx, s.store, maskingException.Member)
				for _, user := range users {
					if user.ID == currentPrincipal.ID {
						maskingExceptionContainsCurrentPrincipal = append(maskingExceptionContainsCurrentPrincipal, maskingException)
						break
					}
				}
			}
		}

		isSensitive, err := m.isSensitiveColumn(database, column, project.DataClassificationConfigID, config, maskingExceptionContainsCurrentPrincipal)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check if column is sensitive")
		}
		if isSensitive {
			result = append(result, column)
		}
	}

	return result, nil
}

func (m *maskingLevelEvaluator) isSensitiveColumn(database *store.DatabaseMessage, column base.ColumnResource, classificationConfigID string, columnConfig *storepb.ColumnCatalog, exception []*storepb.MaskingExceptionPolicy_MaskingException) (bool, error) {
	semanticTypeID, err := m.evaluateSemanticTypeOfColumn(database, column.Schema, column.Table, column.Column, classificationConfigID, columnConfig, exception)
	if err != nil {
		return false, errors.Wrapf(err, "failed to evaluate semantic type of column")
	}
	return semanticTypeID != "", nil
}

func isPredicateColumnsCheckEnabled(engine storepb.Engine) bool {
	return engine == storepb.Engine_MSSQL
}
