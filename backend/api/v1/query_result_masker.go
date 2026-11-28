package v1

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"
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

type queryResultMasker struct {
	store                *store.Store
	evaluator            *maskingLevelEvaluator
	principal            *store.UserMessage
	semanticTypeToMasker map[string]masker.Masker
}

// newQueryResultMasker resolves groups before execution so predicate checks, result masking,
// and error redaction all use the same membership, and a failed lookup cannot expose data.
func newQueryResultMasker(ctx context.Context, stores *store.Store, instance *store.InstanceMessage, principal *store.UserMessage) (*queryResultMasker, error) {
	maskerWorkspaceID := common.GetWorkspaceIDFromContext(ctx)
	classificationSetting, err := stores.GetDataClassificationSetting(ctx, maskerWorkspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find classification setting")
	}

	maskingRulePolicy, err := stores.GetMaskingRulePolicy(ctx, maskerWorkspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find masking rule policy")
	}

	semanticTypesSetting, err := getSemanticTypesSettingWithBuiltins(ctx, stores)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find semantic types setting")
	}
	semanticTypeToMasker, err := semanticTypeMaskersFromSetting(semanticTypesSetting)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build semantic type to masker map")
	}

	m := newEmptyMaskingLevelEvaluator().
		withMaskingRulePolicy(maskingRulePolicy).
		withDataClassificationSetting(classificationSetting).
		withSemanticTypeSetting(semanticTypesSetting)
	if m.usesPrincipalGroups && common.EngineSupportMasking(instance.Metadata.GetEngine()) {
		m.principalGroups, err = resolveMaskingPrincipalGroups(ctx, stores, maskerWorkspaceID, principal)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve groups for masking")
		}
	}

	return &queryResultMasker{store: stores, evaluator: m, principal: principal, semanticTypeToMasker: semanticTypeToMasker}, nil
}

func resolveMaskingPrincipalGroups(ctx context.Context, stores *store.Store, workspace string, principal *store.UserMessage) ([]string, error) {
	if workspace == "" || principal == nil || principal.Email == "" {
		return nil, errors.New("workspace and authenticated principal are required")
	}
	member := common.FormatPrincipalMember(principal.Email, principal.Type)
	groups, err := stores.ListGroups(ctx, &store.FindGroupMessage{Workspace: workspace})
	if err != nil {
		return nil, err
	}
	var memberships []string
	for _, group := range groups {
		for _, groupMember := range group.Payload.GetMembers() {
			if groupMember.GetMember() == member {
				memberships = append(memberships, utils.FormatGroupName(group))
				break
			}
		}
	}
	return memberships, nil
}

// maskResults masks the results in place based on the masking policies and query spans.
func (s *queryResultMasker) maskResults(ctx context.Context, spans []*parserbase.QuerySpan, results []*v1pb.QueryResult, instance *store.InstanceMessage) error {
	// We expect the len(spans) == len(results), but to avoid NPE, we use the min(len(spans), len(results)) here.
	loopBoundary := min(len(spans), len(results))
	for i := 0; i < loopBoundary; i++ {
		// A plan is not row data, so it has no columns to trace to a masking policy.
		if parserbase.StartsWithExplain(results[i].Statement) {
			continue
		}
		if results[i].Error == "" && spans[i].FunctionNotSupportedError != nil {
			return errors.Errorf("masking error: %v", spans[i].FunctionNotSupportedError)
		}
		if results[i].Error == "" && spans[i].NotFoundError != nil {
			return errors.Errorf("masking error: %v", spans[i].NotFoundError)
		}
		// Reject before error handling or masking: results may contain both partial
		// rows and an error, and unresolved lineage cannot safely mask those rows.
		if maskingBlockedByUnresolvedColumns(spans[i], instance) {
			return errors.Errorf(
				"masking cannot be applied: %v, so the query was not returned",
				spans[i].UnresolvedColumnsError)
		}
		// Skip masking for error result, but redact the error message if the
		// statement touches masked columns — database errors can contain
		// actual column values (e.g. "invalid input syntax for type integer: '<value>'").
		// We check SourceColumns (covers SELECT, WHERE, JOIN, etc.) rather than
		// just Results (SELECT output) to catch cases like:
		//   SELECT name FROM t WHERE CAST(masked_col AS int) > 5
		if results[i].Error != "" && len(results[i].Rows) == 0 {
			if i < len(spans) && spans[i] != nil && s.spanTouchesMaskedColumns(ctx, instance, spans[i]) {
				results[i].Error = "Query execution failed. Error details are hidden because the query references columns with data masking policies."
				// Structured error fields can contain sensitive values and SQL too.
				results[i].DetailedError = nil
			}
			continue
		}
		if columnsMisalignedWithSpan(spans[i], results[i]) && s.spanTouchesMaskedColumns(ctx, instance, spans[i]) {
			return errors.Errorf(
				"masking cannot be applied: the query returned %d columns but Bytebase resolved %d from the synced schema, so the query was not returned",
				len(results[i].ColumnNames), len(spans[i].Results))
		}
		maskers, reasons, err := s.getMaskersForQuerySpan(ctx, instance, spans[i])
		if err != nil {
			return errors.Wrapf(err, "failed to get maskers for query span")
		}
		doMaskResult(maskers, reasons, results[i])
	}

	return nil
}

