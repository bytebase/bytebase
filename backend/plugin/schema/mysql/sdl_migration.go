package mysql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	// MySQL-only for now: OceanBase is intentionally NOT registered pending validation
	// against a live OceanBase oracle — an executable-but-unvalidated registration would
	// route OB declarative releases through untested diff/drop-advice paths.
	schema.RegisterDiffSDLMigration(storepb.Engine_MYSQL, mysqlDiffSDLMigrationWithContext)
	schema.RegisterSDLDropAdvices(storepb.Engine_MYSQL, mysqlSDLDropAdvices)
	// Metadata-to-metadata diffs (DatabaseService.DiffMetadata, changelog-target
	// DiffSchema) load the snapshots into omni's catalog rather than round-tripping
	// them through SDL text, so one object omni cannot parse does not fail the diff.
	schema.RegisterDiffMetadataMigration(storepb.Engine_MYSQL, omniDiffMetadataMigration)
}

// sdlDatabaseContext is the synthetic database both SDL inputs are loaded under,
// matching the metadata diff so unqualified DDL resolves consistently.
const sdlDatabaseContext = omnicatalog.Database

// unsupportedSDLVersion returns why the declarative path refuses a MySQL server, ""
// when it does not: a server older than 8.0 is refused because omni's catalog
// canonicalizes only against the 8.0 stored form. A version that names no major
// number is accepted: callers without an instance pass "".
func unsupportedSDLVersion(engineVersion string) string {
	version := strings.TrimSpace(engineVersion)
	major, _, _ := strings.Cut(version, ".")
	if n, err := strconv.Atoi(major); err == nil && n < 8 {
		return fmt.Sprintf("declarative schema change requires MySQL 8.0 or later, but this instance runs MySQL %s", version)
	}
	return ""
}

// buildMigrationPlan delegates SDL loading, session-context stamping, diffing,
// and migration generation to omni's catalog package under sdlDatabaseContext.
func buildMigrationPlan(sourceText, targetText string, sessionCtx *schema.SDLSessionContextMap) (*catalog.MigrationPlan, error) {
	return catalog.BuildSDLMigrationPlan(sourceText, targetText, catalog.SDLMigrationOptions{
		Database:             sdlDatabaseContext,
		SourceSessionContext: toCatalogSessionContextMap(sessionCtx),
	})
}

// toCatalogSessionContextMap converts the engine-neutral schema.SDLSessionContextMap
// into omni's catalog.SessionContextMap so the omni dependency stays in this package.
func toCatalogSessionContextMap(m *schema.SDLSessionContextMap) *catalog.SessionContextMap {
	if m == nil {
		return nil
	}
	conv := func(src map[string]schema.SDLSessionContext) map[string]catalog.SessionContext {
		if len(src) == 0 {
			return nil
		}
		dst := make(map[string]catalog.SessionContext, len(src))
		for name, c := range src {
			dst[name] = catalog.SessionContext{
				SQLMode:             c.SQLMode,
				CharacterSetClient:  c.CharacterSetClient,
				CollationConnection: c.CollationConnection,
				TimeZone:            c.TimeZone,
			}
		}
		return dst
	}
	return &catalog.SessionContextMap{
		Functions:  conv(m.Functions),
		Procedures: conv(m.Procedures),
		Triggers:   conv(m.Triggers),
		Events:     conv(m.Events),
	}
}

// mysqlDiffSDLMigrationWithContext is the registered SDL diff entry point: two schema
// texts plus the target server version in, migration SQL out. A server older than
// MySQL 8.0 is rejected (see unsupportedSDLVersion).
func mysqlDiffSDLMigrationWithContext(sourceSDL, targetSDL, engineVersion string, sessionCtx *schema.SDLSessionContextMap) (string, error) {
	if reason := unsupportedSDLVersion(engineVersion); reason != "" {
		return "", errors.New(reason)
	}
	plan, err := buildMigrationPlan(sourceSDL, targetSDL, sessionCtx)
	if err != nil {
		return "", err
	}
	if plan == nil {
		return "", nil
	}
	return catalog.StripDatabaseContext(plan.SQL(), sdlDatabaseContext), nil
}

// mysqlDiffSDLMigration is the bare (no session context) 3-arg form: two schema texts plus
// the target version, migration SQL out. It is the in-package convenience used by the SDL
// test suites — a recreate through it carries no session framing, matching the old
// behavior. Production callers reach the context-aware form via schema.SDLMigration.
func mysqlDiffSDLMigration(sourceSDL, targetSDL, engineVersion string) (string, error) {
	return mysqlDiffSDLMigrationWithContext(sourceSDL, targetSDL, engineVersion, nil)
}

// mysqlSDLDropAdvices analyzes the SDL migration plan from the current (dumped)
// schema to the user SDL and emits a WARNING advice for each destructive or
// in-place-replace operation, mirroring pg/sdl_migration.go's pgSDLDropAdvices.
// A server older than MySQL 8.0 gets one ERROR advice instead: the release check
// then reports what the rollout would fail on (see unsupportedSDLVersion).
func mysqlSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error) {
	if reason := unsupportedSDLVersion(engineVersion); reason != "" {
		return []*storepb.Advice{{
			Status:  storepb.Advice_ERROR,
			Code:    code.Unsupported.Int32(),
			Title:   "MySQL version not supported",
			Content: reason,
		}}, nil
	}
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_MYSQL, currentSchema)
	if err != nil {
		return nil, err
	}
	// Session context is intentionally not applied here: drop/replace classification
	// depends only on the op TYPES (OpDrop*/OpCreate*), which are identical with or
	// without the recreate framing. Applying context would only add SET wrappers to
	// op.SQL, changing nothing the advice walker inspects.
	plan, err := buildMigrationPlan(sourceSDL, userSDLText, nil)
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
			add(dropAdvice(fmt.Sprintf("Dropping a column from table '%s' will result in data loss. (%s)", op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))
		case catalog.OpModifyColumn:
			add(dropAdvice(fmt.Sprintf("Modifying a column on table '%s' may result in data loss. (%s)", op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))
		case catalog.OpDropIndex:
			add(dropAdvice(fmt.Sprintf("Dropping an index on table '%s'. (%s)", op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))
		case catalog.OpDropConstraint:
			add(dropAdvice(fmt.Sprintf("Dropping a constraint on table '%s'. (%s)", op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))
		case catalog.OpDropForeignKey:
			add(dropAdvice(fmt.Sprintf("Dropping a foreign key on table '%s'. (%s)", op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))
		case catalog.OpDropCheck:
			add(dropAdvice(fmt.Sprintf("Dropping a check constraint on table '%s'. (%s)", op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))

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
