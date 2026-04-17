package pg

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// applyDiffToMetadata applies a SchemaDiff (from user DDL execution) to a
// clone of the original metadata proto, producing the post-DDL metadata.
// The original proto is NOT modified.
func applyDiffToMetadata(original *storepb.DatabaseSchemaMetadata, catBefore, catAfter *catalog.Catalog, diff *catalog.SchemaDiff) *storepb.DatabaseSchemaMetadata {
	if diff == nil || diff.IsEmpty() {
		return original
	}

	result, ok := proto.Clone(original).(*storepb.DatabaseSchemaMetadata)
	if !ok {
		return original
	}

	for _, s := range diff.Schemas {
		switch s.Action {
		case catalog.DiffAdd:
			result.Schemas = append(result.Schemas, &storepb.SchemaMetadata{Name: s.Name})
		case catalog.DiffDrop:
			result.Schemas = removeSchema(result.Schemas, s.Name)
		default:
		}
	}

	for _, rel := range diff.Relations {
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
			sm.EnumTypes = append(sm.EnumTypes, &storepb.EnumTypeMetadata{
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

func findSchema(meta *storepb.DatabaseSchemaMetadata, name string) *storepb.SchemaMetadata {
	for _, s := range meta.Schemas {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func findOrCreateSchema(meta *storepb.DatabaseSchemaMetadata, name string) *storepb.SchemaMetadata {
	for _, s := range meta.Schemas {
		if s.Name == name {
			return s
		}
	}
	s := &storepb.SchemaMetadata{Name: name}
	meta.Schemas = append(meta.Schemas, s)
	return s
}

func removeSchema(schemas []*storepb.SchemaMetadata, name string) []*storepb.SchemaMetadata {
	out := make([]*storepb.SchemaMetadata, 0, len(schemas))
	for _, s := range schemas {
		if s.Name != name {
			out = append(out, s)
		}
	}
	return out
}

func addRelation(sm *storepb.SchemaMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
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

func dropRelation(sm *storepb.SchemaMetadata, rel catalog.RelationDiffEntry) {
	name := rel.Name
	sm.Tables = removeTableByName(sm.Tables, name)
	sm.Views = removeViewByName(sm.Views, name)
	sm.MaterializedViews = removeMatViewByName(sm.MaterializedViews, name)
}

func modifyRelation(sm *storepb.SchemaMetadata, catBefore, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
	if rel.To == nil {
		return
	}
	switch rel.To.RelKind {
	case 'r', 'p', 'f':
		tbl := findTable(sm, rel.Name)
		if tbl == nil {
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

func applyColumnDiffs(tbl *storepb.TableMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
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

func applyIndexDiffs(tbl *storepb.TableMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry) {
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

func applyConstraintDiffs(tbl *storepb.TableMetadata, catBefore, catAfter *catalog.Catalog, rel catalog.RelationDiffEntry) {
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

func addConstraintToTable(tbl *storepb.TableMetadata, cat *catalog.Catalog, rel catalog.RelationDiffEntry, con *catalog.Constraint) {
	switch con.Type {
	case catalog.ConstraintPK, catalog.ConstraintUnique:
		idx := cat.GetIndexByOID(con.IndexOID)
		if idx != nil && rel.To != nil {
			tbl.Indexes = append(tbl.Indexes, indexToProto(cat, rel.To, idx))
		}
	case catalog.ConstraintFK:
		tbl.ForeignKeys = append(tbl.ForeignKeys, constraintToFKProto(cat, rel.To, con))
	case catalog.ConstraintCheck:
		tbl.CheckConstraints = append(tbl.CheckConstraints, &storepb.CheckConstraintMetadata{
			Name:       con.Name,
			Expression: con.CheckExpr,
		})
	case catalog.ConstraintExclude:
		tbl.ExcludeConstraints = append(tbl.ExcludeConstraints, &storepb.ExcludeConstraintMetadata{
			Name:       con.Name,
			Expression: con.CheckExpr,
		})
	default:
	}
}

func removeConstraintFromTable(tbl *storepb.TableMetadata, cat *catalog.Catalog, con *catalog.Constraint) {
	switch con.Type {
	case catalog.ConstraintPK, catalog.ConstraintUnique:
		// Constraint name and backing index name may differ; resolve via IndexOID.
		idxName := con.Name
		if idx := cat.GetIndexByOID(con.IndexOID); idx != nil {
			idxName = idx.Name
		}
		tbl.Indexes = removeIndexByName(tbl.Indexes, idxName)
	case catalog.ConstraintFK:
		out := make([]*storepb.ForeignKeyMetadata, 0, len(tbl.ForeignKeys))
		for _, fk := range tbl.ForeignKeys {
			if fk.Name != con.Name {
				out = append(out, fk)
			}
		}
		tbl.ForeignKeys = out
	case catalog.ConstraintCheck:
		out := make([]*storepb.CheckConstraintMetadata, 0, len(tbl.CheckConstraints))
		for _, c := range tbl.CheckConstraints {
			if c.Name != con.Name {
				out = append(out, c)
			}
		}
		tbl.CheckConstraints = out
	case catalog.ConstraintExclude:
		out := make([]*storepb.ExcludeConstraintMetadata, 0, len(tbl.ExcludeConstraints))
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

func columnToProto(cat *catalog.Catalog, col *catalog.Column) *storepb.ColumnMetadata {
	cm := &storepb.ColumnMetadata{
		Name:     col.Name,
		Position: int32(col.AttNum),
		Type:     cat.FormatType(col.TypeOID, col.TypeMod),
		Nullable: !col.NotNull,
		Default:  col.Default,
	}
	if col.Generated == 's' {
		cm.Generation = &storepb.GenerationMetadata{
			Type:       storepb.GenerationMetadata_TYPE_STORED,
			Expression: col.GenerationExpr,
		}
	}
	if col.Identity != 0 {
		cm.IsIdentity = true
		switch col.Identity {
		case 'a':
			cm.IdentityGeneration = storepb.ColumnMetadata_ALWAYS
		case 'd':
			cm.IdentityGeneration = storepb.ColumnMetadata_BY_DEFAULT
		default:
		}
	}
	return cm
}

func indexToProto(_ *catalog.Catalog, rel *catalog.Relation, idx *catalog.Index) *storepb.IndexMetadata {
	im := &storepb.IndexMetadata{
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

func constraintToFKProto(cat *catalog.Catalog, rel *catalog.Relation, con *catalog.Constraint) *storepb.ForeignKeyMetadata {
	fk := &storepb.ForeignKeyMetadata{
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

func sequenceToProto(cat *catalog.Catalog, seq *catalog.Sequence) *storepb.SequenceMetadata {
	return &storepb.SequenceMetadata{
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

func findTable(sm *storepb.SchemaMetadata, name string) *storepb.TableMetadata {
	for _, t := range sm.Tables {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func removeTableByName(tables []*storepb.TableMetadata, name string) []*storepb.TableMetadata {
	out := make([]*storepb.TableMetadata, 0, len(tables))
	for _, t := range tables {
		if t.Name != name {
			out = append(out, t)
		}
	}
	return out
}

func removeViewByName(views []*storepb.ViewMetadata, name string) []*storepb.ViewMetadata {
	out := make([]*storepb.ViewMetadata, 0, len(views))
	for _, v := range views {
		if v.Name != name {
			out = append(out, v)
		}
	}
	return out
}

func removeMatViewByName(mvs []*storepb.MaterializedViewMetadata, name string) []*storepb.MaterializedViewMetadata {
	out := make([]*storepb.MaterializedViewMetadata, 0, len(mvs))
	for _, m := range mvs {
		if m.Name != name {
			out = append(out, m)
		}
	}
	return out
}

func removeSequenceByName(seqs []*storepb.SequenceMetadata, name string) []*storepb.SequenceMetadata {
	out := make([]*storepb.SequenceMetadata, 0, len(seqs))
	for _, s := range seqs {
		if s.Name != name {
			out = append(out, s)
		}
	}
	return out
}

func removeColumnByName(cols []*storepb.ColumnMetadata, name string) []*storepb.ColumnMetadata {
	out := make([]*storepb.ColumnMetadata, 0, len(cols))
	for _, c := range cols {
		if c.Name != name {
			out = append(out, c)
		}
	}
	return out
}

func removeIndexByName(indexes []*storepb.IndexMetadata, name string) []*storepb.IndexMetadata {
	out := make([]*storepb.IndexMetadata, 0, len(indexes))
	for _, i := range indexes {
		if i.Name != name {
			out = append(out, i)
		}
	}
	return out
}

func removeEnumByName(enums []*storepb.EnumTypeMetadata, name string) []*storepb.EnumTypeMetadata {
	out := make([]*storepb.EnumTypeMetadata, 0, len(enums))
	for _, e := range enums {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return out
}

func functionToProto(cat *catalog.Catalog, up *catalog.UserProc, identity string) *storepb.FunctionMetadata {
	return &storepb.FunctionMetadata{
		Name:       up.Name,
		Definition: buildUserProcDDL(cat, up),
		Signature:  identity,
	}
}

func procedureToProto(cat *catalog.Catalog, up *catalog.UserProc, identity string) *storepb.ProcedureMetadata {
	return &storepb.ProcedureMetadata{
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

func removeProcedureByIdentity(procs []*storepb.ProcedureMetadata, identity string) []*storepb.ProcedureMetadata {
	out := make([]*storepb.ProcedureMetadata, 0, len(procs))
	for _, p := range procs {
		if p.Signature != identity {
			out = append(out, p)
		}
	}
	return out
}

func removeFunctionByIdentity(funcs []*storepb.FunctionMetadata, identity string) []*storepb.FunctionMetadata {
	out := make([]*storepb.FunctionMetadata, 0, len(funcs))
	for _, f := range funcs {
		if f.Signature != identity {
			out = append(out, f)
		}
	}
	return out
}
