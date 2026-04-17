//nolint:unused
package pg

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"
	omniparser "github.com/bytebase/omni/pg/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// loadWalkThroughCatalog installs every schema object from DatabaseSchemaMetadata
// into the omni catalog in dependency-topological order with pseudo fallback.
// Indexes and constraints are installed after their parent tables/views.
func loadWalkThroughCatalog(ctx context.Context, cat *catalog.Catalog, meta *storepb.DatabaseSchemaMetadata) error {
	if cat == nil {
		return errors.New("loadWalkThroughCatalog: nil catalog")
	}
	if meta == nil {
		return nil
	}

	objects := wtCollectObjects(meta)
	sorted := wtTopoSort(objects)
	for _, obj := range sorted {
		if err := ctx.Err(); err != nil {
			return err
		}
		wtInstallOne(cat, obj)
	}
	return nil
}

// wtObjectKind identifies the kind of a walk-through loader object.
type wtObjectKind int

const (
	kindWTSchema wtObjectKind = iota
	kindWTEnum
	kindWTSequence
	kindWTTable
	kindWTView
	kindWTMatView
	kindWTFunction
	kindWTIndex
	kindWTConstraint
)

// wtObjectEntry is one unit of work for the walk-through loader.
type wtObjectEntry struct {
	kind   wtObjectKind
	schema string
	name   string

	// Exactly one of the following is set based on kind.
	enumMeta    *storepb.EnumTypeMetadata
	seqMeta     *storepb.SequenceMetadata
	tableMeta   *storepb.TableMetadata
	viewMeta    *storepb.ViewMetadata
	matViewMeta *storepb.MaterializedViewMetadata
	funcMeta    *storepb.FunctionMetadata
	idxMeta     *storepb.IndexMetadata
	// For kindWTConstraint: the parent table metadata.
	constraintTable *storepb.TableMetadata
	// For kindWTConstraint, kindWTIndex: the parent relation name.
	parentName string
}

func (e *wtObjectEntry) key() string {
	switch e.kind {
	case kindWTSchema:
		return "schema:" + e.schema
	case kindWTEnum:
		return "type:" + e.schema + "." + e.name
	case kindWTSequence:
		return "seq:" + e.schema + "." + e.name
	case kindWTTable, kindWTView, kindWTMatView:
		return "rel:" + e.schema + "." + e.name
	case kindWTFunction:
		return "func:" + e.schema + "." + e.name + "|" + wtFuncSigKey(e.funcMeta)
	case kindWTIndex:
		return "idx:" + e.schema + "." + e.parentName + "." + e.name
	case kindWTConstraint:
		return "con:" + e.schema + "." + e.parentName + "." + e.name
	}
	return "unknown:" + e.schema + "." + e.name
}

func (e *wtObjectEntry) sortKey() string {
	base := e.schema + "\x00" + e.name + "\x00" + wtKindLabel(e.kind)
	if e.kind == kindWTFunction {
		base += "\x00" + wtFuncSigKey(e.funcMeta)
	}
	if e.kind == kindWTIndex || e.kind == kindWTConstraint {
		base += "\x00" + e.parentName
	}
	return base
}

func wtFuncSigKey(fn *storepb.FunctionMetadata) string {
	if fn == nil {
		return ""
	}
	if fn.Signature != "" {
		return fn.Signature
	}
	return fn.Definition
}

func wtKindLabel(k wtObjectKind) string {
	switch k {
	case kindWTSchema:
		return "0schema"
	case kindWTEnum:
		return "1enum"
	case kindWTSequence:
		return "2seq"
	case kindWTTable:
		return "3table"
	case kindWTView:
		return "4view"
	case kindWTMatView:
		return "5matview"
	case kindWTFunction:
		return "6function"
	case kindWTIndex:
		return "7index"
	case kindWTConstraint:
		return "8constraint"
	}
	return "9unknown"
}

