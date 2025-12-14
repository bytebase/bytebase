package v1

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/masker"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
func (s *QueryResultMasker) MaskResults(ctx context.Context, spans []*parserbase.QuerySpan, results []*v1pb.QueryResult, instance *store.InstanceMessage, user *store.UserMessage) error {
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
		if results[i].Error == "" && spans[i].FunctionNotSupportedError != nil {
			return errors.Errorf("masking error: %v", spans[i].FunctionNotSupportedError)
		}
		if results[i].Error == "" && spans[i].NotFoundError != nil {
			return errors.Errorf("masking error: %v", spans[i].NotFoundError)
		}
		// Skip masking for error result.
		if results[i].Error != "" && len(results[i].Rows) == 0 {
			continue
		}
		maskers, reasons, err := s.getMaskersForQuerySpan(ctx, m, instance, user, spans[i])
		if err != nil {
			return errors.Wrapf(err, "failed to get maskers for query span")
		}
		doMaskResult(maskers, reasons, results[i])
	}

	return nil
}

func getAlgorithmName(m masker.Masker) string {
	switch m.(type) {
	case *masker.NoneMasker:
		return "None"
	case *masker.FullMasker:
		return "Full mask"
	case *masker.RangeMasker, *masker.DefaultRangeMasker:
		return "Partial mask"
	case *masker.MD5Masker:
		return "Hash (MD5)"
	case *masker.InnerOuterMasker:
		// Check the actual type by examining the mask result
		return "Inner/Outer mask"
	default:
		return "Unknown"
	}
}

func buildSemanticTypeToMaskerMap(ctx context.Context, stores *store.Store) (map[string]masker.Masker, error) {
	semanticTypeToMasker := map[string]masker.Masker{
		"bb.default":         masker.NewDefaultFullMasker(),
		"bb.default-partial": masker.NewDefaultRangeMasker(),
	}
	semanticTypesSetting, err := stores.GetSemanticTypesSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get semantic types setting")
	}
	for _, semanticType := range semanticTypesSetting.GetTypes() {
		if semanticType.GetId() == "bb.default" || semanticType.GetId() == "bb.default-partial" {
			// Skip the built-in default semantic types.
			continue
		}
		m, err := getMaskerByMaskingAlgorithmAndLevel(semanticType.GetAlgorithm())
		if err != nil {
			return nil, err
		}
		// Only add semantic types that have actual masking configured (not NoneMasker)
		if _, isNoneMasker := m.(*masker.NoneMasker); !isNoneMasker {
			semanticTypeToMasker[semanticType.GetId()] = m
		}
	}

	return semanticTypeToMasker, nil
}

