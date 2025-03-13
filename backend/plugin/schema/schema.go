package schema

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux                            sync.Mutex
	checkColumnTypes               = make(map[storepb.Engine]checkColumnType)
	getDatabaseDefinitions         = make(map[storepb.Engine]getDatabaseDefinition)
	getSchemaDefinitions           = make(map[storepb.Engine]getSchemaDefinition)
	getTableDefinitions            = make(map[storepb.Engine]getTableDefinition)
	getViewDefinitions             = make(map[storepb.Engine]getViewDefinition)
	getMaterializedViewDefinitions = make(map[storepb.Engine]getMaterializedViewDefinition)
	getFunctionDefinitions         = make(map[storepb.Engine]getFunctionDefinition)
	getProcedureDefinitions        = make(map[storepb.Engine]getProcedureDefinition)
	getSequenceDefinitions         = make(map[storepb.Engine]getSequenceDefinition)
)

type checkColumnType func(string) bool
type getDatabaseDefinition func(GetDefinitionContext, *storepb.DatabaseSchemaMetadata) (string, error)
type getSchemaDefinition func(*storepb.SchemaMetadata) (string, error)
type getTableDefinition func(string, *storepb.TableMetadata, []*storepb.SequenceMetadata) (string, error)
type getViewDefinition func(string, *storepb.ViewMetadata) (string, error)
type getMaterializedViewDefinition func(string, *storepb.MaterializedViewMetadata) (string, error)
type getFunctionDefinition func(string, *storepb.FunctionMetadata) (string, error)
type getProcedureDefinition func(string, *storepb.ProcedureMetadata) (string, error)
type getSequenceDefinition func(string, *storepb.SequenceMetadata) (string, error)

type GetDefinitionContext struct {
	SkipBackupSchema bool
	PrintHeader      bool
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

func RegisterCheckColumnType(engine storepb.Engine, f checkColumnType) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := checkColumnTypes[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	checkColumnTypes[engine] = f
}

func CheckColumnType(engine storepb.Engine, tp string) bool {
	f, ok := checkColumnTypes[engine]
	if !ok {
		return false
	}
	return f(tp)
}
