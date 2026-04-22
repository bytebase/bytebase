package mysql

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/bytebase/omni/mysql/catalog"
	mysqlparser "github.com/bytebase/omni/mysql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// autoIncrementSentinel matches the value that MySQL sync writes into
// ColumnMetadata.Default when the column is AUTO_INCREMENT. See
// get_database_definition.go:isAutoIncrement.
const autoIncrementSentinel = "AUTO_INCREMENT"

// loadWalkThroughCatalog installs every schema object from DatabaseSchemaMetadata
// into the omni MySQL catalog with per-object isolation and pseudo fallback.
//
// Rationale: the prior implementation emitted a single concatenated DDL and
// executed it as one Exec call. A single failing CREATE TABLE would leave the
// catalog missing that table, and every downstream view / user query that
// referenced it would also fail. This loader restricts each failure to its
// owning object: if a table cannot be installed as-is, a pseudo table with
// the right column names (but TEXT types, no constraints) is installed in its
// place so that references still resolve.
//
// Preconditions: the caller must already have created and selected the target
// database. This loader mutates foreign_key_checks to allow forward FK
// references during bulk load.
func loadWalkThroughCatalog(ctx context.Context, cat *catalog.Catalog, dbName string, meta *storepb.DatabaseSchemaMetadata) error {
	if cat == nil {
		return errors.New("loadWalkThroughCatalog: nil catalog")
	}
	if meta == nil {
		return nil
	}

	// Disable FK checks for the duration of the bulk load. MySQL tolerates
	// forward FK references only when this flag is off.
	prevFKChecks := cat.ForeignKeyChecks()
	cat.SetForeignKeyChecks(false)
	defer cat.SetForeignKeyChecks(prevFKChecks)

	cat.SetCurrentDatabase(dbName)

	objects := wtCollectObjects(dbName, meta)
	sorted := wtTopoSort(objects)
	for _, obj := range sorted {
		if err := ctx.Err(); err != nil {
			return err
		}
		wtInstallOne(cat, dbName, obj)
	}
	return nil
}

// wtObjectKind identifies the kind of a walk-through loader object.
type wtObjectKind int

const (
	kindWTTable wtObjectKind = iota
	kindWTView
	kindWTFunction
	kindWTProcedure
	kindWTTrigger
	kindWTEvent
)

// wtObjectEntry is one unit of work for the walk-through loader.
type wtObjectEntry struct {
	kind wtObjectKind
	name string
	// parentTable is set for kindWTTrigger — triggers must attach to an
	// existing table in the catalog, so we carry the owning table name
	// separately from the trigger name.
	parentTable string

	// Exactly one of the following is set based on kind.
	tableMeta   *storepb.TableMetadata
	viewMeta    *storepb.ViewMetadata
	funcMeta    *storepb.FunctionMetadata
	procMeta    *storepb.ProcedureMetadata
	triggerMeta *storepb.TriggerMetadata
	eventMeta   *storepb.EventMetadata
}

func (e *wtObjectEntry) key() string {
	switch e.kind {
	case kindWTTable:
		return "table:" + e.name
	case kindWTView:
		return "view:" + e.name
	case kindWTFunction:
		return "func:" + e.name
	case kindWTProcedure:
		return "proc:" + e.name
	case kindWTTrigger:
		return "trigger:" + e.parentTable + "." + e.name
	case kindWTEvent:
		return "event:" + e.name
	}
	return "unknown:" + e.name
}

func (e *wtObjectEntry) sortKey() string {
	base := wtKindLabel(e.kind) + "\x00" + e.name
	if e.kind == kindWTTrigger {
		// Keep triggers grouped by their parent table so stable ordering
		// within a file doesn't depend on trigger name alone.
		base += "\x00" + e.parentTable
	}
	return base
}

func wtKindLabel(k wtObjectKind) string {
	switch k {
	case kindWTTable:
		return "1table"
	case kindWTView:
		return "2view"
	case kindWTFunction:
		return "3function"
	case kindWTProcedure:
		return "4procedure"
	case kindWTTrigger:
		// Triggers must install after their target table. The ordering here
		// alone does the trick because kindWTTable sorts before kindWTTrigger.
		return "5trigger"
	case kindWTEvent:
		return "6event"
	}
	return "9unknown"
}

