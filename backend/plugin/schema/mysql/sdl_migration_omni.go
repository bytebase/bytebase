package mysql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	// MySQL-only for now: OceanBase is intentionally NOT registered pending validation
	// against a live OceanBase oracle — an executable-but-unvalidated registration would
	// route OB declarative releases through untested diff/drop-advice paths.
	schema.RegisterDiffSDLMigration(storepb.Engine_MYSQL, mysqlDiffSDLMigration)
	schema.RegisterSDLDropAdvices(storepb.Engine_MYSQL, mysqlSDLDropAdvices)
	// Metadata-to-metadata diffs (SQLService.DiffMetadata, changelog-target DiffSchema)
	// keep the pre-SDL legacy differ/generator path (mirroring pg's
	// RegisterDiffMetadataMigration split): without this registration,
	// schema.DiffMigration would round-trip synced metadata through SDL text, changing
	// the established output semantics and failing the whole diff whenever any synced
	// object body does not parse.
	schema.RegisterDiffMetadataMigration(storepb.Engine_MYSQL, mysqlDiffMetadataMigration)
}

// sdlDatabaseContext is the synthetic database both SDL inputs are loaded under.
// Bytebase declarative schema text is single-database and unqualified (no leading
// CREATE DATABASE / USE), but the omni catalog resolves unqualified CREATE TABLE,
// foreign-key, and view references against a current database and errors with
// "No database selected" otherwise. Loading both the source (dumped) and target
// (user) SDL under the SAME fixed database name keeps every table identity key
// (database, name) aligned, so the database wrapper is invisible to the diff.
const sdlDatabaseContext = "bbcatalog"

// withDatabaseContext prepends a CREATE DATABASE / USE preamble so unqualified
// declarative DDL resolves. foreign_key_checks is left to LoadSDL, which disables
// it for the whole load.
func withDatabaseContext(text string) string {
	return "CREATE DATABASE IF NOT EXISTS `" + sdlDatabaseContext + "`;\nUSE `" + sdlDatabaseContext + "`;\n" + text
}

// prologueLineCount is the number of lines withDatabaseContext prepends. Omni reports
// parse positions against the wrapped text; errors surfaced to callers must subtract
// the prologue so "(line N, column M)" points into the caller's schema text (the
// user's SDL file), not the synthetic bbcatalog preamble.
const prologueLineCount = 2

var errorLinePattern = regexp.MustCompile(`\(line (\d+), column (\d+)\)`)

// adjustPrologueError rewrites omni parse positions in err's message from
// wrapped-text coordinates to caller-text coordinates. Positions inside the
// prologue itself (line <= prologueLineCount) are left untouched — they cannot
// originate from caller text and losing them would hide an internal bug.
func adjustPrologueError(err error) error {
	if err == nil {
		return nil
	}
	msg := errorLinePattern.ReplaceAllStringFunc(err.Error(), func(m string) string {
		sub := errorLinePattern.FindStringSubmatch(m)
		line, convErr := strconv.Atoi(sub[1])
		if convErr != nil || line <= prologueLineCount {
			return m
		}
		return fmt.Sprintf("(line %d, column %s)", line-prologueLineCount, sub[2])
	})
	return errors.New(msg)
}

// mysqlVersionFor maps a MySQL engine version string (e.g. "5.7.25", "8.0.32", a bare
// "5.7") to the omni catalog Version whose stored form the diff/generate canonicalizer
// must reproduce. The single behavioral split is the utf8mb4 default collation
// (utf8mb4_general_ci on 5.7 vs utf8mb4_0900_ai_ci on 8.0) and integer display widths;
// anything < 8.0.0 takes the 5.7 form, otherwise 8.0. An empty or unparseable version
// falls back to MySQL80 — the historical default — so callers that cannot supply a
// version keep the existing 8.0 behavior.
func mysqlVersionFor(engineVersion string) catalog.Version {
	v := strings.TrimSpace(engineVersion)
	if v == "" {
		return catalog.MySQL80
	}
	sv, err := semver.ParseTolerant(v)
	if err != nil {
		return catalog.MySQL80
	}
	if sv.LT(semver.MustParse("8.0.0")) {
		return catalog.MySQL57
	}
	return catalog.MySQL80
}

