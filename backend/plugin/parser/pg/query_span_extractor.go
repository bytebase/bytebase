package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	pgUnknownFieldName     = "?column?"
	generateSeries         = "generate_series"
	generateSubscripts     = "generate_subscripts"
	unnest                 = "unnest"
	jsonbEach              = "jsonb_each"
	jsonEach               = "json_each"
	jsonbEachText          = "jsonb_each_text"
	jsonEachText           = "json_each_text"
	jsonPopulateRecord     = "json_populate_record"
	jsonbPopulateRecord    = "jsonb_populate_record"
	jsonPopulateRecordset  = "json_populate_recordset"
	jsonbPopulateRecordset = "jsonb_populate_recordset"
	jsonToRecord           = "json_to_record"
	jsonbToRecord          = "jsonb_to_record"
	jsonToRecordset        = "json_to_recordset"
	jsonbToRecordset       = "jsonb_to_recordset"
)

// querySpanExtractor is the extractor to extract the query span from the given pgquery.RawStmt.
type querySpanExtractor struct {
	*postgresql.BasePostgreSQLParserListener
	ctx             context.Context
	defaultDatabase string
	searchPath      []string
	// The metaCache serves as a lazy-load cache for the database metadata and should not be accessed directly.
	// Instead, use querySpanExtractor.getDatabaseMetadata to access it.
	metaCache map[string]*model.DatabaseMetadata

	gCtx base.GetQuerySpanContext

	// Private fields.

	ctes []*base.PseudoTable

	// outerTableSource is the list of table sources from the outer query,
	// it's used to resolve the column name in the correlated subquery.
	outerTableSources []base.TableSource

	// tableSourcesFrom is the list of table sources from the FROM clause.
	tableSourcesFrom []base.TableSource

	// sourceColumnsInFunction is the source columns in the function.
	// It's used to resolve defined functions body as a table source.
	sourceColumnsInFunction base.SourceColumnSet

	// variables are variables declared in the function.
	variables map[string]*base.QuerySpanResult

	err error

	// resultTableSource is used to store the extracted table source from ANTLR parsing
	resultTableSource base.TableSource
}

// newQuerySpanExtractor creates a new query span extractor, the databaseMetadata and the ast are in the read guard.
func newQuerySpanExtractor(defaultDatabase string, searchPath []string, gCtx base.GetQuerySpanContext) *querySpanExtractor {
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}
	return &querySpanExtractor{
		defaultDatabase:         defaultDatabase,
		searchPath:              searchPath,
		metaCache:               make(map[string]*model.DatabaseMetadata),
		gCtx:                    gCtx,
		sourceColumnsInFunction: make(base.SourceColumnSet),
	}
}

func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, database)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}
	q.metaCache[database] = meta
	return meta, nil
}

// Add this field to querySpanExtractor struct (defined in query_span_extractor.go):
// resultTableSource base.TableSource  // Stores the extracted table source from EnterStmtmulti

// getQuerySpan is the ANTLR-based implementation of query span extraction.
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// Parse the statement using ANTLR
	parseResults, err := ParsePostgreSQL(stmt)
	if err != nil {
		return nil, err
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(parseResults))
	}

	parseResult := parseResults[0]

	// Walk the tree to extract access tables

	accessTableExtractor := &accessTableExtractor{
		defaultDatabase:     q.defaultDatabase,
		searchPath:          q.searchPath,
		getDatabaseMetadata: q.gCtx.GetDatabaseMetadataFunc,
		ctx:                 ctx,
		instanceID:          q.gCtx.InstanceID,
	}
	antlr.ParseTreeWalkerDefault.Walk(accessTableExtractor, parseResult.Tree)
	if accessTableExtractor.err != nil {
		return nil, errors.Wrapf(accessTableExtractor.err, "failed to extract query span from statement: %s", stmt)
	}

	// Build access map
	accessesMap := make(base.SourceColumnSet)
	for _, resource := range accessTableExtractor.accessTables {
		accessesMap[resource] = true
	}

	// Check for mixed system/user tables
	allSystems, mixed := isMixedQuery(accessesMap)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	// Get query type using ANTLR-based detection
	queryTypeListener := &queryTypeListener{
		result:     base.QueryTypeUnknown,
		allSystems: allSystems,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, parseResult.Tree)
	if queryTypeListener.err != nil {
		return nil, errors.Wrapf(queryTypeListener.err, "failed to get query type from statement: %s", stmt)
	}

	queryType, isExplainAnalyze := queryTypeListener.result, queryTypeListener.isExplainAnalyze

	// For non-SELECT queries, return early
	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// For EXPLAIN ANALYZE SELECT
	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Extract table source using ANTLR (this was done in EnterStmtmulti)
	// The resultTableSource should have been populated by EnterStmtmulti
	if q.resultTableSource == nil {
		// If not populated, try to extract it now
		root, ok := parseResult.Tree.(*postgresql.RootContext)
		if !ok {
			return nil, errors.Errorf("failed to assert parse tree to RootContext")
		}
		if root.Stmtblock() != nil && root.Stmtblock().Stmtmulti() != nil {
			stmtmulti := root.Stmtblock().Stmtmulti()
			if len(stmtmulti.AllStmt()) == 1 {
				tableSource, err := q.extractTableSourceFromStmt(stmtmulti.AllStmt()[0])
				if err != nil {
					var functionNotSupported *base.FunctionNotSupportedError
					if errors.As(err, &functionNotSupported) {
						if len(accessesMap) == 0 {
							accessesMap[base.ColumnResource{
								Database: q.defaultDatabase,
							}] = true
						}
						return &base.QuerySpan{
							Type:          base.Select,
							SourceColumns: accessesMap,
							Results:       []base.QuerySpanResult{},
							NotFoundError: functionNotSupported,
						}, nil
					}
					var resourceNotFound *base.ResourceNotFoundError
					if errors.As(err, &resourceNotFound) {
						if len(accessesMap) == 0 {
							accessesMap[base.ColumnResource{
								Database: q.defaultDatabase,
							}] = true
						}
						return &base.QuerySpan{
							Type:          base.Select,
							SourceColumns: accessesMap,
							Results:       []base.QuerySpanResult{},
							NotFoundError: resourceNotFound,
						}, nil
					}
					return nil, err
				}
				q.resultTableSource = tableSource
			}
		}
	}

	// Merge the source columns in functions to the access tables
	for source := range q.sourceColumnsInFunction {
		accessesMap[source] = true
	}

	// Build the final query span result
	var results []base.QuerySpanResult
	if q.resultTableSource != nil {
		results = q.resultTableSource.GetQuerySpanResult()
	}

	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       results,
	}, nil
}

type functionDefinition struct {
	schemaName string
	metadata   *storepb.FunctionMetadata
}

func (q *querySpanExtractor) findFunctionDefine(schemaName, funcName string, nArgs int) (base.TableSource, error) {
	dbMetadata, err := q.getDatabaseMetadata(q.defaultDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.defaultDatabase)
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &q.defaultDatabase,
		}
	}
	var funcs []*functionDefinition
	searchPath := q.searchPath
	if schemaName != "" {
		searchPath = []string{schemaName}
	}
	schemas, functions := dbMetadata.SearchFunctions(searchPath, funcName)
	for i, fun := range functions {
		funcs = append(funcs, &functionDefinition{
			schemaName: schemas[i],
			metadata:   fun,
		})
	}

	candidates, err := getFunctionCandidates(funcs, nArgs, funcName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get function candidates")
	}
	if len(candidates) == 0 {
		return nil, &base.ResourceNotFoundError{
			Database: &q.defaultDatabase,
			Schema:   &schemaName,
			Function: &funcName,
		}
	}

	if len(candidates) > 1 && candidates[0].schemaName == candidates[1].schemaName {
		return nil, errors.Errorf("ambiguous function call: %s", funcName)
	}

	function := candidates[0]
	functionName := fmt.Sprintf("%s.%s", function.schemaName, funcName)
	columns, err := q.getColumnsFromFunction(functionName, function.metadata.Definition)
	if err != nil {
		return nil, &base.FunctionNotSupportedError{
			Err:      err,
			Function: functionName,
		}
	}
	return &base.PseudoTable{
		Columns: columns,
	}, nil
}

type functionDefinitionDetail struct {
	nDefaultParam  int
	nVariadicParam int
	nParam         int
	function       *functionDefinition
}

func buildFunctionDefinitionDetail(funcDef *functionDefinition) (*functionDefinitionDetail, error) {
	function := funcDef.metadata
	definition := function.GetDefinition()
	parseResults, err := ParsePostgreSQL(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse function definition: %s", definition)
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement in function definition, got %d", len(parseResults))
	}

	// Navigate: root -> stmtblock -> stmt -> createfunctionstmt
	root, ok := parseResults[0].Tree.(*postgresql.RootContext)
	if !ok {
		return nil, errors.Errorf("expecting RootContext but got %T", parseResults[0].Tree)
	}

	stmtblock := root.Stmtblock()
	if stmtblock == nil {
		return nil, errors.Errorf("expecting stmtblock but got nil")
	}

	stmtmulti := stmtblock.Stmtmulti()
	if stmtmulti == nil {
		return nil, errors.Errorf("expecting stmtmulti but got nil")
	}

	stmts := stmtmulti.AllStmt()
	if len(stmts) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(stmts))
	}

	stmt := stmts[0]
	createFuncStmt := stmt.Createfunctionstmt()
	if createFuncStmt == nil {
		return nil, errors.Errorf("expecting Createfunctionstmt but got nil")
	}

	funcArgsWithDefaults := createFuncStmt.Func_args_with_defaults()
	var params []postgresql.IFunc_arg_with_defaultContext
	if l := funcArgsWithDefaults.Func_args_with_defaults_list(); l != nil {
		params = append(params, l.AllFunc_arg_with_default()...)
	}

	var nDefaultPram, nVariadicParam int
	for _, param := range params {
		if param.A_expr() != nil {
			nDefaultPram++
		}

		if c := param.Func_arg().Arg_class(); c != nil {
			if c.VARIADIC() != nil {
				nVariadicParam++
			}
		}
	}

	return &functionDefinitionDetail{
		nDefaultParam:  nDefaultPram,
		nVariadicParam: nVariadicParam,
		nParam:         len(params),
		function:       funcDef,
	}, nil
}

// getFunctionCandidates returns the function candidates for the function call.
func getFunctionCandidates(functions []*functionDefinition, nArgs int, funcName string) ([]*functionDefinition, error) {
	// Filter by name only.
	var nameFiltered []*functionDefinition
	for _, function := range functions {
		if function.metadata.GetName() != funcName {
			continue
		}
		nameFiltered = append(nameFiltered, function)
	}

	if len(nameFiltered) == 0 {
		return nil, nil
	}
	// If there is only one function with the same name, we return it directly,
	// PostgreSQL would throw an error if the argument does not match the function signature.
	if len(nameFiltered) == 1 {
		return nameFiltered, nil
	}

	detail := make([]*functionDefinitionDetail, 0, len(nameFiltered))
	for _, function := range nameFiltered {
		d, err := buildFunctionDefinitionDetail(function)
		if err != nil {
			return nil, err
		}
		if d == nil {
			continue
		}
		detail = append(detail, d)
	}

	var candidates []*functionDefinition
	for _, d := range detail {
		// If there are no default and variadic parameters, the number of arguments must match the number of parameters.
		if d.nDefaultParam == 0 && d.nVariadicParam == 0 {
			if nArgs == d.nParam {
				candidates = append(candidates, d.function)
			}
			continue
		}

		// Default parameter matches 0 or 1 argument, and variadic parameter matches 0 or more arguments.
		lbound := d.nParam - d.nDefaultParam - d.nVariadicParam
		ubound := d.nParam
		if d.nVariadicParam > 0 && ubound < nArgs {
			// Hack to make variadic parameter match 0 or more arguments.
			ubound = nArgs
		}
		if nArgs >= lbound && nArgs <= ubound {
			candidates = append(candidates, d.function)
		}
	}

	return candidates, nil
}

type languageType int

const (
	languageTypeSQL languageType = iota
	languageTypePLPGSQL
)

func (q *querySpanExtractor) getColumnsFromFunction(name, definition string) ([]base.QuerySpanResult, error) {
	parseResults, err := ParsePostgreSQL(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse function definition: %s", definition)
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement in function definition, got %d", len(parseResults))
	}

	// Navigate: root -> stmtblock -> stmt -> createfunctionstmt
	root, ok := parseResults[0].Tree.(*postgresql.RootContext)
	if !ok {
		return nil, errors.Errorf("expecting RootContext but got %T", parseResults[0].Tree)
	}

	stmtblock := root.Stmtblock()
	if stmtblock == nil {
		return nil, errors.Errorf("expecting stmtblock but got nil")
	}

	stmtmulti := stmtblock.Stmtmulti()
	if stmtmulti == nil {
		return nil, errors.Errorf("expecting stmtmulti but got nil")
	}

	stmts := stmtmulti.AllStmt()
	if len(stmts) != 1 {
		return nil, errors.Errorf("expecting 1 statement, but got %d", len(stmts))
	}

	stmt := stmts[0]
	createFuncStmt := stmt.Createfunctionstmt()
	if createFuncStmt == nil {
		return nil, errors.Errorf("expecting Createfunctionstmt but got nil")
	}

	// Extract language from function options
	language, err := q.extractLanguageFromCreateFunction(createFuncStmt, name)
	if err != nil {
		return nil, err
	}

	// Extract AS body from function options
	_, err = q.extractAsBodyFromCreateFunction(createFuncStmt, name)
	if err != nil {
		return nil, err
	}

	switch language {
	case languageTypeSQL:
		// SQL functions will be handled below
		return q.getColumnsFromSQLFunction(createFuncStmt, name)
	case languageTypePLPGSQL:
		return q.getColumnsFromPLPGSQLFunction(createFuncStmt, name, definition)
	default:
		return nil, errors.Errorf("unsupported language type: %v", language)
	}
}