// wtCollectObjects flattens DatabaseSchemaMetadata into wtObjectEntry values.
// MySQL metadata uses a single empty-named schema, so we look only at the first.
func wtCollectObjects(_ string, meta *storepb.DatabaseSchemaMetadata) []*wtObjectEntry {
	var out []*wtObjectEntry
	for _, sm := range meta.Schemas {
		for _, tbl := range sm.Tables {
			if tbl == nil || tbl.Name == "" {
				continue
			}
			out = append(out, &wtObjectEntry{
				kind:      kindWTTable,
				name:      tbl.Name,
				tableMeta: tbl,
			})
			for _, trg := range tbl.Triggers {
				if trg == nil || trg.Name == "" {
					continue
				}
				out = append(out, &wtObjectEntry{
					kind:        kindWTTrigger,
					name:        trg.Name,
					parentTable: tbl.Name,
					triggerMeta: trg,
				})
			}
		}
		for _, view := range sm.Views {
			if view == nil || view.Name == "" {
				continue
			}
			out = append(out, &wtObjectEntry{
				kind:     kindWTView,
				name:     view.Name,
				viewMeta: view,
			})
		}
		for _, fn := range sm.Functions {
			if fn == nil || fn.Name == "" {
				continue
			}
			out = append(out, &wtObjectEntry{
				kind:     kindWTFunction,
				name:     fn.Name,
				funcMeta: fn,
			})
		}
		for _, proc := range sm.Procedures {
			if proc == nil || proc.Name == "" {
				continue
			}
			out = append(out, &wtObjectEntry{
				kind:     kindWTProcedure,
				name:     proc.Name,
				procMeta: proc,
			})
		}
		for _, ev := range sm.Events {
			if ev == nil || ev.Name == "" {
				continue
			}
			out = append(out, &wtObjectEntry{
				kind:      kindWTEvent,
				name:      ev.Name,
				eventMeta: ev,
			})
		}
	}
	return out
}

// wtTopoSort orders objects by dependency using Tarjan SCC.
//
// For MySQL the dependency graph is extremely thin: tables have no cross-edges
// (FKs are tolerated by SetForeignKeyChecks(false)), views tolerate forward
// refs via DefineView semantics, and functions/procedures reference objects
// only in their body text (opaque to us). So the graph reduces to a fixed
// per-kind ordering: tables → views → functions → procedures. We still run
// Tarjan so the shape of this file mirrors the pg walk-through loader, and so
// the ordering remains deterministic if dependency edges are added later.
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

// wtBuildEdges is intentionally thin today. See wtTopoSort for the rationale.
func wtBuildEdges(_ []*wtObjectEntry, _ map[string]*wtObjectEntry) map[string][]string {
	return map[string][]string{}
}

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

// wtInstallOne attempts a real install and falls back to pseudo for objects
// that have a pseudo form. For objects without one (functions, procedures)
// failure silently drops the object from the catalog.
func wtInstallOne(cat *catalog.Catalog, dbName string, obj *wtObjectEntry) {
	if err := wtInstallReal(cat, obj); err == nil {
		return
	}
	switch obj.kind {
	case kindWTTable:
		_ = wtInstallPseudoTable(cat, dbName, obj.tableMeta)
	case kindWTView:
		_ = wtInstallPseudoView(cat, dbName, obj.viewMeta)
	case kindWTTrigger:
		_ = wtInstallPseudoTrigger(cat, obj.triggerMeta, obj.parentTable)
	case kindWTEvent:
		_ = wtInstallPseudoEvent(cat, obj.eventMeta)
	default:
		// Functions, procedures: no pseudo form. Leave uninstalled.
	}
}