// loadCatalog parses a schema text and returns a catalog reflecting it, recording the
// target MySQL version on the catalog so the later Diff/GenerateMigration canonicalize
// against the right stored form. LoadSDLWithVersion is the canonical, dependency-aware
// path. LoadSQL is tried only as a fallback for raw dumps containing non-SDL statements
// (e.g. SET, SELECT, LOCK TABLES) that LoadSDL legitimately rejects; the fallback catalog's
// version is set explicitly so it is not left at the New() default. When both fail, the
// LoadSDL error is returned because it is far more diagnostic than LoadSQL's order-dependent
// failures.
func loadCatalog(text string, version catalog.Version) (*catalog.Catalog, error) {
	wrapped := withDatabaseContext(text)
	c, sdlErr := catalog.LoadSDLWithVersion(wrapped, version)
	if sdlErr == nil {
		return c, nil
	}
	// SetVersion alone only sets session.Version; it does NOT seed
	// session.explicit_defaults_for_timestamp, which LoadSDLWithVersion seeds from the
	// version normalizer (omni sdl.go) because it governs how a bare TIMESTAMP column
	// materializes at load time. Omni exposes no setter for it, so we prepend an explicit
	// SET to the fallback text: omni's exec handler applies it (before any CREATE TABLE)
	// exactly as it would an in-dump SET, seeding EDFT to this version's server-box default
	// (OFF on 5.7, ON on 8.0). Without this, a 5.7 target taking the LoadSQL fallback keeps
	// the 8.0 default and phantom-diffs against a 5.7 source loaded via LoadSDLWithVersion.
	// A later SET inside the dump still wins (omni processes it afterwards).
	if c, err := catalog.LoadSQL(explicitDefaultsForTimestampSet(version) + wrapped); err == nil {
		c.SetVersion(version)
		return c, nil
	}
	return nil, adjustPrologueError(sdlErr)
}

// explicitDefaultsForTimestampSet returns a leading `SET explicit_defaults_for_timestamp`
// statement that seeds the session variable to version's server-box default, matching what
// LoadSDLWithVersion seeds via the version normalizer.
func explicitDefaultsForTimestampSet(version catalog.Version) string {
	if catalog.NormalizerFor(version).ExplicitDefaultsForTimestamp {
		return "SET explicit_defaults_for_timestamp = 1;\n"
	}
	return "SET explicit_defaults_for_timestamp = 0;\n"
}

// buildMigrationPlan loads two schema texts into catalogs, diffs them, and returns the
// migration plan. Returns nil if there are no changes. Both catalogs carry the same
// target MySQL version, so Diff (which reads the version off the `to` catalog) and
// GenerateMigration reproduce that version's stored form — a 5.7 schema no longer
// phantom-diffs on the utf8mb4 default collation and never emits utf8mb4_0900_ai_ci.
func buildMigrationPlan(sourceText, targetText string, version catalog.Version) (*catalog.MigrationPlan, error) {
	from, err := loadCatalog(sourceText, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load source schema")
	}
	to, err := loadCatalog(targetText, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load target schema")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return nil, nil
	}
	return catalog.GenerateMigration(from, to, diff), nil
}

// mysqlDiffSDLMigration is the registered SDL diff entry point: two schema texts plus
// the target server version in, migration SQL out. engineVersion is the synced
// database's version string (e.g. "5.7.25"), threaded by the release/DiffSchema paths
// so the omni catalog selects the version-correct normalizer (a 5.7 database is
// canonicalized as 5.7 — utf8mb4_general_ci default, integer display widths — instead
// of the 8.0 stored form); an empty or unparseable value falls back to 8.0.
func mysqlDiffSDLMigration(sourceSDL, targetSDL, engineVersion string) (string, error) {
	plan, err := buildMigrationPlan(sourceSDL, targetSDL, mysqlVersionFor(engineVersion))
	if err != nil {
		return "", err
	}
	if plan == nil {
		return "", nil
	}
	return stripDatabaseContext(plan.SQL()), nil
}

