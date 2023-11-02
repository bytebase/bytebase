package base

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux                     sync.Mutex
	queryValidators         = make(map[storepb.Engine]ValidateSQLForEditorFunc)
	fieldMaskers            = make(map[storepb.Engine]GetMaskedFieldsFunc)
	changedResourcesGetters = make(map[storepb.Engine]ExtractChangedResourcesFunc)
	resourcesGetters        = make(map[storepb.Engine]ExtractResourceListFunc)
	splitters               = make(map[storepb.Engine]SplitMultiSQLFunc)
	schemaDiffers           = make(map[storepb.Engine]SchemaDiffFunc)
	completers              = make(map[storepb.Engine]CompletionFunc)
)

type ValidateSQLForEditorFunc func(string) (bool, error)
type GetMaskedFieldsFunc func(string, string, *SensitiveSchemaInfo) ([]SensitiveField, error)
type ExtractChangedResourcesFunc func(string, string, string) ([]SchemaResource, error)
type ExtractResourceListFunc func(string, string, string) ([]SchemaResource, error)
type SplitMultiSQLFunc func(string) ([]SingleSQL, error)
type SchemaDiffFunc func(oldStmt, newStmt string, ignoreCaseSensitivity bool) (string, error)
type CompletionFunc func(statement string, caretLine int, caretOffset int) ([]Candidate, error)

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
func ValidateSQLForEditor(engine storepb.Engine, statement string) (bool, error) {
	f, ok := queryValidators[engine]
	if !ok {
		return false, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement)
}

func RegisterExtractResourceListFunc(engine storepb.Engine, f ExtractResourceListFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := resourcesGetters[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	resourcesGetters[engine] = f
}

func ExtractResourceList(engine storepb.Engine, currentDatabase string, currentSchema string, sql string) ([]SchemaResource, error) {
	f, ok := resourcesGetters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentDatabase, currentSchema, sql)
}

func RegisterGetMaskedFieldsFunc(engine storepb.Engine, f GetMaskedFieldsFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := fieldMaskers[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	fieldMaskers[engine] = f
}

func ExtractSensitiveField(engine storepb.Engine, statement string, currentDatabase string, schemaInfo *SensitiveSchemaInfo) ([]SensitiveField, error) {
	if schemaInfo == nil {
		return nil, nil
	}

	f, ok := fieldMaskers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement, currentDatabase, schemaInfo)
}

func RegisterExtractChangedResourcesFunc(engine storepb.Engine, f ExtractChangedResourcesFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := changedResourcesGetters[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	changedResourcesGetters[engine] = f
}

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engine storepb.Engine, currentDatabase string, currentSchema string, sql string) ([]SchemaResource, error) {
	f, ok := changedResourcesGetters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentDatabase, currentSchema, sql)
}

func RegisterSplitterFunc(engine storepb.Engine, f SplitMultiSQLFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := splitters[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	splitters[engine] = f
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engine storepb.Engine, statement string) ([]SingleSQL, error) {
	f, ok := splitters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement)
}

func RegisterSchemaDiffFunc(engine storepb.Engine, f SchemaDiffFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := schemaDiffers[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	schemaDiffers[engine] = f
}

func SchemaDiff(engine storepb.Engine, oldStmt, newStmt string, ignoreCaseSensitivity bool) (string, error) {
	f, ok := schemaDiffers[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(oldStmt, newStmt, ignoreCaseSensitivity)
}

// RegisterCompleteFunc registers the completion function for the engine.
func RegisterCompleteFunc(engine storepb.Engine, f CompletionFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := completers[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	completers[engine] = f
}

// Completion returns the completion candidates for the statement.
func Completion(engine storepb.Engine, statement string, caretLine int, caretOffset int) ([]Candidate, error) {
	f, ok := completers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement, caretLine, caretOffset)
}