// wtCollectObjects flattens DatabaseSchemaMetadata into wtObjectEntry values.
func wtCollectObjects(meta *storepb.DatabaseSchemaMetadata) []*wtObjectEntry {
	var out []*wtObjectEntry
	for _, sm := range meta.Schemas {
		if sm.Name == "" {
			continue
		}
		out = append(out, &wtObjectEntry{
			kind:   kindWTSchema,
			schema: sm.Name,
			name:   sm.Name,
		})
		for _, enum := range sm.EnumTypes {
			out = append(out, &wtObjectEntry{
				kind:     kindWTEnum,
				schema:   sm.Name,
				name:     enum.Name,
				enumMeta: enum,
			})
		}
		for _, seq := range sm.Sequences {
			if wtIsIdentitySequence(seq, sm.Tables) {
				continue
			}
			out = append(out, &wtObjectEntry{
				kind:    kindWTSequence,
				schema:  sm.Name,
				name:    seq.Name,
				seqMeta: seq,
			})
		}
		for _, tbl := range sm.Tables {
			out = append(out, &wtObjectEntry{
				kind:      kindWTTable,
				schema:    sm.Name,
				name:      tbl.Name,
				tableMeta: tbl,
			})
			for _, idx := range tbl.Indexes {
				out = append(out, &wtObjectEntry{
					kind:       kindWTIndex,
					schema:     sm.Name,
					name:       idx.Name,
					parentName: tbl.Name,
					idxMeta:    idx,
				})
			}
			for _, fk := range tbl.ForeignKeys {
				out = append(out, &wtObjectEntry{
					kind:            kindWTConstraint,
					schema:          sm.Name,
					name:            fk.Name,
					parentName:      tbl.Name,
					constraintTable: tbl,
				})
			}
			for _, chk := range tbl.CheckConstraints {
				out = append(out, &wtObjectEntry{
					kind:            kindWTConstraint,
					schema:          sm.Name,
					name:            chk.Name,
					parentName:      tbl.Name,
					constraintTable: tbl,
				})
			}
			for _, exc := range tbl.ExcludeConstraints {
				out = append(out, &wtObjectEntry{
					kind:            kindWTConstraint,
					schema:          sm.Name,
					name:            exc.Name,
					parentName:      tbl.Name,
					constraintTable: tbl,
				})
			}
		}
		for _, view := range sm.Views {
			out = append(out, &wtObjectEntry{
				kind:     kindWTView,
				schema:   sm.Name,
				name:     view.Name,
				viewMeta: view,
			})
		}
		for _, mv := range sm.MaterializedViews {
			out = append(out, &wtObjectEntry{
				kind:        kindWTMatView,
				schema:      sm.Name,
				name:        mv.Name,
				matViewMeta: mv,
			})
			for _, idx := range mv.Indexes {
				out = append(out, &wtObjectEntry{
					kind:       kindWTIndex,
					schema:     sm.Name,
					name:       idx.Name,
					parentName: mv.Name,
					idxMeta:    idx,
				})
			}
		}
		for _, fn := range sm.Functions {
			out = append(out, &wtObjectEntry{
				kind:     kindWTFunction,
				schema:   sm.Name,
				name:     fn.Name,
				funcMeta: fn,
			})
		}
		for _, proc := range sm.Procedures {
			out = append(out, &wtObjectEntry{
				kind:   kindWTFunction,
				schema: sm.Name,
				name:   proc.Name,
				funcMeta: &storepb.FunctionMetadata{
					Name:       proc.Name,
					Definition: proc.Definition,
					Signature:  proc.Signature,
				},
			})
		}
	}
	return out
}

// wtTopoSort orders objects by dependency using Tarjan SCC.
func wtTopoSort(objects []*wtObjectEntry) []*wtObjectEntry {
	if len(objects) == 0 {
		return nil
	}
	index := make(map[string]*wtObjectEntry, len(objects))
	for _, obj := range objects {
		index[obj.key()] = obj
	}

	edges := wtBuildEdges(objects, index)
	sccs := wtTarjanSCC(objects, edges)

	sccOf := make(map[string]int, len(objects))
	for i, scc := range sccs {
		for _, obj := range scc {
			sccOf[obj.key()] = i
		}
	}

	condensedEdges := make([][]int, len(sccs))
	inDegree := make([]int, len(sccs))
	seenEdge := make(map[[2]int]bool)
	for src, dsts := range edges {
		srcSCC, ok := sccOf[src]
		if !ok {
			continue
		}
		for _, dst := range dsts {
			dstSCC, ok2 := sccOf[dst]
			if !ok2 || dstSCC == srcSCC {
				continue
			}
			edge := [2]int{dstSCC, srcSCC}
			if seenEdge[edge] {
				continue
			}
			seenEdge[edge] = true
			condensedEdges[dstSCC] = append(condensedEdges[dstSCC], srcSCC)
			inDegree[srcSCC]++
		}
	}

	ready := make([]int, 0)
	for i := range sccs {
		if inDegree[i] == 0 {
			ready = append(ready, i)
		}
	}
	wtSortSCCsByMin(ready, sccs)

	var flat []*wtObjectEntry
	for len(ready) > 0 {
		next := ready[0]
		ready = ready[1:]
		flat = append(flat, wtSortedSCCMembers(sccs[next])...)
		for _, nb := range condensedEdges[next] {
			inDegree[nb]--
			if inDegree[nb] == 0 {
				ready = append(ready, nb)
			}
		}
		wtSortSCCsByMin(ready, sccs)
	}

	if len(flat) != len(objects) {
		emitted := make(map[string]bool, len(flat))
		for _, o := range flat {
			emitted[o.key()] = true
		}
		var missed []*wtObjectEntry
		for _, o := range objects {
			if !emitted[o.key()] {
				missed = append(missed, o)
			}
		}
		slices.SortStableFunc(missed, func(a, b *wtObjectEntry) int {
			return cmp.Compare(a.sortKey(), b.sortKey())
		})
		flat = append(flat, missed...)
	}
	return flat
}