// mysqlDiffMetadataMigration computes migration SQL between two metadata states via the
// LEGACY differ/generator (schema.GetDatabaseSchemaDiff + schema.GenerateMigration) —
// exactly the pre-SDL behavior of schema.DiffMigration for MySQL. Metadata-based
// callers (SQLService.DiffMetadata, changelog-target DiffSchema) must not round-trip
// synced metadata through SDL text: the SDL path changes the established output shape
// and fails the whole database diff whenever any synced object body does not parse
// (e.g. a stored view definition the omni loader rejects), which the legacy path
// diffs textually without parsing.
func mysqlDiffMetadataMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_MYSQL, oldSchema, newSchema)
	if err != nil {
		return "", err
	}
	return schema.GenerateMigration(storepb.Engine_MYSQL, diff)
}

// stripDatabaseContext removes the synthetic database qualifier from generated DDL.
// loadCatalog loads both SDL inputs under sdlDatabaseContext so unqualified references
// resolve, which makes the omni migration generator emit fully-qualified identifiers
// (e.g. ALTER TABLE `bbcatalog`.`t`). Bytebase applies the migration within a connection
// already scoped to the target database, so the qualifier must be removed.
//
// The scan is literal- and comment-aware: a '…' or "…" string literal — say a view
// body or CHECK expression whose text happens to contain the byte sequence
// `bbcatalog`.`x` — is copied opaquely (scanViewBodyToken, the shared quote/identifier
// scanner), so only identifier-position qualifiers are stripped, and the identifier
// following a stripped qualifier is itself copied opaquely so even an object literally
// named "bbcatalog" keeps its own quoting. Comments matter because omni emits routine
// bodies VERBATIM into create ops: a user comment ("-- don't …") carries unbalanced
// quotes that would otherwise open a phantom literal swallowing later qualifiers, so
// #…, "-- …", and non-executable /*…*/ comments are copied opaquely, while executable
// /*!…*/ comment interiors are scanned as the live SQL they are.
func stripDatabaseContext(sql string) string {
	quoted := "`" + sdlDatabaseContext + "`"
	if !strings.Contains(sql, quoted+".") {
		return sql
	}
	var out strings.Builder
	out.Grow(len(sql))
	for i := 0; i < len(sql); {
		c := sql[i]
		if isViewBodySpaceByte(c) {
			out.WriteByte(c)
			i++
			continue
		}
		if end, isComment := scanMySQLComment(sql, i); isComment {
			out.WriteString(sql[i:end])
			i = end
			continue
		}
		end := scanViewBodyToken(sql, i)
		tok := sql[i:end]
		if tok == quoted && end < len(sql) && sql[end] == '.' && end+1 < len(sql) && sql[end+1] == '`' {
			identEnd := scanViewBodyToken(sql, end+1)
			out.WriteString(sql[end+1 : identEnd])
			i = identEnd
			continue
		}
		out.WriteString(tok)
		i = end
	}
	return out.String()
}

// scanMySQLComment reports whether a NON-EXECUTABLE MySQL comment starts at i and, if
// so, the offset just past it: "#…" and "-- …" (double dash followed by whitespace or
// end of input, per MySQL's rule) run to end of line; "/*…*/" runs to its terminator.
// Version-gated executable comments ("/*!…*/") are NOT comments — their interior is
// live SQL — so they are left to the normal token scan.
func scanMySQLComment(sql string, i int) (int, bool) {
	switch sql[i] {
	case '#':
		return mysqlLineCommentEnd(sql, i), true
	case '-':
		if i+1 < len(sql) && sql[i+1] == '-' && (i+2 >= len(sql) || isViewBodySpaceByte(sql[i+2])) {
			return mysqlLineCommentEnd(sql, i), true
		}
	case '/':
		if i+1 < len(sql) && sql[i+1] == '*' && (i+2 >= len(sql) || sql[i+2] != '!') {
			if idx := strings.Index(sql[i+2:], "*/"); idx >= 0 {
				return i + 2 + idx + 2, true
			}
			return len(sql), true
		}
	default:
	}
	return 0, false
}