// maskingBlockedByUnresolvedColumns keeps the re-sync trigger and masking
// refusal on the same condition, so stale metadata gets a chance to recover.
func maskingBlockedByUnresolvedColumns(span *parserbase.QuerySpan, instance *store.InstanceMessage) bool {
	if span == nil || span.UnresolvedColumnsError == nil || instance == nil {
		return false
	}
	return common.EngineSupportMasking(instance.Metadata.GetEngine())
}

// columnsMisalignedWithSpan reports whether doMaskResult, which applies the
// span's maskers by position, would put a value beside the wrong masker or past
// the last one. A DML or DDL result reports on the command (Affected Rows)
// rather than projecting span.Results. A result without rows has no values to
// misplace, and BigQuery returns no column names for one.
func columnsMisalignedWithSpan(span *parserbase.QuerySpan, result *v1pb.QueryResult) bool {
	if span.Type == parserbase.DML || span.Type == parserbase.DDL {
		return false
	}
	return len(result.Rows) > 0 && len(result.ColumnNames) != len(span.Results)
}

// spanTouchesMaskedColumns checks whether any column referenced anywhere in
// the query (SELECT, WHERE, JOIN, etc.) has a masking policy applied.
//
// We collect column-level entries from span.Results[*].SourceColumns and
// span.PredicateColumns rather than span.SourceColumns, because some engines
// (e.g. MySQL) populate span.SourceColumns with table-level entries only
// (Column field is empty), which cannot be matched against column configs.
func (s *queryResultMasker) spanTouchesMaskedColumns(ctx context.Context, instance *store.InstanceMessage, span *parserbase.QuerySpan) bool {
	// Collect all column-level source columns from SELECT results and predicates.
	allColumns := make(parserbase.SourceColumnSet)
	for _, r := range span.Results {
		for col := range r.SourceColumns {
			allColumns[col] = true
		}
	}
	for col := range span.PredicateColumns {
		allColumns[col] = true
	}
	if len(allColumns) == 0 {
		return false
	}

	data, err := newMaskingDataProvider(ctx, s.store, instance, span)
	if err != nil {
		return false
	}

	for column := range allColumns {
		mk, _, err := s.getMaskerForColumnResource(ctx, instance, column, data)
		if err != nil {
			continue
		}
		if mk == nil {
			continue
		}
		if _, isNone := mk.(*masker.NoneMasker); !isNone {
			return true
		}
	}
	return false
}

func buildSemanticTypeToMaskerMap(ctx context.Context, stores *store.Store) (map[string]masker.Masker, error) {
	semanticTypesSetting, err := getSemanticTypesSettingWithBuiltins(ctx, stores)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get semantic types setting")
	}
	return semanticTypeMaskersFromSetting(semanticTypesSetting)
}