// wtBuildEdges computes dependency edges for walk-through objects.
func wtBuildEdges(objects []*wtObjectEntry, index map[string]*wtObjectEntry) map[string][]string {
	edges := make(map[string][]string, len(objects))
	for _, obj := range objects {
		var deps []string

		// Every non-schema object depends on its schema.
		if obj.kind != kindWTSchema {
			if key := "schema:" + obj.schema; index[key] != nil {
				deps = append(deps, key)
			}
		}

		switch obj.kind {
		case kindWTTable:
			for _, col := range obj.tableMeta.Columns {
				for _, ref := range wtExtractUserTypeRefs(col.Type) {
					if k := "type:" + ref.Schema + "." + ref.Name; index[k] != nil {
						deps = append(deps, k)
					}
				}
				// Sequence dependency from nextval('schema.seq'::regclass).
				if col.Default != "" {
					if seqKey := wtExtractSeqRef(col.Default, obj.schema, index); seqKey != "" {
						deps = append(deps, seqKey)
					}
				}
			}
		case kindWTView:
			for _, dep := range obj.viewMeta.DependencyColumns {
				if k := "rel:" + dep.Schema + "." + dep.Table; index[k] != nil {
					deps = append(deps, k)
				}
			}
		case kindWTMatView:
			for _, dep := range obj.matViewMeta.DependencyColumns {
				if k := "rel:" + dep.Schema + "." + dep.Table; index[k] != nil {
					deps = append(deps, k)
				}
			}
		case kindWTFunction:
			argTypes, _ := wtParseFuncArgTypes(obj.funcMeta.Signature)
			for _, at := range argTypes {
				for _, ref := range wtExtractUserTypeRefs(at) {
					if k := "type:" + ref.Schema + "." + ref.Name; index[k] != nil {
						deps = append(deps, k)
					}
				}
			}
		case kindWTIndex:
			if k := "rel:" + obj.schema + "." + obj.parentName; index[k] != nil {
				deps = append(deps, k)
			}
		case kindWTConstraint:
			if k := "rel:" + obj.schema + "." + obj.parentName; index[k] != nil {
				deps = append(deps, k)
			}
			if obj.constraintTable != nil {
				for _, fk := range obj.constraintTable.ForeignKeys {
					if fk.Name == obj.name {
						refSchema := fk.ReferencedSchema
						if refSchema == "" {
							refSchema = obj.schema
						}
						if k := "rel:" + refSchema + "." + fk.ReferencedTable; index[k] != nil {
							deps = append(deps, k)
						}
						// FK depends on referenced table's PK/Unique indexes.
						for _, o := range objects {
							if o.kind == kindWTIndex && o.schema == refSchema &&
								o.parentName == fk.ReferencedTable &&
								(o.idxMeta.Primary || (o.idxMeta.IsConstraint && o.idxMeta.Unique)) {
								if index[o.key()] != nil {
									deps = append(deps, o.key())
								}
							}
						}
						break
					}
				}
			}
		default:
			// kindWTSchema, kindWTEnum, kindWTSequence have no additional deps.
		}

		if len(deps) > 0 {
			edges[obj.key()] = wtDedupStrings(deps)
		}
	}
	return edges
}

// wtTarjanSCC runs Tarjan's SCC algorithm over walk-through objects.
func wtTarjanSCC(objects []*wtObjectEntry, edges map[string][]string) [][]*wtObjectEntry {
	type state struct {
		index, low int
		onStack    bool
	}
	st := make(map[string]*state, len(objects))
	byKey := make(map[string]*wtObjectEntry, len(objects))
	for _, obj := range objects {
		byKey[obj.key()] = obj
	}

	keys := make([]string, 0, len(objects))
	for _, obj := range objects {
		keys = append(keys, obj.key())
	}
	slices.Sort(keys)

	var (
		idx    int
		stack  []string
		result [][]*wtObjectEntry
	)

	var strongconnect func(v string)
	strongconnect = func(v string) {
		st[v] = &state{index: idx, low: idx, onStack: true}
		idx++
		stack = append(stack, v)
		for _, w := range edges[v] {
			if _, ok := byKey[w]; !ok {
				continue
			}
			if _, seen := st[w]; !seen {
				strongconnect(w)
				st[v].low = min(st[v].low, st[w].low)
			} else if st[w].onStack {
				st[v].low = min(st[v].low, st[w].index)
			}
		}
		if st[v].low == st[v].index {
			var scc []*wtObjectEntry
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				st[w].onStack = false
				scc = append(scc, byKey[w])
				if w == v {
					break
				}
			}
			result = append(result, scc)
		}
	}

	for _, k := range keys {
		if _, seen := st[k]; !seen {
			strongconnect(k)
		}
	}
	return result
}

func wtSortedSCCMembers(scc []*wtObjectEntry) []*wtObjectEntry {
	out := make([]*wtObjectEntry, len(scc))
	copy(out, scc)
	slices.SortStableFunc(out, func(a, b *wtObjectEntry) int {
		return cmp.Compare(a.sortKey(), b.sortKey())
	})
	return out
}

func wtSortSCCsByMin(indices []int, sccs [][]*wtObjectEntry) {
	slices.SortStableFunc(indices, func(a, b int) int {
		return cmp.Compare(wtMinSortKey(sccs[a]), wtMinSortKey(sccs[b]))
	})
}

func wtMinSortKey(scc []*wtObjectEntry) string {
	if len(scc) == 0 {
		return ""
	}
	best := scc[0].sortKey()
	for _, o := range scc[1:] {
		if k := o.sortKey(); k < best {
			best = k
		}
	}
	return best
}