// getMaskersForQuerySpan returns the maskers for the query span.
func (s *QueryResultMasker) getMaskersForQuerySpan(ctx context.Context, m *maskingLevelEvaluator, instance *store.InstanceMessage, user *store.UserMessage, span *parserbase.QuerySpan) ([]masker.Masker, []*v1pb.MaskingReason, error) {
	if span == nil {
		return nil, nil, nil
	}
	maskers := make([]masker.Masker, 0, len(span.Results))
	masked := make([]*v1pb.MaskingReason, 0, len(span.Results))

	semanticTypesToMasker, err := buildSemanticTypeToMaskerMap(ctx, s.store)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to build semantic type to masker map")
	}
	// Multiple databases may belong to the same project, to reduce the protojson unmarshal cost,
	maskingExemptionPolicyMap := make(map[string]*storepb.MaskingExemptionPolicy)

	for _, spanResult := range span.Results {
		// Likes constant expression, we use the none masker.
		if len(spanResult.SourceColumns) == 0 {
			maskers = append(maskers, masker.NewNoneMasker())
			masked = append(masked, nil)
			continue
		}

		var effectiveMaskers []masker.Masker
		var effectiveReasons []*MaskingEvaluation
		for column := range spanResult.SourceColumns {
			newMasker, reason, err := s.getMaskerForColumnResource(ctx, m, instance, column, maskingExemptionPolicyMap, user, semanticTypesToMasker)
			if err != nil {
				return nil, nil, err
			}
			if newMasker == nil {
				continue
			}
			if _, ok := newMasker.(*masker.NoneMasker); ok {
				continue
			}
			effectiveMaskers = append(effectiveMaskers, newMasker)
			if reason != nil {
				effectiveReasons = append(effectiveReasons, reason)
			}
		}

		switch len(effectiveMaskers) {
		case 0:
			maskers = append(maskers, masker.NewNoneMasker())
			masked = append(masked, nil)
		case 1:
			// If there is only one source column, and comes from the expression, we fall back to the default full masker.
			if !spanResult.IsPlainField && common.EngineSupportQuerySpanPlainField(instance.Metadata.GetEngine()) {
				maskers = append(maskers, masker.NewDefaultFullMasker())
				// For expression-based columns, add a generic reason
				if len(effectiveReasons) > 0 {
					reason := effectiveReasons[0]
					masked = append(masked, &v1pb.MaskingReason{
						SemanticTypeId:      reason.SemanticTypeID,
						SemanticTypeTitle:   reason.SemanticTypeTitle,
						SemanticTypeIcon:    reason.SemanticTypeIcon,
						MaskingRuleId:       reason.MaskingRuleID,
						Algorithm:           "Full mask",
						Context:             "Expression-based column",
						ClassificationLevel: reason.ClassificationLevel,
					})
				} else {
					masked = append(masked, nil)
				}
			} else {
				maskers = append(maskers, effectiveMaskers[0])
				if len(effectiveReasons) > 0 {
					reason := effectiveReasons[0]
					masked = append(masked, &v1pb.MaskingReason{
						SemanticTypeId:      reason.SemanticTypeID,
						SemanticTypeTitle:   reason.SemanticTypeTitle,
						SemanticTypeIcon:    reason.SemanticTypeIcon,
						MaskingRuleId:       reason.MaskingRuleID,
						Algorithm:           reason.Algorithm,
						Context:             reason.Context,
						ClassificationLevel: reason.ClassificationLevel,
					})
				} else {
					masked = append(masked, nil)
				}
			}
		default:
			// If there are more than one source columns, we fall back to the default full masker,
			// because we don't know how the data be made up.
			maskers = append(maskers, masker.NewDefaultFullMasker())
			masked = append(masked, &v1pb.MaskingReason{
				Algorithm: "Full mask",
				Context:   "Multiple source columns",
			})
		}
	}
	return maskers, masked, nil
}

