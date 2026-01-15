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
	"github.com/bytebase/bytebase/backend/store/model"
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

// maskingDataProvider provides cached access to databases, projects, schemas, and policies.
// It pre-fetches data in batch to avoid N+1 queries when evaluating masking for multiple columns.
type maskingDataProvider struct {
	databases map[string]*store.DatabaseMessage
	projects  map[string]*store.ProjectMessage
	schemas   map[string]*model.DatabaseMetadata
	policies  map[string]*storepb.MaskingExemptionPolicy
}

func newMaskingDataProvider(ctx context.Context, s *store.Store, instance *store.InstanceMessage, span *parserbase.QuerySpan) (*maskingDataProvider, error) {
	p := &maskingDataProvider{
		databases: make(map[string]*store.DatabaseMessage),
		projects:  make(map[string]*store.ProjectMessage),
		schemas:   make(map[string]*model.DatabaseMetadata),
		policies:  make(map[string]*storepb.MaskingExemptionPolicy),
	}
	if instance == nil || span == nil {
		return p, nil
	}

	// Collect unique database names.
	dbNameSet := make(map[string]struct{})
	for _, r := range span.Results {
		for col := range r.SourceColumns {
			if col.Database != "" {
				dbNameSet[col.Database] = struct{}{}
			}
		}
	}
	if len(dbNameSet) == 0 {
		return p, nil
	}

	// Convert to slice for batch query.
	dbNames := make([]string, 0, len(dbNameSet))
	for name := range dbNameSet {
		dbNames = append(dbNames, name)
	}

	// Batch fetch databases.
	databases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{
		InstanceID:    &instance.ResourceID,
		DatabaseNames: dbNames,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list databases")
	}

	projectIDSet := make(map[string]struct{})
	for _, db := range databases {
		p.databases[db.DatabaseName] = db
		projectIDSet[db.ProjectID] = struct{}{}
	}

	// Batch fetch projects.
	if len(projectIDSet) > 0 {
		projectIDs := make([]string, 0, len(projectIDSet))
		for id := range projectIDSet {
			projectIDs = append(projectIDs, id)
		}
		projects, err := s.ListProjects(ctx, &store.FindProjectMessage{
			ResourceIDs: projectIDs,
			ShowDeleted: true,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list projects")
		}
		for _, proj := range projects {
			p.projects[proj.ResourceID] = proj
		}
	}

	// Fetch schemas and policies (no batch API available).
	for dbName := range p.databases {
		schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{InstanceID: instance.ResourceID, DatabaseName: dbName})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get db schema for %q", dbName)
		}
		if schema != nil {
			p.schemas[dbName] = schema
		}
	}
	for projectID := range projectIDSet {
		policy, err := s.GetMaskingExemptionPolicyByProject(ctx, projectID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get masking exemption policy for project %q", projectID)
		}
		p.policies[projectID] = policy
	}

	return p, nil
}

func (p *maskingDataProvider) getDatabase(dbName string) *store.DatabaseMessage {
	return p.databases[dbName]
}

func (p *maskingDataProvider) getProject(projectID string) *store.ProjectMessage {
	return p.projects[projectID]
}

func (p *maskingDataProvider) getColumn(col *parserbase.ColumnResource) (*storepb.ColumnMetadata, *storepb.ColumnCatalog) {
	schema := p.schemas[col.Database]
	if schema == nil {
		return nil, nil
	}
	s := schema.GetSchemaMetadata(col.Schema)
	if s == nil {
		return nil, nil
	}
	t := s.GetTable(col.Table)
	if t == nil {
		return nil, nil
	}
	c := t.GetColumn(col.Column)
	if c == nil {
		return nil, nil
	}
	return c.GetProto(), c.GetCatalog()
}

func (p *maskingDataProvider) getMaskingExemptionPolicy(projectID string) *storepb.MaskingExemptionPolicy {
	return p.policies[projectID]
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

	// Pre-fetch all required data to avoid N+1 queries in the loop.
	data, err := newMaskingDataProvider(ctx, s.store, instance, span)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to initialize masking data provider")
	}

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
			newMasker, reason, err := s.getMaskerForColumnResource(ctx, m, instance, column, data, user, semanticTypesToMasker)
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
	data *maskingDataProvider,
	currentPrincipal *store.UserMessage,
	semanticTypeToMasker map[string]masker.Masker,
) (masker.Masker, *MaskingEvaluation, error) {
	if instance != nil && !common.EngineSupportMasking(instance.Metadata.GetEngine()) {
		return masker.NewNoneMasker(), nil, nil
	}

	database := data.getDatabase(sourceColumn.Database)
	if database == nil {
		return masker.NewNoneMasker(), nil, nil
	}

	project := data.getProject(database.ProjectID)
	if project == nil {
		return masker.NewNoneMasker(), nil, nil
	}

	columnMeta, config := data.getColumn(&sourceColumn)
	if columnMeta == nil {
		return masker.NewNoneMasker(), nil, nil
	}

	// Build the filtered maskingExemptionPolicy for current principal.
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

	evaluation, err := m.evaluateSemanticTypeOfColumn(database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column, project.Setting.DataClassificationConfigId, config, exemptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", sourceColumn.Database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column)
	}

	if evaluation == nil || evaluation.SemanticTypeID == "" {
		return masker.NewNoneMasker(), nil, nil
	}

	result, ok := semanticTypeToMasker[evaluation.SemanticTypeID]
	if !ok {
		return masker.NewNoneMasker(), nil, nil
	}

	evaluation.Algorithm = getAlgorithmName(result)
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