// wtInstallReal builds the object's AST directly from metadata and calls the
// corresponding catalog Define* API. This bypasses the bytebase deparser and
// the omni parser on the critical path — both of which can have bugs that
// would otherwise cause an object to fail real install when the metadata
// itself is fine.
//
// Parser use is deliberately confined to cases where the input string comes
// from MySQL itself (COLUMN_TYPE, VIEW_DEFINITION, SHOW CREATE output,
// information_schema expression columns). Parsing those is parsing reality,
// not parsing our own deparse, and that's the distinction that matters.
func wtInstallReal(cat *catalog.Catalog, obj *wtObjectEntry) error {
	switch obj.kind {
	case kindWTTable:
		stmt, err := wtBuildCreateTableStmt(obj.tableMeta)
		if err != nil {
			return err
		}
		return cat.DefineTable(stmt)
	case kindWTView:
		stmt, err := wtBuildCreateViewStmt(obj.viewMeta)
		if err != nil {
			return err
		}
		return cat.DefineView(stmt)
	case kindWTFunction:
		stmt, err := wtParseCreateRoutineStmt(obj.funcMeta.Definition, false)
		if err != nil {
			return err
		}
		return cat.DefineFunction(stmt)
	case kindWTProcedure:
		stmt, err := wtParseCreateRoutineStmt(obj.procMeta.Definition, true)
		if err != nil {
			return err
		}
		return cat.DefineProcedure(stmt)
	case kindWTTrigger:
		tm := obj.triggerMeta
		if tm == nil || tm.Timing == "" || tm.Event == "" {
			// Missing routing metadata — let pseudo fallback synthesize
			// defaults rather than storing a semantically broken trigger.
			return errors.New("trigger: missing Timing or Event")
		}
		return cat.DefineTrigger(wtBuildCreateTriggerStmt(tm, obj.parentTable))
	case kindWTEvent:
		stmt, err := wtParseCreateEventStmt(obj.eventMeta.Definition)
		if err != nil {
			return err
		}
		return cat.DefineEvent(stmt)
	}
	return errors.Errorf("wtInstallReal: unknown kind %d", obj.kind)
}

// wtBuildCreateTableStmt maps TableMetadata directly onto *ast.CreateTableStmt.
// Expression-bearing fields (DEFAULT, ON UPDATE, GENERATED, CHECK) go through
// wtParseExpr, which tolerates parse failures by silently dropping the
// affected feature rather than failing the whole table.
func wtBuildCreateTableStmt(tbl *storepb.TableMetadata) (*ast.CreateTableStmt, error) {
	if tbl == nil || tbl.Name == "" {
		return nil, errors.New("wtBuildCreateTableStmt: empty table")
	}
	stmt := &ast.CreateTableStmt{
		Table: &ast.TableRef{Name: tbl.Name},
	}

	for _, col := range tbl.Columns {
		if col == nil || col.Name == "" {
			continue
		}
		def, err := wtBuildColumnDef(col)
		if err != nil {
			return nil, errors.Wrapf(err, "column %q", col.Name)
		}
		stmt.Columns = append(stmt.Columns, def)
	}

	for _, idx := range tbl.Indexes {
		if idx == nil || len(idx.Expressions) == 0 {
			continue
		}
		if c := wtBuildIndexConstraint(idx); c != nil {
			stmt.Constraints = append(stmt.Constraints, c)
		}
	}

	for _, fk := range tbl.ForeignKeys {
		if fk == nil {
			continue
		}
		stmt.Constraints = append(stmt.Constraints, wtBuildFKConstraint(fk))
	}

	for _, chk := range tbl.CheckConstraints {
		if chk == nil {
			continue
		}
		if c := wtBuildCheckConstraint(chk); c != nil {
			stmt.Constraints = append(stmt.Constraints, c)
		}
	}

	if tbl.Engine != "" {
		stmt.Options = append(stmt.Options, &ast.TableOption{Name: "ENGINE", Value: tbl.Engine})
	}
	if tbl.Charset != "" {
		stmt.Options = append(stmt.Options, &ast.TableOption{Name: "CHARSET", Value: tbl.Charset})
	}
	if tbl.Collation != "" {
		stmt.Options = append(stmt.Options, &ast.TableOption{Name: "COLLATE", Value: tbl.Collation})
	}
	if tbl.Comment != "" {
		stmt.Options = append(stmt.Options, &ast.TableOption{Name: "COMMENT", Value: tbl.Comment})
	}

	// Partitions deliberately skipped: PartitionClause requires parsed
	// expression trees that we don't have metadata for in decomposed form.
	// Losing partition info does not affect column resolution or query shape.

	return stmt, nil
}

