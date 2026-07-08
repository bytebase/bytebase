package schema

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	mux                             sync.Mutex
	getDatabaseDefinitions          = make(map[storepb.Engine]getDatabaseDefinition)
	getSchemaDefinitions            = make(map[storepb.Engine]getSchemaDefinition)
	getTableDefinitions             = make(map[storepb.Engine]getTableDefinition)
	getViewDefinitions              = make(map[storepb.Engine]getViewDefinition)
	getMaterializedViewDefinitions  = make(map[storepb.Engine]getMaterializedViewDefinition)
	getFunctionDefinitions          = make(map[storepb.Engine]getFunctionDefinition)
	getProcedureDefinitions         = make(map[storepb.Engine]getProcedureDefinition)
	getSequenceDefinitions          = make(map[storepb.Engine]getSequenceDefinition)
	getDatabaseMetadataMap          = make(map[storepb.Engine]getDatabaseMetadata)
	generateMigrations              = make(map[storepb.Engine]generateMigration)
	getSDLDiffs                     = make(map[storepb.Engine]getSDLDiff)
	sdlDropAdvicesFns               = make(map[storepb.Engine]sdlDropAdvices)
	diffSDLMigrations               = make(map[storepb.Engine]diffSDLMigration)
	diffMetadataMigrations          = make(map[storepb.Engine]diffMetadataMigration)
	getMultiFileDatabaseDefinitions = make(map[storepb.Engine]getMultiFileDatabaseDefinition)
	walkThroughs                    = make(map[storepb.Engine]walkThrough)
	walkThroughsWithContext         = make(map[storepb.Engine]walkThroughWithContext)
)

type getDatabaseDefinition func(GetDefinitionContext, *storepb.DatabaseSchemaMetadata) (string, error)
type getMultiFileDatabaseDefinition func(GetDefinitionContext, *storepb.DatabaseSchemaMetadata) (*MultiFileSchemaResult, error)
type getSchemaDefinition func(*storepb.SchemaMetadata) (string, error)
type getTableDefinition func(string, *storepb.TableMetadata, []*storepb.SequenceMetadata) (string, error)
type getViewDefinition func(string, *storepb.ViewMetadata) (string, error)
type getMaterializedViewDefinition func(string, *storepb.MaterializedViewMetadata) (string, error)
type getFunctionDefinition func(string, *storepb.FunctionMetadata) (string, error)
type getProcedureDefinition func(string, *storepb.ProcedureMetadata) (string, error)
type getSequenceDefinition func(string, *storepb.SequenceMetadata) (string, error)
type getDatabaseMetadata func(string) (*storepb.DatabaseSchemaMetadata, error)
type generateMigration func(*MetadataDiff) (string, error)
type getSDLDiff func(currentSDLText, previousUserSDLText string, currentSchema *model.DatabaseMetadata) (*MetadataDiff, error)
type sdlDropAdvices func(userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error)

// diffSDLMigration computes migration SQL between two SDL texts. engineVersion is the
// target server's version string (e.g. "5.7.25"); engines that canonicalize
// differently per version (MySQL: utf8mb4 default collation, integer display widths)
// use it to select the correct normalizer, and engines that do not (PostgreSQL) take
// `_ string`. An empty/unparseable version must fall back to the engine's default
// stored form.
//
// sessionCtx is the per-object session context (sql_mode / charset / collation, and
// time_zone for events) captured on the SOURCE (current) schema. The declarative SDL
// text is bare (a clean export carries no `SET sql_mode` framing), so the context that a
// routine/trigger/event must be re-created under enters out-of-band here. Only MySQL
// consumes it (to wrap recreates in a save/restore of the OLD object's context); other
// engines have no session context and take a nil-safe `_`. A nil map means "no context"
// (bare recreates) — the metadata↔metadata and SDL↔SDL callers pass nil.
type diffSDLMigration func(sourceSDL, targetSDL, engineVersion string, sessionCtx *SDLSessionContextMap) (string, error)
type diffMetadataMigration func(oldSchema, newSchema *model.DatabaseMetadata) (string, error)
type walkThrough func(*model.DatabaseMetadata, []base.AST) *storepb.Advice
type walkThroughWithContext func(WalkThroughContext, *model.DatabaseMetadata, []base.AST) *storepb.Advice

