package pg

import (
	"cmp"
	"context"
	"slices"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// e3Loader installs every schema object from a storepb.DatabaseSchemaMetadata
// into an omni catalog, in dependency-topological order, with an inline
// pseudo fallback at each failed slot.
//
// Invariants after Load() returns without error:
//  1. Every object from the metadata is present in the catalog — as either a
//     real install or a pseudo install. `loaderObjects` records its identity.
//  2. `degraded[key]` contains the real-install error for objects whose real
//     install failed and fell back to pseudo.
//  3. `trulyBroken[key]` contains the pseudo-install error for objects whose
//     even-pseudo install failed. These are the few objects callers can
//     expect analyzer errors on.
//
// The loader holds no reference to the catalog after Load() completes; the
// catalog object belongs to the caller.
type e3Loader struct {
	cat  *catalog.Catalog
	meta *storepb.DatabaseSchemaMetadata

	// loaderObjects maps canonical object keys (schema.name) to true for
	// every object the loader collected from metadata. Used by the classifier
	// to tell "loader knew about X and something went wrong" from "X was
	// never in metadata."
	loaderObjects map[string]bool

	// degraded records objects whose real install failed. The pseudo version
	// is in the catalog for these entries.
	degraded map[string]error

	// trulyBroken records objects whose pseudo install also failed. The
	// catalog has nothing at this slot.
	trulyBroken map[string]error
}

// objectKind identifies the kind of an E3 loader object. It drives the
// install switch in installReal/installPseudo and the dependency-edge
// computation in buildDependencyGraph.
type objectKind int

const (
	kindSchema objectKind = iota
	kindEnum
	kindTable
	kindView
	kindMatView
	kindFunction
)

// objectEntry is the flattened, order-free unit of work the loader processes.
// One entry per schema object. Entries for the same (schema, name) across
// different kinds are allowed (e.g. a table and a function with the same
// name) and treated as independent nodes in the dep graph.
type objectEntry struct {
	kind   objectKind
	schema string
	name   string

	// Exactly one of the following is populated based on kind.
	enumMeta    *storepb.EnumTypeMetadata
	tableMeta   *storepb.TableMetadata
	viewMeta    *storepb.ViewMetadata
	matViewMeta *storepb.MaterializedViewMetadata
	funcMeta    *storepb.FunctionMetadata
}

// key returns a canonical kind-prefixed identifier used for dependency
// lookups and set membership. Tables, views, and matviews share a namespace
// (relation namespace in PG), so they use the same prefix "rel:".
func (e *objectEntry) key() string {
	switch e.kind {
	case kindSchema:
		return "schema:" + e.schema
	case kindEnum:
		return "type:" + e.schema + "." + e.name
	case kindTable, kindView, kindMatView:
		return "rel:" + e.schema + "." + e.name
	case kindFunction:
		return "func:" + e.schema + "." + e.name
	}
	return "unknown:" + e.schema + "." + e.name
}

// sortKey returns the intra-SCC lexicographic key. Matches hard contract C10.
func (e *objectEntry) sortKey() string {
	return e.schema + "\x00" + e.name + "\x00" + kindLabel(e.kind)
}

func kindLabel(k objectKind) string {
	switch k {
	case kindSchema:
		return "0schema"
	case kindEnum:
		return "1enum"
	case kindTable:
		return "2table"
	case kindView:
		return "3view"
	case kindMatView:
		return "4matview"
	case kindFunction:
		return "5function"
	}
	return "9unknown"
}

// newE3Loader returns a loader primed for Load(). It does not touch the
// catalog until Load() is called.
func newE3Loader(cat *catalog.Catalog, meta *storepb.DatabaseSchemaMetadata) *e3Loader {
	return &e3Loader{
		cat:           cat,
		meta:          meta,
		loaderObjects: make(map[string]bool),
		degraded:      make(map[string]error),
		trulyBroken:   make(map[string]error),
	}
}

// Load collects every object from metadata, topologically orders them, and
// installs each either real or (on failure) pseudo. The returned error is
// non-nil only for catastrophic conditions (nil catalog, ctx cancellation);
// per-object failures are recorded in degraded/trulyBroken and do not stop
// the loader.
func (l *e3Loader) Load(ctx context.Context) error {
	if l.cat == nil {
		return errors.New("e3Loader: nil catalog")
	}
	if l.meta == nil {
		return nil
	}

	objects := l.collectObjects()
	for _, obj := range objects {
		l.loaderObjects[obj.key()] = true
	}

	sorted := topoSortObjects(objects)
	for _, obj := range sorted {
		if err := ctx.Err(); err != nil {
			return err
		}
		l.installOne(obj)
	}
	return nil
}

// collectObjects flattens DatabaseSchemaMetadata into a list of objectEntry
// values, one per schema object the loader will install. Schemas themselves
// become kindSchema entries so CREATE SCHEMA is part of the topo order.
func (l *e3Loader) collectObjects() []*objectEntry {
	var out []*objectEntry
	for _, sm := range l.meta.Schemas {
		if sm.Name == "" {
			// PG sync sometimes emits a no-schema placeholder; skip.
			continue
		}
		out = append(out, &objectEntry{
			kind:   kindSchema,
			schema: sm.Name,
			name:   sm.Name,
		})
		for _, enum := range sm.EnumTypes {
			out = append(out, &objectEntry{
				kind:     kindEnum,
				schema:   sm.Name,
				name:     enum.Name,
				enumMeta: enum,
			})
		}
		for _, tbl := range sm.Tables {
			out = append(out, &objectEntry{
				kind:      kindTable,
				schema:    sm.Name,
				name:      tbl.Name,
				tableMeta: tbl,
			})
		}
		for _, view := range sm.Views {
			out = append(out, &objectEntry{
				kind:     kindView,
				schema:   sm.Name,
				name:     view.Name,
				viewMeta: view,
			})
		}
		for _, mv := range sm.MaterializedViews {
			out = append(out, &objectEntry{
				kind:        kindMatView,
				schema:      sm.Name,
				name:        mv.Name,
				matViewMeta: mv,
			})
		}
		for _, fn := range sm.Functions {
			out = append(out, &objectEntry{
				kind:     kindFunction,
				schema:   sm.Name,
				name:     fn.Name,
				funcMeta: fn,
			})
		}
	}
	return out
}

// topoSortObjects orders objects by their dependency edges so each object is
// processed only after its dependencies. Cycles are broken via Tarjan SCC
// with intra-SCC lexicographic ordering (C10): one member of an SCC installs
// first (and typically fails → pseudo), the rest install against the pseudo.
func topoSortObjects(objects []*objectEntry) []*objectEntry {
	if len(objects) == 0 {
		return nil
	}

	index := make(map[string]*objectEntry, len(objects))
	for _, obj := range objects {
		index[obj.key()] = obj
	}

	edges := buildDependencyEdges(objects, index)
	sccs := tarjanSCC(objects, edges)

	// Build a condensed DAG of SCCs, then topo sort and flatten.
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
			dstSCC, ok := sccOf[dst]
			if !ok || dstSCC == srcSCC {
				continue
			}
			edge := [2]int{dstSCC, srcSCC} // dst must come before src
			if seenEdge[edge] {
				continue
			}
			seenEdge[edge] = true
			condensedEdges[dstSCC] = append(condensedEdges[dstSCC], srcSCC)
			inDegree[srcSCC]++
		}
	}

	// Kahn's algorithm on the condensed graph with a deterministic tiebreak:
	// ready SCCs are sorted by the lex-smallest sort key of their members.
	readyHeap := make([]int, 0)
	for i := range sccs {
		if inDegree[i] == 0 {
			readyHeap = append(readyHeap, i)
		}
	}
	sortSCCsByMin(readyHeap, sccs)

	var flat []*objectEntry
	for len(readyHeap) > 0 {
		next := readyHeap[0]
		readyHeap = readyHeap[1:]
		flat = append(flat, sortedSCCMembers(sccs[next])...)
		for _, nb := range condensedEdges[next] {
			inDegree[nb]--
			if inDegree[nb] == 0 {
				readyHeap = append(readyHeap, nb)
			}
		}
		sortSCCsByMin(readyHeap, sccs)
	}

	// Safety net: if any objects were not emitted (e.g. invariant violation
	// in Tarjan / edge building), emit them in deterministic order so the
	// loader still covers them.
	if len(flat) != len(objects) {
		emitted := make(map[string]bool, len(flat))
		for _, o := range flat {
			emitted[o.key()] = true
		}
		var missed []*objectEntry
		for _, o := range objects {
			if !emitted[o.key()] {
				missed = append(missed, o)
			}
		}
		slices.SortStableFunc(missed, func(a, b *objectEntry) int {
			return cmp.Compare(a.sortKey(), b.sortKey())
		})
		flat = append(flat, missed...)
	}
	return flat
}