func wtDedupStrings(in []string) []string {
	if len(in) < 2 {
		return in
	}
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// wtInstallOne attempts real install, falls back to pseudo for tables/views/matviews.
func wtInstallOne(cat *catalog.Catalog, obj *wtObjectEntry) {
	if err := wtInstallReal(cat, obj); err == nil {
		return
	}
	// Only pseudo-fallback for relations; other kinds are left uninstalled on failure.
	switch obj.kind {
	case kindWTTable:
		cols := wtColNames(obj.tableMeta)
		_ = cat.DefineRelation(wtPseudoTableStmt(obj.schema, obj.name, cols), 'r')
	case kindWTView:
		cols := wtViewColNames(obj.viewMeta)
		if stmt, err := wtPseudoViewStmt(obj.schema, obj.name, cols); err == nil {
			_ = cat.DefineView(stmt)
		}
	case kindWTMatView:
		cols := wtMatViewColNames(obj.matViewMeta)
		if stmt, err := wtPseudoMatViewStmt(obj.schema, obj.name, cols); err == nil {
			_ = cat.ExecCreateTableAs(stmt)
		}
	default:
		// No pseudo form for schemas, enums, sequences, functions, indexes, constraints.
	}
}

// wtInstallReal performs the real catalog install for one object.
func wtInstallReal(cat *catalog.Catalog, obj *wtObjectEntry) error {
	switch obj.kind {
	case kindWTSchema:
		err := cat.CreateSchemaCommand(&ast.CreateSchemaStmt{Schemaname: obj.schema})
		var cErr *catalog.Error
		if errors.As(err, &cErr) && cErr.Code == catalog.CodeDuplicateSchema {
			return nil
		}
		return err

	case kindWTEnum:
		vals := make([]ast.Node, 0, len(obj.enumMeta.Values))
		for _, v := range obj.enumMeta.Values {
			vals = append(vals, &ast.String{Str: v})
		}
		return cat.DefineEnum(&ast.CreateEnumStmt{
			TypeName: wtQualifiedList(obj.schema, obj.name),
			Vals:     &ast.List{Items: vals},
		})

	case kindWTSequence:
		return wtInstallSequence(cat, obj)

	case kindWTTable:
		return wtInstallTable(cat, obj)

	case kindWTView:
		return wtInstallView(cat, obj)

	case kindWTMatView:
		return wtInstallMatView(cat, obj)

	case kindWTFunction:
		return wtInstallFunction(cat, obj)

	case kindWTIndex:
		if obj.idxMeta.Primary {
			return cat.AddConstraint(obj.schema, obj.parentName, catalog.ConstraintDef{
				Name:    obj.idxMeta.Name,
				Type:    catalog.ConstraintPK,
				Columns: wtUnquoteColumns(obj.idxMeta.Expressions),
			})
		}
		if obj.idxMeta.IsConstraint && obj.idxMeta.Unique {
			return cat.AddConstraint(obj.schema, obj.parentName, catalog.ConstraintDef{
				Name:    obj.idxMeta.Name,
				Type:    catalog.ConstraintUnique,
				Columns: wtUnquoteColumns(obj.idxMeta.Expressions),
			})
		}
		return wtInstallIndex(cat, obj)

	case kindWTConstraint:
		return wtInstallConstraint(cat, obj)
	}
	return errors.Errorf("unknown object kind %d", obj.kind)
}

func wtInstallSequence(cat *catalog.Catalog, obj *wtObjectEntry) error {
	seq := obj.seqMeta
	if seq == nil {
		return errors.New("sequence has no metadata")
	}

	stmt := &ast.CreateSeqStmt{
		Sequence: &ast.RangeVar{
			Schemaname:     obj.schema,
			Relname:        seq.Name,
			Relpersistence: 'p',
		},
	}

	var opts []ast.Node
	if seq.Increment != "" && seq.Increment != "0" {
		opts = append(opts, &ast.DefElem{Defname: "increment", Arg: &ast.Integer{Ival: wtParseInt64(seq.Increment)}})
	}
	if seq.MinValue != "" && seq.MinValue != "0" {
		opts = append(opts, &ast.DefElem{Defname: "minvalue", Arg: &ast.Integer{Ival: wtParseInt64(seq.MinValue)}})
	}
	if seq.MaxValue != "" && seq.MaxValue != "0" {
		opts = append(opts, &ast.DefElem{Defname: "maxvalue", Arg: &ast.Integer{Ival: wtParseInt64(seq.MaxValue)}})
	}
	if seq.Start != "" && seq.Start != "0" {
		opts = append(opts, &ast.DefElem{Defname: "start", Arg: &ast.Integer{Ival: wtParseInt64(seq.Start)}})
	}
	if seq.CacheSize != "" && seq.CacheSize != "0" {
		opts = append(opts, &ast.DefElem{Defname: "cache", Arg: &ast.Integer{Ival: wtParseInt64(seq.CacheSize)}})
	}
	if seq.Cycle {
		opts = append(opts, &ast.DefElem{Defname: "cycle", Arg: &ast.Boolean{Boolval: true}})
	}
	if len(opts) > 0 {
		stmt.Options = &ast.List{Items: opts}
	}

	return cat.DefineSequence(stmt)
}

// wtIsIdentitySequence checks if a sequence is owned by an identity column.
// Identity columns auto-create their sequence during DefineRelation, so
// pre-creating it would cause a duplicate error.
func wtIsIdentitySequence(seq *storepb.SequenceMetadata, tables []*storepb.TableMetadata) bool {
	if seq.OwnerTable == "" || seq.OwnerColumn == "" {
		return false
	}
	for _, tbl := range tables {
		if tbl.Name != seq.OwnerTable {
			continue
		}
		for _, col := range tbl.Columns {
			if col.Name == seq.OwnerColumn && col.IsIdentity {
				return true
			}
		}
	}
	return false
}

func wtParseInt64(s string) int64 {
	var v int64
	_, _ = fmt.Sscanf(s, "%d", &v)
	return v
}

func wtInstallTable(cat *catalog.Catalog, obj *wtObjectEntry) error {
	items := make([]ast.Node, 0, len(obj.tableMeta.Columns))
	for _, col := range obj.tableMeta.Columns {
		if col.Name == "" {
			continue
		}
		if col.Type == "" {
			return errors.Errorf("column %q: empty type", col.Name)
		}
		tn, err := wtTypeNameFromString(col.Type)
		if err != nil {
			return errors.Wrapf(err, "column %q", col.Name)
		}
		colDef := &ast.ColumnDef{
			Colname:   col.Name,
			TypeName:  tn,
			IsNotNull: !col.Nullable,
		}
		if col.Default != "" && col.Generation == nil {
			if rawDefault, err := wtParseDefaultExpr(col.Default); err == nil {
				colDef.RawDefault = rawDefault
			}
		}
		if col.Generation != nil && col.Generation.Type == storepb.GenerationMetadata_TYPE_STORED && col.Generation.Expression != "" {
			colDef.Generated = 's'
			if genExpr := wtParseExpr(col.Generation.Expression); genExpr != nil {
				colDef.Constraints = &ast.List{Items: []ast.Node{
					&ast.Constraint{Contype: ast.CONSTR_GENERATED, RawExpr: genExpr},
				}}
			}
		}
		if col.IsIdentity {
			switch col.IdentityGeneration {
			case storepb.ColumnMetadata_ALWAYS:
				colDef.Identity = 'a'
			case storepb.ColumnMetadata_BY_DEFAULT:
				colDef.Identity = 'd'
			default:
				colDef.Identity = 'd'
			}
		}
		items = append(items, colDef)
	}
	return cat.DefineRelation(&ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     obj.schema,
			Relname:        obj.tableMeta.Name,
			Relpersistence: 'p',
		},
		TableElts: &ast.List{Items: items},
	}, 'r')
}

