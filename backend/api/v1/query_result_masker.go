package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type QueryResultMasker struct {
	store *store.Store
}

func NewQueryResultMasker(store *store.Store) *QueryResultMasker {
	return &QueryResultMasker{
		store: store,
	}
}

// MaskResults masks the result in-place based on the dynamic masking policy, query-span, instance and action.
func (s *QueryResultMasker) MaskResults(ctx context.Context, spans []*base.QuerySpan, results []*v1pb.QueryResult, instance *store.InstanceMessage, action storepb.MaskingExceptionPolicy_MaskingException_Action) error {
	classificationSetting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find classification setting")
	}

	maskingRulePolicy, err := s.store.GetMaskingRulePolicy(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find masking rule policy")
	}

	algorithmSetting, err := s.store.GetMaskingAlgorithmSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find masking algorithm setting")
	}

	semanticTypesSetting, err := s.store.GetSemanticTypesSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find semantic types setting")
	}

	m := newEmptyMaskingLevelEvaluator().
		withMaskingRulePolicy(maskingRulePolicy).
		withDataClassificationSetting(classificationSetting).
		withMaskingAlgorithmSetting(algorithmSetting).
		withSemanticTypeSetting(semanticTypesSetting)

	// We expect the len(spans) == len(results), but to avoid NPE, we use the min(len(spans), len(results)) here.
	loopBoundary := min(len(spans), len(results))
	for i := 0; i < loopBoundary; i++ {
		maskers, err := s.getMaskersForQuerySpan(ctx, m, instance, spans[i], action)
		if err != nil {
			return errors.Wrapf(err, "failed to get maskers for query span")
		}
		mask(maskers, results[i])
	}

	return nil
}