// buildDependencyEdges returns a map from each object's key to the keys of
// objects it depends on. Only edges whose target actually exists in the
// collected set are recorded; references to unknown objects become no-ops
// (the install will fail and trigger pseudo).
func buildDependencyEdges(objects []*objectEntry, index map[string]*objectEntry) map[string][]string {
	edges := make(map[string][]string, len(objects))
	for _, obj := range objects {
		var deps []string
		// Every non-schema object depends on its schema.
		if obj.kind != kindSchema {
			if key := "schema:" + obj.schema; index[key] != nil {
				deps = append(deps, key)
			}
		}
		switch obj.kind {
		case kindTable:
			for _, col := range obj.tableMeta.Columns {
				for _, ref := range extractUserTypeRefs(col.Type) {
					if k := typeRefKey(ref); index[k] != nil {
						deps = append(deps, k)
					}
				}
			}
		case kindView:
			for _, dep := range obj.viewMeta.DependencyColumns {
				if k := relationKey(dep.Schema, dep.Table); index[k] != nil {
					deps = append(deps, k)
				}
			}
		case kindMatView:
			for _, dep := range obj.matViewMeta.DependencyColumns {
				if k := relationKey(dep.Schema, dep.Table); index[k] != nil {
					deps = append(deps, k)
				}
			}
		case kindFunction:
			argTypes, _ := parseFunctionSignatureArgTypes(obj.funcMeta.Signature)
			for _, at := range argTypes {
				for _, ref := range extractUserTypeRefs(at) {
					if k := typeRefKey(ref); index[k] != nil {
						deps = append(deps, k)
					}
				}
			}
		case kindSchema, kindEnum:
			// Schemas have no outgoing edges (they are the top of the graph).
			// Enums depend only on their schema, already captured above.
		default:
			// Unknown kinds contribute no edges; install will surface the
			// problem at real-install time.
		}
		if len(deps) > 0 {
			edges[obj.key()] = dedupStrings(deps)
		}
	}
	return edges
}

