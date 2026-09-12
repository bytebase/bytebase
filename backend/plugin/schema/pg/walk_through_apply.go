package pg

import (
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/pg/catalog"
	omniparser "github.com/bytebase/omni/pg/parser"
)

// applyDiffToMetadata applies a SchemaDiff (from user DDL execution) to a
// clone of the original metadata proto, producing the post-DDL metadata.
// The original proto is NOT modified.
func applyDiffToMetadata(original *metadatapb.DatabaseSchemaMetadata, catBefore, catAfter *catalog.Catalog, diff *catalog.SchemaDiff) *metadatapb.DatabaseSchemaMetadata {
	if diff == nil || diff.IsEmpty() {
		return original
	}

	result, ok := proto.Clone(original).(*metadatapb.DatabaseSchemaMetadata)
	if !ok {
		return original
	}

	for _, s := range diff.Schemas {
		switch s.Action {
		case catalog.DiffAdd:
			result.Schemas = append(result.Schemas, &metadatapb.SchemaMetadata{Name: s.Name})
		case catalog.DiffDrop:
			result.Schemas = removeSchema(result.Schemas, s.Name)
		default:
		}
	}

	renamed := renamePartitions(result, catAfter, diff.Relations)
	for _, rel := range diff.Relations {
		if (rel.From != nil && renamed[rel.From.OID]) || (rel.To != nil && renamed[rel.To.OID]) {
			continue
		}
		if rel.Action == catalog.DiffDrop {
			if sm := findSchema(result, rel.SchemaName); sm != nil {
				dropRelation(sm, rel)
			}
			continue
		}
		sm := findOrCreateSchema(result, rel.SchemaName)
		switch rel.Action {
		case catalog.DiffAdd:
			addRelation(sm, catAfter, rel)
		case catalog.DiffModify:
			modifyRelation(sm, catBefore, catAfter, rel)
		default:
		}
	}

	for _, seq := range diff.Sequences {
		if seq.Action == catalog.DiffDrop {
			if sm := findSchema(result, seq.SchemaName); sm != nil {
				sm.Sequences = removeSequenceByName(sm.Sequences, seq.Name)
			}
			continue
		}
		sm := findOrCreateSchema(result, seq.SchemaName)
		switch seq.Action {
		case catalog.DiffAdd:
			if seq.To != nil {
				sm.Sequences = append(sm.Sequences, sequenceToProto(catAfter, seq.To))
			}
		case catalog.DiffModify:
			if seq.To != nil {
				for i, s := range sm.Sequences {
					if s.Name == seq.Name {
						sm.Sequences[i] = sequenceToProto(catAfter, seq.To)
						break
					}
				}
			}
		default:
		}
	}

	for _, e := range diff.Enums {
		if e.Action == catalog.DiffDrop {
			if sm := findSchema(result, e.SchemaName); sm != nil {
				sm.EnumTypes = removeEnumByName(sm.EnumTypes, e.Name)
			}
			continue
		}
		sm := findOrCreateSchema(result, e.SchemaName)
		switch e.Action {
		case catalog.DiffAdd:
			sm.EnumTypes = append(sm.EnumTypes, &metadatapb.EnumTypeMetadata{
				Name:   e.Name,
				Values: e.ToValues,
			})
		case catalog.DiffModify:
			for i, et := range sm.EnumTypes {
				if et.Name == e.Name {
					sm.EnumTypes[i].Values = e.ToValues
					break
				}
			}
		default:
		}
	}

	for _, ct := range diff.CompositeTypes {
		if ct.Action == catalog.DiffDrop {
			if sm := findSchema(result, ct.SchemaName); sm != nil {
				sm.CompositeTypes = removeCompositeTypeByName(sm.CompositeTypes, ct.Name)
			}
			continue
		}
		if ct.To == nil {
			continue
		}
		sm := findOrCreateSchema(result, ct.SchemaName)
		switch ct.Action {
		case catalog.DiffAdd:
			sm.CompositeTypes = append(sm.CompositeTypes, compositeTypeToProto(catAfter, ct.Name, nil, ct.To, nil))
		case catalog.DiffModify:
			for i, existing := range sm.CompositeTypes {
				if existing.Name == ct.Name {
					sm.CompositeTypes[i] = compositeTypeToProto(catAfter, ct.Name, ct.From, ct.To, existing)
					break
				}
			}
		default:
		}
	}

	for _, f := range diff.Functions {
		isProcedure := f.To != nil && f.To.Kind == 'p'
		wasProcedure := f.From != nil && f.From.Kind == 'p'
		if f.Action == catalog.DiffDrop {
			if sm := findSchema(result, f.SchemaName); sm != nil {
				if wasProcedure {
					sm.Procedures = removeProcedureByIdentity(sm.Procedures, f.Identity)
				} else {
					sm.Functions = removeFunctionByIdentity(sm.Functions, f.Identity)
				}
			}
			continue
		}
		sm := findOrCreateSchema(result, f.SchemaName)
		switch f.Action {
		case catalog.DiffAdd:
			if f.To != nil {
				if isProcedure {
					sm.Procedures = append(sm.Procedures, procedureToProto(catAfter, f.To, f.Identity))
				} else {
					sm.Functions = append(sm.Functions, functionToProto(catAfter, f.To, f.Identity))
				}
			}
		case catalog.DiffModify:
			if wasProcedure {
				sm.Procedures = removeProcedureByIdentity(sm.Procedures, f.Identity)
			} else {
				sm.Functions = removeFunctionByIdentity(sm.Functions, f.Identity)
			}
			if f.To != nil {
				if isProcedure {
					sm.Procedures = append(sm.Procedures, procedureToProto(catAfter, f.To, f.Identity))
				} else {
					sm.Functions = append(sm.Functions, functionToProto(catAfter, f.To, f.Identity))
				}
			}
		default:
		}
	}

	return result
}

