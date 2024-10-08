package common

import (
	"slices"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// GetClassificationAndUserComment parses classification and user comment from the given comment.
func GetClassificationAndUserComment(comment string, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) (string, string) {
	if classificationConfig == nil {
		return "", comment
	}
	if _, ok := classificationConfig.Classification[comment]; ok {
		return comment, ""
	}
	for i := len(comment) - 1; i >= 0; i-- {
		if comment[i] != '-' {
			continue
		}
		if _, ok := classificationConfig.Classification[comment[:i]]; ok {
			return comment[:i], comment[i+1:]
		}
	}
	return "", comment
}

// GetCommentFromClassificationAndUserComment returns the comment from the given classification and user comment.
func GetCommentFromClassificationAndUserComment(classification, userComment string) string {
	if classification == "" {
		return userComment
	}
	if userComment == "" {
		return classification
	}
	return classification + "-" + userComment
}

func EqualDatabaseSchemaMetadataFast(s, t *storepb.DatabaseSchemaMetadata) bool {
	if s.GetName() != t.GetName() {
		return false
	}
	if len(s.GetSchemas()) != len(t.GetSchemas()) {
		return false
	}
	oldSchemaMap := make(map[string]*storepb.SchemaMetadata)
	for _, schema := range s.GetSchemas() {
		oldSchemaMap[schema.GetName()] = schema
	}
	for _, schema := range t.GetSchemas() {
		oldSchema, ok := oldSchemaMap[schema.GetName()]
		if !ok {
			return false
		}
		if !equalSchema(oldSchema, schema) {
			return false
		}
	}
	return true
}

func equalSchema(s, t *storepb.SchemaMetadata) bool {
	if s.GetName() != t.GetName() {
		return false
	}
	if len(s.GetTables()) != len(t.GetTables()) {
		return false
	}
	oldTableMap := make(map[string]*storepb.TableMetadata)
	for _, table := range s.GetTables() {
		oldTableMap[table.GetName()] = table
	}
	for _, table := range t.GetTables() {
		oldTable, ok := oldTableMap[table.GetName()]
		if !ok {
			return false
		}
		if !EqualTable(oldTable, table) {
			return false
		}
	}

	if len(s.GetViews()) != len(t.GetViews()) {
		return false
	}
	oldViewMap := make(map[string]*storepb.ViewMetadata)
	for _, view := range s.GetViews() {
		oldViewMap[view.GetName()] = view
	}
	for _, view := range t.GetViews() {
		oldView, ok := oldViewMap[view.GetName()]
		if !ok {
			return false
		}
		if oldView.GetDefinition() != view.GetDefinition() {
			return false
		}
	}

	if len(s.GetFunctions()) != len(t.GetFunctions()) {
		return false
	}
	oldFunctionMap := make(map[string]*storepb.FunctionMetadata)
	for _, function := range s.GetFunctions() {
		oldFunctionMap[function.GetName()] = function
	}
	for _, function := range t.GetFunctions() {
		oldFunction, ok := oldFunctionMap[function.GetName()]
		if !ok {
			return false
		}
		if oldFunction.GetDefinition() != function.GetDefinition() {
			return false
		}
	}

	if len(s.GetProcedures()) != len(t.GetProcedures()) {
		return false
	}
	oldProcedureMap := make(map[string]*storepb.ProcedureMetadata)
	for _, procedure := range s.GetProcedures() {
		oldProcedureMap[procedure.GetName()] = procedure
	}
	for _, procedure := range t.GetProcedures() {
		oldProcedure, ok := oldProcedureMap[procedure.GetName()]
		if !ok {
			return false
		}
		if oldProcedure.GetDefinition() != procedure.GetDefinition() {
			return false
		}
	}

	if len(s.GetPackages()) != len(t.GetPackages()) {
		return false
	}
	oldPackageMap := make(map[string]*storepb.PackageMetadata)
	for _, p := range s.GetPackages() {
		oldPackageMap[p.GetName()] = p
	}
	for _, p := range t.GetPackages() {
		oldPackage, ok := oldPackageMap[p.GetName()]
		if !ok {
			return false
		}
		if oldPackage.GetDefinition() != p.GetDefinition() {
			return false
		}
	}

	return true
}

// EqualTable compares metadata for two tables.
func EqualTable(s, t *storepb.TableMetadata) bool {
	if len(s.GetColumns()) != len(t.GetColumns()) {
		return false
	}
	if len(s.Indexes) != len(t.Indexes) {
		return false
	}
	if len(s.ForeignKeys) != len(t.ForeignKeys) {
		return false
	}
	if len(s.Partitions) != len(t.Partitions) {
		return false
	}
	if s.GetComment() != t.GetComment() {
		return false
	}
	if s.GetUserComment() != t.GetUserComment() {
		return false
	}
	for i := 0; i < len(s.GetColumns()); i++ {
		sc, tc := s.GetColumns()[i], t.GetColumns()[i]
		if sc.Name != tc.Name {
			return false
		}
		if sc.OnUpdate != tc.OnUpdate {
			return false
		}
		if sc.Comment != tc.Comment {
			return false
		}
		if sc.UserComment != tc.UserComment {
			return false
		}
		if sc.Type != tc.Type {
			return false
		}
		if sc.Nullable != tc.Nullable {
			return false
		}
		if sc.GetDefault().GetValue() != tc.GetDefault().GetValue() {
			return false
		}
		if sc.GetDefaultExpression() != tc.GetDefaultExpression() {
			return false
		}
		if sc.GetDefaultNull() != tc.GetDefaultNull() {
			return false
		}
	}
	for i := 0; i < len(s.GetIndexes()); i++ {
		si, ti := s.GetIndexes()[i], t.GetIndexes()[i]
		if si.GetName() != ti.GetName() {
			return false
		}
		if si.GetDefinition() != ti.GetDefinition() {
			return false
		}
		if si.GetPrimary() != ti.GetPrimary() {
			return false
		}
		if si.GetUnique() != ti.GetUnique() {
			return false
		}
		if si.GetType() != ti.GetType() {
			return false
		}
		if si.GetVisible() != ti.GetVisible() {
			return false
		}
		if si.GetComment() != ti.GetComment() {
			return false
		}
		if !slices.Equal(si.GetExpressions(), ti.GetExpressions()) {
			return false
		}
	}
	for i := 0; i < len(s.GetForeignKeys()); i++ {
		si, ti := s.GetForeignKeys()[i], t.GetForeignKeys()[i]
		if si.GetName() != ti.GetName() {
			return false
		}
		if !slices.Equal(si.GetColumns(), ti.GetColumns()) {
			return false
		}
		if si.GetReferencedSchema() != ti.GetReferencedSchema() {
			return false
		}
		if si.GetReferencedTable() != ti.GetReferencedTable() {
			return false
		}
		if !slices.Equal(si.GetReferencedColumns(), ti.GetReferencedColumns()) {
			return false
		}
		if si.GetOnDelete() != ti.GetOnDelete() {
			return false
		}
		if si.GetOnUpdate() != ti.GetOnUpdate() {
			return false
		}
		if si.GetMatchType() != ti.GetMatchType() {
			return false
		}
	}

	for i := 0; i < len(s.GetPartitions()); i++ {
		si, ti := s.GetPartitions()[i], t.GetPartitions()[i]
		if !equalPartitions(si, ti) {
			return false
		}
	}
	return true
}

func equalPartitions(s, t *storepb.TablePartitionMetadata) bool {
	if s.GetName() != t.GetName() {
		return false
	}
	if s.Type != t.Type {
		return false
	}
	if s.Expression != t.Expression {
		return false
	}
	if s.Value != t.Value {
		return false
	}
	if s.UseDefault != t.UseDefault {
		return false
	}
	if len(s.Subpartitions) != len(t.Subpartitions) {
		return false
	}
	for i := 0; i < len(s.Subpartitions); i++ {
		if !equalPartitions(s.Subpartitions[i], t.Subpartitions[i]) {
			return false
		}
	}
	return true
}