func wtBuildColumnDef(col *storepb.ColumnMetadata) (*ast.ColumnDef, error) {
	typeName, err := wtParseTypeName(col.Type)
	if err != nil {
		return nil, err
	}
	// Column-level charset/collation can come in via the type string or via
	// separate metadata fields. Prefer type-embedded if present, else take
	// the separate fields.
	if typeName.Charset == "" && col.CharacterSet != "" {
		typeName.Charset = col.CharacterSet
	}
	if typeName.Collate == "" && col.Collation != "" {
		typeName.Collate = col.Collation
	}

	def := &ast.ColumnDef{
		Name:     col.Name,
		TypeName: typeName,
		Comment:  col.Comment,
	}
	if !col.Nullable {
		def.Constraints = append(def.Constraints, &ast.ColumnConstraint{Type: ast.ColConstrNotNull})
	}

	// MySQL sync encodes AUTO_INCREMENT as the literal string "AUTO_INCREMENT"
	// in ColumnMetadata.Default. Detect it here and set AutoIncrement instead
	// of treating it as a real default expression.
	isAutoInc := strings.EqualFold(col.Default, autoIncrementSentinel)
	if isAutoInc {
		def.AutoIncrement = true
	}

	if !isAutoInc && col.Default != "" && col.Generation == nil {
		if expr, err := wtParseExpr(col.Default); err == nil {
			def.DefaultValue = expr
		}
		// Silent drop on parse failure — a missing DEFAULT is a closer
		// approximation of reality than a whole-column failure.
	}
	if col.OnUpdate != "" {
		if expr, err := wtParseExpr(col.OnUpdate); err == nil {
			def.OnUpdate = expr
		}
	}
	if col.Generation != nil && col.Generation.Expression != "" {
		if expr, err := wtParseExpr(col.Generation.Expression); err == nil {
			def.Generated = &ast.GeneratedColumn{
				Expr:   expr,
				Stored: col.Generation.Type == storepb.GenerationMetadata_TYPE_STORED,
			}
		}
	}
	return def, nil
}

func wtBuildIndexConstraint(idx *storepb.IndexMetadata) *ast.Constraint {
	// IndexMetadata.Type mixes two axes: the constraint kind (FULLTEXT /
	// SPATIAL) and the access method (BTREE / HASH). Pick them apart so
	// the resulting ast.Constraint sets Type and IndexType correctly.
	idxTypeUpper := strings.ToUpper(strings.TrimSpace(idx.Type))

	c := &ast.Constraint{
		Name: idx.Name,
	}
	switch {
	case idx.Primary:
		c.Type = ast.ConstrPrimaryKey
		c.IndexType = idxTypeUpper
	case idx.Unique:
		c.Type = ast.ConstrUnique
		c.IndexType = idxTypeUpper
	case idxTypeUpper == "FULLTEXT":
		c.Type = ast.ConstrFulltextIndex
	case idxTypeUpper == "SPATIAL":
		c.Type = ast.ConstrSpatialIndex
	default:
		c.Type = ast.ConstrIndex
		c.IndexType = idxTypeUpper
	}
	// Use IndexColumns when per-part length or DESC matters; otherwise the
	// simpler Columns field is enough. Functional indexes (expressions) are
	// left as plain identifiers — rare in practice and omni can't validate
	// them against the catalog anyway.
	needIC := false
	for i := range idx.Expressions {
		if (i < len(idx.KeyLength) && idx.KeyLength[i] > 0) ||
			(i < len(idx.Descending) && idx.Descending[i]) {
			needIC = true
			break
		}
	}
	if needIC {
		for i, expr := range idx.Expressions {
			ic := &ast.IndexColumn{
				Expr: &ast.ColumnRef{Column: wtUnquoteIdent(expr)},
			}
			if i < len(idx.KeyLength) && idx.KeyLength[i] > 0 {
				ic.Length = int(idx.KeyLength[i])
			}
			if i < len(idx.Descending) && idx.Descending[i] {
				ic.Desc = true
			}
			c.IndexColumns = append(c.IndexColumns, ic)
		}
	} else {
		for _, expr := range idx.Expressions {
			c.Columns = append(c.Columns, wtUnquoteIdent(expr))
		}
	}
	return c
}

