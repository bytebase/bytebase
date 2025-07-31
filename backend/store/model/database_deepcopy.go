package model

import (
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// DeepCopy creates a deep copy of the DatabaseMetadata.
func (d *DatabaseMetadata) DeepCopy() *DatabaseMetadata {
	if d == nil {
		return nil
	}

	copiedMetadata := &DatabaseMetadata{
		name:                  d.name,
		owner:                 d.owner,
		isObjectCaseSensitive: d.isObjectCaseSensitive,
		isDetailCaseSensitive: d.isDetailCaseSensitive,
		internal:              make(map[string]*SchemaMetadata),
		linkedDatabase:        make(map[string]*LinkedDatabaseMetadata),
	}

	// Deep copy search path
	if d.searchPath != nil {
		copiedMetadata.searchPath = make([]string, len(d.searchPath))
		copy(copiedMetadata.searchPath, d.searchPath)
	}

	// Deep copy internal schemas
	for schemaName, schemaMetadata := range d.internal {
		copiedMetadata.internal[schemaName] = schemaMetadata.DeepCopy()
	}

	// Deep copy linked databases
	for dbName, linkedDB := range d.linkedDatabase {
		copiedMetadata.linkedDatabase[dbName] = linkedDB.DeepCopy()
	}

	return copiedMetadata
}

// DeepCopy creates a deep copy of the SchemaMetadata.
func (s *SchemaMetadata) DeepCopy() *SchemaMetadata {
	if s == nil {
		return nil
	}

	copiedData := &SchemaMetadata{
		isObjectCaseSensitive:    s.isObjectCaseSensitive,
		isDetailCaseSensitive:    s.isDetailCaseSensitive,
		internalTables:           make(map[string]*TableMetadata),
		internalExternalTable:    make(map[string]*ExternalTableMetadata),
		internalViews:            make(map[string]*ViewMetadata),
		internalMaterializedView: make(map[string]*MaterializedViewMetadata),
		internalFunctions:        make([]*FunctionMetadata, 0, len(s.internalFunctions)),
		internalProcedures:       make(map[string]*ProcedureMetadata),
		internalPackages:         make(map[string]*PackageMetadata),
		internalSequences:        make(map[string]*SequenceMetadata),
	}

	// Deep copy proto
	if s.proto != nil {
		data, _ := protojson.Marshal(s.proto)
		copiedData.proto = &storepb.SchemaMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	// Deep copy tables
	for tableName, tableMetadata := range s.internalTables {
		copiedData.internalTables[tableName] = tableMetadata.DeepCopy()
	}

	// Deep copy external tables
	for tableName, externalTable := range s.internalExternalTable {
		copiedData.internalExternalTable[tableName] = externalTable.DeepCopy()
	}

	// Deep copy views
	for viewName, viewMetadata := range s.internalViews {
		copiedData.internalViews[viewName] = viewMetadata.DeepCopy()
	}

	// Deep copy materialized views
	for viewName, materializedView := range s.internalMaterializedView {
		copiedData.internalMaterializedView[viewName] = materializedView.DeepCopy()
	}

	// Deep copy functions
	for _, function := range s.internalFunctions {
		copiedData.internalFunctions = append(copiedData.internalFunctions, function.DeepCopy())
	}

	// Deep copy procedures
	for procName, procedure := range s.internalProcedures {
		copiedData.internalProcedures[procName] = procedure.DeepCopy()
	}

	// Deep copy packages
	for pkgName, pkg := range s.internalPackages {
		copiedData.internalPackages[pkgName] = pkg.DeepCopy()
	}

	// Deep copy sequences
	for seqName, sequence := range s.internalSequences {
		copiedData.internalSequences[seqName] = sequence.DeepCopy()
	}

	return copiedData
}

// DeepCopy creates a deep copy of the TableMetadata.
func (t *TableMetadata) DeepCopy() *TableMetadata {
	if t == nil {
		return nil
	}

	copiedData := &TableMetadata{
		isDetailCaseSensitive: t.isDetailCaseSensitive,
		partitionOf:           t.partitionOf,
		internalColumn:        make(map[string]*storepb.ColumnMetadata),
		internalIndexes:       make(map[string]*IndexMetadata),
		rowCount:              t.rowCount,
	}

	// Deep copy proto
	if t.proto != nil {
		data, _ := protojson.Marshal(t.proto)
		copiedData.proto = &storepb.TableMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	// Deep copy columns
	for colName, colMetadata := range t.internalColumn {
		if colMetadata != nil {
			data, _ := protojson.Marshal(colMetadata)
			colCopy := &storepb.ColumnMetadata{}
			_ = common.ProtojsonUnmarshaler.Unmarshal(data, colCopy)
			copiedData.internalColumn[colName] = colCopy
		}
	}

	// Deep copy columns slice
	if t.columns != nil {
		copiedData.columns = make([]*storepb.ColumnMetadata, len(t.columns))
		for i, col := range t.columns {
			if col != nil {
				data, _ := protojson.Marshal(col)
				colCopy := &storepb.ColumnMetadata{}
				_ = common.ProtojsonUnmarshaler.Unmarshal(data, colCopy)
				copiedData.columns[i] = colCopy
			}
		}
	}

	// Deep copy indexes
	for indexName, indexMetadata := range t.internalIndexes {
		copiedData.internalIndexes[indexName] = indexMetadata.DeepCopy()
	}

	return copiedData
}

// DeepCopy creates a deep copy of the ExternalTableMetadata.
func (t *ExternalTableMetadata) DeepCopy() *ExternalTableMetadata {
	if t == nil {
		return nil
	}

	copiedData := &ExternalTableMetadata{
		isDetailCaseSensitive: t.isDetailCaseSensitive,
		internal:              make(map[string]*storepb.ColumnMetadata),
	}

	// Deep copy proto
	if t.proto != nil {
		data, _ := protojson.Marshal(t.proto)
		copiedData.proto = &storepb.ExternalTableMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	// Deep copy columns map
	for colName, colMetadata := range t.internal {
		if colMetadata != nil {
			data, _ := protojson.Marshal(colMetadata)
			colCopy := &storepb.ColumnMetadata{}
			_ = common.ProtojsonUnmarshaler.Unmarshal(data, colCopy)
			copiedData.internal[colName] = colCopy
		}
	}

	// Deep copy columns slice
	if t.columns != nil {
		copiedData.columns = make([]*storepb.ColumnMetadata, len(t.columns))
		for i, col := range t.columns {
			if col != nil {
				data, _ := protojson.Marshal(col)
				colCopy := &storepb.ColumnMetadata{}
				_ = common.ProtojsonUnmarshaler.Unmarshal(data, colCopy)
				copiedData.columns[i] = colCopy
			}
		}
	}

	return copiedData
}

// DeepCopy creates a deep copy of the ViewMetadata.
func (v *ViewMetadata) DeepCopy() *ViewMetadata {
	if v == nil {
		return nil
	}

	copiedData := &ViewMetadata{
		Definition: v.Definition,
	}

	// Deep copy proto
	if v.proto != nil {
		data, _ := protojson.Marshal(v.proto)
		copiedData.proto = &storepb.ViewMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the MaterializedViewMetadata.
func (m *MaterializedViewMetadata) DeepCopy() *MaterializedViewMetadata {
	if m == nil {
		return nil
	}

	copiedData := &MaterializedViewMetadata{
		Definition: m.Definition,
	}

	// Deep copy proto
	if m.proto != nil {
		data, _ := protojson.Marshal(m.proto)
		copiedData.proto = &storepb.MaterializedViewMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the FunctionMetadata.
func (f *FunctionMetadata) DeepCopy() *FunctionMetadata {
	if f == nil {
		return nil
	}

	copiedData := &FunctionMetadata{
		Definition: f.Definition,
	}

	// Deep copy proto
	if f.proto != nil {
		data, _ := protojson.Marshal(f.proto)
		copiedData.proto = &storepb.FunctionMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the ProcedureMetadata.
func (p *ProcedureMetadata) DeepCopy() *ProcedureMetadata {
	if p == nil {
		return nil
	}

	copiedData := &ProcedureMetadata{
		Definition: p.Definition,
	}

	// Deep copy proto
	if p.proto != nil {
		data, _ := protojson.Marshal(p.proto)
		copiedData.proto = &storepb.ProcedureMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the PackageMetadata.
func (p *PackageMetadata) DeepCopy() *PackageMetadata {
	if p == nil {
		return nil
	}

	copiedData := &PackageMetadata{
		Definition: p.Definition,
	}

	// Deep copy proto
	if p.proto != nil {
		data, _ := protojson.Marshal(p.proto)
		copiedData.proto = &storepb.PackageMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the SequenceMetadata.
func (p *SequenceMetadata) DeepCopy() *SequenceMetadata {
	if p == nil {
		return nil
	}

	copiedData := &SequenceMetadata{}

	// Deep copy proto
	if p.proto != nil {
		data, _ := protojson.Marshal(p.proto)
		copiedData.proto = &storepb.SequenceMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the IndexMetadata.
func (i *IndexMetadata) DeepCopy() *IndexMetadata {
	if i == nil {
		return nil
	}

	copiedData := &IndexMetadata{}

	// Deep copy tableProto
	if i.tableProto != nil {
		data, _ := protojson.Marshal(i.tableProto)
		copiedData.tableProto = &storepb.TableMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.tableProto)
	}

	// Deep copy proto
	if i.proto != nil {
		data, _ := protojson.Marshal(i.proto)
		copiedData.proto = &storepb.IndexMetadata{}
		_ = common.ProtojsonUnmarshaler.Unmarshal(data, copiedData.proto)
	}

	return copiedData
}

// DeepCopy creates a deep copy of the LinkedDatabaseMetadata.
func (l *LinkedDatabaseMetadata) DeepCopy() *LinkedDatabaseMetadata {
	if l == nil {
		return nil
	}

	return &LinkedDatabaseMetadata{
		name:     l.name,
		username: l.username,
		host:     l.host,
	}
}