func (q *querySpanExtractor) getColumnsFromSQLFunction(createFuncStmt postgresql.ICreatefunctionstmtContext, name string) ([]base.QuerySpanResult, error) {
	asBody, err := q.extractAsBodyFromCreateFunction(createFuncStmt, name)
	if err != nil {
		return nil, err
	}

	columnNames := q.extractParameterNamesFromCreateFunction(createFuncStmt)

	newQ := newQuerySpanExtractor(q.defaultDatabase, q.searchPath, q.gCtx)
	span, err := newQ.getQuerySpan(q.ctx, asBody)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query span for function: %s", name)
	}
	if span.NotFoundError != nil {
		return nil, errors.Wrapf(span.NotFoundError, "failed to get query span for function: %s", name)
	}
	for source := range span.SourceColumns {
		q.sourceColumnsInFunction[source] = true
	}

	if len(columnNames) != len(span.Results) {
		return nil, errors.Errorf("expecting %d columns but got %d for function: %s", len(columnNames), len(span.Results), name)
	}
	for i, columnName := range columnNames {
		span.Results[i].Name = columnName
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getColumnsFromPLPGSQLFunction(createFuncStmt postgresql.ICreatefunctionstmtContext, name, definition string) ([]base.QuerySpanResult, error) {
	// Extract OUT/TABLE parameter names
	columnNames := q.extractParameterNamesFromCreateFunction(createFuncStmt)

	// Extract AS body
	asBody, err := q.extractAsBodyFromCreateFunction(createFuncStmt, name)
	if err != nil {
		return nil, err
	}

	// Try simple extraction first (extract RETURN QUERY statements)
	simpleResult, err := q.getColumnsFromSimplePLPGSQL(name, asBody, columnNames)
	if err == nil {
		return simpleResult, nil
	}

	// Fall back to complex extraction
	return q.getColumnsFromComplexPLPGSQL(name, columnNames, definition)
}

// getColumnsFromSimplePLPGSQL extracts query spans from simple PL/pgSQL functions with RETURN QUERY.
func (q *querySpanExtractor) getColumnsFromSimplePLPGSQL(name, asBody string, columnNames []string) ([]base.QuerySpanResult, error) {
	// Parse the PL/pgSQL body to extract RETURN QUERY statements
	sqlList, err := extractReturnQueryStatements(asBody)
	if err != nil {
		return nil, err
	}

	if len(sqlList) == 0 {
		return nil, errors.Errorf("no RETURN QUERY statements found in function: %s", name)
	}

	// Initialize result with column names and empty source columns
	var leftQuerySpanResult []base.QuerySpanResult
	for _, columnName := range columnNames {
		leftQuerySpanResult = append(leftQuerySpanResult, base.QuerySpanResult{
			Name:          columnName,
			SourceColumns: base.SourceColumnSet{},
		})
	}

	// Process each RETURN QUERY statement and merge source columns
	for _, sql := range sqlList {
		newQ := newQuerySpanExtractor(q.defaultDatabase, q.searchPath, q.gCtx)
		span, err := newQ.getQuerySpan(q.ctx, sql)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get query span for function: %s", name)
		}
		if span.NotFoundError != nil {
			return nil, errors.Wrapf(span.NotFoundError, "failed to get query span for function: %s", name)
		}

		rightQuerySpanResult := span.Results
		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("expecting %d columns but got %d for function: %s", len(leftQuerySpanResult), len(rightQuerySpanResult), name)
		}
		for source := range span.SourceColumns {
			q.sourceColumnsInFunction[source] = true
		}

		// Merge source columns
		var result []base.QuerySpanResult
		for i, leftSpanResult := range leftQuerySpanResult {
			rightSpanResult := rightQuerySpanResult[i]
			newResourceColumns, _ := base.MergeSourceColumnSet(leftSpanResult.SourceColumns, rightSpanResult.SourceColumns)
			result = append(result, base.QuerySpanResult{
				Name:          leftSpanResult.Name,
				SourceColumns: newResourceColumns,
			})
		}
		leftQuerySpanResult = result
	}

	return leftQuerySpanResult, nil
}

// extractReturnQueryStatements extracts all RETURN QUERY SELECT statements from PL/pgSQL body.
func extractReturnQueryStatements(asBody string) ([]string, error) {
	// Parse the PL/pgSQL body as a pl_block (BEGIN...END block)
	res, err := ParsePostgreSQLPLBlock(asBody)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PL/pgSQL body")
	}

	listener := &returnQueryListener{
		tokenStream: res.Tokens,
		sqlList:     []string{},
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, res.Tree)

	return listener.sqlList, nil
}

// returnQueryListener walks the AST to find all RETURN QUERY statements.
type returnQueryListener struct {
	*postgresql.BasePostgreSQLParserListener
	tokenStream antlr.TokenStream
	sqlList     []string
}

// EnterStmt_return is called when entering a RETURN statement.
func (l *returnQueryListener) EnterStmt_return(ctx *postgresql.Stmt_returnContext) {
	// Check if this is RETURN QUERY (not RETURN NEXT or plain RETURN)
	if ctx.QUERY() == nil {
		return
	}

	// Check if it's RETURN QUERY SELECT (not RETURN QUERY EXECUTE)
	if ctx.Selectstmt() == nil {
		return
	}

	// Extract the SELECT statement text using GetTextFromInterval
	selectStmt := ctx.Selectstmt()
	start := selectStmt.GetStart().GetTokenIndex()
	stop := selectStmt.GetStop().GetTokenIndex()
	interval := antlr.NewInterval(start, stop)
	sqlText := l.tokenStream.GetTextFromInterval(interval)

	l.sqlList = append(l.sqlList, sqlText)
}

func (q *querySpanExtractor) getColumnsFromComplexPLPGSQL(name string, columnNames []string, definition string) ([]base.QuerySpanResult, error) {
	parseResults, err := ParsePostgreSQL(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PLpgSQL function body for function %s", name)
	}

	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement in function definition, got %d", len(parseResults))
	}

	listener := &plpgSQLListener{
		q:         q,
		variables: make(map[string]*base.QuerySpanResult),
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, parseResults[0].Tree)
	if listener.err != nil {
		return nil, errors.Wrapf(listener.err, "failed to extract table source from PLpgSQL function body for function %s", name)
	}
	if listener.span == nil {
		return nil, errors.Errorf("failed to extract table source from PLpgSQL function body for function %s", name)
	}
	if listener.span.NotFoundError != nil {
		return nil, errors.Wrapf(listener.span.NotFoundError, "failed to extract table source from PLpgSQL function body for function %s", name)
	}
	if len(columnNames) != len(listener.span.Results) {
		return nil, errors.Errorf("expecting %d columns but got %d for function: %s", len(columnNames), len(listener.span.Results), name)
	}
	var result []base.QuerySpanResult
	for i, columnName := range columnNames {
		result = append(result, base.QuerySpanResult{
			Name:          columnName,
			SourceColumns: listener.span.Results[i].SourceColumns,
		})
	}
	return result, nil
}

// extractLanguageFromCreateFunction extracts the LANGUAGE clause from a CREATE FUNCTION statement.
// Returns languageTypeSQL (default) or languageTypePLPGSQL, or error if the language is unsupported.
func (*querySpanExtractor) extractLanguageFromCreateFunction(createFuncStmt postgresql.ICreatefunctionstmtContext, funcName string) (languageType, error) {
	// Default language is SQL if not specified
	language := languageTypeSQL

	// Parse function options to find LANGUAGE
	createOptList := createFuncStmt.Createfunc_opt_list()
	if createOptList == nil {
		// No options specified, use default
		return language, nil
	}

	for _, item := range createOptList.AllCreatefunc_opt_item() {
		if item == nil {
			continue
		}

		// Check if this option is LANGUAGE
		if item.LANGUAGE() == nil {
			continue
		}

		// Extract language value
		langNode := item.Nonreservedword_or_sconst()
		if langNode == nil {
			continue
		}

		// Get the language text and normalize it
		langText := langNode.GetText()
		// Remove quotes if present (e.g., 'sql' -> sql)

		langText = strings.ToLower(langText)

		switch langText {
		case "sql":
			language = languageTypeSQL
		case "plpgsql":
			language = languageTypePLPGSQL
		default:
			return language, &base.TypeNotSupportedError{
				Type:  "function language",
				Name:  langText,
				Extra: fmt.Sprintf("function: %s", funcName),
			}
		}

		// Found the language, no need to continue
		break
	}

	return language, nil
}

// extractAsBodyFromCreateFunction extracts the AS body (function definition) from a CREATE FUNCTION statement.
// Returns the function body SQL/PL/pgSQL code with quotes removed and escaped quotes unescaped.
func (*querySpanExtractor) extractAsBodyFromCreateFunction(createFuncStmt postgresql.ICreatefunctionstmtContext, funcName string) (string, error) {
	createOptList := createFuncStmt.Createfunc_opt_list()
	if createOptList == nil {
		return "", errors.Errorf("no function options found for function: %s", funcName)
	}

	for _, item := range createOptList.AllCreatefunc_opt_item() {
		if item == nil {
			continue
		}

		// Check if this option is AS
		if item.AS() == nil {
			continue
		}

		// Extract AS body
		funcAs := item.Func_as()
		if funcAs == nil {
			continue
		}

		// Get the function body from the first string constant
		allSconst := funcAs.AllSconst()
		if len(allSconst) == 0 {
			continue
		}

		// Check for C-style function definition: AS 'obj_file', 'link_symbol'
		// This is a legacy format for C language functions that we don't support
		if len(allSconst) > 1 || funcAs.COMMA() != nil {
			return "", &base.TypeNotSupportedError{
				Type:  "function",
				Name:  funcName,
				Extra: "C-style function with object file and link symbol is not supported",
			}
		}

		// Get the raw text preserving all whitespace using token stream interval
		sconstCtx := allSconst[0]
		parser := sconstCtx.GetParser()
		if parser == nil {
			return "", errors.Errorf("parser is nil for function: %s", funcName)
		}
		tokenStream := parser.GetTokenStream()

		// Use GetTextFromInterval to preserve exact whitespace
		start := sconstCtx.GetStart().GetTokenIndex()
		stop := sconstCtx.GetStop().GetTokenIndex()
		interval := antlr.NewInterval(start, stop)
		sconst := tokenStream.GetTextFromInterval(interval)

		if len(sconst) < 2 {
			return "", errors.Errorf("invalid AS body for function: %s", funcName)
		}

		// Unescape PostgreSQL string constant based on its type
		asBody, err := unescapePostgreSQLString(sconst)
		if err != nil {
			return "", errors.Wrapf(err, "failed to unescape AS body for function: %s", funcName)
		}

		return asBody, nil
	}

	return "", errors.Errorf("AS body not found for function: %s", funcName)
}

// unescapePostgreSQLString unescapes a PostgreSQL string constant based on its type.
// Handles: StringConstant, EscapeStringConstant, UnicodeEscapeStringConstant, DollarStringConstant.
func unescapePostgreSQLString(s string) (string, error) {
	if len(s) < 2 {
		return "", errors.Errorf("string too short: %s", s)
	}

	// Dollar-quoted string: $$...$$, $tag$...$tag$
	// No escaping, just remove delimiters
	if strings.HasPrefix(s, "$") {
		firstDollar := strings.Index(s[1:], "$")
		if firstDollar == -1 {
			return "", errors.Errorf("invalid dollar-quoted string")
		}
		delimiter := s[:firstDollar+2] // e.g., "$$" or "$tag$"
		body := strings.TrimPrefix(s, delimiter)
		body = strings.TrimSuffix(body, delimiter)
		return body, nil
	}

	// Escape string constant: E'...'
	// Supports backslash escapes: \n, \t, \r, \\, \', etc.
	if strings.HasPrefix(s, "E'") || strings.HasPrefix(s, "e'") {
		if !strings.HasSuffix(s, "'") {
			return "", errors.Errorf("unterminated escape string constant")
		}
		body := s[2 : len(s)-1]
		return unescapeEscapeString(body), nil
	}

	// Unicode escape string constant: U&'...'
	// Supports unicode escapes: \XXXX or \+XXXXXX
	if strings.HasPrefix(s, "U&'") || strings.HasPrefix(s, "u&'") {
		if !strings.HasSuffix(s, "'") {
			return "", errors.Errorf("unterminated unicode escape string constant")
		}
		body := s[3 : len(s)-1]
		return unescapeUnicodeEscapeString(body), nil
	}

	// Standard string constant: '...'
	// Only '' is escaped as '
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		body := s[1 : len(s)-1]
		// Unescape doubled single quotes
		return strings.ReplaceAll(body, "''", "'"), nil
	}

	return "", errors.Errorf("unknown string constant type: %s", s)
}