// mysqlLineCommentEnd returns the offset of the newline terminating a line comment
// starting at i (or end of input); the newline itself is left to the whitespace copy.
func mysqlLineCommentEnd(sql string, i int) int {
	for i < len(sql) && sql[i] != '\n' {
		i++
	}
	return i
}

// mysqlSDLDropAdvices analyzes the version-aware SDL migration plan from the current
// (dumped) schema to the user SDL and emits a WARNING advice for each destructive or
// in-place-replace operation, mirroring pg/sdl_migration_omni.go's pgSDLDropAdvices.
// engineVersion is threaded so the plan is built against the version-correct stored form
// (a 5.7 schema is not phantom-diffed by the utf8mb4 default-collation gap), keeping the
// drop set free of false positives.
func mysqlSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error) {
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_MYSQL, currentSchema)
	if err != nil {
		return nil, err
	}
	plan, err := buildMigrationPlan(sourceSDL, userSDLText, mysqlVersionFor(engineVersion))
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}

	// Classify view / function / procedure / trigger / event operations as DROP vs REPLACE.
	//
	// This omni build renders a redefinition two different ways depending on object kind:
	//   - view & event: a SINGLE OpCreate<Obj> carrying CREATE OR REPLACE / ALTER SQL (no
	//     paired drop);
	//   - function, procedure & trigger: a paired OpDrop<Obj> + OpCreate<Obj> of the same name.
	// A first-time create is always a lone OpCreate<Obj>, and a removal is always a lone
	// OpDrop<Obj>. The op-type alone therefore cannot tell a redefine from a first-time create,
	// so we cross-reference the per-kind drop/create name sets (for the paired case) and the
	// current schema (for the lone-create redefine case).
	//
	// Rules, keyed by the CREATE op-type:
	//   - drop AND create of a name both present  -> REPLACE, emitted once from the create
	//     (the paired drop is suppressed);
	//   - lone create of a name that already exists in currentSchema -> REPLACE (view/event
	//     redefine);
	//   - lone create of a name absent from currentSchema -> first-time create, no advice;
	//   - lone drop (no matching create) -> DROP.
	dropped := newNameSet()
	created := newNameSet()
	for _, op := range plan.Ops {
		if ct, ok := dropOpToCreateOp(op.Type); ok {
			dropped.add(ct, op.ObjectName)
		}
		if isCreateOp(op.Type) {
			created.add(op.Type, op.ObjectName)
		}
	}

	var advices []*storepb.Advice
	add := func(a *storepb.Advice) { advices = append(advices, a) }
	for _, op := range plan.Ops {
		switch op.Type {
		case catalog.OpDropTable:
			add(dropAdvice(fmt.Sprintf("Dropping table '%s' will result in data loss.", op.ObjectName)))
		// For column/index/constraint/foreign-key/check ops the omni MigrationOp carries the
		// owning TABLE name in both ObjectName and ParentObject; the precise sub-object name
		// lives only in op.SQL (e.g. "... DROP COLUMN `bio`"). So these advices name the table
		// and append the operation statement (with the synthetic bbcatalog qualifier stripped)
		// rather than a (table-valued) ObjectName.
		case catalog.OpDropColumn:
			add(dropAdvice(fmt.Sprintf("Dropping a column from table '%s' will result in data loss. (%s)", op.ParentObject, stripDatabaseContext(op.SQL))))
		case catalog.OpModifyColumn:
			add(dropAdvice(fmt.Sprintf("Modifying a column on table '%s' may result in data loss. (%s)", op.ParentObject, stripDatabaseContext(op.SQL))))
		case catalog.OpDropIndex:
			add(dropAdvice(fmt.Sprintf("Dropping an index on table '%s'. (%s)", op.ParentObject, stripDatabaseContext(op.SQL))))
		case catalog.OpDropConstraint:
			add(dropAdvice(fmt.Sprintf("Dropping a constraint on table '%s'. (%s)", op.ParentObject, stripDatabaseContext(op.SQL))))
		case catalog.OpDropForeignKey:
			add(dropAdvice(fmt.Sprintf("Dropping a foreign key on table '%s'. (%s)", op.ParentObject, stripDatabaseContext(op.SQL))))
		case catalog.OpDropCheck:
			add(dropAdvice(fmt.Sprintf("Dropping a check constraint on table '%s'. (%s)", op.ParentObject, stripDatabaseContext(op.SQL))))

		// Object DROP branches: warn unless this drop is the paired half of a redefinition
		// (the create half emits the single REPLACE advice below).
		case catalog.OpDropView:
			if !created.has(catalog.OpCreateView, op.ObjectName) {
				add(dropAdvice(fmt.Sprintf("Dropping view '%s' will affect dependent objects.", op.ObjectName)))
			}
		case catalog.OpDropFunction:
			if !created.has(catalog.OpCreateFunction, op.ObjectName) {
				add(dropAdvice(fmt.Sprintf("Dropping function '%s' will affect dependent objects.", op.ObjectName)))
			}
		case catalog.OpDropProcedure:
			if !created.has(catalog.OpCreateProcedure, op.ObjectName) {
				add(dropAdvice(fmt.Sprintf("Dropping procedure '%s' will affect dependent objects.", op.ObjectName)))
			}
		case catalog.OpDropTrigger:
			if !created.has(catalog.OpCreateTrigger, op.ObjectName) {
				add(dropAdvice(fmt.Sprintf("Dropping trigger '%s' on table '%s'.", op.ObjectName, op.ParentObject)))
			}
		case catalog.OpDropEvent:
			if !created.has(catalog.OpCreateEvent, op.ObjectName) {
				add(dropAdvice(fmt.Sprintf("Dropping event '%s'.", op.ObjectName)))
			}

		// Object CREATE branches: a create is a REPLACE when it has a paired drop OR when the
		// object already exists in the current schema (a lone-create redefine). A first-time
		// create (no paired drop, not in currentSchema) produces no advice.
		case catalog.OpCreateView:
			if dropped.has(catalog.OpCreateView, op.ObjectName) || objectExistsInSchema(currentSchema, catalog.OpCreateView, op.ObjectName) {
				add(replaceAdvice(fmt.Sprintf("View '%s' definition will be replaced.", op.ObjectName)))
			}
		case catalog.OpCreateFunction:
			if dropped.has(catalog.OpCreateFunction, op.ObjectName) || objectExistsInSchema(currentSchema, catalog.OpCreateFunction, op.ObjectName) {
				add(replaceAdvice(fmt.Sprintf("Function '%s' definition will be replaced.", op.ObjectName)))
			}
		case catalog.OpCreateProcedure:
			if dropped.has(catalog.OpCreateProcedure, op.ObjectName) || objectExistsInSchema(currentSchema, catalog.OpCreateProcedure, op.ObjectName) {
				add(replaceAdvice(fmt.Sprintf("Procedure '%s' definition will be replaced.", op.ObjectName)))
			}
		case catalog.OpCreateTrigger:
			if dropped.has(catalog.OpCreateTrigger, op.ObjectName) || objectExistsInSchema(currentSchema, catalog.OpCreateTrigger, op.ObjectName) {
				add(replaceAdvice(fmt.Sprintf("Trigger '%s' definition will be replaced.", op.ObjectName)))
			}
		case catalog.OpCreateEvent:
			if dropped.has(catalog.OpCreateEvent, op.ObjectName) || objectExistsInSchema(currentSchema, catalog.OpCreateEvent, op.ObjectName) {
				add(replaceAdvice(fmt.Sprintf("Event '%s' definition will be replaced.", op.ObjectName)))
			}
		default:
		}
	}

	return advices, nil
}

