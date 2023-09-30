package base

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux                    sync.Mutex
	queryValidators        = make(map[storepb.Engine]ValidateSQLForEditorFunc)
	fieldMaskers           = make(map[storepb.Engine]GetMaskedFieldsFunc)
	changedResourcesGetter = make(map[storepb.Engine]ExtractChangedResourcesFunc)
	resourcesGetter        = make(map[storepb.Engine]ExtractResourceListFunc)
	splitter               = make(map[storepb.Engine]SplitMultiSQLFunc)
	databaseGetter         = make(map[storepb.Engine]ExtractDatabaseListFunc)
)

type ValidateSQLForEditorFunc func(string) bool
type GetMaskedFieldsFunc func(string, string, *db.SensitiveSchemaInfo) ([]db.SensitiveField, error)
type ExtractChangedResourcesFunc func(string, string, string) ([]SchemaResource, error)
type ExtractResourceListFunc func(string, string, string) ([]SchemaResource, error)
type SplitMultiSQLFunc func(string) ([]SingleSQL, error)
type ExtractDatabaseListFunc func(string, string) ([]string, error)

func RegisterQueryValidator(engine storepb.Engine, f ValidateSQLForEditorFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := queryValidators[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	queryValidators[engine] = f
}

// ValidateSQLForEditor validates the SQL statement for editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE
// 2. SELECT statement
// We also support CTE with SELECT statements, but not with DML statements.
func ValidateSQLForEditor(engine storepb.Engine, statement string) bool {
	f := queryValidators[engine]
	return f(statement)
}

func RegisterGetMaskedFieldsFunc(engine storepb.Engine, f GetMaskedFieldsFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := fieldMaskers[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	fieldMaskers[engine] = f
}

func ExtractSensitiveField(engine storepb.Engine, statement string, currentDatabase string, schemaInfo *db.SensitiveSchemaInfo) ([]db.SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	f, ok := fieldMaskers[engine]
	if !ok {
		return nil, errors.Errorf("engine type is not supported: %s", engine)
	}
	return f(statement, currentDatabase, schemaInfo)
}

func RegisterExtractChangedResourcesFunc(engine storepb.Engine, f ExtractChangedResourcesFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := changedResourcesGetter[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	changedResourcesGetter[engine] = f
}

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engine storepb.Engine, currentDatabase string, currentSchema string, sql string) ([]SchemaResource, error) {
	f, ok := changedResourcesGetter[engine]
	if !ok {
		return nil, errors.Errorf("engine type is not supported: %s", engine)
	}
	return f(currentDatabase, currentSchema, sql)
}

func RegisterExtractResourceListFunc(engine storepb.Engine, f ExtractResourceListFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := resourcesGetter[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	resourcesGetter[engine] = f
}

func ExtractResourceList(engine storepb.Engine, currentDatabase string, currentSchema string, sql string) ([]SchemaResource, error) {
	f, ok := resourcesGetter[engine]
	if !ok {
		return nil, errors.Errorf("engine type is not supported: %s", engine)
	}
	return f(currentDatabase, currentSchema, sql)
}

func RegisterSplitterFunc(engine storepb.Engine, f SplitMultiSQLFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := splitter[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	splitter[engine] = f
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engine storepb.Engine, statement string) ([]SingleSQL, error) {
	f, ok := splitter[engine]
	if !ok {
		return nil, errors.Errorf("engine type is not supported: %s", engine)
	}
	return f(statement)
}

func RegisterExtractDatabaseListFunc(engine storepb.Engine, f ExtractDatabaseListFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := databaseGetter[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	databaseGetter[engine] = f
}

// ExtractDatabaseList extracts all databases from statement.
func ExtractDatabaseList(engine storepb.Engine, statement string, currentDatabase string) ([]string, error) {
	f, ok := databaseGetter[engine]
	if !ok {
		return nil, errors.Errorf("engine type is not supported: %s", engine)
	}
	return f(statement, currentDatabase)
}