// unescapeEscapeString unescapes E'...' style strings with backslash escapes.
func unescapeEscapeString(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case 'b':
				result.WriteByte('\b')
			case 'f':
				result.WriteByte('\f')
			case '\\':
				result.WriteByte('\\')
			case '\'':
				result.WriteByte('\'')
			case '"':
				result.WriteByte('"')
			default:
				// Unknown escape, keep as-is
				result.WriteByte(s[i])
				result.WriteByte(s[i+1])
			}
			i += 2
		} else if s[i] == '\'' && i+1 < len(s) && s[i+1] == '\'' {
			// Handle '' as escaped '
			result.WriteByte('\'')
			i += 2
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// unescapeUnicodeEscapeString unescapes U&'...' style strings with unicode escapes.
func unescapeUnicodeEscapeString(s string) string {
	// For now, implement basic unicode escape support
	// Full implementation would need to handle \XXXX and \+XXXXXX patterns
	// This is a simplified version
	result := s
	// Unescape doubled single quotes first
	result = strings.ReplaceAll(result, "''", "'")
	// TODO: Add proper unicode escape handling if needed
	return result
}

// extractParameterNamesFromCreateFunction extracts OUT/TABLE parameter names from CREATE FUNCTION.
// Returns column names from OUT parameters and RETURNS TABLE columns.
func (q *querySpanExtractor) extractParameterNamesFromCreateFunction(createFuncStmt postgresql.ICreatefunctionstmtContext) []string {
	var columnNames []string

	// Extract OUT parameters from func_args_with_defaults
	funcArgs := createFuncStmt.Func_args_with_defaults()
	if funcArgs != nil && funcArgs.Func_args_with_defaults_list() != nil {
		for _, argWithDefault := range funcArgs.Func_args_with_defaults_list().AllFunc_arg_with_default() {
			if argWithDefault == nil {
				continue
			}
			funcArg := argWithDefault.Func_arg()
			if funcArg == nil {
				continue
			}

			// Check if this is an OUT or INOUT parameter
			argClass := funcArg.Arg_class()
			if argClass == nil {
				continue
			}

			// Check for OUT_P (OUT mode) or INOUT
			if argClass.OUT_P() == nil && argClass.INOUT() == nil {
				continue
			}

			// Extract parameter name
			// func_arg can be: arg_class param_name? func_type | param_name arg_class? func_type | func_type
			var paramName string
			if funcArg.Param_name() != nil {
				paramName = q.extractParamName(funcArg.Param_name())
			}

			// If OUT parameter has a name, add it to column names
			if paramName != "" {
				columnNames = append(columnNames, paramName)
			}
		}
	}

	// Extract TABLE columns from RETURNS TABLE clause
	if createFuncStmt.TABLE() != nil && createFuncStmt.Table_func_column_list() != nil {
		for _, tableCol := range createFuncStmt.Table_func_column_list().AllTable_func_column() {
			if tableCol == nil || tableCol.Param_name() == nil {
				continue
			}
			paramName := q.extractParamName(tableCol.Param_name())
			if paramName != "" {
				columnNames = append(columnNames, paramName)
			}
		}
	}

	return columnNames
}

// extractParamName extracts the parameter name from param_name context.
// param_name can be: type_function_name | builtin_function_name | LEFT | RIGHT
func (*querySpanExtractor) extractParamName(paramNameCtx postgresql.IParam_nameContext) string {
	if paramNameCtx == nil {
		return ""
	}

	// Handle type_function_name (most common case)
	if paramNameCtx.Type_function_name() != nil {
		return normalizePostgreSQLTypeFunctionName(paramNameCtx.Type_function_name())
	}

	// Handle builtin_function_name, LEFT, RIGHT
	return strings.ToLower(paramNameCtx.GetText())
}

type plpgSQLListener struct {
	*postgresql.BasePostgreSQLParserListener
	q *querySpanExtractor

	variables map[string]*base.QuerySpanResult

	span *base.QuerySpan
	err  error
}

func (l *plpgSQLListener) EnterFunc_as(ctx *postgresql.Func_asContext) {
	antlr.ParseTreeWalkerDefault.Walk(l, ctx.Definition)
}

func (l *plpgSQLListener) EnterDecl_statement(ctx *postgresql.Decl_statementContext) {
	vName := normalizePostgreSQLAnyIdentifier(ctx.Decl_varname().Any_identifier())
	l.variables[vName] = &base.QuerySpanResult{
		SourceColumns: base.SourceColumnSet{},
	}
}

func (l *plpgSQLListener) EnterStmt_assign(ctx *postgresql.Stmt_assignContext) {
	names := NormalizePostgreSQLAnyName(ctx.Assign_var().Any_name())
	if len(names) != 1 {
		return
	}
	assignVar := names[0]

	newQ := newQuerySpanExtractor(l.q.defaultDatabase, l.q.searchPath, l.q.gCtx)
	span, err := newQ.getQuerySpan(l.q.ctx, fmt.Sprintf("SELECT %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Sql_expression())))
	if err != nil {
		return
	}
	if span.NotFoundError != nil {
		l.err = span.NotFoundError
		return
	}
	for source := range span.SourceColumns {
		l.q.sourceColumnsInFunction[source] = true
	}
	leftVarQuerySpan, exists := l.variables[assignVar]
	if !exists {
		leftVarQuerySpan = &base.QuerySpanResult{
			SourceColumns: base.SourceColumnSet{},
		}
		l.variables[assignVar] = leftVarQuerySpan
	}
	if len(span.Results) != 1 {
		return
	}
	newResourceColumns, _ := base.MergeSourceColumnSet(leftVarQuerySpan.SourceColumns, span.Results[0].SourceColumns)
	l.variables[assignVar] = &base.QuerySpanResult{
		SourceColumns: newResourceColumns,
	}
}

func (l *plpgSQLListener) EnterStmt_return(ctx *postgresql.Stmt_returnContext) {
	if ctx.QUERY() == nil {
		return
	}

	if ctx.Selectstmt() == nil {
		return
	}

	newQ := newQuerySpanExtractor(l.q.defaultDatabase, l.q.searchPath, l.q.gCtx)
	newQ.variables = l.variables
	l.span, l.err = newQ.getQuerySpan(l.q.ctx, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Selectstmt()))
}

func (q *querySpanExtractor) getAllTableColumnSources(schemaName, tableName string) ([]base.QuerySpanResult, bool) {
	findInTableSource := func(tableSource base.TableSource) ([]base.QuerySpanResult, bool) {
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		return tableSource.GetQuerySpanResult(), true
	}

	// One sub-query may have multi-outer schemas and the multi-outer schemas can use the same name, such as:
	//
	//  select (
	//    select (
	//      select max(a) > x1.a from t
	//    )
	//    from t1 as x1
	//    limit 1
	//  )
	//  from t as x1;
	//
	// This query has two tables can be called `x1`, and the expression x1.a uses the closer x1 table.
	// This is the reason we loop the slice in reversed order.

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet, true
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	return []base.QuerySpanResult{}, false
}

func (q *querySpanExtractor) getFieldColumnSource(schemaName, tableName, fieldName string) (base.SourceColumnSet, bool) {
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if tableSource == nil {
			return nil, false
		}
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}
		// If the table name is empty, we should check if there are ambiguous fields,
		// but we delegate this responsibility to the db-server, we do the fail-open strategy here.

		querySpanResult := tableSource.GetQuerySpanResult()
		for _, field := range querySpanResult {
			if field.Name == fieldName {
				return field.SourceColumns, true
			}
		}
		return nil, false
	}

	// One sub-query may have multi-outer schemas and the multi-outer schemas can use the same name, such as:
	//
	//  select (
	//    select (
	//      select max(a) > x1.a from t
	//    )
	//    from t1 as x1
	//    limit 1
	//  )
	//  from t as x1;
	//
	// This query has two tables can be called `x1`, and the expression x1.a uses the closer x1 table.
	// This is the reason we loop the slice in reversed order.

	for i := len(q.outerTableSources) - 1; i >= 0; i-- {
		if sourceColumnSet, ok := findInTableSource(q.outerTableSources[i]); ok {
			return sourceColumnSet, true
		}
	}

	for _, tableSource := range q.tableSourcesFrom {
		if sourceColumnSet, ok := findInTableSource(tableSource); ok {
			return sourceColumnSet, true
		}
	}

	if schemaName == "" && tableName == "" && q.variables != nil {
		if v, exists := q.variables[fieldName]; exists {
			return v.SourceColumns, true
		}
	}

	return base.SourceColumnSet{}, false
}

func (q *querySpanExtractor) findTableInFrom(schemaName string, tableName string) base.TableSource {
	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	if schemaName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table
			}
		}
	}

	for i := len(q.tableSourcesFrom) - 1; i >= 0; i-- {
		tableSource := q.tableSourcesFrom[i]
		if tableSource == nil {
			continue
		}
		emptySchemaNameMatch := schemaName == "" && (tableSource.GetSchemaName() == "" || slices.Contains(q.searchPath, tableSource.GetSchemaName())) && tableName == tableSource.GetTableName()
		nonEmptySchemaNameMatch := schemaName != "" && tableSource.GetSchemaName() == schemaName && tableName == tableSource.GetTableName()
		if emptySchemaNameMatch || nonEmptySchemaNameMatch {
			return tableSource
		}
	}

	return nil
}

func (q *querySpanExtractor) findTableSchema(schemaName string, tableName string) (base.TableSource, error) {
	// Each CTE name in one WITH clause must be unique, but we can use the same name in the different level CTE, such as:
	//
	//  with tt2 as (
	//    with tt2 as (select * from t)
	//    select max(a) from tt2)
	//  select * from tt2
	//
	// This query has two CTE can be called `tt2`, and the FROM clause 'from tt2' uses the closer tt2 CTE.
	// This is the reason we loop the slice in reversed order.
	if schemaName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			table := q.ctes[i]
			if table.Name == tableName {
				return table, nil
			}
		}
	}

	// FIXME: consider cross database query which is supported in Redshift.
	dbMetadata, err := q.getDatabaseMetadata(q.defaultDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.defaultDatabase)
	}
	if dbMetadata == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &q.defaultDatabase,
		}
	}
	searcher := dbMetadata.NewSearcher(schemaName, q.searchPath)
	tableSchemaName, table := searcher.SearchTable(tableName)
	viewSchemaName, view := searcher.SearchView(tableName)
	materializedViewSchemaName, materializedView := searcher.SearchMaterializedView(tableName)
	foreignTableSchemaName, foreignTable := searcher.SearchExternalTable(tableName)
	sequenceSchemaName, sequence := searcher.SearchSequence(tableName)

	if table == nil && view == nil && foreignTable == nil && materializedView == nil && sequence == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &q.defaultDatabase,
			Schema:   &schemaName,
			Table:    &tableName,
		}
	}

	if table != nil {
		var columns []string
		for _, column := range table.GetProto().GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: q.defaultDatabase,
			Schema:   tableSchemaName,
			Name:     table.GetProto().Name,
			Columns:  columns,
		}, nil
	}

	if foreignTable != nil {
		var columns []string
		for _, column := range foreignTable.GetProto().GetColumns() {
			columns = append(columns, column.Name)
		}
		return &base.PhysicalTable{
			Server:   "",
			Database: q.defaultDatabase,
			Schema:   foreignTableSchemaName,
			Name:     foreignTable.GetProto().Name,
			Columns:  columns,
		}, nil
	}

	if view != nil && view.Definition != "" {
		columns, err := q.getColumnsForView(view.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PhysicalView{
			Server:   "",
			Database: q.defaultDatabase,
			Schema:   viewSchemaName,
			Name:     view.Name,
			Columns:  columns,
		}, nil
	}

	if materializedView != nil && materializedView.Definition != "" {
		columns, err := q.getColumnsForMaterializedView(materializedView.Definition)
		if err != nil {
			return nil, err
		}
		return &base.PhysicalView{
			Server:   "",
			Database: q.defaultDatabase,
			Schema:   materializedViewSchemaName,
			Name:     materializedView.Name,
			Columns:  columns,
		}, nil
	}

	if sequence != nil {
		// The default columns for sequence in PostgreSQL.
		columns := []string{"last_value", "log_cnt", "is_called"}
		return &base.Sequence{
			Server:   "",
			Database: q.defaultDatabase,
			Schema:   sequenceSchemaName,
			Name:     sequence.Name,
			Columns:  columns,
		}, nil
	}
	return nil, nil
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.defaultDatabase, q.searchPath, q.gCtx)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func (q *querySpanExtractor) getColumnsForMaterializedView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.defaultDatabase, q.searchPath, q.gCtx)
	span, err := newQ.getQuerySpan(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}

func (q *querySpanExtractor) EnterStmtmulti(sm *postgresql.StmtmultiContext) {
	if len(sm.AllStmt()) != 1 {
		q.err = errors.Errorf("malformed query, expected 1 query but got %d", len(sm.AllStmt()))
		return
	}

	// Process the statement and store the result
	stmt := sm.AllStmt()[0]
	tableSource, err := q.extractTableSourceFromStmt(stmt)
	if err != nil {
		q.err = err
		return
	}

	// Store the extracted table source for later retrieval
	q.resultTableSource = tableSource
}

func (q *querySpanExtractor) extractTableSourceFromStmt(stmt postgresql.IStmtContext) (base.TableSource, error) {
	switch {
	case stmt.Explainstmt() != nil:
		// Handle EXPLAIN statement
		explainStmt := stmt.Explainstmt()
		if explainStmt.Explainablestmt() != nil {
			// Extract from the explained statement
			if explainableStmt := explainStmt.Explainablestmt(); explainableStmt != nil {
				if explainableStmt.Selectstmt() != nil {
					return q.extractTableSourceFromSelectstmt(explainableStmt.Selectstmt())
				}
				if explainableStmt.Insertstmt() != nil ||
					explainableStmt.Updatestmt() != nil ||
					explainableStmt.Deletestmt() != nil {
					// For DML in EXPLAIN, return empty pseudo table
					return &base.PseudoTable{
						Name:    "",
						Columns: []base.QuerySpanResult{},
					}, nil
				}
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}, nil
	case stmt.Selectstmt() != nil:
		return q.extractTableSourceFromSelectstmt(stmt.Selectstmt())
	default:
		// For non-SELECT statements, return empty pseudo table
		return &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}, nil
	}
}

func (q *querySpanExtractor) extractTableSourceFromSelectstmt(selectstmt postgresql.ISelectstmtContext) (base.TableSource, error) {
	selectNoParens := getSelectNoParensFromSelectStmt(selectstmt)
	return q.extractTableSourceFromSelectNoParens(selectNoParens)
}

func (q *querySpanExtractor) extractTableSourceFromSelectNoParens(selectNoParens postgresql.ISelect_no_parensContext) (base.TableSource, error) {
	// Reset the table sources from the FROM clause after exit the SELECT statement.
	previousTableSourcesFromLength := len(q.tableSourcesFrom)
	defer func() {
		q.tableSourcesFrom = q.tableSourcesFrom[:previousTableSourcesFromLength]
	}()

	// Analyze the CTE first.
	withClause := selectNoParens.With_clause()
	if withClause != nil {
		cteLength, err := q.recordCTEs(withClause)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to record CTEs")
		}
		defer func() {
			q.ctes = q.ctes[:cteLength]
		}()
	}

	// Process the SELECT clause
	selectClause := selectNoParens.Select_clause()
	if selectClause == nil {
		return nil, errors.New("select_no_parens has no select_clause")
	}

	// Handle UNION/INTERSECT/EXCEPT at the top level
	allSimpleSelects := selectClause.AllSimple_select_intersect()
	if len(allSimpleSelects) == 0 {
		return nil, errors.New("select_clause has no simple_select_intersect")
	}

	// If there's only one simple_select_intersect, process it directly
	if len(allSimpleSelects) == 1 {
		return q.extractTableSourceFromSimpleSelectIntersect(allSimpleSelects[0])
	}

	// Multiple simple_select_intersect with operators between them
	// Extract operators from the select_clause
	operators, err := q.extractOperatorsFromSelectClause(selectClause)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract operators from select_clause")
	}

	return q.mergeSimpleSelectIntersectsWithOperators(allSimpleSelects, operators)
}

func (q *querySpanExtractor) recordCTEs(with postgresql.IWith_clauseContext) (restoreCTELength int, err error) {
	previousCTELength := len(q.ctes)
	recursive := with.RECURSIVE() != nil
	for _, cte := range with.Cte_list().AllCommon_table_expr() {
		cteName := normalizePostgreSQLName(cte.Name())
		var colAliasNames []string
		if cte.Opt_name_list() != nil {
			for _, name := range cte.Opt_name_list().Name_list().AllName() {
				colAliasNames = append(colAliasNames, normalizePostgreSQLName(name))
			}
		}

		var cteTableSource *base.PseudoTable
		if !recursive {
			// Non-recursive WITH clause - all CTEs are non-recursive
			cteTableSource, err = q.extractTableSourceFromNonRecursiveCTE(cte, cteName, colAliasNames)
			if err != nil {
				return previousCTELength, err
			}
		} else {
			// WITH RECURSIVE clause - try to extract as recursive
			// If it's not actually recursive, extractTableSourceFromRecursiveCTE will fall back to non-recursive
			cteTableSource, err = q.extractTableSourceFromRecursiveCTE(cte, cteName, colAliasNames)
			if err != nil {
				return previousCTELength, err
			}
		}

		q.ctes = append(q.ctes, cteTableSource)
	}

	return previousCTELength, nil
}