// getMaskersForQuerySpan returns the maskers for the query span.
func (s *QueryResultMasker) getMaskersForQuerySpan(ctx context.Context, m *maskingLevelEvaluator, instance *store.InstanceMessage, span *base.QuerySpan, action storepb.MaskingExceptionPolicy_MaskingException_Action) ([]masker.Masker, error) {
	if span == nil {
		return nil, nil
	}
	maskers := make([]masker.Masker, 0, len(span.Results))

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	currentPrincipal, err := s.store.GetUser(ctx, &store.FindUserMessage{
		ID: &principalID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find current principal")
	}
	if currentPrincipal == nil {
		return nil, status.Errorf(codes.Internal, "current principal not found")
	}

	// Multiple databases may belong to the same project, to reduce the protojson unmarshal cost,
	// we store the projectResourceID - maskingExceptionPolicy in a map.
	maskingExceptionPolicyMap := make(map[string]*storepb.MaskingExceptionPolicy)

	for _, spanResult := range span.Results {
		// Likes constant expression, we use the none masker.
		if len(spanResult.SourceColumns) == 0 {
			maskers = append(maskers, masker.NewNoneMasker())
			continue
		}

		var effectiveMaskers []masker.Masker
		for column := range spanResult.SourceColumns {
			newMasker, err := s.getMasterForColumnResource(ctx, m, instance, column, maskingExceptionPolicyMap, action, currentPrincipal)
			if err != nil {
				return nil, err
			}
			if newMasker == nil {
				continue
			}
			if _, ok := newMasker.(*masker.NoneMasker); ok {
				continue
			}
			effectiveMaskers = append(effectiveMaskers, newMasker)
		}

		switch len(effectiveMaskers) {
		case 0:
			maskers = append(maskers, masker.NewNoneMasker())
		case 1:
			maskers = append(maskers, effectiveMaskers[0])
		default:
			// If there are more than one source columns, we fall back to the default full masker,
			// because we don't know how the data be made up.
			maskers = append(maskers, masker.NewDefaultFullMasker())
		}
	}
	return maskers, nil
}

func (s *QueryResultMasker) getMasterForColumnResource(
	ctx context.Context,
	m *maskingLevelEvaluator,
	instance *store.InstanceMessage,
	sourceColumn base.ColumnResource,
	maskingExceptionPolicyMap map[string]*storepb.MaskingExceptionPolicy,
	action storepb.MaskingExceptionPolicy_MaskingException_Action,
	currentPrincipal *store.UserMessage,
) (masker.Masker, error) {
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return masker.NewNoneMasker(), nil
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &database.ProjectID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project: %q", database.ProjectID)
	}
	if project == nil {
		return masker.NewNoneMasker(), nil
	}

	meta, config, err := s.getColumnForColumnResource(ctx, instance.ResourceID, &sourceColumn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for column resource: %q", sourceColumn.String())
	}
	// Span and metadata are not the same in real time, so we fall back to none masker.
	if meta == nil {
		return masker.NewNoneMasker(), nil
	}

	semanticTypeID := ""
	if config != nil {
		semanticTypeID = config.SemanticTypeId
	}

	maskingPolicy, err := s.store.GetMaskingPolicyByDatabaseUID(ctx, database.UID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get masking policy for database: %q", database.DatabaseName)
	}
	maskingPolicyMap := make(map[maskingPolicyKey]*storepb.MaskData)
	if maskingPolicy != nil {
		for _, maskData := range maskingPolicy.MaskData {
			maskingPolicyMap[maskingPolicyKey{
				schema: maskData.Schema,
				table:  maskData.Table,
				column: maskData.Column,
			}] = maskData
		}
	}

	var maskingExceptionPolicy *storepb.MaskingExceptionPolicy
	// If we cannot find the maskingExceptionPolicy before, we need to find it from the database and record it in cache.

	if _, ok := maskingExceptionPolicyMap[database.ProjectID]; !ok {
		policy, err := s.store.GetMaskingExceptionPolicyByProjectUID(ctx, project.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find masking exception policy for project %q", project.ResourceID)
		}
		// It is safe if policy is nil.
		maskingExceptionPolicyMap[database.ProjectID] = policy
	}
	maskingExceptionPolicy = maskingExceptionPolicyMap[database.ProjectID]

	// Build the filtered maskingExceptionPolicy for current principal.
	var maskingExceptionContainsCurrentPrincipal []*storepb.MaskingExceptionPolicy_MaskingException
	if maskingExceptionPolicy != nil {
		for _, maskingException := range maskingExceptionPolicy.MaskingExceptions {
			if maskingException.Action != action {
				continue
			}
			if maskingException.Member == currentPrincipal.Email {
				maskingExceptionContainsCurrentPrincipal = append(maskingExceptionContainsCurrentPrincipal, maskingException)
			}
		}
	}

	maskingAlgorithm, maskingLevel, err := m.evaluateMaskingAlgorithmOfColumn(database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column, semanticTypeID, meta.Classification, project.DataClassificationConfigID, maskingPolicyMap, maskingExceptionContainsCurrentPrincipal)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", sourceColumn.Database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column)
	}
	return getMaskerByMaskingAlgorithmAndLevel(maskingAlgorithm, maskingLevel), nil
}
func (s *QueryResultMasker) getColumnForColumnResource(ctx context.Context, instanceID string, sourceColumn *base.ColumnResource) (*storepb.ColumnMetadata, *storepb.ColumnConfig, error) {
	if sourceColumn == nil {
		return nil, nil, nil
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return nil, nil, nil
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database schema: %q", sourceColumn.Database)
	}
	if dbSchema == nil {
		return nil, nil, nil
	}

	var columnMetadata *storepb.ColumnMetadata
	metadata := dbSchema.GetDatabaseMetadata()
	if metadata == nil {
		return nil, nil, nil
	}
	schema := metadata.GetSchema(sourceColumn.Schema)
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
	columnMetadata = column

	var columnConfig *storepb.ColumnConfig
	config := dbSchema.GetDatabaseConfig()
	if config == nil {
		return columnMetadata, nil, nil
	}
	schemaConfig := config.GetSchemaConfig(sourceColumn.Schema)
	if schemaConfig == nil {
		return columnMetadata, nil, nil
	}
	tableConfig := schemaConfig.GetTableConfig(sourceColumn.Table)
	if tableConfig == nil {
		return columnMetadata, nil, nil
	}

	columnConfig = tableConfig.GetColumnConfig(sourceColumn.Column)
	return columnMetadata, columnConfig, nil
}
