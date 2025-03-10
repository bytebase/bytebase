package schema

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux                    sync.Mutex
	getDesignSchemas       = make(map[storepb.Engine]getDesignSchema)
	checkColumnTypes       = make(map[storepb.Engine]checkColumnType)
	stringifyTables        = make(map[storepb.Engine]stringifyTable)
	getDatabaseDefinitions = make(map[storepb.Engine]getDatabaseDefinition)
)

type getDesignSchema func(*storepb.DatabaseSchemaMetadata) (string, error)
type checkColumnType func(string) bool
type stringifyTable func(*storepb.TableMetadata) (string, error)
type getDatabaseDefinition func(GetDefinitionContext, *storepb.DatabaseSchemaMetadata) (string, error)

type GetDefinitionContext struct {
	SkipBackupSchema bool
	PrintHeader      bool
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

func RegisterGetDesignSchema(engine storepb.Engine, f getDesignSchema) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getDesignSchemas[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getDesignSchemas[engine] = f
}

func GetDesignSchema(engine storepb.Engine, to *storepb.DatabaseSchemaMetadata) (string, error) {
	f, ok := getDesignSchemas[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(to)
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

func RegisterStringifyTable(engine storepb.Engine, f stringifyTable) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := stringifyTables[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	stringifyTables[engine] = f
}

func StringifyTable(engine storepb.Engine, table *storepb.TableMetadata) (string, error) {
	f, ok := stringifyTables[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(table)
}
