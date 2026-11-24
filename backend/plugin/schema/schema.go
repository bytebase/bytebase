package schema

import (
	"context"
	"fmt"
	"strings"
	"sync"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	registryMu sync.Mutex
	engines    = make(map[storepb.Engine]Engine)
)

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

// Definitions groups the schema definition renderers an engine registers.
// Any nil field is skipped.
type Definitions struct {
	Database          func(GetDefinitionContext, *metadatapb.DatabaseSchemaMetadata) (string, error)
	MultiFileDatabase func(GetDefinitionContext, *metadatapb.DatabaseSchemaMetadata) (*MultiFileSchemaResult, error)
	Schema            func(*metadatapb.SchemaMetadata) (string, error)
	Table             func(string, *metadatapb.TableMetadata, []*metadatapb.SequenceMetadata) (string, error)
	View              func(string, *metadatapb.ViewMetadata) (string, error)
	MaterializedView  func(string, *metadatapb.MaterializedViewMetadata) (string, error)
	Function          func(string, *metadatapb.FunctionMetadata) (string, error)
	Procedure         func(string, *metadatapb.ProcedureMetadata) (string, error)
	Sequence          func(string, *metadatapb.SequenceMetadata) (string, error)
}

// Engine defines the schema capabilities an engine plugin registers.
// Any nil field is omitted. Separate Register calls may supply disjoint
// capabilities for the same engine (for example, omnicatalog registers shared
// definitions while engine packages add walkthrough or metadata extraction).
type Engine struct {
	Definitions
	DatabaseMetadata func(string) (*metadatapb.DatabaseSchemaMetadata, error)
	SDLDropAdvices   func(userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error)
	// DiffSDLMigration computes migration SQL between two SDL texts.
	// engineVersion is the target server's version string (e.g. "8.0.32");
	// engines that do not inspect it accept "". sessionCtx carries per-object
	// session context from the source schema for MySQL bare-SDL recreates;
	// nil means bare recreates.
	DiffSDLMigration      func(sourceSDL, targetSDL, engineVersion string, sessionCtx *SDLSessionContextMap) (string, error)
	DiffMetadataMigration func(oldSchema, newSchema *model.DatabaseMetadata) (string, error)
	WalkThrough           func(context.Context, WalkThroughContext, *model.DatabaseMetadata, []base.AST) *storepb.Advice
}

// Register registers every non-nil capability in spec for each target engine.
// It panics if any capability is already registered for a target engine.
func Register(spec Engine, targetEngines ...storepb.Engine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, engine := range targetEngines {
		cur := engines[engine]
		mergeEngine(&cur, spec, engine)
		engines[engine] = cur
	}
}

func mergeEngine(cur *Engine, spec Engine, engine storepb.Engine) {
	checkDup := func(alreadySet bool, name string) {
		if alreadySet {
			panic(fmt.Sprintf("schema: %s registered twice for %s", name, engine))
		}
	}
	if spec.Database != nil {
		checkDup(cur.Database != nil, "Database")
		cur.Database = spec.Database
	}
	if spec.MultiFileDatabase != nil {
		checkDup(cur.MultiFileDatabase != nil, "MultiFileDatabase")
		cur.MultiFileDatabase = spec.MultiFileDatabase
	}
	if spec.Schema != nil {
		checkDup(cur.Schema != nil, "Schema")
		cur.Schema = spec.Schema
	}
	if spec.Table != nil {
		checkDup(cur.Table != nil, "Table")
		cur.Table = spec.Table
	}
	if spec.View != nil {
		checkDup(cur.View != nil, "View")
		cur.View = spec.View
	}
	if spec.MaterializedView != nil {
		checkDup(cur.MaterializedView != nil, "MaterializedView")
		cur.MaterializedView = spec.MaterializedView
	}
	if spec.Function != nil {
		checkDup(cur.Function != nil, "Function")
		cur.Function = spec.Function
	}
	if spec.Procedure != nil {
		checkDup(cur.Procedure != nil, "Procedure")
		cur.Procedure = spec.Procedure
	}
	if spec.Sequence != nil {
		checkDup(cur.Sequence != nil, "Sequence")
		cur.Sequence = spec.Sequence
	}
	if spec.DatabaseMetadata != nil {
		checkDup(cur.DatabaseMetadata != nil, "DatabaseMetadata")
		cur.DatabaseMetadata = spec.DatabaseMetadata
	}
	if spec.SDLDropAdvices != nil {
		checkDup(cur.SDLDropAdvices != nil, "SDLDropAdvices")
		cur.SDLDropAdvices = spec.SDLDropAdvices
	}
	if spec.DiffSDLMigration != nil {
		checkDup(cur.DiffSDLMigration != nil, "DiffSDLMigration")
		cur.DiffSDLMigration = spec.DiffSDLMigration
	}
	if spec.DiffMetadataMigration != nil {
		checkDup(cur.DiffMetadataMigration != nil, "DiffMetadataMigration")
		cur.DiffMetadataMigration = spec.DiffMetadataMigration
	}
	if spec.WalkThrough != nil {
		checkDup(cur.WalkThrough != nil, "WalkThrough")
		cur.WalkThrough = spec.WalkThrough
	}
}