// extractTableSourceFromRecursiveCTE processes a recursive CTE by separating it into base and recursive parts,
// then performs iterative closure computation to resolve all column dependencies.
// A recursive CTE has the structure: base_case UNION [ALL] recursive_case
// We use a simple strategy: the last part after UNION is the recursive part.
func (q *querySpanExtractor) extractTableSourceFromRecursiveCTE(cte postgresql.ICommon_table_exprContext, cteName string, colAliasNames []string) (*base.PseudoTable, error) {
	selectStmt := cte.Preparablestmt().Selectstmt()
	if selectStmt == nil {
		return nil, errors.Errorf("malformed recursive CTE, expected SELECT statement in CTE but got others")
	}
	selectNoParens := getSelectNoParensFromSelectStmt(selectStmt)

	// Split the CTE into base and recursive parts using the simple strategy
	baseParts, baseOperators, recursivePart, recursiveOperator, err := splitRecursiveCTEParts(selectNoParens)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split recursive CTE %q", cteName)
	}

	if len(baseParts) == 0 || recursivePart == nil || recursiveOperator == nil {
		// Not a valid recursive structure, treat as non-recursive
		return q.extractTableSourceFromNonRecursiveCTE(cte, cteName, colAliasNames)
	}

	// Process the base parts (can be multiple parts combined with various operators)
	var baseTableSource base.TableSource
	if len(baseParts) == 1 {
		// Single base part
		baseTableSource, err = q.extractTableSourceFromSimpleSelectIntersect(baseParts[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract base part for recursive CTE %q", cteName)
		}
	} else {
		// Multiple base parts - need to merge them with their respective operators
		baseTableSource, err = q.mergeSimpleSelectIntersectsWithOperators(baseParts, baseOperators)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to merge base parts for recursive CTE %q", cteName)
		}
	}

	// Apply column aliases
	querySpanResult := baseTableSource.GetQuerySpanResult()
	if len(colAliasNames) > 0 {
		if len(colAliasNames) != len(querySpanResult) {
			return nil, errors.Errorf("CTE %q has %d columns but alias has %d columns", cteName, len(querySpanResult), len(colAliasNames))
		}
		for i, colAlias := range colAliasNames {
			querySpanResult[i].Name = colAlias
		}
	}

	cteTableSource := &base.PseudoTable{
		Name:    cteName,
		Columns: querySpanResult,
	}

	// Add to CTEs for recursive part to reference
	q.ctes = append(q.ctes, cteTableSource)
	defer func() {
		q.ctes = q.ctes[:len(q.ctes)-1]
	}()

	// Iterative closure computation
	for {
		recursiveTableSource, err := q.extractTableSourceFromSimpleSelectIntersect(recursivePart)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract recursive part for CTE %q", cteName)
		}

		recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
		if len(recursiveQuerySpanResult) != len(querySpanResult) {
			return nil, errors.Errorf("CTE %q base has %d columns but recursive has %d", cteName, len(querySpanResult), len(recursiveQuerySpanResult))
		}

		changed := false
		for i, spanResult := range recursiveQuerySpanResult {
			newSourceColumns, hasDiff := base.MergeSourceColumnSet(querySpanResult[i].SourceColumns, spanResult.SourceColumns)
			if hasDiff {
				changed = true
				querySpanResult[i].SourceColumns = newSourceColumns
			}
		}

		if !changed {
			break
		}
		q.ctes[len(q.ctes)-1].Columns = querySpanResult
	}

	return cteTableSource, nil
}

func (q *querySpanExtractor) extractTableSourceFromNonRecursiveCTE(cte postgresql.ICommon_table_exprContext, cteName string, colAliasNames []string) (*base.PseudoTable, error) {
	selectStmt := cte.Preparablestmt().Selectstmt()
	if selectStmt == nil {
		return nil, errors.Errorf("malformed CTE, expected SELECT statement in CTE but got others")
	}
	// Non-recursive CTE.
	tableSource, err := q.extractTableSourceFromSelectstmt(selectStmt)
	if err != nil {
		return nil, err
	}
	querySpanResult := tableSource.GetQuerySpanResult()
	if len(colAliasNames) > 0 && len(colAliasNames) != len(querySpanResult) {
		return nil, errors.Errorf("the column alias number %d does not match the actual column number %d in CTE %q", len(colAliasNames), len(querySpanResult), cteName)
	}
	for i, colAlias := range colAliasNames {
		querySpanResult[i].Name = colAlias
	}
	cteTableSource := &base.PseudoTable{
		Name:    cteName,
		Columns: querySpanResult,
	}
	return cteTableSource, nil
}

// extractOperatorsFromSelectClause extracts set operators from a select_clause
func (*querySpanExtractor) extractOperatorsFromSelectClause(selectClause postgresql.ISelect_clauseContext) ([]SetOperator, error) {
	if selectClause == nil {
		return nil, errors.New("nil select_clause")
	}

	allSimpleSelects := selectClause.AllSimple_select_intersect()
	if len(allSimpleSelects) <= 1 {
		return nil, nil // No operators needed for 0 or 1 select
	}

	var operators []SetOperator
	children := selectClause.GetChildren()

	// Parse children to find operators between simple_select_intersect nodes
	for i := 0; i < len(children); i++ {
		if token, ok := children[i].(antlr.TerminalNode); ok {
			tokenType := token.GetSymbol().GetTokenType()
			var op SetOperator

			switch tokenType {
			case postgresql.PostgreSQLParserUNION:
				op.Type = "UNION"
				op.IsDistinct = true // Default to DISTINCT
				// Check for ALL/DISTINCT modifier
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						if nextToken.GetSymbol().GetTokenType() == postgresql.PostgreSQLParserALL {
							op.IsDistinct = false
							i++ // Skip ALL token
						} else if nextToken.GetSymbol().GetTokenType() == postgresql.PostgreSQLParserDISTINCT {
							op.IsDistinct = true
							i++ // Skip DISTINCT token
						}
					}
				}
				operators = append(operators, op)

			case postgresql.PostgreSQLParserEXCEPT:
				op.Type = "EXCEPT"
				op.IsDistinct = true // Default to DISTINCT
				// Check for ALL/DISTINCT modifier
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						if nextToken.GetSymbol().GetTokenType() == postgresql.PostgreSQLParserALL {
							op.IsDistinct = false
							i++ // Skip ALL token
						} else if nextToken.GetSymbol().GetTokenType() == postgresql.PostgreSQLParserDISTINCT {
							op.IsDistinct = true
							i++ // Skip DISTINCT token
						}
					}
				}
				operators = append(operators, op)

			case postgresql.PostgreSQLParserINTERSECT:
				op.Type = "INTERSECT"
				op.IsDistinct = true // Default to DISTINCT
				// Check for ALL/DISTINCT modifier
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						if nextToken.GetSymbol().GetTokenType() == postgresql.PostgreSQLParserALL {
							op.IsDistinct = false
							i++ // Skip ALL token
						} else if nextToken.GetSymbol().GetTokenType() == postgresql.PostgreSQLParserDISTINCT {
							op.IsDistinct = true
							i++ // Skip DISTINCT token
						}
					}
				}
				operators = append(operators, op)
			default:
				// Ignore other tokens
				continue
			}
		}
	}

	if len(operators) != len(allSimpleSelects)-1 {
		return nil, errors.Errorf("expected %d operators but found %d", len(allSimpleSelects)-1, len(operators))
	}

	return operators, nil
}

// SetOperator represents a set operation between SELECT statements
type SetOperator struct {
	Type       string // "UNION", "EXCEPT", "INTERSECT"
	IsDistinct bool   // false means ALL, true means DISTINCT (or implicit distinct)
}

// splitRecursiveCTEParts analyzes a recursive CTE's SELECT structure and separates it into:
// - baseParts: Everything before the last UNION/UNION ALL (the non-recursive anchor queries)
// - baseOperators: The operators between base parts (UNION, EXCEPT, etc.)
// - recursivePart: The part after the last UNION/UNION ALL (the recursive query)
// - recursiveOperator: The operator connecting the base to the recursive part (must be UNION [ALL])
// Note: The recursive part must be connected with UNION [ALL], not EXCEPT or INTERSECT.
func splitRecursiveCTEParts(selectNoParens postgresql.ISelect_no_parensContext) (
	baseParts []postgresql.ISimple_select_intersectContext,
	baseOperators []SetOperator,
	recursivePart postgresql.ISimple_select_intersectContext,
	recursiveOperator *SetOperator,
	err error) {
	if selectNoParens == nil || selectNoParens.Select_clause() == nil {
		return nil, nil, nil, nil, errors.New("invalid select_no_parens")
	}

	selectClause := selectNoParens.Select_clause()
	allParts := selectClause.AllSimple_select_intersect()

	if len(allParts) < 2 {
		// No UNION, cannot be a recursive CTE
		return nil, nil, nil, nil, nil
	}

	// Track all operators between parts
	var allOperators []SetOperator
	lastUnionIndex := -1

	// Parse the children to extract operators
	// Pattern: part0 [OPERATOR [ALL|DISTINCT]] part1 [OPERATOR [ALL|DISTINCT]] part2 ...
	children := selectClause.GetChildren()
	partIndex := 0
	i := 0

	for i < len(children) {
		child := children[i]
		if token, ok := child.(antlr.TerminalNode); ok {
			tokenType := token.GetSymbol().GetTokenType()
			var op SetOperator

			switch tokenType {
			case postgresql.PostgreSQLParserUNION:
				op.Type = "UNION"
				// Check if next token is ALL or DISTINCT
				op.IsDistinct = true // Default to DISTINCT
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						nextType := nextToken.GetSymbol().GetTokenType()
						switch nextType {
						case postgresql.PostgreSQLParserALL:
							op.IsDistinct = false
							i++
						case postgresql.PostgreSQLParserDISTINCT:
							op.IsDistinct = true
							i++
						default:
						}
					}
				}
				allOperators = append(allOperators, op)
				lastUnionIndex = partIndex
				partIndex++

			case postgresql.PostgreSQLParserEXCEPT:
				op.Type = "EXCEPT"
				// Check if next token is ALL or DISTINCT
				op.IsDistinct = true // Default to DISTINCT
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						nextType := nextToken.GetSymbol().GetTokenType()
						switch nextType {
						case postgresql.PostgreSQLParserALL:
							op.IsDistinct = false
							i++
						case postgresql.PostgreSQLParserDISTINCT:
							op.IsDistinct = true
							i++
						default:
						}
					}
				}
				allOperators = append(allOperators, op)
				partIndex++

			case postgresql.PostgreSQLParserINTERSECT:
				op.Type = "INTERSECT"
				// Check if next token is ALL or DISTINCT
				op.IsDistinct = true // Default to DISTINCT
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						nextType := nextToken.GetSymbol().GetTokenType()
						switch nextType {
						case postgresql.PostgreSQLParserALL:
							op.IsDistinct = false
							i++
						case postgresql.PostgreSQLParserDISTINCT:
							op.IsDistinct = true
							i++
						default:
						}
					}
				}
				allOperators = append(allOperators, op)
				partIndex++
			default:
				// Ignore other tokens
			}
		}
		i++
	}

	if lastUnionIndex < 0 {
		// No UNION found, not a recursive CTE
		return nil, nil, nil, nil, nil
	}

	// Split at the last UNION position
	// Everything before the last UNION is the base part(s)
	// The part after the last UNION is the recursive part
	baseParts = allParts[:lastUnionIndex+1]
	recursivePart = allParts[lastUnionIndex+1]

	// Split operators too
	if lastUnionIndex > 0 {
		baseOperators = allOperators[:lastUnionIndex]
	}
	recursiveOperator = &allOperators[lastUnionIndex]

	return baseParts, baseOperators, recursivePart, recursiveOperator, nil
}

// extractTableSourceFromSimpleSelectIntersect processes a simple_select_intersect node and returns its table source.
// This handles the INTERSECT operations between multiple simple_select_pramary nodes.
func (q *querySpanExtractor) extractTableSourceFromSimpleSelectIntersect(simpleSelect postgresql.ISimple_select_intersectContext) (base.TableSource, error) {
	if simpleSelect == nil {
		return nil, errors.New("nil simple_select_intersect")
	}

	allPrimaries := simpleSelect.AllSimple_select_pramary()
	if len(allPrimaries) == 0 {
		return nil, errors.New("no primary selects in simple_select_intersect")
	}

	// If there's only one primary, just process it directly
	if len(allPrimaries) == 1 {
		return q.extractTableSourceFromSimpleSelectPrimary(allPrimaries[0])
	}

	// Multiple primaries - need to handle INTERSECT operations
	// Start with the first primary
	result, err := q.extractTableSourceFromSimpleSelectPrimary(allPrimaries[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract first primary")
	}

	// Process remaining primaries with INTERSECT
	for i := 1; i < len(allPrimaries); i++ {
		nextResult, err := q.extractTableSourceFromSimpleSelectPrimary(allPrimaries[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract primary %d", i)
		}

		// Merge with INTERSECT semantics
		leftQuerySpanResult := result.GetQuerySpanResult()
		rightQuerySpanResult := nextResult.GetQuerySpanResult()

		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("INTERSECT requires same number of columns: %d vs %d",
				len(leftQuerySpanResult), len(rightQuerySpanResult))
		}

		var mergedColumns []base.QuerySpanResult
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			// For INTERSECT, we take columns that exist in both
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name,
				SourceColumns: mergedSourceColumns,
			})
		}

		result = &base.PseudoTable{
			Name:    "",
			Columns: mergedColumns,
		}
	}

	return result, nil
}

// extractTableSourceFromSimpleSelectPrimary processes a simple_select_pramary node and returns its table source.
// This is the basic building block of SELECT statements, containing the SELECT list, FROM clause, etc.
func (q *querySpanExtractor) extractTableSourceFromSimpleSelectPrimary(primary postgresql.ISimple_select_pramaryContext) (base.TableSource, error) {
	if primary == nil {
		return nil, errors.New("nil simple_select_pramary")
	}

	// Handle VALUES clause
	if v := primary.Values_clause(); v != nil {
		columnNumber := len(v.AllExpr_list()[0].AllA_expr())
		// Check other rows have the same number of columns
		for _, exprList := range v.AllExpr_list()[1:] {
			if len(exprList.AllA_expr()) != columnNumber {
				return nil, errors.New("VALUES clause rows have different column counts")
			}
		}
		columnSourceSets := make([]base.SourceColumnSet, columnNumber)
		for i, exprList := range v.AllExpr_list() {
			for j, expr := range exprList.AllA_expr() {
				cols, err := q.extractSourceColumnSetFromAExpr(expr)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract columns from VALUES clause row %d, column %d", i, j)
				}
				columnSourceSets[j], _ = base.MergeSourceColumnSet(columnSourceSets[j], cols)
			}
		}
		tableSource := &base.PseudoTable{
			Name: "",
			Columns: func() []base.QuerySpanResult {
				var results []base.QuerySpanResult
				for i, colSet := range columnSourceSets {
					results = append(results, base.QuerySpanResult{
						Name:          fmt.Sprintf("column%d", i+1),
						SourceColumns: colSet,
					})
				}
				return results
			}(),
		}

		return tableSource, nil
	}

	// Handle TABLE clause (TABLE tablename)
	// TODO: Implement TABLE clause handling when needed

	// Handle regular SELECT
	if primary.SELECT() != nil {
		// Save current FROM table sources
		previousTableSourcesFromLength := len(q.tableSourcesFrom)
		defer func() {
			q.tableSourcesFrom = q.tableSourcesFrom[:previousTableSourcesFromLength]
		}()

		// Process FROM clause first
		fromFieldList, err := q.extractTableSourceFromFromClause(primary.From_clause())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table sources from FROM clause")
		}

		// Process target list (SELECT items)
		var targetListContext postgresql.ITarget_listContext
		if primary.Opt_target_list() != nil {
			targetListContext = primary.Opt_target_list().Target_list()
		} else if primary.Target_list() != nil {
			targetListContext = primary.Target_list()
		}

		// PostgreSQL allow SELECT without target list (e.g., SELECT FROM table), it returns no columns.
		if targetListContext == nil {
			return &base.PseudoTable{
				Name:    "",
				Columns: []base.QuerySpanResult{},
			}, nil
		}

		columns, err := q.extractColumnsFromTargetList(targetListContext, fromFieldList)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract columns from target list")
		}

		result := &base.PseudoTable{
			Name:    "",
			Columns: columns,
		}

		return result, nil
	}

	// Handle parenthesized SELECT
	if s := primary.Select_with_parens(); s != nil {
		selectNoParens := getSelectNoParensFromSelectWithParens(s)
		return q.extractTableSourceFromSelectNoParens(selectNoParens)
	}

	// Default case
	return &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}, nil
}

