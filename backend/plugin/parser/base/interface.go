package base

import (
	"context"
	"fmt"
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
	registryMu sync.Mutex
	engines    = make(map[storepb.Engine]Engine)
)

type ValidateSQLForEditorFunc func(string) (bool, bool, error)
type ExtractChangedResourcesFunc func(string, string, *model.DatabaseMetadata, []AST, string) (*ChangeSummary, error)
type SplitMultiSQLFunc func(string) ([]Statement, error)
type CompletionFunc func(ctx context.Context, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error)
type DiagnoseFunc func(ctx context.Context, dCtx DiagnoseContext, statement string) ([]Diagnostic, error)
type StatementRangeFunc func(ctx context.Context, sCtx StatementRangeContext, statement string) ([]Range, error)

// GetQuerySpanFunc is the interface of getting the query span for a single statement.
// It accepts a pre-split Statement instead of raw string to avoid redundant splitting.
type GetQuerySpanFunc func(ctx context.Context, gCtx GetQuerySpanContext, stmt Statement, database, schema string, ignoreCaseSensitive bool) (*QuerySpan, error)

// TransformDMLToSelectFunc is the interface of transforming DML statements to SELECT statements.
type TransformDMLToSelectFunc func(ctx context.Context, tCtx TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]BackupStatement, error)

type GenerateRestoreSQLFunc func(ctx context.Context, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error)

// ParseStatementsFunc is the interface for parsing SQL statements and returning []ParsedStatement.
type ParseStatementsFunc func(statement string) ([]ParsedStatement, error)

// GetStatementTypesFunc returns the types of statements in the ASTs.
// Statement types include: INSERT, UPDATE, DELETE (DML), CREATE_TABLE, ALTER_TABLE, DROP_TABLE, etc. (DDL).
type GetStatementTypesFunc func([]AST) ([]storepb.StatementType, error)

// Engine defines the parser capabilities an engine plugin registers.
// Any nil field is omitted. Separate Register calls may supply disjoint
// capabilities for the same engine (for example, googlesql registers shared
// capabilities while bigquery and spanner add ResultLimit).
type Engine struct {
	Split                SplitMultiSQLFunc
	Parse                ParseStatementsFunc
	StatementTypes       GetStatementTypesFunc
	StatementRanges      StatementRangeFunc
	Diagnose             DiagnoseFunc
	Complete             CompletionFunc
	ValidateQuery        ValidateSQLForEditorFunc
	QuerySpan            GetQuerySpanFunc
	Explain              ExplainStatementFunc
	ResultLimit          ResultLimitFunc
	ExtractChanged       ExtractChangedResourcesFunc
	TransformDMLToSelect TransformDMLToSelectFunc
	GenerateRestoreSQL   GenerateRestoreSQLFunc
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
			panic(fmt.Sprintf("parser/base: %s registered twice for %s", name, engine))
		}
	}
	if spec.Split != nil {
		checkDup(cur.Split != nil, "Split")
		cur.Split = spec.Split
	}
	if spec.Parse != nil {
		checkDup(cur.Parse != nil, "Parse")
		cur.Parse = spec.Parse
	}
	if spec.StatementTypes != nil {
		checkDup(cur.StatementTypes != nil, "StatementTypes")
		cur.StatementTypes = spec.StatementTypes
	}
	if spec.StatementRanges != nil {
		checkDup(cur.StatementRanges != nil, "StatementRanges")
		cur.StatementRanges = spec.StatementRanges
	}
	if spec.Diagnose != nil {
		checkDup(cur.Diagnose != nil, "Diagnose")
		cur.Diagnose = spec.Diagnose
	}
	if spec.Complete != nil {
		checkDup(cur.Complete != nil, "Complete")
		cur.Complete = spec.Complete
	}
	if spec.ValidateQuery != nil {
		checkDup(cur.ValidateQuery != nil, "ValidateQuery")
		cur.ValidateQuery = spec.ValidateQuery
	}
	if spec.QuerySpan != nil {
		checkDup(cur.QuerySpan != nil, "QuerySpan")
		cur.QuerySpan = spec.QuerySpan
	}
	if spec.Explain != nil {
		checkDup(cur.Explain != nil, "Explain")
		cur.Explain = spec.Explain
	}
	if spec.ResultLimit != nil {
		checkDup(cur.ResultLimit != nil, "ResultLimit")
		cur.ResultLimit = spec.ResultLimit
	}
	if spec.ExtractChanged != nil {
		checkDup(cur.ExtractChanged != nil, "ExtractChanged")
		cur.ExtractChanged = spec.ExtractChanged
	}
	if spec.TransformDMLToSelect != nil {
		checkDup(cur.TransformDMLToSelect != nil, "TransformDMLToSelect")
		cur.TransformDMLToSelect = spec.TransformDMLToSelect
	}
	if spec.GenerateRestoreSQL != nil {
		checkDup(cur.GenerateRestoreSQL != nil, "GenerateRestoreSQL")
		cur.GenerateRestoreSQL = spec.GenerateRestoreSQL
	}
}

