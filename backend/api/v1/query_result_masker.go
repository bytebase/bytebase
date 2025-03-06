package v1

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/masker"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
func (s *QueryResultMasker) MaskResults(ctx context.Context, spans []*base.QuerySpan, results []*v1pb.QueryResult, instance *store.InstanceMessage, user *store.UserMessage, action storepb.MaskingExceptionPolicy_MaskingException_Action) error {
	classificationSetting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find classification setting")
	}

	maskingRulePolicy, err := s.store.GetMaskingRulePolicy(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find masking rule policy")
	}

	semanticTypesSetting, err := s.store.GetSemanticTypesSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find semantic types setting")
	}

	m := newEmptyMaskingLevelEvaluator().
		withMaskingRulePolicy(maskingRulePolicy).
		withDataClassificationSetting(classificationSetting).
		withSemanticTypeSetting(semanticTypesSetting)

	// We expect the len(spans) == len(results), but to avoid NPE, we use the min(len(spans), len(results)) here.
	loopBoundary := min(len(spans), len(results))
	for i := 0; i < loopBoundary; i++ {
		if strings.HasPrefix(strings.TrimSpace(results[i].Statement), "EXPLAIN") {
			continue
		}
		if results[i].Error == "" && spans[i].NotFoundError != nil {
			return errors.Errorf("query span error: %v", spans[i].NotFoundError)
		}
		// Skip masking for error result.
		if results[i].Error != "" && len(results[i].Rows) == 0 {
			continue
		}
		maskers, err := s.getMaskersForQuerySpan(ctx, m, instance, user, spans[i], action)
		if err != nil {
			return errors.Wrapf(err, "failed to get maskers for query span")
		}
		doMaskResult(maskers, results[i])
	}

	return nil
}

// getMaskersForQuerySpan returns the maskers for the query span.
func (s *QueryResultMasker) getMaskersForQuerySpan(ctx context.Context, m *maskingLevelEvaluator, instance *store.InstanceMessage, user *store.UserMessage, span *base.QuerySpan, action storepb.MaskingExceptionPolicy_MaskingException_Action) ([]masker.Masker, error) {
	if span == nil {
		return nil, nil
	}
	maskers := make([]masker.Masker, 0, len(span.Results))

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
			newMasker, err := s.getMaskerForColumnResource(ctx, m, instance, column, maskingExceptionPolicyMap, action, user)
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
			// If there is only one source column, and comes from the expression, we fall back to the default full masker.
			if !spanResult.IsPlainField {
				maskers = append(maskers, masker.NewDefaultFullMasker())
			} else {
				maskers = append(maskers, effectiveMaskers[0])
			}
		default:
			// If there are more than one source columns, we fall back to the default full masker,
			// because we don't know how the data be made up.
			maskers = append(maskers, masker.NewDefaultFullMasker())
		}
	}
	return maskers, nil
}

func (s *QueryResultMasker) getMaskerForColumnResource(
	ctx context.Context,
	m *maskingLevelEvaluator,
	instance *store.InstanceMessage,
	sourceColumn base.ColumnResource,
	maskingExceptionPolicyMap map[string]*storepb.MaskingExceptionPolicy,
	action storepb.MaskingExceptionPolicy_MaskingException_Action,
	currentPrincipal *store.UserMessage,
) (masker.Masker, error) {
	if instance != nil && !isMaskingSupported(instance.Metadata.GetEngine()) {
		return masker.NewNoneMasker(), nil
	}
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

	var maskingExceptionPolicy *storepb.MaskingExceptionPolicy
	// If we cannot find the maskingExceptionPolicy before, we need to find it from the database and record it in cache.

	if _, ok := maskingExceptionPolicyMap[database.ProjectID]; !ok {
		policy, err := s.store.GetMaskingExceptionPolicyByProject(ctx, project.ResourceID)
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

			users := utils.GetUsersByMember(ctx, s.store, maskingException.Member)
			for _, user := range users {
				if user.ID == currentPrincipal.ID {
					maskingExceptionContainsCurrentPrincipal = append(maskingExceptionContainsCurrentPrincipal, maskingException)
					break
				}
			}
		}
	}

	semanticTypeID, err := m.evaluateSemanticTypeOfColumn(database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column, project.DataClassificationConfigID, config, maskingExceptionContainsCurrentPrincipal)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", sourceColumn.Database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column)
	}

	// Built-in algorithm.
	switch semanticTypeID {
	case "bb.default":
		return masker.NewDefaultFullMasker(), nil
	case "bb.default-partial":
		return masker.NewDefaultRangeMasker(), nil
	}

	semanticType := m.semanticTypesMap[semanticTypeID]
	return getMaskerByMaskingAlgorithmAndLevel(semanticType.GetAlgorithm()), nil
}

