package schema

import (
	"fmt"
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
	getMultiFileDatabaseDefinitions = make(map[storepb.Engine]getMultiFileDatabaseDefinition)
	walkThroughs                    = make(map[storepb.Engine]walkThrough)
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
type getSDLDiff func(currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseMetadata) (*MetadataDiff, error)
type walkThrough func(*model.DatabaseMetadata, []base.AST) *storepb.Advice

type GetDefinitionContext struct {
	SkipBackupSchema bool
	PrintHeader      bool
	SDLFormat        bool
	// MultiFileFormat indicates whether to generate multi-file SDL output.
	// When true, the result should be organized as multiple files.
	MultiFileFormat bool
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

func GetSDLDiff(engine storepb.Engine, currentSDLText, previousUserSDLText string, currentSchema, previousSchema *model.DatabaseMetadata) (*MetadataDiff, error) {
	f, ok := getSDLDiffs[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentSDLText, previousUserSDLText, currentSchema, previousSchema)
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

func WalkThrough(engine storepb.Engine, d *model.DatabaseMetadata, ast []base.AST) *storepb.Advice {
	f, ok := walkThroughs[engine]
	if !ok {
		return nil
	}
	return f(d, ast)
}