func wtInstallView(cat *catalog.Catalog, obj *wtObjectEntry) error {
	if obj.viewMeta.Definition == "" {
		return errors.New("empty view definition")
	}
	sel, err := wtParseSelectBody(obj.viewMeta.Definition)
	if err != nil {
		return errors.Wrapf(err, "view %q", obj.name)
	}
	return cat.DefineView(&ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     obj.schema,
			Relname:        obj.name,
			Relpersistence: 'p',
		},
		Query: sel,
	})
}

func wtInstallMatView(cat *catalog.Catalog, obj *wtObjectEntry) error {
	if obj.matViewMeta.Definition == "" {
		return errors.New("empty matview definition")
	}
	sel, err := wtParseSelectBody(obj.matViewMeta.Definition)
	if err != nil {
		return errors.Wrapf(err, "matview %q", obj.name)
	}
	return cat.ExecCreateTableAs(&ast.CreateTableAsStmt{
		Query:   sel,
		Objtype: ast.OBJECT_MATVIEW,
		Into: &ast.IntoClause{
			Rel: &ast.RangeVar{
				Schemaname:     obj.schema,
				Relname:        obj.name,
				Relpersistence: 'p',
			},
		},
	})
}

func wtInstallFunction(cat *catalog.Catalog, obj *wtObjectEntry) error {
	fn := obj.funcMeta
	if fn.Definition != "" {
		nodes, err := omniparser.Parse(fn.Definition)
		if err == nil && nodes != nil && len(nodes.Items) == 1 {
			node := nodes.Items[0]
			if raw, ok := node.(*ast.RawStmt); ok {
				node = raw.Stmt
			}
			if parsed, ok := node.(*ast.CreateFunctionStmt); ok {
				return cat.CreateFunctionStmt(parsed)
			}
		}
	}
	// Fallback: build from signature.
	argTypes, err := wtParseFuncArgTypes(fn.Signature)
	if err != nil {
		return errors.Wrapf(err, "function %q signature %q", fn.Name, fn.Signature)
	}
	params := make([]ast.Node, 0, len(argTypes))
	for i, at := range argTypes {
		tn, err := wtTypeNameFromString(at)
		if err != nil {
			return errors.Wrapf(err, "function %q arg %d", fn.Name, i)
		}
		params = append(params, &ast.FunctionParameter{
			ArgType: tn,
			Mode:    ast.FUNC_PARAM_IN,
		})
	}
	stmt := &ast.CreateFunctionStmt{
		Funcname:   wtQualifiedList(obj.schema, fn.Name),
		ReturnType: wtPseudoTextTypeName(),
	}
	if len(params) > 0 {
		stmt.Parameters = &ast.List{Items: params}
	}
	stmt.Options = &ast.List{Items: []ast.Node{
		&ast.DefElem{Defname: "language", Arg: &ast.String{Str: "sql"}},
		&ast.DefElem{
			Defname: "as",
			Arg:     &ast.List{Items: []ast.Node{&ast.String{Str: "SELECT NULL::text"}}},
		},
	}}
	return cat.CreateFunctionStmt(stmt)
}