// extractTableSourceFromSelectWithParens handles SELECT statements wrapped in parentheses
func (q *querySpanExtractor) extractTableSourceFromSelectWithParens(selectWithParens postgresql.ISelect_with_parensContext) (base.TableSource, error) {
	if selectWithParens == nil {
		return nil, errors.New("nil select_with_parens")
	}

	if selectWithParens.Select_no_parens() != nil {
		return q.extractTableSourceFromSelectNoParens(selectWithParens.Select_no_parens())
	}

	if selectWithParens.Select_with_parens() != nil {
		// Nested parentheses
		return q.extractTableSourceFromSelectWithParens(selectWithParens.Select_with_parens())
	}

	return nil, errors.New("select_with_parens has no valid content")
}

// mergeSimpleSelectIntersectsWithOperators combines multiple simple_select_intersect nodes with their respective operators.
// This properly handles UNION, EXCEPT, INTERSECT with ALL/DISTINCT modifiers.
func (q *querySpanExtractor) mergeSimpleSelectIntersectsWithOperators(parts []postgresql.ISimple_select_intersectContext, operators []SetOperator) (base.TableSource, error) {
	if len(parts) == 0 {
		return nil, errors.New("no parts to merge")
	}

	if len(parts) == 1 {
		// Single part, no operators needed
		return q.extractTableSourceFromSimpleSelectIntersect(parts[0])
	}

	if len(operators) != len(parts)-1 {
		return nil, errors.Errorf("invalid operators count: %d operators for %d parts", len(operators), len(parts))
	}

	// Start with the first part
	result, err := q.extractTableSourceFromSimpleSelectIntersect(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract first simple_select_intersect")
	}

	// Apply each operator with the next part
	for i, op := range operators {
		nextResult, err := q.extractTableSourceFromSimpleSelectIntersect(parts[i+1])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract simple_select_intersect %d", i+1)
		}

		// Apply the specific operator semantics
		result, err = q.applySetOperator(result, nextResult, op)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply %s at position %d", op.Type, i)
		}
	}

	return result, nil
}

// applySetOperator applies a set operation (UNION, EXCEPT, INTERSECT) between two table sources
func (*querySpanExtractor) applySetOperator(left, right base.TableSource, op SetOperator) (base.TableSource, error) {
	leftQuerySpanResult := left.GetQuerySpanResult()
	rightQuerySpanResult := right.GetQuerySpanResult()

	if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
		return nil, errors.Errorf("%s requires same number of columns: %d vs %d",
			op.Type, len(leftQuerySpanResult), len(rightQuerySpanResult))
	}

	var mergedColumns []base.QuerySpanResult

	switch op.Type {
	case "UNION":
		// UNION combines rows from both queries
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name, // Use the name from the first query
				SourceColumns: mergedSourceColumns,
			})
		}

	case "EXCEPT":
		// EXCEPT returns rows from left that don't appear in right
		// For column tracking, we only track columns from the left side
		for _, leftCol := range leftQuerySpanResult {
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name,
				SourceColumns: leftCol.SourceColumns, // Only left side columns matter
			})
		}

	case "INTERSECT":
		// INTERSECT returns rows that appear in both
		// We need to track columns from both sides
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name,
				SourceColumns: mergedSourceColumns,
			})
		}

	default:
		return nil, errors.Errorf("unsupported set operator: %s", op.Type)
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: mergedColumns,
	}, nil
}

// extractTableSourceFromFromClause processes the FROM clause and returns all table sources
func (q *querySpanExtractor) extractTableSourceFromFromClause(fromClause postgresql.IFrom_clauseContext) ([]base.TableSource, error) {
	if fromClause == nil || fromClause.From_list() == nil {
		return nil, nil
	}

	var fromFieldList []base.TableSource
	for _, fromItem := range fromClause.From_list().AllTable_ref() {
		tableSource, err := q.extractTableSourceFromTableRef(fromItem)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from FROM item")
		}
		q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
		fromFieldList = append(fromFieldList, tableSource)
	}

	return fromFieldList, nil
}

// extractTableSourceFromTableRef extracts table source from a table reference in FROM clause
func (q *querySpanExtractor) extractTableSourceFromTableRef(tableRef postgresql.ITable_refContext) (base.TableSource, error) {
	if tableRef == nil {
		return nil, errors.New("nil table_ref")
	}

	var anchor base.TableSource
	var err error

	// Handle simple table reference (e.g., schema.table)
	if tableRef.Relation_expr() != nil {
		anchor, err = q.extractTableSourceFromRelationExpr(tableRef.Relation_expr(), tableRef.Opt_alias_clause())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from relation_expr")
		}
		if tableRef.Opt_alias_clause() != nil {
			anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to relation_expr")
			}
		}
	}

	// Handle function tables (e.g., generate_series, unnest)
	if tableRef.Func_table() != nil {
		anchor, err = q.extractTableSourceFromFuncTable(tableRef.Func_table(), tableRef.Func_alias_clause())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from func_table")
		}
		if tableRef.Func_alias_clause() != nil {
			anchor, err = applyFuncAliasClauseToTableSource(anchor, tableRef.Func_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to func_table")
			}
		}
	}

	// TODO(zp): Handle xmltable candidates.

	// Handle subquery (SELECT in parentheses)
	if tableRef.Select_with_parens() != nil {
		anchor, err = q.extractTableSourceFromSelectWithParens(tableRef.Select_with_parens())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract subquery")
		}

		// Apply alias if present
		if tableRef.Opt_alias_clause() != nil {
			anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to subquery")
			}
		}
	}

	if tableRef.Table_ref() != nil {
		anchor, err = q.extractTableSourceFromTableRef(tableRef.Table_ref())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract nested table_ref")
		}
		if len(tableRef.AllJoined_table()) == 0 && tableRef.Opt_alias_clause() != nil {
			anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to nested table_ref")
			}
		}
	}

	q.tableSourcesFrom = append(q.tableSourcesFrom, anchor)

	// Handle JOIN expressions
	if len(tableRef.AllJoined_table()) != 0 {
		joinedTables := tableRef.AllJoined_table()
		for i, join := range joinedTables {
			tableSource, err := q.extractTableSourceFromTableRef(join.Table_ref())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract joined table in JOIN")
			}
			// According to the grammar:
			// OPEN_PAREN table_ref joined_table? CLOSE_PAREN opt_alias_clause?
			// ) joined_table*
			// The alias appears before the second joined_table.
			// Determine join type and conditions
			naturalJoin := join.NATURAL() != nil
			var usingColumns []string
			if join.Join_qual() != nil && join.Join_qual().USING() != nil {
				usingColumns = normalizePostgreSQLNameList(join.Join_qual().Name_list())
			}
			anchor = q.joinTableSources(anchor, tableSource, naturalJoin, usingColumns)
			if i == 0 && tableRef.Opt_alias_clause() != nil && tableRef.Table_ref() != nil {
				anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to apply alias to table source")
				}
			}
		}
	}

	return anchor, nil
}

func (*querySpanExtractor) joinTableSources(left, right base.TableSource, naturalJoin bool, usingColumn []string) base.TableSource {
	leftSpanResult, rightSpanResult := left.GetQuerySpanResult(), right.GetQuerySpanResult()

	result := new(base.PseudoTable)

	if naturalJoin {
		leftSpanResultIdx, rightSpanResultIdx := make(map[string]int, len(leftSpanResult)), make(map[string]int, len(rightSpanResult))
		for i, spanResult := range leftSpanResult {
			leftSpanResultIdx[spanResult.Name] = i
		}
		for i, spanResult := range rightSpanResult {
			rightSpanResultIdx[spanResult.Name] = i
		}

		// NaturalJoin will merge the same column name field.
		for idx, spanResult := range leftSpanResult {
			if _, ok := rightSpanResultIdx[spanResult.Name]; ok {
				spanResult.SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[idx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}
		for _, spanResult := range rightSpanResult {
			if _, ok := leftSpanResultIdx[spanResult.Name]; !ok {
				result.Columns = append(result.Columns, spanResult)
			}
		}
	} else {
		if len(usingColumn) > 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool, len(usingColumn))
			for _, using := range usingColumn {
				usingMap[using] = true
			}

			result.Columns = append(result.Columns, leftSpanResult...)
			for _, spanResult := range rightSpanResult {
				if _, ok := usingMap[spanResult.Name]; !ok {
					result.Columns = append(result.Columns, spanResult)
				}
			}
		} else {
			result.Columns = append(result.Columns, leftSpanResult...)
			result.Columns = append(result.Columns, rightSpanResult...)
		}
	}

	return result
}

// extractTableSourceFromRelationExpr extracts table source from a relation expression (simple table)
func (q *querySpanExtractor) extractTableSourceFromRelationExpr(relationExpr postgresql.IRelation_exprContext, aliasClause postgresql.IOpt_alias_clauseContext) (base.TableSource, error) {
	if relationExpr == nil {
		return nil, errors.New("nil relation_expr")
	}

	qualifiedName := relationExpr.Qualified_name()
	if qualifiedName == nil {
		return nil, errors.New("relation_expr has no qualified_name")
	}

	// Extract the table name parts
	nameParts := NormalizePostgreSQLQualifiedName(qualifiedName)

	var schemaName, tableName string
	switch len(nameParts) {
	case 1:
		tableName = nameParts[0]
		schemaName = "" // Will be resolved using search path
	case 2:
		schemaName = nameParts[0]
		tableName = nameParts[1]
	case 3:
		// Cross-database query: database.schema.table
		// For now, we ignore the database part and use schema.table
		schemaName = nameParts[1]
		tableName = nameParts[2]
	default:
		return nil, errors.Errorf("invalid qualified name with %d parts", len(nameParts))
	}

	// Extract alias and column names from alias clause
	aliasName := tableName
	var columnAliases []string
	if aliasClause != nil && aliasClause.Table_alias_clause() != nil {
		tableAlias := aliasClause.Table_alias_clause()
		if tableAlias.Table_alias() != nil {
			aliasName = normalizePostgreSQLTableAlias(tableAlias.Table_alias())
		}
		if tableAlias.Name_list() != nil {
			columnAliases = normalizePostgreSQLNameList(tableAlias.Name_list())
		}
	}

	// Find the actual table schema with columns
	tableSource, err := q.findTableSchema(schemaName, tableName)
	if err != nil {
		return nil, err
	}

	// Apply alias if specified
	if aliasName != tableName || len(columnAliases) > 0 {
		// If we have column aliases, use them; otherwise use the original columns
		var columns []base.QuerySpanResult
		if len(columnAliases) > 0 {
			for _, alias := range columnAliases {
				columns = append(columns, base.QuerySpanResult{Name: alias})
			}
		} else {
			columns = tableSource.GetQuerySpanResult()
		}
		return &base.PseudoTable{
			Name:    aliasName,
			Columns: columns,
		}, nil
	}

	return tableSource, nil
}

func (q *querySpanExtractor) extractTableSourceFromUDF2(funcExprWindowless postgresql.IFunc_expr_windowlessContext, funcAlias postgresql.IFunc_alias_clauseContext) (base.TableSource, error) {
	if funcExprWindowless == nil {
		return nil, errors.New("nil func_expr_windowless")
	}

	schemaName, funcName, args, err := q.extractFunctionElementFromFuncExprWindowless(funcExprWindowless)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract function element from func_expr_windowless")
	}

	tableSource, err := q.findFunctionDefine(schemaName, funcName, len(args))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find function definition for %s.%s with %d args", schemaName, funcName, len(args))
	}
	if funcAlias != nil {
		tableSource, err = applyFuncAliasClauseToTableSource(tableSource, funcAlias)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply function alias clause")
		}
	}
	return tableSource, nil
}

// extractTableSourceFromFuncTable extracts table source from a function table
func (q *querySpanExtractor) extractTableSourceFromFuncTable(funcTable postgresql.IFunc_tableContext, funcAlias postgresql.IFunc_alias_clauseContext) (base.TableSource, error) {
	result := &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}

	if funcTable == nil {
		return result, nil
	}

	if funcTable.Func_expr_windowless() != nil {
		schemaName, funcName, args, err := q.extractFunctionElementFromFuncExprWindowless(funcTable.Func_expr_windowless())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract function name from func_expr_windowless")
		}
		if funcName == "" {
			return nil, errors.Wrapf(err, "empty function name in func_expr_windowless")
		}
		if schemaName == "" && IsSystemFunction(funcName, "") {
			// If the schemaName is empty, we try to match the system function first.
			return q.extractTableSourceFromSystemFunctionNew(funcName, args, funcAlias)
		}
		return q.extractTableSourceFromUDF2(funcTable.Func_expr_windowless(), funcAlias)
	}

	// The actual column count and names will be determined by the alias
	// For now, just return an empty pseudo table
	// The applyAliasToTableSource function will handle the column names

	// Check if this is a ROWS FROM expression
	if funcTable.ROWS() != nil && funcTable.Rowsfrom_list() != nil {
		// Handle ROWS FROM (...)
		// Each function in ROWS FROM produces columns
		// ROWS FROM expands multiple functions in parallel, creating a single result set where the first row contains
		// the first result from each function, the second row contains the second result from each function, and so on.
		tableSource := &base.PseudoTable{
			Columns: []base.QuerySpanResult{},
		}
		for _, item := range funcTable.Rowsfrom_list().AllRowsfrom_item() {
			if item.Func_expr_windowless() != nil {
				schemaName, funcName, args, err := q.extractFunctionElementFromFuncExprWindowless(funcTable.Func_expr_windowless())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract function name from func_expr_windowless")
				}
				if funcName == "" {
					return nil, errors.Wrapf(err, "empty function name in func_expr_windowless")
				}
				var t base.TableSource
				if schemaName == "" && IsSystemFunction(funcName, "") {
					// If the schemaName is empty, we try to match the system function first.
					t, err = q.extractTableSourceFromSystemFunctionNew(funcName, args, nil)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to extract table source from system function %s", funcName)
					}
				} else {
					t, err = q.extractTableSourceFromUDF2(item.Func_expr_windowless(), nil)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to extract table source from UDF %s", funcName)
					}
				}
				if t != nil {
					// Append columns from this function to the overall table source
					tableSource.Columns = append(tableSource.Columns, t.GetQuerySpanResult()...)
				}
			}
		}
		if funcAlias != nil {
			tableSource, err := applyFuncAliasClauseToTableSource(tableSource, funcAlias)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply function alias clause")
			}
			return tableSource, nil
		}
		return tableSource, nil
	}

	// For function tables, the columns are typically determined by:
	// 1. The function's return type (which we don't have access to without metadata)
	// 2. The alias provided (which will be applied later)
	// So we return an empty pseudo table that will be populated by the alias

	return result, nil
}