func wtBuildFKConstraint(fk *storepb.ForeignKeyMetadata) *ast.Constraint {
	return &ast.Constraint{
		Type:       ast.ConstrForeignKey,
		Name:       fk.Name,
		Columns:    fk.Columns,
		RefTable:   &ast.TableRef{Schema: fk.ReferencedSchema, Name: fk.ReferencedTable},
		RefColumns: fk.ReferencedColumns,
		OnUpdate:   wtFKAction(fk.OnUpdate),
		OnDelete:   wtFKAction(fk.OnDelete),
		Match:      strings.ToUpper(fk.MatchType),
	}
}

func wtFKAction(s string) ast.ReferenceAction {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "RESTRICT":
		return ast.RefActRestrict
	case "CASCADE":
		return ast.RefActCascade
	case "SET NULL":
		return ast.RefActSetNull
	case "SET DEFAULT":
		return ast.RefActSetDefault
	case "NO ACTION":
		return ast.RefActNoAction
	default:
		return ast.RefActNone
	}
}

func wtBuildCheckConstraint(chk *storepb.CheckConstraintMetadata) *ast.Constraint {
	if chk.Expression == "" {
		return nil
	}
	expr, err := wtParseExpr(chk.Expression)
	if err != nil {
		return nil
	}
	return &ast.Constraint{
		Type: ast.ConstrCheck,
		Name: chk.Name,
		Expr: expr,
	}
}

// wtBuildCreateViewStmt builds the CreateViewStmt directly; the SELECT body is
// parsed from VIEW_DEFINITION (a MySQL-produced string, not our deparse).
// DefineView tolerates a nil Select, so forward references, cyclic views and
// body-parse failures are all survivable.
func wtBuildCreateViewStmt(view *storepb.ViewMetadata) (*ast.CreateViewStmt, error) {
	if view == nil || view.Name == "" {
		return nil, errors.New("wtBuildCreateViewStmt: empty view")
	}
	stmt := &ast.CreateViewStmt{
		OrReplace:  true,
		Name:       &ast.TableRef{Name: view.Name},
		SelectText: view.Definition,
	}
	for _, col := range view.Columns {
		if col != nil && col.Name != "" {
			stmt.Columns = append(stmt.Columns, col.Name)
		}
	}
	if view.Definition != "" {
		if sel, err := wtParseSelect(view.Definition); err == nil {
			stmt.Select = sel
		}
	}
	return stmt, nil
}

// wtParseCreateRoutineStmt parses the full SHOW CREATE FUNCTION / PROCEDURE
// text that MySQL sync stored in FunctionMetadata.Definition /
// ProcedureMetadata.Definition. The result is the AST form DefineFunction and
// DefineProcedure consume.
func wtParseCreateRoutineStmt(definition string, isProcedure bool) (*ast.CreateFunctionStmt, error) {
	if strings.TrimSpace(definition) == "" {
		return nil, errors.New("empty definition")
	}
	list, err := mysqlparser.Parse(definition)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, errors.New("nil parse result")
	}
	for _, n := range list.Items {
		if stmt, ok := n.(*ast.CreateFunctionStmt); ok {
			stmt.IsProcedure = isProcedure
			return stmt, nil
		}
	}
	return nil, errors.New("no CreateFunctionStmt in parse result")
}

// wtParseTypeName parses a MySQL column-type string (the output of
// information_schema.COLUMNS.COLUMN_TYPE, e.g. "varchar(255)",
// "int unsigned", "enum('a','b')") into *ast.DataType by wrapping it in a
// throwaway CREATE TABLE and letting the omni parser do the work.
func wtParseTypeName(typeStr string) (*ast.DataType, error) {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return nil, errors.New("empty type string")
	}
	sql := "CREATE TABLE `__bb_wt_type_probe` (`__bb_c` " + typeStr + ")"
	list, err := mysqlparser.Parse(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "parse type %q", typeStr)
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.Errorf("parse type %q: empty result", typeStr)
	}
	ct, ok := list.Items[0].(*ast.CreateTableStmt)
	if !ok {
		return nil, errors.Errorf("parse type %q: expected CreateTableStmt, got %T", typeStr, list.Items[0])
	}
	if len(ct.Columns) == 0 || ct.Columns[0].TypeName == nil {
		return nil, errors.Errorf("parse type %q: no type in parsed column", typeStr)
	}
	return ct.Columns[0].TypeName, nil
}

