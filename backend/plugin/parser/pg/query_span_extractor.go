package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"

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

type functionDefinition struct {
	schemaName string
	metadata   *model.FunctionMetadata
}

func (q *querySpanExtractor) findFunctionDefine(schemaName, funcName string, nArgs int) (base.TableSource, error) {
	dbSchema, err := q.getDatabaseMetadata(q.defaultDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.defaultDatabase)
	}
	if dbSchema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.defaultDatabase,
		}
	}
	var funcs []*functionDefinition
	searchPath := q.searchPath
	if schemaName != "" {
		searchPath = []string{schemaName}
	}
	schemas, functions := dbSchema.SearchFunctions(searchPath, funcName)
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
		return nil, &parsererror.ResourceNotFoundError{
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
		return nil, &parsererror.FunctionNotSupportedError{
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
	definition := function.GetProto().GetDefinition()
	res, err := ParsePostgreSQL(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse function definition: %s", definition)
	}

	// Navigate: root -> stmtblock -> stmt -> createfunctionstmt
	root, ok := res.Tree.(*postgresql.RootContext)
	if !ok {
		return nil, errors.Errorf("expecting RootContext but got %T", res.Tree)
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
		if function.metadata.GetProto().GetName() != funcName {
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
	res, err := ParsePostgreSQL(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse function definition: %s", definition)
	}

	// Navigate: root -> stmtblock -> stmt -> createfunctionstmt
	root, ok := res.Tree.(*postgresql.RootContext)
	if !ok {
		return nil, errors.Errorf("expecting RootContext but got %T", res.Tree)
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
	span, err := newQ.getQuerySpanNew(q.ctx, asBody)
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
		span, err := newQ.getQuerySpanNew(q.ctx, sql)
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
	res, err := ParsePostgreSQL(definition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PLpgSQL function body for function %s", name)
	}

	listener := &plpgSQLListener{
		q:         q,
		variables: make(map[string]*base.QuerySpanResult),
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, res.Tree)
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
			return language, &parsererror.TypeNotSupportedError{
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
			return "", &parsererror.TypeNotSupportedError{
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
	span, err := newQ.getQuerySpanNew(l.q.ctx, fmt.Sprintf("SELECT %s", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Sql_expression())))
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
	l.span, l.err = newQ.getQuerySpanNew(l.q.ctx, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Selectstmt()))
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
	dbSchema, err := q.getDatabaseMetadata(q.defaultDatabase)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", q.defaultDatabase)
	}
	if dbSchema == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.defaultDatabase,
		}
	}
	searchPath := q.searchPath
	if schemaName != "" {
		searchPath = []string{schemaName}
	}
	tableSchemaName, table := dbSchema.SearchTable(searchPath, tableName)
	viewSchemaName, view := dbSchema.SearchView(searchPath, tableName)
	materializedViewSchemaName, materializedView := dbSchema.SearchMaterializedView(searchPath, tableName)
	foreignTableSchemaName, foreignTable := dbSchema.SearchExternalTable(searchPath, tableName)
	sequenceSchemaName, sequence := dbSchema.SearchSequence(searchPath, tableName)

	if table == nil && view == nil && foreignTable == nil && materializedView == nil && sequence == nil {
		return nil, &parsererror.ResourceNotFoundError{
			Database: &q.defaultDatabase,
			Schema:   &schemaName,
			Table:    &tableName,
		}
	}

	if table != nil {
		var columns []string
		for _, column := range table.GetColumns() {
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
		for _, column := range foreignTable.GetColumns() {
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
			Name:     view.GetProto().Name,
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
			Name:     materializedView.GetProto().Name,
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
			Name:     sequence.GetProto().Name,
			Columns:  columns,
		}, nil
	}
	return nil, nil
}

func (q *querySpanExtractor) getColumnsForView(definition string) ([]base.QuerySpanResult, error) {
	newQ := newQuerySpanExtractor(q.defaultDatabase, q.searchPath, q.gCtx)
	span, err := newQ.getQuerySpanNew(q.ctx, definition)
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
	span, err := newQ.getQuerySpanNew(q.ctx, definition)
	if err != nil {
		return nil, err
	}
	if span.NotFoundError != nil {
		return nil, span.NotFoundError
	}
	return span.Results, nil
}
