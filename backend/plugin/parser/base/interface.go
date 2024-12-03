package base

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	mux                     sync.Mutex
	queryValidators         = make(map[storepb.Engine]ValidateSQLForEditorFunc)
	changedResourcesGetters = make(map[storepb.Engine]ExtractChangedResourcesFunc)
	resourcesGetters        = make(map[storepb.Engine]ExtractResourceListFunc)
	splitters               = make(map[storepb.Engine]SplitMultiSQLFunc)
	schemaDiffers           = make(map[storepb.Engine]SchemaDiffFunc)
	completers              = make(map[storepb.Engine]CompletionFunc)
	diagnoseCollectors      = make(map[storepb.Engine]DiagnoseFunc)
	spans                   = make(map[storepb.Engine]GetQuerySpanFunc)
	transformDMLToSelect    = make(map[storepb.Engine]TransformDMLToSelectFunc)
	generateRestoreSQL      = make(map[storepb.Engine]GenerateRestoreSQLFunc)
)

type ValidateSQLForEditorFunc func(string) (bool, bool, error)
type ExtractChangedResourcesFunc func(string, string, *model.DBSchema, any, string) (*ChangeSummary, error)
type ExtractResourceListFunc func(string, string, string) ([]SchemaResource, error)
type SplitMultiSQLFunc func(string) ([]SingleSQL, error)
type SchemaDiffFunc func(ctx DiffContext, oldStmt, newStmt string) (string, error)
type CompletionFunc func(ctx context.Context, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error)
type DiagnoseFunc func(ctx context.Context, dCtx DiagnoseContext, statement string) ([]Diagnostic, error)

// GetQuerySpanFunc is the interface of getting the query span for a query.
type GetQuerySpanFunc func(ctx context.Context, gCtx GetQuerySpanContext, statement, database, schema string, ignoreCaseSensitive bool) (*QuerySpan, error)

// TransformDMLToSelectFunc is the interface of transforming DML statements to SELECT statements.
type TransformDMLToSelectFunc func(ctx context.Context, tCtx TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]BackupStatement, error)

type GenerateRestoreSQLFunc func(ctx context.Context, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error)

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
// The first bool indicates whether the query can run in read-only mode, and the second bool determines whether all queries return data.
func ValidateSQLForEditor(engine storepb.Engine, statement string) (bool, bool, error) {
	f, ok := queryValidators[engine]
	if !ok {
		return true, true, nil
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

func RegisterExtractChangedResourcesFunc(engine storepb.Engine, f ExtractChangedResourcesFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := changedResourcesGetters[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	changedResourcesGetters[engine] = f
}

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engine storepb.Engine, currentDatabase string, currentSchema string, dbSchema *model.DBSchema, ast any, statement string) (*ChangeSummary, error) {
	f, ok := changedResourcesGetters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentDatabase, currentSchema, dbSchema, ast, statement)
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

func SchemaDiff(engine storepb.Engine, ctx DiffContext, oldStmt, newStmt string) (string, error) {
	f, ok := schemaDiffers[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, oldStmt, newStmt)
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
func Completion(ctx context.Context, engine storepb.Engine, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error) {
	f, ok := completers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, cCtx, statement, caretLine, caretOffset)
}

// RegisterDiagnoseFunc registers the diagnose function for the engine.
func RegisterDiagnoseFunc(engine storepb.Engine, f DiagnoseFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := diagnoseCollectors[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	diagnoseCollectors[engine] = f
}

// Diagnose returns the diagnostics for the statement. The diagnostics never be nil, and may not be empty although the error is not nil.
func Diagnose(ctx context.Context, dCtx DiagnoseContext, engine storepb.Engine, statement string) ([]Diagnostic, error) {
	f, ok := diagnoseCollectors[engine]
	if !ok {
		return []Diagnostic{}, nil
	}
	return f(ctx, dCtx, statement)
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
// The interface will return the query spans with non-critical errors, or return an error if the query is invalid.
func GetQuerySpan(ctx context.Context, gCtx GetQuerySpanContext, engine storepb.Engine, statement, database, schema string, ignoreCaseSensitive bool) ([]*QuerySpan, error) {
	f, ok := spans[engine]
	if !ok {
		return nil, nil
	}
	gCtx.Engine = engine
	statements, err := SplitMultiSQL(engine, statement)
	if err != nil {
		return nil, err
	}
	gCtx.TempTables = make(map[string]*PhysicalTable)
	var results []*QuerySpan
	var nonEmptyStatement []string
	for _, stmt := range statements {
		if stmt.Empty {
			continue
		}
		nonEmptyStatement = append(nonEmptyStatement, stmt.Text)
		result, err := f(ctx, gCtx, stmt.Text, database, schema, ignoreCaseSensitive)
		if err != nil {
			// Try to unwrap the error to see if it's a ResourceNotFoundError to decrease the error noise.
			// TODO(d): remove resource not found error checks.
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
	if engine == storepb.Engine_MSSQL {
		TSQLRecognizeExplainType(results, nonEmptyStatement)
	}
	return results, nil
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
func TransformDMLToSelect(ctx context.Context, engine storepb.Engine, tCtx TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]BackupStatement, error) {
	f, ok := transformDMLToSelect[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, tCtx, statement, sourceDatabase, targetDatabase, tablePrefix)
}

func RegisterGenerateRestoreSQL(engine storepb.Engine, f GenerateRestoreSQLFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := generateRestoreSQL[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	generateRestoreSQL[engine] = f
}

func GenerateRestoreSQL(ctx context.Context, engine storepb.Engine, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	f, ok := generateRestoreSQL[engine]
	if !ok {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, rCtx, statement, backupItem)
}

type ChangeSummary struct {
	ChangedResources *model.ChangedResources
	SampleDMLS       []string
	DMLCount         int
	InsertCount      int
}

type Range struct {
	Start int32
	End   int32
}

// NewRange creates a new Range with index range of singleSQL in statement.
func NewRange(statement, singleSQL string) *storepb.Range {
	statementBytes := []byte(statement)
	singleSQLBytes := []byte(singleSQL)
	start := bytes.Index(statementBytes, singleSQLBytes)
	return &storepb.Range{
		Start: int32(start),
		End:   int32(start + len(singleSQLBytes)),
	}
}

// REFACTOR(zp): Put it here to avoid circular import for now.
var (
	showplanReg = regexp.MustCompile(`(?mi)^\s*SET\s+SHOWPLAN_(ALL|XML|TEXT)\s+(?P<status>(ON|OFF))\s*;?$`)
)

// TSQLRecognizeExplainType walks the spans, and rewrite the select type to explain type if previous statement is SET SHOWPLAN_ALL.
func TSQLRecognizeExplainType(spans []*QuerySpan, stmt []string) {
	if len(spans) != len(stmt) {
		return
	}
	on := false
	for i := range spans {
		matches := showplanReg.FindStringSubmatch(stmt[i])
		if matches != nil {
			for k, name := range showplanReg.SubexpNames() {
				if k != 0 && name == "status" {
					switch strings.ToLower(matches[k]) {
					case "on":
						on = true
					case "off":
						on = false
					}
				}
			}
		}
		if on {
			if matches != nil {
				spans[i].Type = Explain
			} else if spans[i].Type == Select {
				spans[i].Type = Explain
			}
		} else {
			// SET SHOW_PLANALL OFF, this statement is explain either.
			if matches != nil {
				spans[i].Type = Explain
			}
		}
	}
}