// wtParseExpr parses an expression string via a SELECT wrapper. Used for
// DEFAULT, ON UPDATE, generated-column and CHECK expressions — all of which
// originate in MySQL's information_schema or SHOW CREATE output (reality,
// not our deparse).
func wtParseExpr(exprStr string) (ast.ExprNode, error) {
	exprStr = strings.TrimSpace(exprStr)
	if exprStr == "" {
		return nil, errors.New("empty expression")
	}
	sql := "SELECT (" + exprStr + ") AS `__bb_wt_expr_probe`"
	list, err := mysqlparser.Parse(sql)
	if err != nil {
		return nil, errors.Wrapf(err, "parse expr %q", exprStr)
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.Errorf("parse expr %q: empty result", exprStr)
	}
	sel, ok := list.Items[0].(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("parse expr %q: expected SelectStmt, got %T", exprStr, list.Items[0])
	}
	if len(sel.TargetList) == 0 {
		return nil, errors.Errorf("parse expr %q: empty target list", exprStr)
	}
	rt, ok := sel.TargetList[0].(*ast.ResTarget)
	if !ok {
		return nil, errors.Errorf("parse expr %q: unexpected target %T", exprStr, sel.TargetList[0])
	}
	return rt.Val, nil
}

// wtParseSelect parses a SELECT body (as returned by VIEW_DEFINITION) into
// *ast.SelectStmt.
func wtParseSelect(body string) (*ast.SelectStmt, error) {
	list, err := mysqlparser.Parse(body)
	if err != nil {
		return nil, err
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("empty parse result")
	}
	sel, ok := list.Items[0].(*ast.SelectStmt)
	if !ok {
		return nil, errors.Errorf("expected SelectStmt, got %T", list.Items[0])
	}
	return sel, nil
}

func wtUnquoteIdent(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return strings.ReplaceAll(s[1:len(s)-1], "``", "`")
	}
	return s
}

// wtInstallPseudoTable installs a degraded table with all-TEXT columns and no
// constraints. Built as an AST and handed to DefineTable directly — no SQL
// string and no parser on the critical path.
func wtInstallPseudoTable(cat *catalog.Catalog, _ string, tbl *storepb.TableMetadata) error {
	if tbl == nil || tbl.Name == "" {
		return errors.New("pseudo table: missing name")
	}

	stmt := &ast.CreateTableStmt{
		Table: &ast.TableRef{Name: tbl.Name},
	}

	seen := make(map[string]bool, len(tbl.Columns))
	for _, col := range tbl.Columns {
		if col == nil || col.Name == "" || seen[col.Name] {
			continue
		}
		seen[col.Name] = true
		stmt.Columns = append(stmt.Columns, &ast.ColumnDef{
			Name:     col.Name,
			TypeName: wtPseudoTextType(),
		})
	}
	if len(stmt.Columns) == 0 {
		// MySQL requires at least one column — give it a placeholder so the
		// object still exists and blocks future duplicate-name installs.
		stmt.Columns = append(stmt.Columns, &ast.ColumnDef{
			Name:     "__bb_placeholder",
			TypeName: wtPseudoTextType(),
		})
	}
	return cat.DefineTable(stmt)
}

// wtInstallPseudoView installs a degraded view whose SELECT list is a series
// of NULL literals aliased to the original view's column names. Built in AST
// form and installed via DefineView directly.
func wtInstallPseudoView(cat *catalog.Catalog, _ string, view *storepb.ViewMetadata) error {
	if view == nil || view.Name == "" {
		return errors.New("pseudo view: missing name")
	}

	var targets []ast.ExprNode
	seen := make(map[string]bool)
	addTarget := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		targets = append(targets, &ast.ResTarget{
			Name: name,
			Val:  &ast.NullLit{},
		})
	}
	for _, col := range view.Columns {
		if col != nil {
			addTarget(col.Name)
		}
	}
	if len(targets) == 0 {
		for _, dc := range view.DependencyColumns {
			if dc != nil {
				addTarget(dc.Column)
			}
		}
	}
	if len(targets) == 0 {
		addTarget("__bb_placeholder")
	}

	return cat.DefineView(&ast.CreateViewStmt{
		OrReplace: true,
		Name:      &ast.TableRef{Name: view.Name},
		Select:    &ast.SelectStmt{TargetList: targets},
	})
}