func applyAliasToTableSource(source base.TableSource, tableAliasName string, colAliases []string) (base.TableSource, error) {
	// Create a pseudo table with the alias
	pseudoTable := &base.PseudoTable{
		Name:    source.GetTableName(),
		Columns: source.GetQuerySpanResult(),
	}
	if tableAliasName != "" {
		pseudoTable.Name = tableAliasName
	}

	// Apply column aliases if provided
	if len(colAliases) > 0 {
		// Special handling for function tables that return no columns initially
		if len(pseudoTable.Columns) == 0 {
			// Create columns based on aliases
			for _, alias := range colAliases {
				pseudoTable.Columns = append(pseudoTable.Columns, base.QuerySpanResult{
					Name:          alias,
					SourceColumns: base.SourceColumnSet{},
				})
			}
		} else if len(colAliases) != len(pseudoTable.Columns) {
			// For non-empty tables, the alias count must match
			return nil, errors.Errorf("alias has %d columns but table has %d",
				len(colAliases), len(pseudoTable.Columns))
		} else {
			// Apply aliases to existing columns
			for i, alias := range colAliases {
				pseudoTable.Columns[i].Name = alias
			}
		}
	}
	return pseudoTable, nil
}

func applyTableAliasClauseToTableSource(source base.TableSource, tableAlias postgresql.ITable_alias_clauseContext) (base.TableSource, error) {
	aliasName := ""
	var colAliases []string

	// Get table alias name
	if tableAlias.Table_alias() != nil {
		aliasName = normalizePostgreSQLTableAlias(tableAlias.Table_alias())
	}

	// Get column aliases if present
	if tableAlias.Name_list() != nil {
		for _, name := range tableAlias.Name_list().AllName() {
			colAliases = append(colAliases, normalizePostgreSQLName(name))
		}
	}

	return applyAliasToTableSource(source, aliasName, colAliases)
}

func applyAliasClauseToTableSource(source base.TableSource, aliasClause postgresql.IAlias_clauseContext) (base.TableSource, error) {
	if aliasClause == nil {
		return source, nil
	}

	aliasName := ""
	var colAliases []string

	// Get table alias name
	if aliasClause.Colid() != nil {
		aliasName = NormalizePostgreSQLColid(aliasClause.Colid())
	}

	// Get column aliases if present
	if aliasClause.Name_list() != nil {
		for _, name := range aliasClause.Name_list().AllName() {
			colAliases = append(colAliases, normalizePostgreSQLName(name))
		}
	}

	return applyAliasToTableSource(source, aliasName, colAliases)
}

// applyAliasToTableSource applies an alias to a table source
func applyOptAliasClauseToTableSource(source base.TableSource, aliasClause postgresql.IOpt_alias_clauseContext) (base.TableSource, error) {
	if aliasClause == nil || aliasClause.Table_alias_clause() == nil {
		return source, nil
	}

	return applyTableAliasClauseToTableSource(source, aliasClause.Table_alias_clause())
}

func applyFuncAliasClauseToTableSource(source base.TableSource, funcAlias postgresql.IFunc_alias_clauseContext) (base.TableSource, error) {
	if funcAlias == nil {
		return source, nil
	}

	if funcAlias.Alias_clause() != nil {
		return applyAliasClauseToTableSource(source, funcAlias.Alias_clause())
	}

	aliasName := ""
	if funcAlias.Colid() != nil {
		aliasName = NormalizePostgreSQLColid(funcAlias.Colid())
	}

	var colAliases []string
	if funcAlias.Tablefuncelementlist() != nil {
		for _, name := range funcAlias.Tablefuncelementlist().AllTablefuncelement() {
			if name.Colid() != nil {
				colAliases = append(colAliases, NormalizePostgreSQLColid(name.Colid()))
			}
		}
	}

	return applyAliasToTableSource(source, aliasName, colAliases)
}

// extractColumnsFromTargetList processes the SELECT target list and returns the result columns.
// It handles *, table.*, column references, expressions, and aliases.
func (q *querySpanExtractor) extractColumnsFromTargetList(targetList postgresql.ITarget_listContext, fromFieldList []base.TableSource) ([]base.QuerySpanResult, error) {
	if targetList == nil {
		return nil, nil
	}

	var result []base.QuerySpanResult

	for _, targetEl := range targetList.AllTarget_el() {
		// Check if this is a star expansion
		if _, ok := targetEl.(*postgresql.Target_starContext); ok {
			// Handle SELECT * or SELECT table.*
			starColumns := q.handleStarExpansion(fromFieldList)
			result = append(result, starColumns...)
			continue
		}

		if _, ok := targetEl.(*postgresql.Target_columnrefContext); ok {
			columns, err := q.getColumnsFromColumnRef(targetEl.(*postgresql.Target_columnrefContext).Columnref())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process column reference in target element")
			}
			result = append(result, columns...)
			continue
		}

		// Check if this is a labeled target (expression with optional alias)
		if labelCtx, ok := targetEl.(*postgresql.Target_labelContext); ok {
			column, err := q.handleTargetLabel(labelCtx)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process target element")
			}
			result = append(result, column)
			continue
		}
	}

	return result, nil
}

// handleStarExpansion handles SELECT * expansion
func (*querySpanExtractor) handleStarExpansion(fromFieldList []base.TableSource) []base.QuerySpanResult {
	// Simple * - expand all columns from all tables in FROM clause
	var result []base.QuerySpanResult
	for _, tableSource := range fromFieldList {
		result = append(result, tableSource.GetQuerySpanResult()...)
	}
	return result
}

// handleTargetLabel handles a labeled target element (expression with optional alias)
func (q *querySpanExtractor) handleTargetLabel(labelCtx *postgresql.Target_labelContext) (base.QuerySpanResult, error) {
	// Get the expression
	expr := labelCtx.A_expr()
	if expr == nil {
		return base.QuerySpanResult{}, errors.New("target_label without expression")
	}

	// Extract source columns from the expression
	sourceColumns, err := q.extractSourceColumnSetFromAExpr(expr)
	if err != nil {
		return base.QuerySpanResult{}, errors.Wrapf(err, "failed to extract source columns from expression")
	}

	// Get the column name or alias
	columnName := ""
	if labelCtx.Target_alias() != nil {
		// Has an explicit alias
		targetAlias := labelCtx.Target_alias()
		if targetAlias.Collabel() != nil {
			columnName = normalizePostgreSQLCollabel(targetAlias.Collabel())
		} else if targetAlias.Bare_col_label() != nil {
			columnName = normalizePostgreSQLBareColLabel(targetAlias.Bare_col_label())
		}
	}

	if columnName == "" {
		// No alias, try to extract name from expression
		columnName = q.getFieldNameFromAExpr(expr)
		if columnName == "" {
			// Use default unknown column name
			columnName = pgUnknownFieldName
		}
	}

	return base.QuerySpanResult{
		Name:          columnName,
		SourceColumns: sourceColumns,
	}, nil
}

// getFieldNameFromAExpr attempts to extract a meaningful column name from an expression.
// Returns "?column?" if no meaningful name can be derived.
func (q *querySpanExtractor) getFieldNameFromAExpr(expr postgresql.IA_exprContext) string {
	if expr == nil {
		return pgUnknownFieldName
	}

	// Try to find a column reference in the expression
	columnName := q.findColumnNameInExpression(expr)
	if columnName != "" {
		return columnName
	}

	// Default to unknown column name for other expression types
	return pgUnknownFieldName
}

// findColumnNameInExpression recursively searches for a column name in an expression
func (q *querySpanExtractor) findColumnNameInExpression(node antlr.ParseTree) string {
	if node == nil {
		return ""
	}

	// Check if this is a column reference
	if columnRef, ok := node.(postgresql.IColumnrefContext); ok {
		// Get the column name from the columnref
		if columnRef.Colid() != nil {
			colName := NormalizePostgreSQLColid(columnRef.Colid())

			// Check if there's indirection (qualified name)
			if columnRef.Indirection() != nil {
				parts := normalizePostgreSQLIndirection(columnRef.Indirection())
				if len(parts) > 0 && parts[len(parts)-1] != "*" {
					// Return the last part as the column name
					return parts[len(parts)-1]
				}
			}
			// Simple column name
			return colName
		}
	}

	switch ctx := node.(type) {
	case postgresql.IColumnrefContext:
		if ctx.Colid() != nil {
			return NormalizePostgreSQLColid(ctx.Colid())
		}
	case postgresql.IFunc_exprContext:
		// return the function name as the column name
		_, funcName, _, err := q.extractFunctionElementFromFuncExpr(ctx)
		if err == nil && funcName != "" {
			return funcName
		}
		return pgUnknownFieldName
	case postgresql.IFunc_expr_windowlessContext:
		_, funcName, _, err := q.extractFunctionElementFromFuncExprWindowless(ctx)
		if err == nil && funcName != "" {
			return funcName
		}
		return pgUnknownFieldName
	}

	// Recursively search children
	for i := 0; i < node.GetChildCount(); i++ {
		if child, ok := node.GetChild(i).(antlr.ParseTree); ok {
			// Skip terminal nodes
			if _, isTerminal := child.(antlr.TerminalNode); isTerminal {
				continue
			}
			if name := q.findColumnNameInExpression(child); name != "" {
				return name
			}
		}
	}

	return ""
}

// extractSourceColumnSetFromAExpr extracts source columns from an a_expr node
func (q *querySpanExtractor) extractSourceColumnSetFromAExpr(expr postgresql.IA_exprContext) (base.SourceColumnSet, error) {
	if expr == nil {
		return base.SourceColumnSet{}, nil
	}

	// Use the generic node traversal
	return q.extractSourceColumnSetFromNode(expr)
}

// extractSourceColumnSetFromNode recursively extracts source columns from any parse tree node.
// This follows the pg_query_go pattern but adapted for ANTLR's tree structure.
func (q *querySpanExtractor) extractSourceColumnSetFromNode(node antlr.ParseTree) (base.SourceColumnSet, error) {
	if node == nil {
		return base.SourceColumnSet{}, nil
	}

	result := make(base.SourceColumnSet)

	// Handle specific node types (similar to pg_query_go's switch statement)
	switch ctx := node.(type) {
	case postgresql.IColumnrefContext:
		// Handle column reference directly
		return q.getSourceColumnSetFromColumnRef(ctx)
	case postgresql.IFunc_exprContext:
		// Handle function calls
		// Process function arguments recursively
		argsResult, err := q.extractSourceColumnSetFromFuncExpr(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract from function expression")
		}
		if udfResult, err := q.extractSourceColumnSetFromUDF2(ctx); err == nil {
			argsResult, _ = base.MergeSourceColumnSet(argsResult, udfResult)
		}
		return argsResult, nil
	case postgresql.ISelect_with_parensContext:
		sourceColumnSet := make(base.SourceColumnSet)
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:                     q.ctx,
			defaultDatabase:         q.defaultDatabase,
			searchPath:              q.searchPath,
			metaCache:               q.metaCache,
			gCtx:                    q.gCtx,
			ctes:                    q.ctes,
			outerTableSources:       append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:        []base.TableSource{},
			sourceColumnsInFunction: make(base.SourceColumnSet),
		}
		tableSource, err := subqueryExtractor.extractTableSourceFromSelectWithParens(ctx)
		if err != nil {
			return base.SourceColumnSet{}, errors.Wrapf(err, "failed to extract span result from subquery %s", ctx.GetText())
		}
		spanResult := tableSource.GetQuerySpanResult()

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return sourceColumnSet, nil
	// Add more specific cases as we discover we need them...

	default:
		// For any unhandled node type, recursively visit all children
		// This is the generic fallback that ensures we don't miss any column references
		for i := 0; i < node.GetChildCount(); i++ {
			if child, ok := node.GetChild(i).(antlr.ParseTree); ok {
				childSources, err := q.extractSourceColumnSetFromNode(child)
				if err != nil {
					// For now, propagate errors. Could also log and continue.
					return nil, err
				}
				result, _ = base.MergeSourceColumnSet(result, childSources)
			}
		}
	}

	return result, nil
}

func (q *querySpanExtractor) getColumnsFromColumnRef(ctx postgresql.IColumnrefContext) ([]base.QuerySpanResult, error) {
	if ctx == nil {
		return nil, nil
	}

	// Extract schema, table, column from the columnref
	var schemaName, tableName, columnName string

	// Get the base column name
	if ctx.Colid() != nil {
		columnName = NormalizePostgreSQLColid(ctx.Colid())
	}

	// Handle indirection (qualified names like table.column or schema.table.column)
	if ctx.Indirection() != nil {
		parts := normalizePostgreSQLIndirection(ctx.Indirection())

		// Check for star expansion (should have been handled elsewhere, but check anyway)
		if len(parts) > 0 && parts[len(parts)-1] == "*" {
			// This is table.* or schema.table.* - should not happen in regular column refs
			// Return empty set as this should be handled by star expansion logic
			switch len(parts) {
			case 1:
				// table.* case
				columnName = ""
				tableName = columnName
				schemaName = ""
			case 2:
				// schema.table.* case
				columnName = ""
				schemaName = columnName
				tableName = parts[0]
			default:
				// More complex indirection with star, ignore
				return []base.QuerySpanResult{}, nil
			}
			sources, ok := q.getAllTableColumnSources(schemaName, tableName)
			if !ok {
				return []base.QuerySpanResult{}, &base.ResourceNotFoundError{
					Err:      errors.New("cannot find the table for star expansion"),
					Database: &q.defaultDatabase,
					Schema:   &schemaName,
					Table:    &tableName,
				}
			}
			return sources, nil
		}

		switch len(parts) {
		case 1:
			// table.column case
			tableName = columnName
			columnName = parts[0]
		case 2:
			// schema.table.column case
			schemaName = columnName
			tableName = parts[0]
			columnName = parts[1]
		default:
			// More complex indirection, just take the last part as column
			if len(parts) > 0 {
				columnName = parts[len(parts)-1]
			}
		}
	}

	// Special handling for record/composite types
	// If we have a table name but no column name, it might be a whole-row reference
	if tableName != "" && columnName == "" {
		// This is a whole-row reference like "table" in row_to_json(table)
		// We need to return all columns from that table
		tableSource := q.findTableInFromNew(tableName)
		return tableSource.GetQuerySpanResult(), nil
	}

	// Look up the column source
	sources, columnSourceOk := q.getFieldColumnSource(schemaName, tableName, columnName)
	// The column ref in function call can be record type, such as row_to_json.
	if !columnSourceOk {
		if schemaName == "" {
			tableSource := q.findTableInFrom(tableName, columnName)
			if tableSource == nil {
				return []base.QuerySpanResult{}, &base.ResourceNotFoundError{
					Err:      errors.New("cannot find the column ref"),
					Database: &q.defaultDatabase,
					Schema:   &schemaName,
					Table:    &tableName,
					Column:   &columnName,
				}
			}
			querySpanResult := tableSource.GetQuerySpanResult()
			var result []base.QuerySpanResult
			result = append(result, querySpanResult...)
			return result, nil
		}
		return []base.QuerySpanResult{}, &base.ResourceNotFoundError{
			Err:      errors.New("cannot find the column ref"),
			Database: &q.defaultDatabase,
			Schema:   &schemaName,
			Table:    &tableName,
			Column:   &columnName,
		}
	}
	return []base.QuerySpanResult{
		{
			Name:          columnName,
			SourceColumns: sources,
		},
	}, nil
}