func wtInstallIndex(cat *catalog.Catalog, obj *wtObjectEntry) error {
	idx := obj.idxMeta
	if idx.Name == "" {
		return errors.New("index has no name")
	}
	params := make([]ast.Node, 0, len(idx.Expressions))
	for i, expr := range idx.Expressions {
		if expr == "" {
			continue
		}
		elem := &ast.IndexElem{}
		if i < len(idx.Descending) && idx.Descending[i] {
			elem.Ordering = ast.SORTBY_DESC
		}
		bare := wtUnquoteIdent(expr)
		if wtIsExpressionIndex(bare) {
			if parsed := wtParseExpr(expr); parsed != nil {
				elem.Expr = parsed
			} else {
				elem.Name = bare
			}
		} else {
			elem.Name = bare
		}
		params = append(params, elem)
	}
	return cat.DefineIndex(&ast.IndexStmt{
		Idxname:      idx.Name,
		Relation:     &ast.RangeVar{Schemaname: obj.schema, Relname: obj.parentName},
		AccessMethod: idx.Type,
		IndexParams:  &ast.List{Items: params},
		Unique:       idx.Unique,
		Primary:      idx.Primary,
	})
}

func wtInstallConstraint(cat *catalog.Catalog, obj *wtObjectEntry) error {
	tbl := obj.constraintTable
	if tbl == nil {
		return errors.New("constraint has no parent table")
	}

	// Find which constraint this entry represents.
	for _, fk := range tbl.ForeignKeys {
		if fk.Name != obj.name {
			continue
		}
		refSchema := fk.ReferencedSchema
		if refSchema == "" {
			refSchema = obj.schema
		}
		def := catalog.ConstraintDef{
			Name:        fk.Name,
			Type:        catalog.ConstraintFK,
			Columns:     fk.Columns,
			RefSchema:   refSchema,
			RefTable:    fk.ReferencedTable,
			RefColumns:  fk.ReferencedColumns,
			FKUpdAction: wtFKActionByte(fk.OnUpdate),
			FKDelAction: wtFKActionByte(fk.OnDelete),
			FKMatchType: wtFKMatchByte(fk.MatchType),
		}
		return cat.AddConstraint(obj.schema, obj.parentName, def)
	}
	for _, chk := range tbl.CheckConstraints {
		if chk.Name != obj.name {
			continue
		}
		return cat.AddConstraint(obj.schema, obj.parentName, catalog.ConstraintDef{
			Name:      chk.Name,
			Type:      catalog.ConstraintCheck,
			CheckExpr: chk.Expression,
		})
	}
	for _, exc := range tbl.ExcludeConstraints {
		if exc.Name != obj.name {
			continue
		}
		return cat.AddConstraint(obj.schema, obj.parentName, catalog.ConstraintDef{
			Name:      exc.Name,
			Type:      catalog.ConstraintExclude,
			CheckExpr: exc.Expression,
		})
	}
	return errors.Errorf("constraint %q not found in table %q", obj.name, obj.parentName)
}

// ---- pseudo forms ----

func wtPseudoTableStmt(schema, name string, cols []string) *ast.CreateStmt {
	items := make([]ast.Node, 0, len(cols))
	for _, col := range cols {
		if col == "" {
			continue
		}
		items = append(items, &ast.ColumnDef{
			Colname:  col,
			TypeName: wtPseudoTextTypeName(),
		})
	}
	return &ast.CreateStmt{
		Relation: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        name,
			Relpersistence: 'p',
		},
		TableElts: &ast.List{Items: items},
	}
}

func wtPseudoViewStmt(schema, name string, cols []string) (*ast.ViewStmt, error) {
	sel, err := wtPseudoConstantSelect(cols)
	if err != nil {
		return nil, err
	}
	return &ast.ViewStmt{
		View: &ast.RangeVar{
			Schemaname:     schema,
			Relname:        name,
			Relpersistence: 'p',
		},
		Query: sel,
	}, nil
}

func wtPseudoMatViewStmt(schema, name string, cols []string) (*ast.CreateTableAsStmt, error) {
	sel, err := wtPseudoConstantSelect(cols)
	if err != nil {
		return nil, err
	}
	return &ast.CreateTableAsStmt{
		Query:   sel,
		Objtype: ast.OBJECT_MATVIEW,
		Into: &ast.IntoClause{
			Rel: &ast.RangeVar{
				Schemaname:     schema,
				Relname:        name,
				Relpersistence: 'p',
			},
		},
	}, nil
}

func wtPseudoConstantSelect(cols []string) (*ast.SelectStmt, error) {
	var targets []string
	for _, col := range cols {
		if col == "" {
			continue
		}
		targets = append(targets, "NULL::text AS "+wtQuoteIdent(col))
	}
	if len(targets) == 0 {
		targets = []string{"NULL::text"}
	}
	sql := "SELECT " + strings.Join(targets, ", ")
	nodes, err := omniparser.Parse(sql)
	if err != nil {
		return nil, errors.Wrap(err, "parse pseudo constant select")
	}
	if nodes == nil || len(nodes.Items) != 1 {
		return nil, errors.New("pseudo constant select: expected 1 statement")
	}
	node := nodes.Items[0]
	if raw, ok := node.(*ast.RawStmt); ok {
		node = raw.Stmt
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("pseudo constant select: expected SelectStmt, got %T", node)
	}
	return sel, nil
}

// ---- column name helpers ----

func wtColNames(tbl *storepb.TableMetadata) []string {
	if tbl == nil {
		return nil
	}
	seen := make(map[string]bool, len(tbl.Columns))
	out := make([]string, 0, len(tbl.Columns))
	for _, col := range tbl.Columns {
		if col.Name == "" || seen[col.Name] {
			continue
		}
		seen[col.Name] = true
		out = append(out, col.Name)
	}
	return out
}