func wtPseudoTextType() *ast.DataType {
	return &ast.DataType{Name: "TEXT"}
}

// wtBuildCreateTriggerStmt hand-constructs *ast.CreateTriggerStmt from
// TriggerMetadata. Sync fills only the body (information_schema
// TRIGGERS.ACTION_STATEMENT) without the CREATE TRIGGER header, so we wire
// Name/Timing/Event/Table/BodyText directly. Body is a parsed sp_proc_stmt —
// best-effort only; on parse failure we still hand DefineTrigger a usable
// stmt with BodyText populated.
func wtBuildCreateTriggerStmt(tm *storepb.TriggerMetadata, tableName string) *ast.CreateTriggerStmt {
	stmt := &ast.CreateTriggerStmt{
		Name:     tm.Name,
		Timing:   tm.Timing,
		Event:    tm.Event,
		Table:    &ast.TableRef{Name: tableName},
		BodyText: tm.Body,
	}
	if body := wtParseTriggerBody(tableName, tm.Timing, tm.Event, tm.Body); body != nil {
		stmt.Body = body
	}
	return stmt
}

// wtParseTriggerBody wraps the raw trigger body in a known-good CREATE TRIGGER
// template so the omni parser can produce a Body node. Source is MySQL's
// information_schema output, not our own deparse.
func wtParseTriggerBody(tableName, timing, event, body string) ast.Node {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	if tableName == "" {
		tableName = "__bb_probe_table"
	}
	if timing == "" {
		timing = "BEFORE"
	}
	if event == "" {
		event = "INSERT"
	}
	sql := "CREATE TRIGGER `__bb_probe_trigger` " + timing + " " + event +
		" ON `" + tableName + "` FOR EACH ROW " + body
	list, err := mysqlparser.Parse(sql)
	if err != nil || list == nil {
		return nil
	}
	for _, n := range list.Items {
		if ct, ok := n.(*ast.CreateTriggerStmt); ok {
			return ct.Body
		}
	}
	return nil
}

// wtInstallPseudoTrigger installs a trigger stub when the real install path
// fails (e.g. body node construction tripped up DefineTrigger, or the parent
// table name was empty in metadata). The stub preserves Name and routing
// (Timing/Event/Table) so references still resolve; Body collapses to the
// trivial no-op "BEGIN END".
func wtInstallPseudoTrigger(cat *catalog.Catalog, tm *storepb.TriggerMetadata, tableName string) error {
	if tm == nil || tm.Name == "" {
		return errors.New("pseudo trigger: missing name")
	}
	timing := tm.Timing
	if timing == "" {
		timing = "BEFORE"
	}
	event := tm.Event
	if event == "" {
		event = "INSERT"
	}
	if tableName == "" {
		tableName = "__bb_placeholder"
	}
	return cat.DefineTrigger(&ast.CreateTriggerStmt{
		Name:     tm.Name,
		Timing:   timing,
		Event:    event,
		Table:    &ast.TableRef{Name: tableName},
		BodyText: "BEGIN END",
	})
}

// wtParseCreateEventStmt parses the full SHOW CREATE EVENT text that sync
// stored in EventMetadata.Definition into *ast.CreateEventStmt.
func wtParseCreateEventStmt(definition string) (*ast.CreateEventStmt, error) {
	if strings.TrimSpace(definition) == "" {
		return nil, errors.New("empty event definition")
	}
	list, err := mysqlparser.Parse(definition)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, errors.New("nil parse result")
	}
	for _, n := range list.Items {
		if ev, ok := n.(*ast.CreateEventStmt); ok {
			return ev, nil
		}
	}
	return nil, errors.New("no CreateEventStmt in parse result")
}

// wtInstallPseudoEvent installs a bare name-only event when the real install
// path fails (Definition was empty, malformed, or rejected by DefineEvent).
func wtInstallPseudoEvent(cat *catalog.Catalog, em *storepb.EventMetadata) error {
	if em == nil || em.Name == "" {
		return errors.New("pseudo event: missing name")
	}
	return cat.DefineEvent(&ast.CreateEventStmt{
		Name: em.Name,
	})
}
