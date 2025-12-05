package trino

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/trino"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// querySpanExtractor extracts query spans from Trino statements.
// This follows the TSQL pattern for consistency.
type querySpanExtractor struct {
	ctx context.Context

	defaultDatabase     string
	defaultSchema       string
	ignoreCaseSensitive bool

	gCtx base.GetQuerySpanContext

	// CTEs for handling WITH clauses - following TSQL pattern
	ctes []*base.PseudoTable

	// Table sources for resolving columns - following TSQL pattern
	tableSourcesFrom []base.TableSource

	// Parse results - following TSQL pattern
	sourceColumns    base.SourceColumnSet
	predicateColumns base.SourceColumnSet

	// Outer table sources for correlated subqueries
	outerTableSources []base.TableSource

	// Metadata cache
	metaCache map[string]*model.DatabaseMetadata
}

// newQuerySpanExtractor creates a new Trino query span extractor.
// This follows the TSQL constructor pattern.
func newQuerySpanExtractor(defaultDatabase, defaultSchema string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		defaultSchema:       defaultSchema,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
		metaCache:           make(map[string]*model.DatabaseMetadata),
		sourceColumns:       make(base.SourceColumnSet),
		predicateColumns:    make(base.SourceColumnSet),
	}
}

// getQuerySpan extracts the query span for a Trino statement.
// This method follows the TSQL pattern for consistency.
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.sourceColumns = make(base.SourceColumnSet)
	q.predicateColumns = make(base.SourceColumnSet)
	q.tableSourcesFrom = []base.TableSource{}
	q.ctes = []*base.PseudoTable{}

	// Parse the statement
	parseResults, err := ParseTrino(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Trino statement")
	}

	// Query span extraction expects exactly one statement
	if len(parseResults) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(parseResults))
	}

	result := parseResults[0]

	if result.Tree == nil {
		return nil, errors.New("failed to parse Trino statement, no parse tree found")
	}

	// Get accessed tables for basic query type determination
	accessTables := q.extractAccessedTables(result.Tree)
	queryType, isExplainAnalyze := getQueryType(result.Tree)

	// For non-SELECT queries, return basic span following TSQL pattern
	if queryType != base.Select && queryType != base.Explain && queryType != base.SelectInfoSchema {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessTables,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Special handling for EXPLAIN following TSQL pattern
	if isExplainAnalyze || queryType == base.Explain {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessTables,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Use ANTLR listener to extract source columns and results
	listener := newTrinoQuerySpanListener(q)
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)

	if listener.err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(listener.err, &resourceNotFound) {
			return &base.QuerySpan{
				Type:          queryType,
				SourceColumns: accessTables,
				Results:       []base.QuerySpanResult{},
				NotFoundError: resourceNotFound,
			}, nil
		}
		return nil, listener.err
	}

	// Expand table references to columns following TSQL pattern
	fullSourceColumns := q.expandTableReferencesToColumns(accessTables)

	// Process predicate columns following TSQL pattern
	fullPredicateColumns := q.expandPredicateColumns(fullSourceColumns)

	// Expand SELECT * results following TSQL pattern
	expandedResults := q.expandSelectAsteriskResults(listener.results, fullSourceColumns)

	return &base.QuerySpan{
		Type:             queryType,
		SourceColumns:    fullSourceColumns,
		PredicateColumns: fullPredicateColumns,
		Results:          expandedResults,
	}, nil
}

// expandPredicateColumns expands predicate columns to their fully qualified forms.
// This follows the TSQL pattern for consistent predicate handling.
func (q *querySpanExtractor) expandPredicateColumns(fullSourceColumns base.SourceColumnSet) base.SourceColumnSet {
	fullPredicateColumns := make(base.SourceColumnSet)

	for col := range q.predicateColumns {
		if col.Column != "" {
			// Find all fully qualified versions in source columns
			for sourceCol := range fullSourceColumns {
				if sourceCol.Column == col.Column {
					// Match table if specified, otherwise accept any table
					if col.Table == "" || col.Table == sourceCol.Table {
						fullPredicateColumns[sourceCol] = true
					}
				}
			}
		}
	}

	return fullPredicateColumns
}

// expandSelectAsteriskResults expands SELECT * results into individual column results
// This is similar to how TSQL handles SELECT * queries
func (q *querySpanExtractor) expandSelectAsteriskResults(results []base.QuerySpanResult, fullSourceColumns base.SourceColumnSet) []base.QuerySpanResult {
	var expandedResults []base.QuerySpanResult

	for _, result := range results {
		if result.SelectAsterisk && result.Name == "*" {
			// Use table sources to get columns in the correct order (same as TSQL approach)
			for _, tableSource := range q.tableSourcesFrom {
				tableColumns := tableSource.GetQuerySpanResult()

				for _, columnResult := range tableColumns {
					// Only include columns that are in our source columns set
					columnIncluded := false
					for sourceCol := range columnResult.SourceColumns {
						if _, exists := fullSourceColumns[sourceCol]; exists {
							columnIncluded = true
							break
						}
					}

					if columnIncluded {
						expandedResults = append(expandedResults, columnResult)
					}
				}
			}
		} else {
			// Keep non-asterisk results as-is
			expandedResults = append(expandedResults, result)
		}
	}

	return expandedResults
}

