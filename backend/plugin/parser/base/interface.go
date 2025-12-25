package base

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"github.com/zeebo/xxh3"

	lsp "github.com/bytebase/lsp-protocol"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

type isAllDMLCacheKey struct {
	engine storepb.Engine
	hash   uint64
}

type isAllDMLResult struct {
	sync.Mutex
	computed bool
	value    bool
}

var (
	isAllDMLCacheMu sync.Mutex
	isAllDMLCache   = func() *lru.Cache[isAllDMLCacheKey, *isAllDMLResult] {
		cache, err := lru.New[isAllDMLCacheKey, *isAllDMLResult](1024)
		if err != nil {
			panic(err)
		}
		return cache
	}()
)

var (
	mux                     sync.Mutex
	queryValidators         = make(map[storepb.Engine]ValidateSQLForEditorFunc)
	changedResourcesGetters = make(map[storepb.Engine]ExtractChangedResourcesFunc)
	splitters               = make(map[storepb.Engine]SplitMultiSQLFunc)
	completers              = make(map[storepb.Engine]CompletionFunc)
	diagnoseCollectors      = make(map[storepb.Engine]DiagnoseFunc)
	statementRanges         = make(map[storepb.Engine]StatementRangeFunc)
	spans                   = make(map[storepb.Engine]GetQuerySpanFunc)
	transformDMLToSelect    = make(map[storepb.Engine]TransformDMLToSelectFunc)
	generateRestoreSQL      = make(map[storepb.Engine]GenerateRestoreSQLFunc)
	parsers                 = make(map[storepb.Engine]ParseFunc)
	statementParsers        = make(map[storepb.Engine]ParseStatementsFunc)
	statementTypeGetters    = make(map[storepb.Engine]GetStatementTypesFunc)
)

type ValidateSQLForEditorFunc func(string) (bool, bool, error)
type ExtractChangedResourcesFunc func(string, string, *model.DatabaseMetadata, []AST, string) (*ChangeSummary, error)
type SplitMultiSQLFunc func(string) ([]Statement, error)
type CompletionFunc func(ctx context.Context, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error)
type DiagnoseFunc func(ctx context.Context, dCtx DiagnoseContext, statement string) ([]Diagnostic, error)
type StatementRangeFunc func(ctx context.Context, sCtx StatementRangeContext, statement string) ([]Range, error)

// GetQuerySpanFunc is the interface of getting the query span for a query.
type GetQuerySpanFunc func(ctx context.Context, gCtx GetQuerySpanContext, statement, database, schema string, ignoreCaseSensitive bool) (*QuerySpan, error)

// TransformDMLToSelectFunc is the interface of transforming DML statements to SELECT statements.
type TransformDMLToSelectFunc func(ctx context.Context, tCtx TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]BackupStatement, error)

type GenerateRestoreSQLFunc func(ctx context.Context, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error)

// ParseFunc is the interface for parsing SQL statements and returning []AST.
// Each parser package is responsible for creating AST instances with the appropriate data.
// Parser packages can return *ANTLRAST for ANTLR-based parsers or their own concrete types.
type ParseFunc func(statement string) ([]AST, error)

// ParseStatementsFunc is the interface for parsing SQL statements and returning []ParsedStatement.
// This is the new unified parsing function that returns complete ParsedStatement objects with AST.
type ParseStatementsFunc func(statement string) ([]ParsedStatement, error)

// GetStatementTypesFunc returns the types of statements in the ASTs.
// Statement types include: INSERT, UPDATE, DELETE (DML), CREATE_TABLE, ALTER_TABLE, DROP_TABLE, etc. (DDL).
type GetStatementTypesFunc func([]AST) ([]string, error)

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

func RegisterExtractChangedResourcesFunc(engine storepb.Engine, f ExtractChangedResourcesFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := changedResourcesGetters[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	changedResourcesGetters[engine] = f
}

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engine storepb.Engine, currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []AST, statement string) (*ChangeSummary, error) {
	f, ok := changedResourcesGetters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentDatabase, currentSchema, dbMetadata, asts, statement)
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
func SplitMultiSQL(engine storepb.Engine, statement string) ([]Statement, error) {
	f, ok := splitters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement)
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