// WalkThroughContext carries optional session state into schema walk-through implementations.
type WalkThroughContext struct {
	SessionUser string
	// RawSQL is the original SQL text to be executed.
	// Used by the omni-based walkthrough for catalog.Exec().
	RawSQL string
}

type GetDefinitionContext struct {
	SkipBackupSchema bool
	PrintHeader      bool
	SDLFormat        bool
}

// File represents a single file in a multi-file schema output.
type File struct {
	// Name is the file path or name (e.g., "schemas/public/tables/users.sql")
	Name string
	// Content is the file content
	Content string
}

// MultiFileSchemaResult represents the result of multi-file schema generation.
type MultiFileSchemaResult struct {
	// Files is the list of schema files organized by type
	Files []File
}

func RegisterGetSequenceDefinition(engine storepb.Engine, f getSequenceDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getSequenceDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getSequenceDefinitions[engine] = f
}

func GetSequenceDefinition(engine storepb.Engine, sequenceName string, sequence *storepb.SequenceMetadata) (string, error) {
	f, ok := getSequenceDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(sequenceName, sequence)
}

func RegisterGetFunctionDefinition(engine storepb.Engine, f getFunctionDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getFunctionDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getFunctionDefinitions[engine] = f
}

func GetFunctionDefinition(engine storepb.Engine, functionName string, function *storepb.FunctionMetadata) (string, error) {
	f, ok := getFunctionDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(functionName, function)
}

func RegisterGetProcedureDefinition(engine storepb.Engine, f getProcedureDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getProcedureDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getProcedureDefinitions[engine] = f
}

func GetProcedureDefinition(engine storepb.Engine, procedureName string, procedure *storepb.ProcedureMetadata) (string, error) {
	f, ok := getProcedureDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(procedureName, procedure)
}

func RegisterGetMaterializedViewDefinition(engine storepb.Engine, f getMaterializedViewDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getMaterializedViewDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getMaterializedViewDefinitions[engine] = f
}

func GetMaterializedViewDefinition(engine storepb.Engine, viewName string, view *storepb.MaterializedViewMetadata) (string, error) {
	f, ok := getMaterializedViewDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(viewName, view)
}

func RegisterGetViewDefinition(engine storepb.Engine, f getViewDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getViewDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getViewDefinitions[engine] = f
}

func GetViewDefinition(engine storepb.Engine, viewName string, view *storepb.ViewMetadata) (string, error) {
	f, ok := getViewDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(viewName, view)
}

func RegisterGetTableDefinition(engine storepb.Engine, f getTableDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getTableDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getTableDefinitions[engine] = f
}

func GetTableDefinition(engine storepb.Engine, tableName string, table *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) (string, error) {
	f, ok := getTableDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(tableName, table, sequences)
}

func RegisterGetSchemaDefinition(engine storepb.Engine, f getSchemaDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getSchemaDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getSchemaDefinitions[engine] = f
}

func GetSchemaDefinition(engine storepb.Engine, schema *storepb.SchemaMetadata) (string, error) {
	f, ok := getSchemaDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(schema)
}

func RegisterGetDatabaseDefinition(engine storepb.Engine, f getDatabaseDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getDatabaseDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getDatabaseDefinitions[engine] = f
}

