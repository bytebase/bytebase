package schema

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux              sync.Mutex
	getDesignSchemas = make(map[storepb.Engine]getDesignSchema)
	parseToMetadatas = make(map[storepb.Engine]parseToMetadata)
	checkColumnTypes = make(map[storepb.Engine]checkColumnType)
	stringifyTables  = make(map[storepb.Engine]stringifyTable)
)

type getDesignSchema func(string, *storepb.DatabaseSchemaMetadata) (string, error)
type parseToMetadata func(string, string) (*storepb.DatabaseSchemaMetadata, error)
type checkColumnType func(string) bool
type stringifyTable func(*storepb.TableMetadata) (string, error)

func RegisterGetDesignSchema(engine storepb.Engine, f getDesignSchema) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := getDesignSchemas[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	getDesignSchemas[engine] = f
}

func GetDesignSchema(engine storepb.Engine, defaultSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	f, ok := getDesignSchemas[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(defaultSchema, to)
}

func RegisterParseToMetadatas(engine storepb.Engine, f parseToMetadata) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := parseToMetadatas[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	parseToMetadatas[engine] = f
}

func ParseToMetadata(engine storepb.Engine, defaultSchemaName, schema string) (*storepb.DatabaseSchemaMetadata, error) {
	f, ok := parseToMetadatas[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(defaultSchemaName, schema)
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