func wtViewColNames(v *storepb.ViewMetadata) []string {
	if v == nil {
		return nil
	}
	if len(v.Columns) > 0 {
		seen := make(map[string]bool, len(v.Columns))
		out := make([]string, 0, len(v.Columns))
		for _, col := range v.Columns {
			if col.Name == "" || seen[col.Name] {
				continue
			}
			seen[col.Name] = true
			out = append(out, col.Name)
		}
		return out
	}
	seen := make(map[string]bool)
	var out []string
	for _, dc := range v.DependencyColumns {
		if dc.Column == "" || seen[dc.Column] {
			continue
		}
		seen[dc.Column] = true
		out = append(out, dc.Column)
	}
	return out
}

func wtMatViewColNames(m *storepb.MaterializedViewMetadata) []string {
	if m == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	for _, dc := range m.DependencyColumns {
		if dc.Column == "" || seen[dc.Column] {
			continue
		}
		seen[dc.Column] = true
		out = append(out, dc.Column)
	}
	return out
}

// ---- type parsing helpers ----

// wtTypeNameFromString parses a PG type string into *ast.TypeName via
// "SELECT NULL::<type>" to reuse omni's SELECT parse path.
func wtTypeNameFromString(typeStr string) (*ast.TypeName, error) {
	nodes, err := omniparser.Parse("SELECT NULL::" + typeStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse type %q", typeStr)
	}
	if nodes == nil || len(nodes.Items) != 1 {
		return nil, errors.Errorf("type %q: expected 1 statement", typeStr)
	}
	node := nodes.Items[0]
	if raw, ok := node.(*ast.RawStmt); ok {
		node = raw.Stmt
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("type %q: expected SelectStmt, got %T", typeStr, node)
	}
	if sel.TargetList == nil || len(sel.TargetList.Items) != 1 {
		return nil, errors.Errorf("type %q: expected 1 target", typeStr)
	}
	rt, ok := sel.TargetList.Items[0].(*ast.ResTarget)
	if !ok {
		return nil, errors.Errorf("type %q: expected ResTarget, got %T", typeStr, sel.TargetList.Items[0])
	}
	cast, ok := rt.Val.(*ast.TypeCast)
	if !ok {
		return nil, errors.Errorf("type %q: expected TypeCast, got %T", typeStr, rt.Val)
	}
	return cast.TypeName, nil
}

// wtParseDefaultExpr parses a default expression string and returns the AST node.
func wtParseDefaultExpr(expr string) (ast.Node, error) {
	nodes, err := omniparser.Parse("SELECT " + expr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse default %q", expr)
	}
	if nodes == nil || len(nodes.Items) != 1 {
		return nil, errors.Errorf("default %q: expected 1 statement", expr)
	}
	node := nodes.Items[0]
	if raw, ok := node.(*ast.RawStmt); ok {
		node = raw.Stmt
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("default %q: expected SelectStmt", expr)
	}
	if sel.TargetList == nil || len(sel.TargetList.Items) != 1 {
		return nil, errors.Errorf("default %q: expected 1 target", expr)
	}
	rt, ok := sel.TargetList.Items[0].(*ast.ResTarget)
	if !ok {
		return nil, errors.Errorf("default %q: expected ResTarget", expr)
	}
	return rt.Val, nil
}

// wtParseSelectBody parses a SQL string into *ast.SelectStmt.
func wtParseSelectBody(sql string) (*ast.SelectStmt, error) {
	nodes, err := omniparser.Parse(sql)
	if err != nil {
		return nil, errors.Wrap(err, "parse")
	}
	if nodes == nil || len(nodes.Items) != 1 {
		return nil, errors.Errorf("expected 1 statement, got %d", len(nodes.Items))
	}
	node := nodes.Items[0]
	if raw, ok := node.(*ast.RawStmt); ok {
		node = raw.Stmt
	}
	sel, ok := node.(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SelectStmt, got %T", node)
	}
	return sel, nil
}

// ---- type ref extraction (local, no import from parser/pg) ----

type wtUserTypeRef struct {
	Schema string
	Name   string
}

func wtExtractUserTypeRefs(typeStr string) []wtUserTypeRef {
	if typeStr == "" {
		return nil
	}
	base := wtStripTypeModifiers(typeStr)
	if base == "" || wtIsBuiltinType(base) {
		return nil
	}
	if strings.HasPrefix(base, "_") {
		return nil
	}
	schema, name, ok := wtSplitQualifiedName(base)
	if !ok {
		return nil
	}
	if wtIsSystemSchema(schema) {
		return nil
	}
	return []wtUserTypeRef{{Schema: schema, Name: name}}
}