// getSourceColumnSetFromColumnRef extracts source columns from a column reference.
func (q *querySpanExtractor) getSourceColumnSetFromColumnRef(ctx postgresql.IColumnrefContext) (base.SourceColumnSet, error) {
	if ctx == nil {
		return base.SourceColumnSet{}, nil
	}

	columns, err := q.getColumnsFromColumnRef(ctx)
	if err != nil {
		return nil, err
	}

	result := make(base.SourceColumnSet)
	for _, col := range columns {
		result, _ = base.MergeSourceColumnSet(result, col.SourceColumns)
	}
	return result, nil
}

// findTableInFromNew finds a table source by name in the FROM clause or CTEs.
func (q *querySpanExtractor) findTableInFromNew(tableName string) base.TableSource {
	// Search in FROM clause tables
	for _, tableSource := range q.tableSourcesFrom {
		if tableSource.GetTableName() == tableName {
			return tableSource
		}
	}

	// Search in CTEs
	for i := len(q.ctes) - 1; i >= 0; i-- {
		if q.ctes[i].Name == tableName {
			return q.ctes[i]
		}
	}

	// Search in outer tables (for correlated subqueries)
	for _, tableSource := range q.outerTableSources {
		if tableSource.GetTableName() == tableName {
			return tableSource
		}
	}

	return nil
}

// extractSourceColumnSetFromFuncExpr extracts source columns from a function expression
func (q *querySpanExtractor) extractSourceColumnSetFromFuncExpr(funcExpr postgresql.IFunc_exprContext) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)

	// Handle function application
	if funcExpr.Func_application() != nil {
		funcApp := funcExpr.Func_application()
		// Process function arguments
		if funcApp.Func_arg_list() != nil {
			for _, arg := range funcApp.Func_arg_list().AllFunc_arg_expr() {
				if arg.A_expr() != nil {
					argSources, err := q.extractSourceColumnSetFromAExpr(arg.A_expr())
					if err != nil {
						return nil, err
					}
					result, _ = base.MergeSourceColumnSet(result, argSources)
				}
			}
		}
	}

	// Handle common subexpressions (COALESCE, GREATEST, LEAST, etc.)
	if funcExpr.Func_expr_common_subexpr() != nil {
		// Recursive extract all nodes in common subexpr
		for i := 0; i < funcExpr.Func_expr_common_subexpr().GetChildCount(); i++ {
			if child, ok := funcExpr.Func_expr_common_subexpr().GetChild(i).(antlr.ParseTree); ok {
				// Skip terminal nodes
				if _, isTerminal := child.(antlr.TerminalNode); isTerminal {
					continue
				}
				childSources, err := q.extractSourceColumnSetFromNode(child)
				if err != nil {
					return nil, err
				}
				result, _ = base.MergeSourceColumnSet(result, childSources)
			}
		}
		// Handle XML functions that may have additional expressions
		// TODO: Add more specific handling for XML functions if needed
	}

	// TODO: Handle window functions if needed

	filterClause := funcExpr.Filter_clause()
	if filterClause != nil {
		r, err := q.extractSourceColumnSetFromNode(filterClause)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, r)
	}
	return result, nil
}

func getSelectNoParensFromSelectStmt(selectstmt postgresql.ISelectstmtContext) postgresql.ISelect_no_parensContext {
	if selectstmt == nil {
		return nil
	}

	// Direct check
	if selectstmt.Select_no_parens() != nil {
		return selectstmt.Select_no_parens()
	}

	// Check nested select_with_parens
	if selectstmt.Select_with_parens() != nil {
		return getSelectNoParensFromSelectWithParens(selectstmt.Select_with_parens())
	}

	return nil
}

func getSelectNoParensFromSelectWithParens(selectWithParens postgresql.ISelect_with_parensContext) postgresql.ISelect_no_parensContext {
	if selectWithParens == nil {
		return nil
	}

	// Direct check
	if selectWithParens.Select_no_parens() != nil {
		return selectWithParens.Select_no_parens()
	}

	// Recurse for nested parentheses
	if selectWithParens.Select_with_parens() != nil {
		return getSelectNoParensFromSelectWithParens(selectWithParens.Select_with_parens())
	}

	return nil
}

func (q *querySpanExtractor) extractSourceColumnSetFromUDF2(funcExpr postgresql.IFunc_exprContext) (base.SourceColumnSet, error) {
	schemaName, funcName, err := q.extractFunctionNameFromFuncExpr(funcExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract function name")
	}
	if schemaName == "" && IsSystemFunction(funcName, "") {
		return base.SourceColumnSet{}, nil
	}
	nArgs := q.extractNFunctionArgsFromFuncExpr(funcExpr)

	result := make(base.SourceColumnSet)
	tableSource, err := q.findFunctionDefine(schemaName, funcName, nArgs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find function definition for %s.%s/%d", schemaName, funcName, nArgs)
	}
	for _, span := range tableSource.GetQuerySpanResult() {
		result, _ = base.MergeSourceColumnSet(result, span.SourceColumns)
	}
	return result, nil
}

func (*querySpanExtractor) extractFunctionElementFromFuncExpr(funcExpr postgresql.IFunc_exprContext) (string, string, []antlr.ParseTree, error) {
	args := getArgumentsFromFunctionExpr(funcExpr)
	switch {
	case funcExpr.Func_application() != nil:
		funcName := funcExpr.Func_application().Func_name()
		names := NormalizePostgreSQLFuncName(funcName)
		switch len(names) {
		case 2:
			return names[0], names[1], args, nil
		case 1:
			return "", names[0], args, nil
		default:
			return "", "", nil, errors.New("invalid function name")
		}
	case funcExpr.Json_aggregate_func() != nil:
		if funcExpr.Json_aggregate_func().JSON_OBJECTAGG() != nil {
			return "", "json_objectagg", args, nil
		} else if funcExpr.Json_aggregate_func().JSON_ARRAYAGG() != nil {
			return "", "json_arrayagg", args, nil
		}
	case funcExpr.Func_expr_common_subexpr() != nil:
		// Builtin function, return the first token text as function name
		if funcExpr.Func_expr_common_subexpr().GetStart() != nil {
			return "", strings.ToLower(funcExpr.Func_expr_common_subexpr().GetStart().GetText()), args, nil
		}
	default:
	}
	return "", "", nil, errors.New("unable to extract function name")
}

func (*querySpanExtractor) extractFunctionElementFromFuncExprWindowless(funcExprWindowless postgresql.IFunc_expr_windowlessContext) (string, string, []antlr.ParseTree, error) {
	args := getArgumentsFromFunctionExprWindowless(funcExprWindowless)
	switch {
	case funcExprWindowless.Func_application() != nil:
		funcName := funcExprWindowless.Func_application().Func_name()
		names := NormalizePostgreSQLFuncName(funcName)
		switch len(names) {
		case 2:
			return names[0], names[1], args, nil
		case 1:
			return "", names[0], args, nil
		default:
			return "", "", nil, errors.New("invalid function name")
		}
	case funcExprWindowless.Json_aggregate_func() != nil:
		if funcExprWindowless.Json_aggregate_func().JSON_OBJECTAGG() != nil {
			return "", "json_objectagg", args, nil
		} else if funcExprWindowless.Json_aggregate_func().JSON_ARRAYAGG() != nil {
			return "", "json_arrayagg", args, nil
		}
	case funcExprWindowless.Func_expr_common_subexpr() != nil:
		// Builtin function, return the first token text as function name
		if funcExprWindowless.Func_expr_common_subexpr().GetStart() != nil {
			return "", strings.ToLower(funcExprWindowless.Func_expr_common_subexpr().GetStart().GetText()), args, nil
		}
	default:
	}
	return "", "", nil, errors.New("unable to extract function name")
}

func (*querySpanExtractor) extractFunctionNameFromFuncExpr(funcExpr postgresql.IFunc_exprContext) (string, string, error) {
	switch {
	case funcExpr.Func_application() != nil:
		funcName := funcExpr.Func_application().Func_name()
		if funcName.Colid() != nil {
			names := NormalizePostgreSQLFuncName(funcName)
			switch len(names) {
			case 2:
				return names[0], names[1], nil
			case 1:
				return "", names[0], nil
			default:
				return "", "", errors.New("invalid function name")
			}
		} else if t := funcName.Type_function_name(); t != nil {
			// No indirection, just a simple function name
			s := t.GetText()
			if t.Identifier() != nil {
				s = normalizePostgreSQLIdentifier(t.Identifier())
			}
			return "", s, nil
		} else {
			// Builtin function without schema
			return "", strings.ToLower(funcName.GetText()), nil
		}
	case funcExpr.Json_aggregate_func() != nil:
		if funcExpr.Json_aggregate_func().JSON_OBJECTAGG() != nil {
			return "", "json_objectagg", nil
		} else if funcExpr.Json_aggregate_func().JSON_ARRAYAGG() != nil {
			return "", "json_arrayagg", nil
		}
	case funcExpr.Func_expr_common_subexpr() != nil:
		// Builtin function, return the first token text as function name
		if funcExpr.Func_expr_common_subexpr().GetStart() != nil {
			return "", strings.ToLower(funcExpr.Func_expr_common_subexpr().GetStart().GetText()), nil
		}
	default:
	}
	return "", "", errors.New("unable to extract function name")
}

func (*querySpanExtractor) extractNFunctionArgsFromFuncExpr(funcExpr postgresql.IFunc_exprContext) int {
	if funcExpr == nil {
		return 0
	}

	// This handles generic function calls like my_func(a, b).
	if fa := funcExpr.Func_application(); fa != nil {
		if fa.Func_arg_list() != nil {
			return len(fa.Func_arg_list().AllFunc_arg_expr())
		}
		// This handles cases like my_func(a) without a list rule.
		if fa.Func_arg_expr() != nil {
			return 1
		}
		return 0
	}

	// This handles JSON aggregate functions like JSON_AGG(expr).
	if f := funcExpr.Json_aggregate_func(); f != nil {
		// Based on Postgres grammar, these functions (JSON_AGG, JSONB_AGG, JSON_OBJECT_AGG, JSONB_OBJECT_AGG)
		// typically take one or two main arguments. Returning 1 is a reasonable assumption for the primary expression argument.
		return 1
	}

	// This handles a large set of built-in functions with special syntax.
	if f := funcExpr.Func_expr_common_subexpr(); f != nil {
		// Functions with no parentheses have zero arguments (e.g., CURRENT_DATE, CURRENT_USER).
		if f.OPEN_PAREN() == nil {
			return 0
		}

		// --- List-based Functions ---
		// These functions take a variable number of arguments in a simple list.
		if f.COALESCE() != nil || f.GREATEST() != nil || f.LEAST() != nil || f.XMLCONCAT() != nil {
			if list := f.Expr_list(); list != nil {
				return len(list.AllA_expr())
			}
			return 0
		}

		// --- Functions with Specific Keyword-based Syntax ---

		if f.COLLATION() != nil { // COLLATION FOR (a_expr)
			return 1
		}

		if f.CURRENT_TIME() != nil || f.CURRENT_TIMESTAMP() != nil || f.LOCALTIME() != nil || f.LOCALTIMESTAMP() != nil {
			// e.g., CURRENT_TIME(precision)
			if f.Iconst() != nil {
				return 1
			}
			return 0
		}

		if f.CAST() != nil || f.TREAT() != nil { // CAST(expr AS type), TREAT(expr AS type)
			return 2
		}

		if f.EXTRACT() != nil { // EXTRACT(field FROM source)
			if f.Extract_list() != nil {
				// extract_list contains 'extract_arg' and 'a_expr'.
				return 2
			}
			return 0
		}

		if f.NORMALIZE() != nil { // NORMALIZE(string [, form])
			count := 1 // 'string' argument is mandatory
			if f.Unicode_normal_form() != nil {
				count++
			}
			return count
		}

		if f.OVERLAY() != nil { // OVERLAY(string PLACING new_substring FROM start [FOR count])
			if list := f.Overlay_list(); list != nil {
				return len(list.AllA_expr()) // Counts all expressions
			}
			return 0
		}

		if f.POSITION() != nil { // POSITION(substring IN string)
			if f.Position_list() != nil {
				return 2
			}
			return 0
		}

		if f.SUBSTRING() != nil { // SUBSTRING(string [FROM start] [FOR count])
			if list := f.Substr_list(); list != nil {
				return len(list.AllA_expr())
			}
			return 0
		}

		if f.TRIM() != nil { // TRIM([LEADING|TRAILING|BOTH] [characters] FROM string)
			if list := f.Trim_list(); list != nil {
				count := 0
				if list.A_expr() != nil { // The optional 'characters' argument
					count++
				}
				if list.Expr_list() != nil { // The mandatory 'string' argument
					count += len(list.Expr_list().AllA_expr())
				}
				return count
			}
			return 0
		}

		if f.NULLIF() != nil { // NULLIF(value1, value2)
			return len(f.AllA_expr()) // Expects 2
		}

		// --- XML Functions ---
		if f.XMLELEMENT() != nil {
			count := 0
			if f.Collabel() != nil {
				count++
			} // The element name
			if f.Xml_attributes() != nil {
				count += len(f.Xml_attributes().Xml_attribute_list().AllXml_attribute_el())
			}
			if f.Expr_list() != nil {
				count += len(f.Expr_list().AllA_expr())
			}
			return count
		}
		if f.XMLEXISTS() != nil {
			return 2
		} // XMLEXISTS(xpath PASSING xml)
		if f.XMLFOREST() != nil {
			if list := f.Xml_attribute_list(); list != nil {
				return len(list.AllXml_attribute_el())
			}
			return 0
		}
		if f.XMLPARSE() != nil { // XMLPARSE(DOCUMENT|CONTENT string_value [WHITESPACE option])
			count := 1 // string_value
			if f.Xml_whitespace_option() != nil {
				count++
			}
			return count
		}
		if f.XMLPI() != nil { // XMLPI(NAME target [, content])
			count := 1 // target
			if len(f.AllA_expr()) > 0 {
				count++
			}
			return count
		}
		if f.XMLROOT() != nil {
			count := 2 // xml and version
			if f.Opt_xml_root_standalone() != nil {
				count++
			}
			return count
		}
		if f.XMLSERIALIZE() != nil {
			return 2
		} // XMLSERIALIZE(CONTENT value AS type)

		// --- JSON Functions ---
		if f.JSON_OBJECT() != nil {
			if list := f.Func_arg_list(); list != nil {
				return len(list.AllFunc_arg_expr())
			}
			if list := f.Json_name_and_value_list(); list != nil {
				return len(list.AllJson_name_and_value())
			}
			return 0
		}
		if f.JSON_ARRAY() != nil {
			if list := f.Json_value_expr_list(); list != nil {
				return len(list.AllJson_value_expr())
			}
			if f.Select_no_parens() != nil {
				return 1
			} // subquery
			return 0
		}
		if f.JSON() != nil || f.JSON_SCALAR() != nil || f.JSON_SERIALIZE() != nil {
			return 1
		}
		if f.MERGE_ACTION() != nil {
			return 0
		}
		if f.JSON_QUERY() != nil || f.JSON_EXISTS() != nil || f.JSON_VALUE() != nil {
			count := 2 // json_value_expr, a_expr
			return count
		}
	}

	return 0
}