func GetDatabaseDefinition(engine storepb.Engine, ctx GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (string, error) {
	f, ok := getDatabaseDefinitions[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, metadata)
}

func RegisterGetDatabaseMetadata(engine storepb.Engine, f getDatabaseMetadata) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getDatabaseMetadataMap[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getDatabaseMetadataMap[engine] = f
}

func GetDatabaseMetadata(engine storepb.Engine, schemaText string) (*storepb.DatabaseSchemaMetadata, error) {
	f, ok := getDatabaseMetadataMap[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(schemaText)
}

func RegisterGenerateMigration(engine storepb.Engine, f generateMigration) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := generateMigrations[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	generateMigrations[engine] = f
}

func GenerateMigration(engine storepb.Engine, diff *MetadataDiff) (string, error) {
	f, ok := generateMigrations[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(diff)
}

func RegisterGetSDLDiff(engine storepb.Engine, f getSDLDiff) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getSDLDiffs[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getSDLDiffs[engine] = f
}

func GetSDLDiff(engine storepb.Engine, currentSDLText, previousUserSDLText string, currentSchema *model.DatabaseMetadata) (*MetadataDiff, error) {
	f, ok := getSDLDiffs[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentSDLText, previousUserSDLText, currentSchema)
}

// SDLSessionContext is the session state a routine/trigger/event was created under, as
// stored in the synced metadata: sql_mode, character_set_client, collation_connection,
// and — events only — the session time_zone. It is the engine-neutral mirror of omni's
// catalog.SessionContext (the MySQL SDL implementation converts to it); other engines
// never populate it. Empty strings are legal and preserved verbatim.
type SDLSessionContext struct {
	SQLMode             string
	CharacterSetClient  string
	CollationConnection string
	TimeZone            string // events only
}

// SDLSessionContextMap carries per-object session context for a whole schema, keyed by
// lower-cased object name within each object kind. It is the out-of-band structured input
// bytebase hands to the MySQL SDL diff so a declarative recreate re-emits an object under
// its ORIGINAL session context (the bare SDL text carries none). A nil map or a missing
// entry simply leaves that object without context (a bare recreate). This lives in the
// engine-neutral schema layer so SDLMigration can build it from synced metadata without
// importing any engine's omni catalog; only the MySQL implementation consumes it.
type SDLSessionContextMap struct {
	Functions  map[string]SDLSessionContext
	Procedures map[string]SDLSessionContext
	Triggers   map[string]SDLSessionContext
	Events     map[string]SDLSessionContext
}

// buildSDLSessionContextMap extracts per-object session context from the synced current
// schema. MySQL stores functions/procedures/events on the schema and triggers on their
// owning table; a routine/trigger carries sql_mode/charset/collation and an event also
// carries time_zone. Keys are lower-cased object names (matching omni's identity folding).
// Returns nil only when there is no metadata to read; otherwise it returns an allocated
// map (with empty submaps when the schema has no such objects), which the consumer applies
// as "no context for any object". Engines without session context (their metadata leaves
// these fields empty) still produce a map, but only MySQL's diff consumes it.
func buildSDLSessionContextMap(currentSchema *model.DatabaseMetadata) *SDLSessionContextMap {
	if currentSchema == nil {
		return nil
	}
	proto := currentSchema.GetProto()
	if proto == nil {
		return nil
	}
	m := &SDLSessionContextMap{
		Functions:  map[string]SDLSessionContext{},
		Procedures: map[string]SDLSessionContext{},
		Triggers:   map[string]SDLSessionContext{},
		Events:     map[string]SDLSessionContext{},
	}
	for _, sm := range proto.GetSchemas() {
		for _, fn := range sm.GetFunctions() {
			m.Functions[strings.ToLower(fn.GetName())] = SDLSessionContext{
				SQLMode:             fn.GetSqlMode(),
				CharacterSetClient:  fn.GetCharacterSetClient(),
				CollationConnection: fn.GetCollationConnection(),
			}
		}
		for _, proc := range sm.GetProcedures() {
			m.Procedures[strings.ToLower(proc.GetName())] = SDLSessionContext{
				SQLMode:             proc.GetSqlMode(),
				CharacterSetClient:  proc.GetCharacterSetClient(),
				CollationConnection: proc.GetCollationConnection(),
			}
		}
		for _, event := range sm.GetEvents() {
			m.Events[strings.ToLower(event.GetName())] = SDLSessionContext{
				SQLMode:             event.GetSqlMode(),
				CharacterSetClient:  event.GetCharacterSetClient(),
				CollationConnection: event.GetCollationConnection(),
				TimeZone:            event.GetTimeZone(),
			}
		}
		for _, table := range sm.GetTables() {
			for _, trigger := range table.GetTriggers() {
				// Keyed by lower-cased name to match omni's trigger identity, which is
				// itself lower-folded (catalog stores triggers as map[string]*Trigger by
				// toLower(name)). Triggers whose names differ only by case therefore
				// collapse to one entry — an inherited omni limitation, not introduced
				// here; MySQL trigger names can be case-sensitive on lower_case_table_names=0.
				m.Triggers[strings.ToLower(trigger.GetName())] = SDLSessionContext{
					SQLMode:             trigger.GetSqlMode(),
					CharacterSetClient:  trigger.GetCharacterSetClient(),
					CollationConnection: trigger.GetCollationConnection(),
				}
			}
		}
	}
	return m
}

// SDLMigration computes the migration SQL from a user-provided SDL text and the
// current database schema. It converts the current metadata to SDL, then diffs
// against the user SDL. engineVersion is the target server's version string (e.g.
// "5.7.25"); for engines that canonicalize per version (MySQL) it selects the
// version-correct normalizer. An empty/unparseable version falls back to the
// engine default, and engines without a version-aware path ignore it entirely.
//
// The per-object session context (sql_mode/charset/collation, and event time_zone) is
// extracted from currentSchema and threaded to the diff so MySQL re-creates a routine/
// trigger/event under its ORIGINAL session context — the bare SDL source carries none.
// Other engines ignore it.
func SDLMigration(engine storepb.Engine, userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) (string, error) {
	sourceSDL, err := MetadataToSDL(engine, currentSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert current schema to SDL")
	}
	return diffSDLMigrationWithContext(engine, sourceSDL, userSDLText, engineVersion, buildSDLSessionContextMap(currentSchema))
}

func RegisterSDLDropAdvices(engine storepb.Engine, f sdlDropAdvices) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := sdlDropAdvicesFns[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	sdlDropAdvicesFns[engine] = f
}

// SDLDropAdvices analyzes the SDL migration for destructive operations and returns warnings.
// engineVersion is the target server's version string; engines that canonicalize per
// version (MySQL) use it so the destructive-op detection runs against the version-correct
// plan, while engines like PostgreSQL ignore it.
func SDLDropAdvices(engine storepb.Engine, userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error) {
	f, ok := sdlDropAdvicesFns[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported for SDL drop advices", engine)
	}
	return f(userSDLText, currentSchema, engineVersion)
}

// DiffMigration computes the migration SQL between two database metadata states.
// Engines may register a metadata migration to avoid round-tripping synced
// metadata through SDL text. Otherwise, engines with DiffSDLMigration registered
// convert both sides to SDL and diff (no server version is available here, so the
// engine's default stored form applies). The legacy MetadataDiff + GenerateMigration
// path remains the fallback for engines that haven't migrated yet.
func DiffMigration(engine storepb.Engine, oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	if f, ok := diffMetadataMigrations[engine]; ok {
		return f(oldSchema, newSchema)
	}
	if _, ok := diffSDLMigrations[engine]; ok {
		sourceSDL, err := MetadataToSDL(engine, oldSchema)
		if err != nil {
			return "", errors.Wrap(err, "failed to convert source schema to SDL")
		}
		targetSDL, err := MetadataToSDL(engine, newSchema)
		if err != nil {
			return "", errors.Wrap(err, "failed to convert target schema to SDL")
		}
		return DiffSDLMigration(engine, sourceSDL, targetSDL, "")
	}
	// Fallback to legacy path for engines that haven't migrated yet.
	diff, err := GetDatabaseSchemaDiff(engine, oldSchema, newSchema)
	if err != nil {
		return "", err
	}
	return GenerateMigration(engine, diff)
}

func RegisterDiffMetadataMigration(engine storepb.Engine, f diffMetadataMigration) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := diffMetadataMigrations[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	diffMetadataMigrations[engine] = f
}

// MetadataToSDL converts database metadata to SDL text using the registered GetDatabaseDefinition.
func MetadataToSDL(engine storepb.Engine, meta *model.DatabaseMetadata) (string, error) {
	if meta == nil {
		return "", nil
	}
	proto := meta.GetProto()
	if proto == nil {
		return "", nil
	}
	return GetDatabaseDefinition(engine, GetDefinitionContext{
		SkipBackupSchema: true,
		SDLFormat:        true,
	}, proto)
}

func RegisterDiffSDLMigration(engine storepb.Engine, f diffSDLMigration) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := diffSDLMigrations[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	diffSDLMigrations[engine] = f
}

// DiffSDLMigration computes migration SQL between two SDL texts. engineVersion is the
// target server's version string, threaded to engines that canonicalize per version
// (MySQL) and ignored by the rest (PostgreSQL); pass "" when no version is known to get
// the engine's default stored form.
//
// This SDL↔SDL entry point carries no per-object session context (there is no synced
// metadata to source it from), so a MySQL recreate through this path is bare. The
// live rollout uses SDLMigration, which builds the context from the current schema.
func DiffSDLMigration(engine storepb.Engine, sourceSDL, targetSDL, engineVersion string) (string, error) {
	return diffSDLMigrationWithContext(engine, sourceSDL, targetSDL, engineVersion, nil)
}

// diffSDLMigrationWithContext is DiffSDLMigration plus the optional per-object session
// context threaded to the engine's registered diff. sessionCtx is nil for the callers
// without synced metadata (SDL↔SDL DiffSchema, metadata↔metadata DiffMigration); only
// SDLMigration supplies it.
func diffSDLMigrationWithContext(engine storepb.Engine, sourceSDL, targetSDL, engineVersion string, sessionCtx *SDLSessionContextMap) (string, error) {
	f, ok := diffSDLMigrations[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported for SDL diff migration", engine)
	}
	return f(sourceSDL, targetSDL, engineVersion, sessionCtx)
}

func RegisterGetMultiFileDatabaseDefinition(engine storepb.Engine, f getMultiFileDatabaseDefinition) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getMultiFileDatabaseDefinitions[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getMultiFileDatabaseDefinitions[engine] = f
}

func GetMultiFileDatabaseDefinition(engine storepb.Engine, ctx GetDefinitionContext, metadata *storepb.DatabaseSchemaMetadata) (*MultiFileSchemaResult, error) {
	f, ok := getMultiFileDatabaseDefinitions[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported for multi-file database definition", engine)
	}
	return f(ctx, metadata)
}

func RegisterWalkThrough(engine storepb.Engine, f walkThrough) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := walkThroughs[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	walkThroughs[engine] = f
}

func RegisterWalkThroughWithContext(engine storepb.Engine, f walkThroughWithContext) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := walkThroughsWithContext[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	walkThroughsWithContext[engine] = f
}

func WalkThrough(engine storepb.Engine, d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	return WalkThroughWithContext(engine, WalkThroughContext{}, d, ast)
}

func WalkThroughWithContext(engine storepb.Engine, ctx WalkThroughContext, d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	if f, ok := walkThroughsWithContext[engine]; ok {
		return f(ctx, d, ast)
	}
	f, ok := walkThroughs[engine]
	if !ok {
		return nil
	}
	return f(d, ast)
}