func RegisterStatementRangesFunc(engine storepb.Engine, f StatementRangeFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := statementRanges[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	statementRanges[engine] = f
}

// GetStatementRanges returns a list of ranges for the statement.
// Start is inclusive and end is exclusive. Character is 0-based UTF-16 code unit offset, Line is 0-based line number.
func GetStatementRanges(ctx context.Context, sCtx StatementRangeContext, engine storepb.Engine, statement string) ([]Range, error) {
	f, ok := statementRanges[engine]
	if !ok {
		return []lsp.Range{}, nil
	}
	return f(ctx, sCtx, statement)
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
			var resourceNotFound *ResourceNotFoundError
			if errors.As(err, &resourceNotFound) {
				return nil, resourceNotFound
			}
			var typeNotSupported *TypeNotSupportedError
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

func RegisterParseFunc(engine storepb.Engine, f ParseFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := parsers[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	parsers[engine] = f
}

// Parse parses the SQL statement and returns an AST representation.
// Each parser is responsible for creating []AST instances directly.
func Parse(engine storepb.Engine, statement string) ([]AST, error) {
	f, ok := parsers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement)
}

func RegisterParseStatementsFunc(engine storepb.Engine, f ParseStatementsFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := statementParsers[engine]; dup {
		panic(fmt.Sprintf("RegisterParseStatementsFunc called twice %s", engine))
	}
	statementParsers[engine] = f
}

// ParseStatements parses the SQL statement and returns ParsedStatement objects with both text and AST.
// This is the new unified parsing function.
func ParseStatements(engine storepb.Engine, statement string) ([]ParsedStatement, error) {
	f, ok := statementParsers[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported for ParseStatements", engine)
	}
	return f(statement)
}

func RegisterGetStatementTypes(engine storepb.Engine, f GetStatementTypesFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, dup := statementTypeGetters[engine]; dup {
		panic(fmt.Sprintf("Register called twice %s", engine))
	}
	statementTypeGetters[engine] = f
}

// GetStatementTypes returns the types of statements in the ASTs.
func GetStatementTypes(engine storepb.Engine, asts []AST) ([]string, error) {
	f, ok := statementTypeGetters[engine]
	if !ok {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(asts)
}

// IsAllDML checks if all statements are DML (INSERT, UPDATE, DELETE).
// Returns false for unsupported engines or parse errors (conservative approach).
// Results are cached to avoid repeated parsing of the same statement.
// Safe for concurrent calls with the same statement.
func IsAllDML(engine storepb.Engine, statement string) bool {
	key := isAllDMLCacheKey{engine: engine, hash: xxh3.HashString(statement)}

	isAllDMLCacheMu.Lock()
	result, ok := isAllDMLCache.Get(key)
	if !ok {
		result = &isAllDMLResult{}
		isAllDMLCache.Add(key, result)
	}
	isAllDMLCacheMu.Unlock()

	result.Lock()
	defer result.Unlock()
	if result.computed {
		return result.value
	}
	result.value = isAllDMLImpl(engine, statement)
	result.computed = true
	return result.value
}

func isAllDMLImpl(engine storepb.Engine, statement string) bool {
	asts, err := Parse(engine, statement)
	if err != nil {
		return false
	}
	if len(asts) == 0 {
		return false
	}
	types, err := GetStatementTypes(engine, asts)
	if err != nil {
		return false
	}
	if len(types) == 0 {
		return false
	}
	for _, t := range types {
		switch t {
		case "INSERT", "UPDATE", "DELETE":
			// DML statement, continue
		default:
			return false
		}
	}
	return true
}

type ChangeSummary struct {
	ChangedResources *model.ChangedResources
	SampleDMLS       []string
	DMLCount         int
	InsertCount      int
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
					default:
						// Unknown status, keep default value
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

type StatementRangeContext struct{}

type Range = lsp.Range