func semanticTypeMaskersFromSetting(semanticTypesSetting *storepb.SemanticTypeSetting) (map[string]masker.Masker, error) {
	semanticTypeToMasker := make(map[string]masker.Masker)
	for _, semanticType := range semanticTypesSetting.GetTypes() {
		m, err := masker.ForSemanticType(semanticType)
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
	if instance == nil || span == nil {
		return newEmptyMaskingDataProvider(), nil
	}

	dbNameSet := make(map[string]struct{})
	for _, r := range span.Results {
		for col := range r.SourceColumns {
			if col.Database != "" {
				dbNameSet[col.Database] = struct{}{}
			}
		}
	}

	return newMaskingDataProviderFromDBNames(ctx, s, instance, dbNameSet)
}

func newMaskingDataProviderFromColumns(ctx context.Context, s *store.Store, instance *store.InstanceMessage, columns parserbase.SourceColumnSet) (*maskingDataProvider, error) {
	if instance == nil || len(columns) == 0 {
		return newEmptyMaskingDataProvider(), nil
	}

	dbNameSet := make(map[string]struct{})
	for col := range columns {
		if col.Database != "" {
			dbNameSet[col.Database] = struct{}{}
		}
	}

	return newMaskingDataProviderFromDBNames(ctx, s, instance, dbNameSet)
}

func newEmptyMaskingDataProvider() *maskingDataProvider {
	return &maskingDataProvider{
		databases: make(map[string]*store.DatabaseMessage),
		projects:  make(map[string]*store.ProjectMessage),
		schemas:   make(map[string]*model.DatabaseMetadata),
		policies:  make(map[string]*storepb.MaskingExemptionPolicy),
	}
}

func newMaskingDataProviderFromDBNames(ctx context.Context, s *store.Store, instance *store.InstanceMessage, dbNameSet map[string]struct{}) (*maskingDataProvider, error) {
	p := newEmptyMaskingDataProvider()
	if len(dbNameSet) == 0 {
		return p, nil
	}

	dbNames := make([]string, 0, len(dbNameSet))
	for name := range dbNameSet {
		dbNames = append(dbNames, name)
	}

	databases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{
		Workspace:     common.GetWorkspaceIDFromContext(ctx),
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

	if len(projectIDSet) > 0 {
		projectIDs := make([]string, 0, len(projectIDSet))
		for id := range projectIDSet {
			projectIDs = append(projectIDs, id)
		}
		projects, err := s.ListProjects(ctx, &store.FindProjectMessage{
			Workspace:   common.GetWorkspaceIDFromContext(ctx),
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

	for dbName := range p.databases {
		schema, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{Workspace: common.GetWorkspaceIDFromContext(ctx), InstanceID: instance.ResourceID, DatabaseName: dbName})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get db schema for %q", dbName)
		}
		if schema != nil {
			p.schemas[dbName] = schema
		}
	}

	for projectID := range projectIDSet {
		policy, err := s.GetMaskingExemptionPolicyByProject(ctx, common.GetWorkspaceIDFromContext(ctx), projectID)
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

func (p *maskingDataProvider) getColumn(col *parserbase.ColumnResource) (*metadatapb.ColumnMetadata, *storepb.ColumnCatalog) {
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
func (s *queryResultMasker) getMaskersForQuerySpan(ctx context.Context, instance *store.InstanceMessage, span *parserbase.QuerySpan) ([]masker.Masker, []*v1pb.MaskingReason, error) {
	if span == nil {
		return nil, nil, nil
	}
	maskers := make([]masker.Masker, 0, len(span.Results))
	masked := make([]*v1pb.MaskingReason, 0, len(span.Results))

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
			newMasker, reason, err := s.getMaskerForColumnResource(ctx, instance, column, data)
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

// Apply the same exemptions to results and predicates so a predicate cannot expose a masked value.
func (s *queryResultMasker) exemptionsForPrincipal(ctx context.Context, data *maskingDataProvider, projectID string) []*storepb.MaskingExemptionPolicy_Exemption {
	policy := data.getMaskingExemptionPolicy(projectID)
	if policy == nil {
		return nil
	}
	var exemptions []*storepb.MaskingExemptionPolicy_Exemption
	for _, e := range policy.Exemptions {
		for _, member := range e.Members {
			if utils.MemberContainsUser(ctx, s.store, common.GetWorkspaceIDFromContext(ctx), member, s.principal) {
				exemptions = append(exemptions, e)
				break
			}
		}
	}
	return exemptions
}

func (s *queryResultMasker) getMaskerForColumnResource(
	ctx context.Context,
	instance *store.InstanceMessage,
	sourceColumn parserbase.ColumnResource,
	data *maskingDataProvider,
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

	exemptions := s.exemptionsForPrincipal(ctx, data, database.ProjectID)

	evaluation, err := s.evaluator.evaluateSemanticTypeOfColumn(database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column, project.Setting.DataClassificationConfigId, config, exemptions)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to evaluate masking level of database %q, schema %q, table %q, column %q", sourceColumn.Database, sourceColumn.Schema, sourceColumn.Table, sourceColumn.Column)
	}

	if evaluation == nil || evaluation.SemanticTypeID == "" {
		return masker.NewNoneMasker(), nil, nil
	}

	if result, ok := s.semanticTypeToMasker[evaluation.SemanticTypeID]; ok {
		return result, evaluation, nil
	}
	if _, known := s.evaluator.semanticTypesMap[evaluation.SemanticTypeID]; known {
		return masker.NewNoneMasker(), nil, nil
	}

	// A live catalog can reference a semantic type newer than the request's settings.
	// Do not treat a missing definition as permission to return unmasked data.
	evaluation.Algorithm = "Full mask"
	return masker.NewDefaultFullMasker(), evaluation, nil
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