// nameSet groups object names by their CREATE op-type so a drop and a create of the same
// object are bucketed under the same key (dropOpToCreateOp maps the drop type to its create
// type), letting the advice walker pair them. Names are compared case-insensitively because
// MySQL view/routine/trigger/event identities are not case-sensitive.
type nameSet struct {
	m map[catalog.MigrationOpType]map[string]bool
}

func newNameSet() nameSet { return nameSet{m: map[catalog.MigrationOpType]map[string]bool{}} }

func (s nameSet) add(createOp catalog.MigrationOpType, name string) {
	if s.m[createOp] == nil {
		s.m[createOp] = map[string]bool{}
	}
	s.m[createOp][strings.ToLower(name)] = true
}

func (s nameSet) has(createOp catalog.MigrationOpType, name string) bool {
	return s.m[createOp] != nil && s.m[createOp][strings.ToLower(name)]
}

// dropOpToCreateOp maps a DROP op-type for a view/routine/trigger/event to its paired CREATE
// op-type, returning false for any other op-type. Bucketing drops under the create key lets
// the walker detect a drop+create redefinition pair.
func dropOpToCreateOp(t catalog.MigrationOpType) (catalog.MigrationOpType, bool) {
	switch t {
	case catalog.OpDropView:
		return catalog.OpCreateView, true
	case catalog.OpDropFunction:
		return catalog.OpCreateFunction, true
	case catalog.OpDropProcedure:
		return catalog.OpCreateProcedure, true
	case catalog.OpDropTrigger:
		return catalog.OpCreateTrigger, true
	case catalog.OpDropEvent:
		return catalog.OpCreateEvent, true
	default:
		return "", false
	}
}