func findSchema(meta *metadatapb.DatabaseSchemaMetadata, name string) *metadatapb.SchemaMetadata {
	for _, s := range meta.Schemas {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func findOrCreateSchema(meta *metadatapb.DatabaseSchemaMetadata, name string) *metadatapb.SchemaMetadata {
	for _, s := range meta.Schemas {
		if s.Name == name {
			return s
		}
	}
	s := &metadatapb.SchemaMetadata{Name: name}
	meta.Schemas = append(meta.Schemas, s)
	return s
}

func removeSchema(schemas []*metadatapb.SchemaMetadata, name string) []*metadatapb.SchemaMetadata {
	out := make([]*metadatapb.SchemaMetadata, 0, len(schemas))
	for _, s := range schemas {
		if s.Name != name {
			out = append(out, s)
		}
	}
	return out
}

func addRelation(sm *metadatapb.SchemaMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
	if rel.To == nil {
		return
	}
	switch rel.To.RelKind {
	case 'r', 'p', 'f':
		sm.Tables = append(sm.Tables, relationToTableProto(cat, rel.To))
	case 'v':
		sm.Views = append(sm.Views, relationToViewProto(cat, rel.To))
	case 'm':
		sm.MaterializedViews = append(sm.MaterializedViews, relationToMatViewProto(cat, rel.To))
	default:
	}
}

func dropRelation(sm *metadatapb.SchemaMetadata, rel catalog.RelationDiffEntry) {
	name := rel.Name
	sm.Tables = removeTableByName(sm.Tables, name)
	sm.Views = removeViewByName(sm.Views, name)
	sm.MaterializedViews = removeMatViewByName(sm.MaterializedViews, name)
	for _, t := range sm.Tables {
		t.Partitions = removePartitionByName(t.Partitions, name)
	}
}

func modifyRelation(sm *metadatapb.SchemaMetadata, catBefore, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
	if rel.To == nil {
		return
	}
	switch rel.To.RelKind {
	case 'r', 'p', 'f':
		tbl := findTable(sm, rel.Name)
		if tbl == nil {
			if p := findPartition(sm, rel.Name); p != nil {
				applyPartitionDiffs(p, catBefore, cat, rel)
				return
			}
			sm.Tables = append(sm.Tables, relationToTableProto(cat, rel.To))
			return
		}
		applyColumnDiffs(tbl, cat, rel)
		applyIndexDiffs(tbl, cat, rel)
		applyConstraintDiffs(tbl, catBefore, cat, rel)
	case 'v':
		sm.Views = removeViewByName(sm.Views, rel.Name)
		sm.Views = append(sm.Views, relationToViewProto(cat, rel.To))
	case 'm':
		sm.MaterializedViews = removeMatViewByName(sm.MaterializedViews, rel.Name)
		sm.MaterializedViews = append(sm.MaterializedViews, relationToMatViewProto(cat, rel.To))
	default:
	}
}

// renamePartitions renames each partition the diff renames where it stands,
// with its indexes and constraints as the catalog now has them. The diff shows
// a rename as a drop and an add of the same relation, which would otherwise
// turn the partition into a table of its own. It returns the OIDs it handled.
func renamePartitions(result *metadatapb.DatabaseSchemaMetadata, catAfter *catalog.Catalog, relations []catalog.RelationDiffEntry) map[uint32]bool {
	added := make(map[uint32]catalog.RelationDiffEntry)
	for _, rel := range relations {
		if rel.Action == catalog.DiffAdd && rel.To != nil {
			added[rel.To.OID] = rel
		}
	}
	renamed := make(map[uint32]bool)
	for _, rel := range relations {
		if rel.Action != catalog.DiffDrop || rel.From == nil {
			continue
		}
		add, ok := added[rel.From.OID]
		sm := findSchema(result, rel.SchemaName)
		if !ok || add.SchemaName != rel.SchemaName || sm == nil {
			continue
		}
		if p := findPartition(sm, rel.Name); p != nil {
			tbl := relationToTableProto(catAfter, add.To)
			p.Name = add.Name
			p.Indexes, p.CheckConstraints, p.ExcludeConstraints = tbl.Indexes, tbl.CheckConstraints, tbl.ExcludeConstraints
			renamed[rel.From.OID] = true
		}
	}
	return renamed
}

// applyPartitionDiffs applies a partition's index and constraint changes. A
// partition's metadata holds only those; its columns are its parent table's.
func applyPartitionDiffs(p *metadatapb.TablePartitionMetadata, catBefore, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
	tbl := &metadatapb.TableMetadata{Indexes: p.Indexes, CheckConstraints: p.CheckConstraints, ExcludeConstraints: p.ExcludeConstraints}
	applyIndexDiffs(tbl, cat, rel)
	applyConstraintDiffs(tbl, catBefore, cat, rel)
	p.Indexes, p.CheckConstraints, p.ExcludeConstraints = tbl.Indexes, tbl.CheckConstraints, tbl.ExcludeConstraints
}

func applyColumnDiffs(tbl *metadatapb.TableMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
	for _, cd := range rel.Columns {
		switch cd.Action {
		case catalog.DiffAdd:
			if cd.To != nil {
				tbl.Columns = append(tbl.Columns, columnToProto(cat, cd.To))
			}
		case catalog.DiffDrop:
			tbl.Columns = removeColumnByName(tbl.Columns, cd.Name)
		case catalog.DiffModify:
			if cd.To != nil {
				for i, col := range tbl.Columns {
					if col.Name == cd.Name {
						tbl.Columns[i] = columnToProto(cat, cd.To)
						break
					}
				}
			}
		default:
		}
	}
}

func applyIndexDiffs(tbl *metadatapb.TableMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
	for _, id := range rel.Indexes {
		switch id.Action {
		case catalog.DiffAdd:
			if id.To != nil {
				tbl.Indexes = append(tbl.Indexes, indexToProto(cat, rel.To, id.To))
			}
		case catalog.DiffDrop:
			tbl.Indexes = removeIndexByName(tbl.Indexes, id.Name)
		case catalog.DiffModify:
			if id.To != nil {
				for i, idx := range tbl.Indexes {
					if idx.Name == id.Name {
						tbl.Indexes[i] = indexToProto(cat, rel.To, id.To)
						break
					}
				}
			}
		default:
		}
	}
}

func applyConstraintDiffs(tbl *metadatapb.TableMetadata, catBefore, catAfter *catalog.Catalog, rel catalog.RelationDiffEntry) {
	for _, cd := range rel.Constraints {
		switch cd.Action {
		case catalog.DiffAdd:
			if cd.To != nil {
				addConstraintToTable(tbl, catAfter, rel, cd.To)
			}
		case catalog.DiffDrop:
			if cd.From != nil {
				removeConstraintFromTable(tbl, catBefore, cd.From)
			}
		case catalog.DiffModify:
			if cd.From != nil {
				removeConstraintFromTable(tbl, catBefore, cd.From)
			}
			if cd.To != nil {
				addConstraintToTable(tbl, catAfter, rel, cd.To)
			}
		default:
		}
	}
}

func addConstraintToTable(tbl *metadatapb.TableMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry, con *catalog.Constraint) {
	switch con.Type {
	case catalog.ConstraintPK, catalog.ConstraintUnique:
		idx := cat.GetIndexByOID(con.IndexOID)
		if idx != nil && rel.To != nil {
			tbl.Indexes = append(tbl.Indexes, indexToProto(cat, rel.To, idx))
		}
	case catalog.ConstraintFK:
		tbl.ForeignKeys = append(tbl.ForeignKeys, constraintToFKProto(cat, rel.To, con))
	case catalog.ConstraintCheck:
		tbl.CheckConstraints = append(tbl.CheckConstraints, &metadatapb.CheckConstraintMetadata{
			Name:       con.Name,
			Expression: con.CheckExpr,
		})
	case catalog.ConstraintExclude:
		tbl.ExcludeConstraints = append(tbl.ExcludeConstraints, &metadatapb.ExcludeConstraintMetadata{
			Name:       con.Name,
			Expression: con.CheckExpr,
		})
	default:
	}
}

func removeConstraintFromTable(tbl *metadatapb.TableMetadata, cat *catalog.Catalog, con *catalog.Constraint) {
	switch con.Type {
	case catalog.ConstraintPK, catalog.ConstraintUnique:
		// Constraint name and backing index name may differ; resolve via IndexOID.
		idxName := con.Name
		if idx := cat.GetIndexByOID(con.IndexOID); idx != nil {
			idxName = idx.Name
		}
		tbl.Indexes = removeIndexByName(tbl.Indexes, idxName)
	case catalog.ConstraintFK:
		out := make([]*metadatapb.ForeignKeyMetadata, 0, len(tbl.ForeignKeys))
		for _, fk := range tbl.ForeignKeys {
			if fk.Name != con.Name {
				out = append(out, fk)
			}
		}
		tbl.ForeignKeys = out
	case catalog.ConstraintCheck:
		out := make([]*metadatapb.CheckConstraintMetadata, 0, len(tbl.CheckConstraints))
		for _, c := range tbl.CheckConstraints {
			if c.Name != con.Name {
				out = append(out, c)
			}
		}
		tbl.CheckConstraints = out
	case catalog.ConstraintExclude:
		out := make([]*metadatapb.ExcludeConstraintMetadata, 0, len(tbl.ExcludeConstraints))
		for _, c := range tbl.ExcludeConstraints {
			if c.Name != con.Name {
				out = append(out, c)
			}
		}
		tbl.ExcludeConstraints = out
	default:
	}
}

// --- Conversion helpers: omni types → proto types ---

func columnToProto(cat *catalog.Catalog, col *catalog.Column) *metadatapb.ColumnMetadata {
	cm := &metadatapb.ColumnMetadata{
		Name:     col.Name,
		Position: int32(col.AttNum),
		Type:     cat.FormatType(col.TypeOID, col.TypeMod),
		Nullable: !col.NotNull,
		Default:  col.Default,
	}
	if col.Generated == 's' {
		cm.Generation = &metadatapb.GenerationMetadata{
			Type:       metadatapb.GenerationMetadata_TYPE_STORED,
			Expression: col.GenerationExpr,
		}
	}
	if col.Identity != 0 {
		cm.IsIdentity = true
		switch col.Identity {
		case 'a':
			cm.IdentityGeneration = metadatapb.ColumnMetadata_ALWAYS
		case 'd':
			cm.IdentityGeneration = metadatapb.ColumnMetadata_BY_DEFAULT
		default:
		}
	}
	return cm
}

func indexToProto(_ *catalog.Catalog, rel *catalog.Relation, idx *catalog.Index) *metadatapb.IndexMetadata {
	im := &metadatapb.IndexMetadata{
		Name:         idx.Name,
		Type:         idx.AccessMethod,
		Unique:       idx.IsUnique,
		Primary:      idx.IsPrimary,
		IsConstraint: idx.ConstraintOID != 0,
	}

	exprIdx := 0
	for i, attnum := range idx.Columns {
		if attnum == 0 {
			if exprIdx < len(idx.Exprs) {
				im.Expressions = append(im.Expressions, idx.Exprs[exprIdx])
				exprIdx++
			}
		} else {
			colName := ""
			if rel != nil {
				for _, col := range rel.Columns {
					if col.AttNum == attnum {
						colName = col.Name
						break
					}
				}
			}
			im.Expressions = append(im.Expressions, colName)
		}

		if i < len(idx.IndOption) {
			im.Descending = append(im.Descending, idx.IndOption[i]&1 != 0)
		}
	}

	return im
}

func constraintToFKProto(cat *catalog.Catalog, rel *catalog.Relation, con *catalog.Constraint) *metadatapb.ForeignKeyMetadata {
	fk := &metadatapb.ForeignKeyMetadata{
		Name: con.Name,
	}

	for _, attnum := range con.Columns {
		if rel != nil {
			for _, col := range rel.Columns {
				if col.AttNum == attnum {
					fk.Columns = append(fk.Columns, col.Name)
					break
				}
			}
		}
	}

	fRel := cat.GetRelationByOID(con.FRelOID)
	if fRel != nil {
		if fRel.Schema != nil {
			fk.ReferencedSchema = fRel.Schema.Name
		}
		fk.ReferencedTable = fRel.Name
		for _, attnum := range con.FColumns {
			for _, col := range fRel.Columns {
				if col.AttNum == attnum {
					fk.ReferencedColumns = append(fk.ReferencedColumns, col.Name)
					break
				}
			}
		}
	}

	fk.OnUpdate = wtFKActionToString(con.FKUpdAction)
	fk.OnDelete = wtFKActionToString(con.FKDelAction)
	fk.MatchType = wtFKMatchToString(con.FKMatchType)
	return fk
}

func sequenceToProto(cat *catalog.Catalog, seq *catalog.Sequence) *metadatapb.SequenceMetadata {
	return &metadatapb.SequenceMetadata{
		Name:      seq.Name,
		DataType:  cat.FormatType(seq.TypeOID, -1),
		Start:     fmt.Sprintf("%d", seq.Start),
		MinValue:  fmt.Sprintf("%d", seq.MinValue),
		MaxValue:  fmt.Sprintf("%d", seq.MaxValue),
		Increment: fmt.Sprintf("%d", seq.Increment),
		Cycle:     seq.Cycle,
		CacheSize: fmt.Sprintf("%d", seq.CacheValue),
	}
}

func wtFKActionToString(action byte) string {
	switch action {
	case 'r':
		return "RESTRICT"
	case 'c':
		return "CASCADE"
	case 'n':
		return "SET NULL"
	case 'd':
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

func wtFKMatchToString(match byte) string {
	switch match {
	case 'f':
		return "FULL"
	case 'p':
		return "PARTIAL"
	default:
		return "SIMPLE"
	}
}

// --- Slice helpers ---

func findTable(sm *metadatapb.SchemaMetadata, name string) *metadatapb.TableMetadata {
	for _, t := range sm.Tables {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// findPartition returns the partition named name, at any depth, of a table in sm.
func findPartition(sm *metadatapb.SchemaMetadata, name string) *metadatapb.TablePartitionMetadata {
	var find func(partitions []*metadatapb.TablePartitionMetadata) *metadatapb.TablePartitionMetadata
	find = func(partitions []*metadatapb.TablePartitionMetadata) *metadatapb.TablePartitionMetadata {
		for _, p := range partitions {
			if p.Name == name {
				return p
			}
			if sub := find(p.Subpartitions); sub != nil {
				return sub
			}
		}
		return nil
	}
	for _, t := range sm.Tables {
		if p := find(t.Partitions); p != nil {
			return p
		}
	}
	return nil
}

// removePartitionByName removes the partition named name, at any depth.
func removePartitionByName(partitions []*metadatapb.TablePartitionMetadata, name string) []*metadatapb.TablePartitionMetadata {
	partitions = slices.DeleteFunc(partitions, func(p *metadatapb.TablePartitionMetadata) bool { return p.Name == name })
	for _, p := range partitions {
		p.Subpartitions = removePartitionByName(p.Subpartitions, name)
	}
	return partitions
}

func removeTableByName(tables []*metadatapb.TableMetadata, name string) []*metadatapb.TableMetadata {
	out := make([]*metadatapb.TableMetadata, 0, len(tables))
	for _, t := range tables {
		if t.Name != name {
			out = append(out, t)
		}
	}
	return out
}

func removeViewByName(views []*metadatapb.ViewMetadata, name string) []*metadatapb.ViewMetadata {
	out := make([]*metadatapb.ViewMetadata, 0, len(views))
	for _, v := range views {
		if v.Name != name {
			out = append(out, v)
		}
	}
	return out
}

func removeMatViewByName(mvs []*metadatapb.MaterializedViewMetadata, name string) []*metadatapb.MaterializedViewMetadata {
	out := make([]*metadatapb.MaterializedViewMetadata, 0, len(mvs))
	for _, m := range mvs {
		if m.Name != name {
			out = append(out, m)
		}
	}
	return out
}

func removeSequenceByName(seqs []*metadatapb.SequenceMetadata, name string) []*metadatapb.SequenceMetadata {
	out := make([]*metadatapb.SequenceMetadata, 0, len(seqs))
	for _, s := range seqs {
		if s.Name != name {
			out = append(out, s)
		}
	}
	return out
}

func removeColumnByName(cols []*metadatapb.ColumnMetadata, name string) []*metadatapb.ColumnMetadata {
	out := make([]*metadatapb.ColumnMetadata, 0, len(cols))
	for _, c := range cols {
		if c.Name != name {
			out = append(out, c)
		}
	}
	return out
}

func removeIndexByName(indexes []*metadatapb.IndexMetadata, name string) []*metadatapb.IndexMetadata {
	out := make([]*metadatapb.IndexMetadata, 0, len(indexes))
	for _, i := range indexes {
		if i.Name != name {
			out = append(out, i)
		}
	}
	return out
}

func removeEnumByName(enums []*metadatapb.EnumTypeMetadata, name string) []*metadatapb.EnumTypeMetadata {
	out := make([]*metadatapb.EnumTypeMetadata, 0, len(enums))
	for _, e := range enums {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return out
}

func functionToProto(cat *catalog.Catalog, up *catalog.UserProc, identity string) *metadatapb.FunctionMetadata {
	return &metadatapb.FunctionMetadata{
		Name:       up.Name,
		Definition: buildUserProcDDL(cat, up),
		Signature:  identity,
	}
}

func procedureToProto(cat *catalog.Catalog, up *catalog.UserProc, identity string) *metadatapb.ProcedureMetadata {
	return &metadatapb.ProcedureMetadata{
		Name:       up.Name,
		Definition: buildUserProcDDL(cat, up),
		Signature:  identity,
	}
}

func buildUserProcDDL(cat *catalog.Catalog, up *catalog.UserProc) string {
	var b strings.Builder
	if up.Kind == 'p' {
		b.WriteString("CREATE OR REPLACE PROCEDURE ")
	} else {
		b.WriteString("CREATE OR REPLACE FUNCTION ")
	}
	if up.Schema != nil && up.Schema.Name != "" {
		b.WriteString(up.Schema.Name)
		b.WriteByte('.')
	}
	b.WriteString(up.Name)
	b.WriteByte('(')
	argTypes := up.ArgTypes
	if len(up.AllArgTypes) > 0 {
		argTypes = up.AllArgTypes
	}
	for i, t := range argTypes {
		if i > 0 {
			b.WriteString(", ")
		}
		if i < len(up.ArgModes) {
			switch up.ArgModes[i] {
			case 'o':
				b.WriteString("OUT ")
			case 'b':
				b.WriteString("INOUT ")
			case 'v':
				b.WriteString("VARIADIC ")
			default:
			}
		}
		if i < len(up.ArgNames) && up.ArgNames[i] != "" {
			b.WriteString(up.ArgNames[i])
			b.WriteByte(' ')
		}
		b.WriteString(cat.FormatType(t, -1))
	}
	b.WriteString(")\n")
	if up.Kind != 'p' {
		b.WriteString(" RETURNS ")
		if up.RetSet {
			b.WriteString("SETOF ")
		}
		b.WriteString(cat.FormatType(up.RetType, -1))
		b.WriteByte('\n')
	}
	b.WriteString(" LANGUAGE ")
	b.WriteString(up.Language)
	b.WriteByte('\n')
	switch up.Volatile {
	case 'i':
		b.WriteString(" IMMUTABLE\n")
	case 's':
		b.WriteString(" STABLE\n")
	default:
	}
	if up.IsStrict {
		b.WriteString(" STRICT\n")
	}
	if up.SecDef {
		b.WriteString(" SECURITY DEFINER\n")
	}
	if up.LeakProof {
		b.WriteString(" LEAKPROOF\n")
	}
	switch up.Parallel {
	case 's':
		b.WriteString(" PARALLEL SAFE\n")
	case 'r':
		b.WriteString(" PARALLEL RESTRICTED\n")
	default:
	}
	b.WriteString("AS $function$")
	b.WriteString(up.Body)
	b.WriteString("$function$\n")
	return b.String()
}

func removeProcedureByIdentity(procs []*metadatapb.ProcedureMetadata, identity string) []*metadatapb.ProcedureMetadata {
	out := make([]*metadatapb.ProcedureMetadata, 0, len(procs))
	for _, p := range procs {
		if p.Signature != identity {
			out = append(out, p)
		}
	}
	return out
}

func removeFunctionByIdentity(funcs []*metadatapb.FunctionMetadata, identity string) []*metadatapb.FunctionMetadata {
	out := make([]*metadatapb.FunctionMetadata, 0, len(funcs))
	for _, f := range funcs {
		if f.Signature != identity {
			out = append(out, f)
		}
	}
	return out
}

// compositeTypeToProto builds metadata from the catalog relation, preserving
// the type comment and per-attribute comments/collations from the previous
// metadata (the catalog does not track comments, and it only keeps the bare
// collation name) for attributes that survive by name. Attributes unchanged
// between from and to keep their previous metadata verbatim, so a composite
// degraded to a text-backed stand-in does not rewrite untouched
// attributes to text — mirroring the per-column granularity of table diffs.
//
// Within a degraded composite this is a deliberate trade-off: an attribute
// genuinely altered TO text is indistinguishable from an unchanged one (both
// read text in from and to), so that rare change is missed in favor of not
// corrupting every untouched attribute. Walk-through metadata is advisory;
// post-execution sync restores ground truth. The root fix is modeling the
// types LoadMetadata cannot install today (e.g. domains).
func compositeTypeToProto(cat *catalog.Catalog, name string, from, to *catalog.Relation, previous *metadatapb.CompositeTypeMetadata) *metadatapb.CompositeTypeMetadata {
	previousAttributes := make(map[string]*metadatapb.CompositeTypeAttribute)
	composite := &metadatapb.CompositeTypeMetadata{Name: name}
	if previous != nil {
		composite.Comment = previous.Comment
		composite.SkipDump = previous.SkipDump
		for _, attribute := range previous.Attributes {
			previousAttributes[attribute.Name] = attribute
		}
	}
	fromByAttNum := make(map[int16]*catalog.Column)
	if from != nil {
		for _, c := range from.Columns {
			if c != nil {
				fromByAttNum[c.AttNum] = c
			}
		}
	}
	for _, c := range to.Columns {
		if c == nil || c.Name == "" {
			continue
		}
		// The attribute number is the stable column identity: it survives
		// RENAME ATTRIBUTE and is never reused, so a dropped-and-re-added
		// attribute (same name, new attnum) correctly reads as a new column
		// while a renamed survivor still carries its previous metadata.
		fc := fromByAttNum[c.AttNum]
		if fc != nil &&
			fc.TypeOID == c.TypeOID && fc.TypeMod == c.TypeMod && fc.CollationName == c.CollationName {
			if prev := previousAttributes[fc.Name]; prev != nil {
				composite.Attributes = append(composite.Attributes, &metadatapb.CompositeTypeAttribute{
					Name:      c.Name,
					Type:      prev.Type,
					Collation: prev.Collation,
					Comment:   prev.Comment,
				})
				continue
			}
		}
		attribute := &metadatapb.CompositeTypeAttribute{
			Name: c.Name,
			Type: qualifyWalkThroughAttributeType(cat, c.TypeOID, cat.FormatType(c.TypeOID, c.TypeMod)),
		}
		// Previous metadata is only consulted for the column identity that
		// survived (same attnum): a dropped-and-re-added attribute must not
		// inherit the old attribute's collation reference or comment.
		var surviving *metadatapb.CompositeTypeAttribute
		if fc != nil {
			surviving = previousAttributes[fc.Name]
		}
		if c.CollationName != "" {
			// The catalog keeps only the bare collation name, so a collation
			// introduced by walk-through DDL loses its schema qualifier here
			// (same limitation as below). Walk-through metadata is advisory —
			// the authoritative metadata is re-synced from the database after
			// execution, which stores the properly qualified reference.
			attribute.Collation = "\"" + c.CollationName + "\""
			// Substitute the previous emit-ready (possibly schema-qualified)
			// reference only when it names the same collation — a changed
			// collation must reflect the catalog's current value.
			if surviving != nil && collationReferenceBareName(surviving.Collation) == c.CollationName {
				attribute.Collation = surviving.Collation
			}
		}
		if surviving != nil {
			attribute.Comment = surviving.Comment
		}
		composite.Attributes = append(composite.Attributes, attribute)
	}
	return composite
}

func removeCompositeTypeByName(composites []*metadatapb.CompositeTypeMetadata, name string) []*metadatapb.CompositeTypeMetadata {
	out := make([]*metadatapb.CompositeTypeMetadata, 0, len(composites))
	for _, composite := range composites {
		if composite.Name != name {
			out = append(out, composite)
		}
	}
	return out
}

// collationReferenceBareName extracts the bare collation name from a stored
// emit-ready reference (e.g. `"C"` -> C, `locale.en_us` -> en_us) for
// comparison with the catalog's collation name. The comparison cannot see
// schema qualifiers — the omni catalog only records the bare collation name —
// so switching an attribute between two same-named collations in different
// schemas keeps the previous reference. Known, accepted limitation.
func collationReferenceBareName(reference string) string {
	if _, name, ok := parseQualifiedTypeIdent(reference); ok {
		return name
	}
	name := strings.Trim(reference, `"`)
	return strings.ReplaceAll(name, `""`, `"`)
}

// qualifyWalkThroughAttributeType enforces the CompositeTypeAttribute.Type
// contract: user-defined types are always schema-qualified. FormatType
// renders search-path-visible types unqualified, so the (element) type's
// namespace is resolved and prepended when missing. System-schema types come
// back unqualified from QueryPgNamespace and stay untouched.
func qualifyWalkThroughAttributeType(cat *catalog.Catalog, typeOID uint32, formatted string) string {
	t := cat.TypeByOID(typeOID)
	if t == nil {
		return formatted
	}
	element := t
	if t.Elem != 0 && t.Len == -1 {
		if el := cat.TypeByOID(t.Elem); el != nil {
			element = el
		}
	}
	schemaName := ""
	for _, row := range cat.QueryPgNamespace() {
		if row.OID == element.Namespace {
			schemaName = row.NspName
			break
		}
	}
	// QueryPgNamespace already omits builtin-OID schemas; the explicit check
	// keeps the invariant independent of the catalog's OID layout and covers
	// information_schema, mirroring the sync query's exclusions.
	if schemaName == "" || wtIsSystemSchema(schemaName) {
		return formatted
	}
	prefix := quoteWalkThroughIdent(schemaName) + "."
	quotedPrefix := `"` + strings.ReplaceAll(schemaName, `"`, `""`) + `".`
	if strings.HasPrefix(formatted, prefix) || strings.HasPrefix(formatted, quotedPrefix) {
		return formatted
	}
	return prefix + formatted
}

// quoteWalkThroughIdent mirrors quote_ident: quote when the name needs it,
// including reserved keywords.
func quoteWalkThroughIdent(name string) string {
	simple := name != ""
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || r == '_' || (i > 0 && ((r >= '0' && r <= '9') || r == '$')) {
			continue
		}
		simple = false
		break
	}
	if simple && !omniparser.IsReservedKeyword(name) {
		return name
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func wtIsSystemSchema(s string) bool {
	switch s {
	case "pg_catalog", "pg_toast", "information_schema":
		return true
	}
	return strings.HasPrefix(s, "pg_")
}
