package base

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"
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
	spans                   = make(map[storepb.Engine]GetQuerySpanFunc)
	affectedRows            = make(map[storepb.Engine]GetAffectedRowsFunc)
	transformDMLToSelect    = make(map[storepb.Engine]TransformDMLToSelectFunc)
)

type ValidateSQLForEditorFunc func(string) (bool, error)
type GetMaskedFieldsFunc func(string, string, *SensitiveSchemaInfo) ([]SensitiveField, error)
type ExtractChangedResourcesFunc func(string, string, string) ([]SchemaResource, error)
type ExtractResourceListFunc func(string, string, string) ([]SchemaResource, error)
type SplitMultiSQLFunc func(string) ([]SingleSQL, error)
type SchemaDiffFunc func(oldStmt, newStmt string, ignoreCaseSensitivity bool) (string, error)
type CompletionFunc func(ctx context.Context, statement string, caretLine int, caretOffset int, defaultDatabase string, metadata GetDatabaseMetadataFunc) ([]Candidate, error)

// GetQuerySpanFunc is the interface of getting the query span for a query.
type GetQuerySpanFunc func(ctx context.Context, statement, database string, metadataFunc GetDatabaseMetadataFunc) (*QuerySpan, error)

// GetAffectedRows is the interface of getting the affected rows for a statement.
type GetAffectedRowsFunc func(ctx context.Context, stmt any, getAffectedRowsByQuery GetAffectedRowsCountByQueryFunc, getTableDataSizeFunc GetTableDataSizeFunc) (int64, error)

// TransformDMLToSelectFunc is the interface of transforming DML statements to SELECT statements.
type TransformDMLToSelectFunc func(statement string, sourceDatabase string, targetDatabase string, tableSuffix string) ([]RollbackStatement, error)

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
func Completion(ctx context.Context, engine storepb.Engine, statement string, caretLine int, caretOffset int, defaultDatabase string, metadata GetDatabaseMetadataFunc) ([]Candidate, error) {
	f, ok := completers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, statement, caretLine, caretOffset, defaultDatabase, metadata)
}

func RegisterGetQuerySpan(engine storepb.Engine, f GetQuerySpanFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := spans[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	spans[engine] = f
}

// GetQuerySpan gets the span of a query.
func GetQuerySpan(ctx context.Context, engine storepb.Engine, statement, database string, getMetadataFunc GetDatabaseMetadataFunc) ([]*QuerySpan, error) {
	f, ok := spans[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	statements, err := SplitMultiSQL(engine, statement)
	if err != nil {
		return nil, err
	}
	var results []*QuerySpan
	for _, stmt := range statements {
		if stmt.Empty {
			continue
		}
		result, err := f(ctx, stmt.Text, database, getMetadataFunc)
		if err != nil {
			// Try to unwrap the error to see if it's a ResourceNotFoundError to decrease the error noise.
			var resourceNotFound *parsererror.ResourceNotFoundError
			if errors.As(err, &resourceNotFound) {
				return nil, resourceNotFound
			}
			var typeNotSupported *parsererror.TypeNotSupportedError
			if errors.As(err, &typeNotSupported) {
				return nil, typeNotSupported
			}
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// RegisterGetAffectedRows registers the getAffectedRows function for the engine.
func RegisterGetAffectedRows(engine storepb.Engine, f GetAffectedRowsFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := affectedRows[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	affectedRows[engine] = f
}

// GetAffectedRows returns the affected rows for the parse result.
func GetAffectedRows(ctx context.Context, engine storepb.Engine, stmt any, getAffectedRowsByQueryFunc GetAffectedRowsCountByQueryFunc, getTableDataSizeFunc GetTableDataSizeFunc) (int64, error) {
	f, ok := affectedRows[engine]
	if !ok {
		return 0, errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, stmt, getAffectedRowsByQueryFunc, getTableDataSizeFunc)
}

// RegisterTransformDMLToSelect registers the transformDMLToSelect function for the engine.
func RegisterTransformDMLToSelect(engine storepb.Engine, f TransformDMLToSelectFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := transformDMLToSelect[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	transformDMLToSelect[engine] = f
}

// TransformDMLToSelect transforms the DML statement to SELECT statement.
func TransformDMLToSelect(engine storepb.Engine, statement string, sourceDatabase string, targetDatabase string, tableSuffix string) ([]RollbackStatement, error) {
	f, ok := transformDMLToSelect[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement, sourceDatabase, targetDatabase, tableSuffix)
}