func notSupportedError(engine storepb.Engine, feature string) error {
	if feature == "" {
		return errors.Errorf("engine %s is not supported", engine)
	}
	return errors.Errorf("engine %s is not supported for %s", engine, feature)
}

func GetSequenceDefinition(engine storepb.Engine, sequenceName string, sequence *metadatapb.SequenceMetadata) (string, error) {
	f := engines[engine].Sequence
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(sequenceName, sequence)
}

func GetFunctionDefinition(engine storepb.Engine, functionName string, function *metadatapb.FunctionMetadata) (string, error) {
	f := engines[engine].Function
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(functionName, function)
}

func GetProcedureDefinition(engine storepb.Engine, procedureName string, procedure *metadatapb.ProcedureMetadata) (string, error) {
	f := engines[engine].Procedure
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(procedureName, procedure)
}

func GetMaterializedViewDefinition(engine storepb.Engine, viewName string, view *metadatapb.MaterializedViewMetadata) (string, error) {
	f := engines[engine].MaterializedView
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(viewName, view)
}

func GetViewDefinition(engine storepb.Engine, viewName string, view *metadatapb.ViewMetadata) (string, error) {
	f := engines[engine].View
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(viewName, view)
}

func GetTableDefinition(engine storepb.Engine, tableName string, table *metadatapb.TableMetadata, sequences []*metadatapb.SequenceMetadata) (string, error) {
	f := engines[engine].Table
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(tableName, table, sequences)
}

func GetSchemaDefinition(engine storepb.Engine, schema *metadatapb.SchemaMetadata) (string, error) {
	f := engines[engine].Schema
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(schema)
}

func GetDatabaseDefinition(engine storepb.Engine, ctx GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (string, error) {
	f := engines[engine].Database
	if f == nil {
		return "", notSupportedError(engine, "")
	}
	return f(ctx, metadata)
}

func GetDatabaseMetadata(engine storepb.Engine, schemaText string) (*metadatapb.DatabaseSchemaMetadata, error) {
	f := engines[engine].DatabaseMetadata
	if f == nil {
		return nil, notSupportedError(engine, "")
	}
	return f(schemaText)
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
// schema. Keys are lower-cased object names (matching omni's identity folding).
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
				// Keyed by lower-cased name to match omni's trigger identity folding.
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
// "8.0.32"); MySQL returns an error for a server older than 8.0, and other engines
// ignore it.
func SDLMigration(engine storepb.Engine, userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) (string, error) {
	sourceSDL, err := MetadataToSDL(engine, currentSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert current schema to SDL")
	}
	return diffSDLMigrationWithContext(engine, sourceSDL, userSDLText, engineVersion, buildSDLSessionContextMap(currentSchema))
}

// SDLDropAdvices analyzes the SDL migration for destructive operations and returns warnings.
// engineVersion is the target server's version string; MySQL reports a server older
// than 8.0 as an error advice, and other engines ignore it.
func SDLDropAdvices(engine storepb.Engine, userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error) {
	f := engines[engine].SDLDropAdvices
	if f == nil {
		return nil, notSupportedError(engine, "SDL drop advices")
	}
	return f(userSDLText, currentSchema, engineVersion)
}

// DiffMigration computes the migration SQL between two database metadata states.
// Engines may register DiffMetadataMigration to avoid round-tripping synced
// metadata through SDL text; otherwise engines with DiffSDLMigration convert
// both sides to SDL and diff.
func DiffMigration(engine storepb.Engine, oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	e := engines[engine]
	if e.DiffMetadataMigration != nil {
		return e.DiffMetadataMigration(oldSchema, newSchema)
	}
	if e.DiffSDLMigration != nil {
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
	return "", notSupportedError(engine, "")
}

// MetadataToSDL converts database metadata to SDL text using the registered Database definition renderer.
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

// DiffSDLMigration computes migration SQL between two SDL texts. engineVersion is the
// target server's version string: MySQL returns an error for a server older than 8.0,
// and other engines ignore it. Pass "" when no version is known.
func DiffSDLMigration(engine storepb.Engine, sourceSDL, targetSDL, engineVersion string) (string, error) {
	return diffSDLMigrationWithContext(engine, sourceSDL, targetSDL, engineVersion, nil)
}

func diffSDLMigrationWithContext(engine storepb.Engine, sourceSDL, targetSDL, engineVersion string, sessionCtx *SDLSessionContextMap) (string, error) {
	f := engines[engine].DiffSDLMigration
	if f == nil {
		return "", notSupportedError(engine, "SDL diff migration")
	}
	return f(sourceSDL, targetSDL, engineVersion, sessionCtx)
}

func GetMultiFileDatabaseDefinition(engine storepb.Engine, ctx GetDefinitionContext, metadata *metadatapb.DatabaseSchemaMetadata) (*MultiFileSchemaResult, error) {
	f := engines[engine].MultiFileDatabase
	if f == nil {
		return nil, notSupportedError(engine, "multi-file database definition")
	}
	return f(ctx, metadata)
}

func WalkThroughWithContext(ctx context.Context, engine storepb.Engine, wtCtx WalkThroughContext, d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	f := engines[engine].WalkThrough
	if f == nil {
		return nil
	}
	return f(ctx, wtCtx, d, ast)
}