// isCreateOp reports whether t is a view/routine/trigger/event CREATE op-type.
func isCreateOp(t catalog.MigrationOpType) bool {
	switch t {
	case catalog.OpCreateView, catalog.OpCreateFunction, catalog.OpCreateProcedure, catalog.OpCreateTrigger, catalog.OpCreateEvent:
		return true
	default:
		return false
	}
}

// objectExistsInSchema reports whether an object of the given CREATE kind and name already
// exists in the current (synced) schema. It is used to classify a lone OpCreate<Obj> (no
// paired drop) as a redefinition (object present) versus a first-time create (absent) — the
// view/event redefine case this omni build renders without a paired drop. MySQL is single
// schema; the synced metadata stores it under the empty schema name.
func objectExistsInSchema(currentSchema *model.DatabaseMetadata, createOp catalog.MigrationOpType, name string) bool {
	if currentSchema == nil {
		return false
	}
	sm := currentSchema.GetSchemaMetadata("")
	if sm == nil {
		return false
	}
	switch createOp {
	case catalog.OpCreateView:
		return sm.GetView(name) != nil
	case catalog.OpCreateFunction:
		return sm.GetFunction(name) != nil
	case catalog.OpCreateProcedure:
		return sm.GetProcedure(name) != nil
	case catalog.OpCreateTrigger:
		for _, table := range sm.GetProto().GetTables() {
			for _, trigger := range table.GetTriggers() {
				if strings.EqualFold(trigger.GetName(), name) {
					return true
				}
			}
		}
		return false
	case catalog.OpCreateEvent:
		for _, event := range sm.GetProto().GetEvents() {
			if strings.EqualFold(event.GetName(), name) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func dropAdvice(content string) *storepb.Advice {
	return &storepb.Advice{
		Status:  storepb.Advice_WARNING,
		Code:    code.SDLDropOperation.Int32(),
		Title:   "DROP operation detected",
		Content: content,
	}
}

func replaceAdvice(content string) *storepb.Advice {
	return &storepb.Advice{
		Status:  storepb.Advice_WARNING,
		Code:    code.SDLReplaceOperation.Int32(),
		Title:   "CREATE OR REPLACE operation detected",
		Content: content,
	}
}