func (s *QueryResultMasker) getMaskerForColumnResource(
	ctx context.Context,
	m *maskingLevelEvaluator,
	instance *store.InstanceMessage,
	sourceColumn parserbase.ColumnResource,
	maskingExemptionPolicyMap map[string]*storepb.MaskingExemptionPolicy,
	currentPrincipal *store.UserMessage,
	semanticTypeToMasker map[string]masker.Masker,
) (masker.Masker, *MaskingEvaluation, error) {
	if instance != nil && !common.EngineSupportMasking(instance.Metadata.GetEngine()) {
		return masker.NewNoneMasker(), nil, nil
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instance.ResourceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return masker.NewNoneMasker(), nil, nil
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &database.ProjectID,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find project: %q", database.ProjectID)
	}
	if project == nil {
		return masker.NewNoneMasker(), nil, nil
	}

	meta, config, err := s.getColumnForColumnResource(ctx, instance.ResourceID, &sourceColumn)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get database metadata for column resource: %q", sourceColumn.String())
	}
	// Span and metadata are not the same in real time, so we fall back to none masker.
	if meta == nil {
		return masker.NewNoneMasker(), nil, nil
	}

	var maskingExemptionPolicy *storepb.MaskingExemptionPolicy
	// If we cannot find the maskingExemptionPolicy before, we need to find it from the database and record it in cache.

	if _, ok := maskingExemptionPolicyMap[database.ProjectID]; !ok {
		policy, err := s.store.GetMaskingExemptionPolicyByProject(ctx, project.ResourceID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find masking exemption policy for project %q", project.ResourceID)
		}
		// It is safe if policy is nil.
		maskingExemptionPolicyMap[database.ProjectID] = policy
	}
	maskingExemptionPolicy = maskingExemptionPolicyMap[database.ProjectID]

	// Build the filtered maskingExemptionPolicy for current principal.
	var maskingExemptionContainsCurrentPrincipal []*storepb.MaskingExemptionPolicy_Exemption
	if maskingExemptionPolicy != nil {
		for _, maskingExemption := range maskingExemptionPolicy.Exemptions {
			for _, member := range maskingExemption.Members {
				if utils.MemberContainsUser(ctx, s.store, member, currentPrincipal) {
					maskingExemptionContainsCurrentPrincipal = append(maskingExemptionContainsCurrentPrincipal, maskingExemption)
					break
				}
			}
		}
	}

	evaluation, err := m.evaluateSemanticTypeOfColumn(database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column, project.DataClassificationConfigID, config, maskingExemptionContainsCurrentPrincipal)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", sourceColumn.Database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column)
	}

	if evaluation == nil || evaluation.SemanticTypeID == "" {
		return masker.NewNoneMasker(), nil, nil
	}

	result, ok := semanticTypeToMasker[evaluation.SemanticTypeID]
	if !ok {
		// No masker configured for this semantic type, return NoneMasker without evaluation
		return masker.NewNoneMasker(), nil, nil
	}

	// Get algorithm name from the masker
	algorithmName := getAlgorithmName(result)
	evaluation.Algorithm = algorithmName

	return result, evaluation, nil
}

func (s *QueryResultMasker) getColumnForColumnResource(ctx context.Context, instanceID string, sourceColumn *parserbase.ColumnResource) (*storepb.ColumnMetadata, *storepb.ColumnCatalog, error) {
	if sourceColumn == nil {
		return nil, nil, nil
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &sourceColumn.Database,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database: %q", sourceColumn.Database)
	}
	if database == nil {
		return nil, nil, nil
	}
	dbMetadata, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find database schema: %q", sourceColumn.Database)
	}
	if dbMetadata == nil {
		return nil, nil, nil
	}

	var columnMetadata *storepb.ColumnMetadata
	schema := dbMetadata.GetSchemaMetadata(sourceColumn.Schema)
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
	columnMetadata = column.GetProto()

	columnConfig := column.GetCatalog()
	return columnMetadata, columnConfig, nil
}

func getMaskerByMaskingAlgorithmAndLevel(algorithm *storepb.Algorithm) (masker.Masker, error) {
	if algorithm == nil {
		return masker.NewNoneMasker(), nil
	}

	switch m := algorithm.Mask.(type) {
	case *storepb.Algorithm_FullMask_:
		return masker.NewFullMasker(m.FullMask.Substitution), nil
	case *storepb.Algorithm_RangeMask_:
		return masker.NewRangeMasker(convertRangeMaskSlices(m.RangeMask.Slices)), nil
	case *storepb.Algorithm_Md5Mask:
		return masker.NewMD5Masker(m.Md5Mask.Salt), nil
	case *storepb.Algorithm_InnerOuterMask_:
		return masker.NewInnerOuterMasker(m.InnerOuterMask.Type, m.InnerOuterMask.PrefixLen, m.InnerOuterMask.SuffixLen, m.InnerOuterMask.Substitution)
	}
	return masker.NewNoneMasker(), nil
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

func doMaskResult(maskers []masker.Masker, reasons []*v1pb.MaskingReason, result *v1pb.QueryResult) {
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

	result.Masked = reasons
}