func (s *QueryResultMasker) getColumnForColumnResource(ctx context.Context, instanceID string, sourceColumn *base.ColumnResource) (*storepb.ColumnMetadata, *storepb.ColumnCatalog, error) {
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
	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
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

	var columnConfig *storepb.ColumnCatalog
	config := dbSchema.GetInternalConfig()
	if config == nil {
		return columnMetadata, nil, nil
	}
	schemaConfig := config.CreateOrGetSchemaConfig(sourceColumn.Schema)
	tableConfig := schemaConfig.CreateOrGetTableConfig(sourceColumn.Table)
	columnConfig = tableConfig.CreateOrGetColumnConfig(sourceColumn.Column)
	return columnMetadata, columnConfig, nil
}

func getMaskerByMaskingAlgorithmAndLevel(algorithm *storepb.Algorithm) masker.Masker {
	if algorithm == nil {
		return masker.NewNoneMasker()
	}

	switch m := algorithm.Mask.(type) {
	case *storepb.Algorithm_FullMask_:
		return masker.NewFullMasker(m.FullMask.Substitution)
	case *storepb.Algorithm_RangeMask_:
		return masker.NewRangeMasker(convertRangeMaskSlices(m.RangeMask.Slices))
	case *storepb.Algorithm_Md5Mask:
		return masker.NewMD5Masker(m.Md5Mask.Salt)
	case *storepb.Algorithm_InnerOuterMask_:
		return masker.NewInnerOuterMasker(m.InnerOuterMask.Type, m.InnerOuterMask.PrefixLen, m.InnerOuterMask.SuffixLen, m.InnerOuterMask.Substitution)
	}
	return masker.NewNoneMasker()
}

func convertRangeMaskSlices(slices []*storepb.Algorithm_RangeMask_Slice) []*masker.MaskRangeSlice {
	var result []*masker.MaskRangeSlice
	for _, slice := range slices {
		result = append(result, &masker.MaskRangeSlice{
			Start:        slice.Start,
			End:          slice.End,
			Substitution: slice.Substitution,
		})
	}
	return result
}

func doMaskResult(maskers []masker.Masker, result *v1pb.QueryResult) {
	sensitive := make([]bool, len(result.ColumnNames))
	for i := range result.ColumnNames {
		if i < len(maskers) {
			switch maskers[i].(type) {
			case *masker.NoneMasker:
				sensitive[i] = false
			default:
				sensitive[i] = true
			}
		}
	}

	for i, row := range result.Rows {
		for j, value := range row.Values {
			if value == nil {
				continue
			}
			maskedValue := row.Values[j]
			if j < len(maskers) && maskers[j] != nil {
				maskedValue = maskers[j].Mask(&masker.MaskData{
					Data: row.Values[j],
				})
			}
			result.Rows[i].Values[j] = maskedValue
		}
	}

	result.Sensitive = sensitive
	result.Masked = sensitive
}

func isMaskingSupported(e storepb.Engine) bool {
	var supportedEngines = map[storepb.Engine]bool{
		storepb.Engine_MYSQL:     true,
		storepb.Engine_POSTGRES:  true,
		storepb.Engine_ORACLE:    true,
		storepb.Engine_MSSQL:     true,
		storepb.Engine_MARIADB:   true,
		storepb.Engine_OCEANBASE: true,
		storepb.Engine_TIDB:      true,
		storepb.Engine_BIGQUERY:  true,
		storepb.Engine_SPANNER:   true,
	}

	if _, ok := supportedEngines[e]; !ok {
		return false
	}
	return true
}