// expandTableReferencesToColumns expands table references to individual columns
func (q *querySpanExtractor) expandTableReferencesToColumns(accessTables base.SourceColumnSet) base.SourceColumnSet {
	fullSourceColumns := make(base.SourceColumnSet)

	// First, copy all explicitly collected source columns
	for col := range q.sourceColumns {
		fullSourceColumns[col] = true
	}

	// Expand table references to columns
	for resource := range accessTables {
		// Check if this is a table reference (has no column set)
		if resource.Column == "" {
			// Get the database metadata to find all columns for this table
			db, schema, table := resource.Database, resource.Schema, resource.Table
			tableMeta, err := q.findTableSchema(db, schema, table)
			if err == nil && tableMeta != nil {
				// Add each column from the table as a source column with full path
				for _, col := range tableMeta.GetProto().GetColumns() {
					colResource := base.ColumnResource{
						Database: db,
						Schema:   schema,
						Table:    table,
						Column:   col.Name,
					}
					fullSourceColumns[colResource] = true
				}
			}
		} else {
			// This is already a column reference, keep it
			fullSourceColumns[resource] = true
		}
	}

	return fullSourceColumns
}

// extractAccessedTables analyzes the parse tree to find all table resources accessed
func (q *querySpanExtractor) extractAccessedTables(tree antlr.Tree) base.SourceColumnSet {
	resources := make(base.SourceColumnSet)
	tableListener := &tableExtractorListener{
		extractor: q,
		resources: resources,
	}

	antlr.ParseTreeWalkerDefault.Walk(tableListener, tree)
	return resources
}

// getDatabaseMetadata fetches metadata for the given database.
func (q *querySpanExtractor) getDatabaseMetadata(database string) (*model.DatabaseMetadata, error) {
	if database == "" {
		database = q.defaultDatabase
	}

	// Return cached metadata if available
	if meta, ok := q.metaCache[database]; ok {
		return meta, nil
	}

	// Skip if metadata function not provided (for testing)
	if q.gCtx.GetDatabaseMetadataFunc == nil {
		return nil, &base.ResourceNotFoundError{Database: &database}
	}

	// Fetch metadata using the provided function
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, database)
	if err != nil {
		var resourceNotFound *base.ResourceNotFoundError
		if errors.As(err, &resourceNotFound) {
			return nil, err
		}
		return nil, errors.Wrapf(err, "failed to get database metadata for database: %s", database)
	}

	if meta == nil {
		// Return empty metadata for testing purposes
		emptyMeta := &model.DatabaseMetadata{}
		q.metaCache[database] = emptyMeta
		return emptyMeta, nil
	}

	// Cache and return the metadata
	q.metaCache[database] = meta
	return meta, nil
}

// addSourceColumn adds a column to the tracked source columns.
func (q *querySpanExtractor) addSourceColumn(col base.ColumnResource) {
	q.sourceColumns[col] = true
}

// findTableSchema locates a table or view and returns its metadata.
func (q *querySpanExtractor) findTableSchema(db, schema, name string) (*model.TableMetadata, error) {
	// Get database metadata
	metadata, err := q.getDatabaseMetadata(db)
	if err != nil {
		return nil, err
	}

	// Get schema metadata
	schemaMeta := metadata.GetSchemaMetadata(schema)
	if schemaMeta == nil {
		return nil, &base.ResourceNotFoundError{
			Database: &db,
			Schema:   &schema,
		}
	}

	// Look for table
	var tableMeta *model.TableMetadata
	if q.ignoreCaseSensitive {
		for _, tblName := range schemaMeta.ListTableNames() {
			if strings.EqualFold(tblName, name) {
				tableMeta = schemaMeta.GetTable(tblName)
				break
			}
		}
	} else {
		tableMeta = schemaMeta.GetTable(name)
	}

	if tableMeta != nil {
		return tableMeta, nil
	}

	// Look for view
	var viewMeta *storepb.ViewMetadata
	if q.ignoreCaseSensitive {
		for _, viewName := range schemaMeta.ListViewNames() {
			if strings.EqualFold(viewName, name) {
				viewMeta = schemaMeta.GetView(viewName)
				break
			}
		}
	} else {
		viewMeta = schemaMeta.GetView(name)
	}

	if viewMeta != nil {
		// For views, return a table metadata with columns from the view
		tableMeta := &model.TableMetadata{}
		// In a more complete implementation, we would add columns from the view here
		return tableMeta, nil
	}

	// Not found
	return nil, &base.ResourceNotFoundError{
		Database: &db,
		Schema:   &schema,
		Table:    &name,
	}
}

// tableExtractorListener extracts table resources from the parse tree
type tableExtractorListener struct {
	parser.BaseTrinoParserListener

	extractor *querySpanExtractor
	resources base.SourceColumnSet
}

// EnterTableName is called when the parser enters a table name
func (l *tableExtractorListener) EnterTableName(ctx *parser.TableNameContext) {
	if ctx.QualifiedName() == nil {
		return
	}

	db, schema, table := ExtractDatabaseSchemaName(
		ctx.QualifiedName(),
		l.extractor.defaultDatabase,
		l.extractor.defaultSchema,
	)

	// Add a resource for the table
	l.resources[base.ColumnResource{
		Database: db,
		Schema:   schema,
		Table:    table,
	}] = true
}