func typeRefKey(r UserTypeRef) string {
	return "type:" + r.Schema + "." + r.Name
}

func relationKey(schema, name string) string {
	return "rel:" + schema + "." + name
}

func dedupStrings(in []string) []string {
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

// tarjanSCC runs Tarjan's strongly-connected-components algorithm on the
// dependency graph and returns SCCs. Each SCC's members are returned in the
// order the algorithm discovered them; callers sort for determinism.
func tarjanSCC(objects []*objectEntry, edges map[string][]string) [][]*objectEntry {
	type state struct {
		index, low int
		onStack    bool
	}
	st := make(map[string]*state, len(objects))
	byKey := make(map[string]*objectEntry, len(objects))
	for _, obj := range objects {
		byKey[obj.key()] = obj
	}

	// Walk nodes in deterministic order so tests and shadow diffs reproduce.
	keys := make([]string, 0, len(objects))
	for _, obj := range objects {
		keys = append(keys, obj.key())
	}
	slices.Sort(keys)

	var (
		index  int
		stack  []string
		result [][]*objectEntry
	)

	var strongconnect func(v string)
	strongconnect = func(v string) {
		st[v] = &state{index: index, low: index, onStack: true}
		index++
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
			var scc []*objectEntry
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

// sortedSCCMembers returns a copy of scc sorted by sortKey for deterministic
// install order.
func sortedSCCMembers(scc []*objectEntry) []*objectEntry {
	out := make([]*objectEntry, len(scc))
	copy(out, scc)
	slices.SortStableFunc(out, func(a, b *objectEntry) int {
		return cmp.Compare(a.sortKey(), b.sortKey())
	})
	return out
}

// sortSCCsByMin sorts a slice of SCC indices in place by the lex-smallest
// sortKey among each SCC's members. Stable ordering.
func sortSCCsByMin(indices []int, sccs [][]*objectEntry) {
	slices.SortStableFunc(indices, func(a, b int) int {
		return cmp.Compare(minSortKey(sccs[a]), minSortKey(sccs[b]))
	})
}

func minSortKey(scc []*objectEntry) string {
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

// installOne attempts a real install; on failure installs a pseudo at the
// same slot. All outcomes are recorded in l.degraded / l.trulyBroken; this
// function never returns an error to the caller because per-object failures
// are not fatal to the loader.
//
// Functions are a special case: when a function's real install fails, we
// deliberately leave the slot empty. Query span has a dedicated fallback for
// user functions (`tryMetadataFuncLookup` in query_span_omni.go) that
// reconstructs a synthetic proc from metadata at query time. Installing a
// text-backed pseudo function would crowd out that path, producing wrong
// overload resolution and losing body-level lineage. Tables/views/types are
// fine to pseudo-install because their lookups only need the object to exist.
func (l *e3Loader) installOne(obj *objectEntry) {
	realErr := l.installReal(obj)
	if realErr == nil {
		return
	}
	l.degraded[obj.key()] = realErr
	if obj.kind == kindFunction {
		// No pseudo — rely on tryMetadataFuncLookup at query time.
		return
	}
	if pErr := l.installPseudo(obj); pErr != nil {
		l.trulyBroken[obj.key()] = pErr
	}
}

// installReal translates the object's metadata into an AST and calls the
// matching omni DefineX / ExecCreateTableAs / CreateFunctionStmt. Returns
// the first error encountered.
func (l *e3Loader) installReal(obj *objectEntry) error {
	switch obj.kind {
	case kindSchema:
		err := l.cat.CreateSchemaCommand(&ast.CreateSchemaStmt{
			Schemaname: obj.schema,
		})
		// Built-in schemas (public, pg_catalog, pg_toast) are preloaded by
		// catalog.New(). Treating duplicate-schema as success keeps the
		// loader's "every metadata object is present post-Load" invariant
		// without needing a hardcoded exclusion list.
		var cErr *catalog.Error
		if errors.As(err, &cErr) && cErr.Code == catalog.CodeDuplicateSchema {
			return nil
		}
		return err
	case kindEnum:
		return l.cat.DefineEnum(buildCreateEnumStmt(obj.schema, obj.enumMeta))
	case kindTable:
		stmt, err := buildCreateStmt(obj.schema, obj.tableMeta)
		if err != nil {
			return err
		}
		return l.cat.DefineRelation(stmt, 'r')
	case kindView:
		stmt, err := buildViewStmt(obj.schema, obj.viewMeta)
		if err != nil {
			return err
		}
		return l.cat.DefineView(stmt)
	case kindMatView:
		stmt, err := buildCreateTableAsStmt(obj.schema, obj.matViewMeta)
		if err != nil {
			return err
		}
		return l.cat.ExecCreateTableAs(stmt)
	case kindFunction:
		stmt, err := buildCreateFunctionStmt(obj.schema, obj.funcMeta)
		if err != nil {
			return err
		}
		return l.cat.CreateFunctionStmt(stmt)
	}
	return errors.Errorf("unknown object kind %d", obj.kind)
}

// installPseudo builds and installs the pseudo variant of an object. Every
// pseudo form is text-backed and free of user-object dependencies, so this
// function essentially never fails; the few error paths exist for defense in
// depth.
func (l *e3Loader) installPseudo(obj *objectEntry) error {
	switch obj.kind {
	case kindSchema:
		// Schemas have no pseudo form — they either exist or they don't.
		// If CreateSchemaCommand failed, treat as trulyBroken.
		return errors.New("schema has no pseudo form")
	case kindEnum:
		return l.cat.DefineEnum(pseudoCreateEnumStmt(obj.schema, obj.name))
	case kindTable:
		cols := columnNamesFromTableMetadata(obj.tableMeta)
		return l.cat.DefineRelation(pseudoCreateTableStmt(obj.schema, obj.name, cols), 'r')
	case kindView:
		cols := viewColumnNames(obj.viewMeta)
		stmt, err := pseudoViewStmt(obj.schema, obj.name, cols)
		if err != nil {
			return err
		}
		return l.cat.DefineView(stmt)
	case kindMatView:
		cols := matviewColumnNames(obj.matViewMeta)
		stmt, err := pseudoCreateTableAsStmt(obj.schema, obj.name, cols)
		if err != nil {
			return err
		}
		return l.cat.ExecCreateTableAs(stmt)
	case kindFunction:
		argCount := functionArgCountFromSignature(obj.funcMeta.Signature)
		return l.cat.CreateFunctionStmt(pseudoCreateFunctionStmt(obj.schema, obj.name, argCount))
	}
	return errors.Errorf("unknown object kind %d", obj.kind)
}

// columnNamesFromTableMetadata returns the deduplicated column names of a
// table's metadata, in order, for use as pseudo column names.
func columnNamesFromTableMetadata(tbl *storepb.TableMetadata) []string {
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

// viewColumnNames returns the pseudo column-name list for a view. Prefers
// explicit Columns (when sync populated them); falls back to the set of
// dependency_columns.column values.
func viewColumnNames(v *storepb.ViewMetadata) []string {
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
	seen := make(map[string]bool, len(v.DependencyColumns))
	out := make([]string, 0, len(v.DependencyColumns))
	for _, dc := range v.DependencyColumns {
		if dc.Column == "" || seen[dc.Column] {
			continue
		}
		seen[dc.Column] = true
		out = append(out, dc.Column)
	}
	return out
}

// matviewColumnNames returns pseudo column names for a materialized view.
// MaterializedViewMetadata does not carry a Columns field, so we fall back
// to dependency_columns.
func matviewColumnNames(m *storepb.MaterializedViewMetadata) []string {
	if m == nil {
		return nil
	}
	seen := make(map[string]bool, len(m.DependencyColumns))
	out := make([]string, 0, len(m.DependencyColumns))
	for _, dc := range m.DependencyColumns {
		if dc.Column == "" || seen[dc.Column] {
			continue
		}
		seen[dc.Column] = true
		out = append(out, dc.Column)
	}
	return out
}