func getArgumentsFromFunctionExpr(funcExpr postgresql.IFunc_exprContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if funcExpr == nil {
		return result
	}
	if fa := funcExpr.Func_application(); fa != nil {
		if fa.Func_arg_list() != nil {
			result = append(result, getArgumentsFromFuncArgList(fa.Func_arg_list())...)
		}
		if fa.Func_arg_expr() != nil {
			result = append(result, fa.Func_arg_expr())
		}
		// TODO(zp): Handle *, ALL, DISTINCT if needed
		return result
	}
	if f := funcExpr.Func_expr_common_subexpr(); f != nil {
		if f.COLLATION() != nil {
			result = append(result, f.A_expr(0))
		} else if f.CURRENT_TIME() != nil || f.CURRENT_TIMESTAMP() != nil || f.LOCALTIME() != nil || f.LOCALTIMESTAMP() != nil {
			if f.Iconst() != nil {
				result = append(result, f.Iconst())
			}
		} else if f.CAST() != nil || f.TREAT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.EXTRACT() != nil {
			if f.Extract_list() != nil {
				result = append(result, f.Extract_list().A_expr())
			}
		} else if f.NORMALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.OVERLAY() != nil {
			if list := f.Overlay_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.POSITION() != nil {
			if list := f.Position_list(); list != nil {
				for _, child := range list.GetChildren() {
					if expr, ok := child.(postgresql.IA_exprContext); ok {
						result = append(result, expr)
					}
				}
			}
		} else if f.SUBSTRING() != nil {
			if list := f.Substr_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.TRIM() != nil {
			if list := f.Trim_list(); list != nil {
				if list.A_expr() != nil {
					result = append(result, list.A_expr())
				}
				if list.Expr_list() != nil {
					for _, expr := range list.Expr_list().AllA_expr() {
						result = append(result, expr)
					}
				}
			}
		} else if f.NULLIF() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.COALESCE() != nil || f.GREATEST() != nil || f.LEAST() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLELEMENT() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLEXISTS() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.XMLFOREST() != nil {
			if list := f.Xml_attribute_list(); list != nil {
				for _, attr := range list.AllXml_attribute_el() {
					result = append(result, attr.A_expr())
				}
			}
		} else if f.XMLPARSE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLPI() != nil {
			if len(f.AllA_expr()) > 0 {
				result = append(result, f.A_expr(0))
			}
		} else if f.XMLROOT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLSERIALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_OBJECT() != nil {
			if list := f.Func_arg_list(); list != nil {
				result = append(result, getArgumentsFromFuncArgList(list)...)
			}
			if list := f.Json_name_and_value_list(); list != nil {
				result = append(result, getArgumentsFromJSONNameAndValueList(list)...)
			}
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJSONValueExprList(list)...)
			}
		} else if f.JSON_ARRAY() != nil {
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJSONValueExprList(list)...)
			}
			if list := f.Select_no_parens(); list != nil {
				result = append(result, list)
			}
		} else if f.JSON() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_SCALAR() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_SERIALIZE() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_QUERY() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_EXISTS() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_VALUE() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		}
	}
	return result
}

func getArgumentsFromFunctionExprWindowless(funcExprWindowless postgresql.IFunc_expr_windowlessContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if funcExprWindowless == nil {
		return result
	}
	if fa := funcExprWindowless.Func_application(); fa != nil {
		if fa.Func_arg_list() != nil {
			result = append(result, getArgumentsFromFuncArgList(fa.Func_arg_list())...)
		}
		if fa.Func_arg_expr() != nil {
			result = append(result, fa.Func_arg_expr())
		}
		// TODO(zp): Handle *, ALL, DISTINCT if needed
		return result
	}
	if f := funcExprWindowless.Func_expr_common_subexpr(); f != nil {
		if f.COLLATION() != nil {
			result = append(result, f.A_expr(0))
		} else if f.CURRENT_TIME() != nil || f.CURRENT_TIMESTAMP() != nil || f.LOCALTIME() != nil || f.LOCALTIMESTAMP() != nil {
			if f.Iconst() != nil {
				result = append(result, f.Iconst())
			}
		} else if f.CAST() != nil || f.TREAT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.EXTRACT() != nil {
			if f.Extract_list() != nil {
				result = append(result, f.Extract_list().A_expr())
			}
		} else if f.NORMALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.OVERLAY() != nil {
			if list := f.Overlay_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.POSITION() != nil {
			if list := f.Position_list(); list != nil {
				for _, child := range list.GetChildren() {
					if expr, ok := child.(postgresql.IA_exprContext); ok {
						result = append(result, expr)
					}
				}
			}
		} else if f.SUBSTRING() != nil {
			if list := f.Substr_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.TRIM() != nil {
			if list := f.Trim_list(); list != nil {
				if list.A_expr() != nil {
					result = append(result, list.A_expr())
				}
				if list.Expr_list() != nil {
					for _, expr := range list.Expr_list().AllA_expr() {
						result = append(result, expr)
					}
				}
			}
		} else if f.NULLIF() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.COALESCE() != nil || f.GREATEST() != nil || f.LEAST() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLELEMENT() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLEXISTS() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.XMLFOREST() != nil {
			if list := f.Xml_attribute_list(); list != nil {
				for _, attr := range list.AllXml_attribute_el() {
					result = append(result, attr.A_expr())
				}
			}
		} else if f.XMLPARSE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLPI() != nil {
			if len(f.AllA_expr()) > 0 {
				result = append(result, f.A_expr(0))
			}
		} else if f.XMLROOT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLSERIALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_OBJECT() != nil {
			if list := f.Func_arg_list(); list != nil {
				result = append(result, getArgumentsFromFuncArgList(list)...)
			}
			if list := f.Json_name_and_value_list(); list != nil {
				result = append(result, getArgumentsFromJSONNameAndValueList(list)...)
			}
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJSONValueExprList(list)...)
			}
		} else if f.JSON_ARRAY() != nil {
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJSONValueExprList(list)...)
			}
			if list := f.Select_no_parens(); list != nil {
				result = append(result, list)
			}
		} else if f.JSON() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_SCALAR() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_SERIALIZE() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_QUERY() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_EXISTS() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_VALUE() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		}
	}
	return result
}

func getArgumentsFromFuncArgList(funcArgList postgresql.IFunc_arg_listContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if funcArgList == nil {
		return result
	}
	for _, argExpr := range funcArgList.AllFunc_arg_expr() {
		result = append(result, argExpr)
	}
	return result
}

func getArgumentsFromJSONNameAndValueList(jsonNameAndValueList postgresql.IJson_name_and_value_listContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if jsonNameAndValueList == nil {
		return result
	}
	for _, nameAndValue := range jsonNameAndValueList.AllJson_name_and_value() {
		result = append(result, nameAndValue)
	}
	return result
}

func getArgumentsFromJSONValueExprList(jsonValueExprList postgresql.IJson_value_expr_listContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if jsonValueExprList == nil {
		return result
	}
	for _, valueExpr := range jsonValueExprList.AllJson_value_expr() {
		result = append(result, valueExpr)
	}
	return result
}

func (q *querySpanExtractor) extractTableSourceFromSystemFunctionNew(funcName string, args []antlr.ParseTree, alias postgresql.IFunc_alias_clauseContext) (base.TableSource, error) {
	switch strings.ToLower(funcName) {
	case generateSeries:
		// https://neon.com/postgresql/postgresql-tutorial/postgresql-generate_series
		tableSource := &base.PseudoTable{
			Name: generateSeries,
			Columns: []base.QuerySpanResult{
				{
					Name:          generateSeries,
					SourceColumns: make(base.SourceColumnSet, 0),
				},
			},
		}
		return applyFuncAliasClauseToTableSource(tableSource, alias)
	case generateSubscripts:
		tableSource := &base.PseudoTable{
			Name: generateSubscripts,
			Columns: []base.QuerySpanResult{
				{
					Name:          generateSubscripts,
					SourceColumns: make(base.SourceColumnSet, 0),
				},
			},
		}
		return applyFuncAliasClauseToTableSource(tableSource, alias)
	case unnest:
		tableSource := &base.PseudoTable{
			Name:    unnest,
			Columns: []base.QuerySpanResult{},
		}
		if alias == nil || (alias.Alias_clause() != nil && alias.Alias_clause().Name_list() == nil) {
			for range args {
				tableSource.Columns = append(tableSource.Columns, base.QuerySpanResult{
					Name:          unnest,
					SourceColumns: make(base.SourceColumnSet, 0),
				})
			}
		}
		return applyFuncAliasClauseToTableSource(tableSource, alias)
	case jsonbEach, jsonEach, jsonbEachText, jsonEachText:
		// Should be only called while jsonb_each act as table source.
		// SELECT * FROM json_test, jsonb_each(jb) AS hh(key, value) WHERE id = 1;
		var tableSource base.TableSource = &base.PseudoTable{
			Name: "",
			Columns: []base.QuerySpanResult{
				{
					Name:          "key",
					SourceColumns: make(base.SourceColumnSet, 0),
				},
				{
					Name:          "value",
					SourceColumns: make(base.SourceColumnSet, 0),
				},
			},
		}
		tableSource, err := applyFuncAliasClauseToTableSource(tableSource, alias)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply alias clause to %s", funcName)
		}

		if len(args) == 0 {
			return nil, errors.Errorf("unexpected empty args for function %s", funcName)
		}
		fieldArg := args[0]
		set, err := q.extractSourceColumnSetFromNode(fieldArg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract source columns from argument of %s", funcName)
		}
		for i := range tableSource.GetQuerySpanResult() {
			tableSource.GetQuerySpanResult()[i].SourceColumns, _ = base.MergeSourceColumnSet(tableSource.GetQuerySpanResult()[i].SourceColumns, set)
		}
		return tableSource, nil
	case jsonToRecordset, jsonbToRecordset:
		if alias == nil {
			return nil, &base.TypeNotSupportedError{
				Extra: fmt.Sprintf("function %s must explicitly define the structure of the record with an AS clause", funcName),
				Type:  "function",
				Name:  funcName,
			}
		}

		sourceColumn := make(base.SourceColumnSet)
		for _, arg := range args {
			argSources, err := q.extractSourceColumnSetFromNode(arg)
			if err != nil {
				return nil, err
			}
			sourceColumn, _ = base.MergeSourceColumnSet(sourceColumn, argSources)
		}

		tableSource := &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}

		tableName := ""
		var columnNames []string
		if alias.Colid() != nil {
			tableName = NormalizePostgreSQLColid(alias.Colid())
			if alias.Tablefuncelementlist() != nil {
				for _, name := range alias.Tablefuncelementlist().AllTablefuncelement() {
					if name.Colid() != nil {
						columnNames = append(columnNames, NormalizePostgreSQLColid(name.Colid()))
					}
				}
			}
		} else if a := alias.Alias_clause(); a != nil {
			// Get table alias name
			if a.Colid() != nil {
				tableName = NormalizePostgreSQLColid(a.Colid())
			}

			// Get column aliases if present
			if a.Name_list() != nil {
				for _, name := range a.Name_list().AllName() {
					columnNames = append(columnNames, normalizePostgreSQLName(name))
				}
			}
		}

		for _, colName := range columnNames {
			tableSource.Columns = append(tableSource.Columns, base.QuerySpanResult{
				Name:          colName,
				SourceColumns: sourceColumn,
			})
		}
		tableSource.Name = tableName
		return tableSource, nil
	case jsonPopulateRecord, jsonbPopulateRecord, jsonPopulateRecordset, jsonbPopulateRecordset,
		jsonToRecord, jsonbToRecord:
		return nil, &base.TypeNotSupportedError{
			Extra: fmt.Sprintf("Unsupport function %s", funcName),
			Type:  "function",
			Name:  funcName,
		}
	default:
		// For unknown functions, continue with the generic handling below
		if alias == nil {
			return nil, &base.TypeNotSupportedError{
				Extra: "Use system function result as the table source must have the alias clause to specify table and columns name",
				Type:  "function",
				Name:  funcName,
			}
		}

		tableName := ""
		var columnNames []string
		if alias.Colid() != nil {
			tableName = NormalizePostgreSQLColid(alias.Colid())
			if alias.Tablefuncelementlist() != nil {
				for _, name := range alias.Tablefuncelementlist().AllTablefuncelement() {
					if name.Colid() != nil {
						columnNames = append(columnNames, NormalizePostgreSQLColid(name.Colid()))
					}
				}
			}
		} else if a := alias.Alias_clause(); a != nil {
			// Get table alias name
			if a.Colid() != nil {
				tableName = NormalizePostgreSQLColid(a.Colid())
			}

			// Get column aliases if present
			if a.Name_list() != nil {
				for _, name := range a.Name_list().AllName() {
					columnNames = append(columnNames, normalizePostgreSQLName(name))
				}
			}
		}
		tableSource := &base.PseudoTable{
			Name:    tableName,
			Columns: []base.QuerySpanResult{},
		}
		for _, colName := range columnNames {
			tableSource.Columns = append(tableSource.Columns, base.QuerySpanResult{
				Name:          colName,
				SourceColumns: make(base.SourceColumnSet, 0),
			})
		}
		return tableSource, nil
	}
}