// ValidateSQLForEditor validates the SQL statement for editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE
// 2. SELECT statement
// We also support CTE with SELECT statements, but not with DML statements.
// The first bool indicates whether the query can run in read-only mode, and the second bool determines whether all queries return data.
//
// An engine with no registered validator answers "read-only, all queries
// return data". That default suits the callers who only route or format with
// it; a caller that must REFUSE a write has to ask HasQueryValidator first,
// because here the answer for such an engine is a default rather than a
// verdict.
func ValidateSQLForEditor(engine storepb.Engine, statement string) (bool, bool, error) {
	f := engines[engine].ValidateQuery
	if f == nil {
		return true, true, nil
	}
	return f(statement)
}

// HasQueryValidator reports whether a read-only query validator is registered
// for engine, so a caller can tell ValidateSQLForEditor's verdict apart from
// its no-validator default.
//
// Lock-free like every other lookup here: capabilities are registered from
// package init functions before any request runs.
func HasQueryValidator(engine storepb.Engine) bool {
	return engines[engine].ValidateQuery != nil
}

// ExtractChangedResources extracts the changed resources from the SQL.
func ExtractChangedResources(engine storepb.Engine, currentDatabase string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []AST, statement string) (*ChangeSummary, error) {
	f := engines[engine].ExtractChanged
	if f == nil {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(currentDatabase, currentSchema, dbMetadata, asts, statement)
}

// SplitMultiSQL splits statement into a slice of the single SQL.
func SplitMultiSQL(engine storepb.Engine, statement string) ([]Statement, error) {
	f := engines[engine].Split
	if f == nil {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(statement)
}

// Completion returns the completion candidates for the statement.
func Completion(ctx context.Context, engine storepb.Engine, cCtx CompletionContext, statement string, caretLine int, caretOffset int) ([]Candidate, error) {
	f := engines[engine].Complete
	if f == nil {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, cCtx, statement, caretLine, caretOffset)
}

// Diagnose returns the diagnostics for the statement. The diagnostics never be nil, and may not be empty although the error is not nil.
// A non-nil result matters: publishing diagnostics as JSON null instead of an empty array
// does not clear previously published diagnostics on the LSP client.
func Diagnose(ctx context.Context, dCtx DiagnoseContext, engine storepb.Engine, statement string) ([]Diagnostic, error) {
	f := engines[engine].Diagnose
	if f == nil {
		return []Diagnostic{}, nil
	}
	diagnostics, err := f(ctx, dCtx, statement)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	return diagnostics, err
}

// GetStatementRanges returns a list of ranges for the statement.
// Start is inclusive and end is exclusive. Character is 0-based UTF-16 code unit offset, Line is 0-based line number.
func GetStatementRanges(ctx context.Context, sCtx StatementRangeContext, engine storepb.Engine, statement string) ([]Range, error) {
	f := engines[engine].StatementRanges
	if f == nil {
		return []lsp.Range{}, nil
	}
	return f(ctx, sCtx, statement)
}

// GetQuerySpan gets the span of a query from pre-split statements.
// The interface will return the query spans with non-critical errors, or return an error if the query is invalid.
// Callers should split the SQL first using SplitMultiSQL and pass the resulting []Statement.
func GetQuerySpan(ctx context.Context, gCtx GetQuerySpanContext, engine storepb.Engine, statements []Statement, database, schema string, ignoreCaseSensitive bool) ([]*QuerySpan, error) {
	f := engines[engine].QuerySpan
	if f == nil {
		return nil, nil
	}
	gCtx.Engine = engine
	gCtx.TempTables = make(map[string]*PhysicalTable)
	gCtx.PlanOutput = make(PlanOutput)
	var results []*QuerySpan
	for _, stmt := range statements {
		if stmt.Empty {
			continue
		}
		result, err := f(ctx, gCtx, stmt, database, schema, ignoreCaseSensitive)
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
	return results, nil
}

// TransformDMLToSelect transforms the DML statement to SELECT statement.
func TransformDMLToSelect(ctx context.Context, engine storepb.Engine, tCtx TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]BackupStatement, error) {
	f := engines[engine].TransformDMLToSelect
	if f == nil {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, tCtx, statement, sourceDatabase, targetDatabase, tablePrefix)
}

func GenerateRestoreSQL(ctx context.Context, engine storepb.Engine, rCtx RestoreContext, statement string, backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	f := engines[engine].GenerateRestoreSQL
	if f == nil {
		return "", errors.Errorf("engine %s is not supported", engine)
	}
	return f(ctx, rCtx, statement, backupItem)
}

// ParseStatements parses the SQL statement and returns ParsedStatement objects with both text and AST.
func ParseStatements(engine storepb.Engine, statement string) ([]ParsedStatement, error) {
	f := engines[engine].Parse
	if f == nil {
		return nil, errors.Errorf("engine %s is not supported for ParseStatements", engine)
	}
	return f(statement)
}

// GetStatementTypes returns the types of statements in the ASTs.
func GetStatementTypes(engine storepb.Engine, asts []AST) ([]storepb.StatementType, error) {
	f := engines[engine].StatementTypes
	if f == nil {
		return nil, errors.Errorf("engine %s is not supported", engine)
	}
	return f(asts)
}

// IsAllDML checks if all statements are DML (INSERT, UPDATE, DELETE, MERGE).
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
	stmts, err := ParseStatements(engine, statement)
	if err != nil {
		return false
	}
	asts := ExtractASTs(stmts)
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
		case storepb.StatementType_INSERT, storepb.StatementType_UPDATE, storepb.StatementType_DELETE, storepb.StatementType_MERGE:
			// DML statement, continue
		default:
			return false
		}
	}
	return true
}

type ChangeSummary struct {
	ChangedResources *model.ChangedResources
	// DMLStatements holds the text of every DML statement whose rows can be estimated, in order.
	// A statement that runs after the sheet changes the search path starts with WithSearchPath's
	// prefix, which the driver splits off and runs before the EXPLAIN.
	DMLStatements []string
	// DMLCount counts the DML statements, including those DMLStatements omits.
	DMLCount int
	// InsertCount counts the rows of INSERT ... VALUES statements.
	InsertCount int
	// SkippedStatements counts the statements the extractor could not read, whose
	// changes ChangedResources misses.
	SkippedStatements int
}

// searchPathPrefix starts the statement WithSearchPath puts before a DML statement.
const searchPathPrefix = "SET LOCAL search_path TO "

// WithSearchPath returns statement preceded by a SET LOCAL search_path statement for schemas.
func WithSearchPath(statement string, schemas []string) string {
	quoted := make([]string, len(schemas))
	for i, schema := range schemas {
		quoted[i] = `"` + strings.ReplaceAll(schema, `"`, `""`) + `"`
	}
	return searchPathPrefix + strings.Join(quoted, ", ") + ";\n" + statement
}

type StatementRangeContext struct{}

type Range = lsp.Range