func wtStripTypeModifiers(s string) string {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	for _, sfx := range []string{" with time zone", " without time zone"} {
		if strings.HasSuffix(lower, sfx) {
			s = s[:len(s)-len(sfx)]
			lower = lower[:len(lower)-len(sfx)]
		}
	}
	if i := strings.Index(s, "("); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func wtSplitQualifiedName(s string) (schema, name string, ok bool) {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == '.' && !inQuote:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if inQuote {
		return "", "", false
	}
	parts = append(parts, cur.String())
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func wtIsSystemSchema(s string) bool {
	switch s {
	case "pg_catalog", "pg_toast", "information_schema":
		return true
	}
	return strings.HasPrefix(s, "pg_")
}

func wtIsBuiltinType(s string) bool {
	_, ok := wtBuiltinTypes[strings.ToLower(s)]
	return ok
}

var wtBuiltinTypes = map[string]struct{}{
	"smallint": {}, "integer": {}, "int": {}, "int2": {}, "int4": {}, "int8": {},
	"bigint": {}, "decimal": {}, "numeric": {}, "real": {}, "double precision": {},
	"float4": {}, "float8": {}, "smallserial": {}, "serial": {}, "bigserial": {},
	"money": {}, "character": {}, "character varying": {}, "char": {}, "varchar": {},
	"text": {}, "bytea": {}, "bpchar": {}, "name": {}, "bit": {}, "bit varying": {},
	"varbit": {}, "date": {}, "time": {}, "timetz": {}, "timestamp": {}, "timestamptz": {},
	"interval": {}, "boolean": {}, "bool": {}, "json": {}, "jsonb": {}, "xml": {},
	"uuid": {}, "point": {}, "line": {}, "lseg": {}, "box": {}, "path": {},
	"polygon": {}, "circle": {}, "cidr": {}, "inet": {}, "macaddr": {}, "macaddr8": {},
	"tsvector": {}, "tsquery": {}, "void": {}, "record": {}, "anyelement": {},
	"anyarray": {}, "anynonarray": {}, "anyenum": {}, "anyrange": {}, "any": {},
	"trigger": {}, "event_trigger": {}, "cstring": {}, "internal": {},
	"language_handler": {}, "fdw_handler": {}, "index_am_handler": {}, "tsm_handler": {},
	"pg_lsn": {}, "oid": {}, "regclass": {}, "regproc": {}, "regprocedure": {},
	"regoper": {}, "regoperator": {}, "regtype": {}, "regconfig": {}, "regdictionary": {},
	"regnamespace": {}, "regrole": {},
}

// wtExtractSeqRef extracts the seq key from a nextval('schema.seq'::regclass) default.
// Returns the "seq:schema.name" key if found, empty string otherwise.
var reNextval = regexp.MustCompile(`nextval\('([^']+)'`)

func wtExtractSeqRef(defaultExpr, tableSchema string, index map[string]*wtObjectEntry) string {
	m := reNextval.FindStringSubmatch(defaultExpr)
	if m == nil {
		return ""
	}
	// m[1] is the content: "schema.seq_name" or "seq_name"
	raw := strings.TrimSuffix(m[1], "::regclass")
	raw = strings.Trim(raw, "\"")
	var seqSchema, seqName string
	if dot := strings.Index(raw, "."); dot >= 0 {
		seqSchema = strings.Trim(raw[:dot], "\"")
		seqName = strings.Trim(raw[dot+1:], "\"")
	} else {
		seqSchema = tableSchema
		seqName = raw
	}
	k := "seq:" + seqSchema + "." + seqName
	if index[k] != nil {
		return k
	}
	return ""
}

// ---- function signature helpers ----

func wtParseFuncArgTypes(signature string) ([]string, error) {
	open := strings.Index(signature, "(")
	if open < 0 {
		return nil, nil
	}
	closeIdx := strings.LastIndex(signature, ")")
	if closeIdx < 0 || closeIdx <= open {
		return nil, errors.Errorf("signature %q: unbalanced parens", signature)
	}
	return wtSplitTopLevelCommas(signature[open+1 : closeIdx]), nil
}

func wtSplitTopLevelCommas(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		default:
			// other bytes are part of the current token
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

// ---- FK action helpers ----

func wtFKActionByte(action string) byte {
	switch strings.ToUpper(action) {
	case "RESTRICT":
		return 'r'
	case "CASCADE":
		return 'c'
	case "SET NULL":
		return 'n'
	case "SET DEFAULT":
		return 'd'
	default:
		return 'a' // NO ACTION
	}
}

func wtFKMatchByte(match string) byte {
	switch strings.ToUpper(match) {
	case "FULL":
		return 'f'
	case "PARTIAL":
		return 'p'
	default:
		return 's' // SIMPLE
	}
}

// ---- small AST helpers ----

func wtQualifiedList(schema, name string) *ast.List {
	items := make([]ast.Node, 0, 2)
	if schema != "" {
		items = append(items, &ast.String{Str: schema})
	}
	items = append(items, &ast.String{Str: name})
	return &ast.List{Items: items}
}

func wtPseudoTextTypeName() *ast.TypeName {
	return &ast.TypeName{
		Names:   &ast.List{Items: []ast.Node{&ast.String{Str: "text"}}},
		Typemod: -1,
	}
}

func wtQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func wtIsExpressionIndex(expr string) bool {
	return strings.ContainsAny(expr, "( ") || strings.Contains(expr, "::")
}

func wtUnquoteIdent(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
	}
	return s
}

func wtUnquoteColumns(cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = wtUnquoteIdent(c)
	}
	return out
}

func wtParseExpr(expr string) ast.Node {
	stmts, err := omniparser.Parse("SELECT (" + expr + ")")
	if err != nil || stmts == nil || len(stmts.Items) == 0 {
		return nil
	}
	raw, ok := stmts.Items[0].(*ast.RawStmt)
	if !ok {
		return nil
	}
	sel, ok := raw.Stmt.(*ast.SelectStmt)
	if !ok || sel.TargetList == nil || len(sel.TargetList.Items) == 0 {
		return nil
	}
	rt, ok := sel.TargetList.Items[0].(*ast.ResTarget)
	if !ok {
		return nil
	}
	return rt.Val
}
