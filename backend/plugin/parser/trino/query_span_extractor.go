package trino

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/trino-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"
	"github.com/bytebase/bytebase/backend/store/model"
)

// querySpanExtractor extracts query spans from Trino statements.
type querySpanExtractor struct {
	ctx                 context.Context
	gCtx                base.GetQuerySpanContext
	defaultDatabase     string
	defaultSchema       string
	statement           string
	ignoreCaseSensitive bool

	// Metadata cache
	metaCache map[string]*model.DatabaseMetadata

	// Parse results
	sourceColumns    base.SourceColumnSet
	predicateColumns base.SourceColumnSet

	// Table sources for resolving columns
	tableSourcesFrom  []base.TableSource
	outerTableSources []base.TableSource

	// CTEs for handling WITH clauses
	ctes []*base.PseudoTable
}

// newQuerySpanExtractor creates a new Trino query span extractor.
func newQuerySpanExtractor(defaultDatabase, defaultSchema string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		defaultDatabase:     defaultDatabase,
		defaultSchema:       defaultSchema,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
		metaCache:           make(map[string]*model.DatabaseMetadata),
		sourceColumns:       make(base.SourceColumnSet),
		predicateColumns:    make(base.SourceColumnSet),
		tableSourcesFrom:    []base.TableSource{},
		outerTableSources:   []base.TableSource{},
		ctes:                []*base.PseudoTable{},
	}
}

// getQuerySpan extracts the query span for a Trino statement.
func (q *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	q.ctx = ctx
	q.statement = statement
	q.sourceColumns = make(base.SourceColumnSet)
	q.predicateColumns = make(base.SourceColumnSet)
	q.tableSourcesFrom = []base.TableSource{}
	q.ctes = []*base.PseudoTable{}

	// Parse the statement
	result, err := ParseTrino(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse Trino statement")
	}

	if result.Tree == nil {
		return nil, errors.New("failed to parse Trino statement, no parse tree found")
	}

	// Determine query type
	queryType, isExplainAnalyze := getQueryType(result.Tree, false)

	// Collect accessed table resources
	accessTables := make(base.SourceColumnSet)
	for resource := range q.extractAccessedTables(result.Tree) {
		accessTables[resource] = true
	}

	// For non-SELECT queries, return a basic span
	if queryType != base.Select && queryType != base.Explain && queryType != base.SelectInfoSchema {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessTables,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Special handling for EXPLAIN/ANALYZE
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
		var resourceNotFound *parsererror.ResourceNotFoundError
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

	// Process all table references to ensure we have full column information
	fullSourceColumns := q.expandTableReferencesToColumns(accessTables)

	// Create the final query span with properly processed predicate columns
	// For data access control, we need to ensure that all predicates are fully qualified
	fullPredicateColumns := make(base.SourceColumnSet)

	// Add every predicate column from our tracked set
	for col := range q.predicateColumns {
		// For each predicate column, find the fully qualified version in source columns
		if col.Column != "" {
			for sourceCol := range fullSourceColumns {
				// Match column name and ensure table matches if specified
				if sourceCol.Column == col.Column {
					if col.Table == "" || col.Table == sourceCol.Table {
						fullPredicateColumns[sourceCol] = true
					}
				}
			}
		}
	}

	// Return the complete query span
	return &base.QuerySpan{
		Type:             queryType,
		SourceColumns:    fullSourceColumns,
		PredicateColumns: fullPredicateColumns,
		Results:          listener.results,
	}, nil
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
				for _, col := range tableMeta.GetColumns() {
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
		return nil, &parsererror.ResourceNotFoundError{Database: &database}
	}

	// Fetch metadata using the provided function
	_, meta, err := q.gCtx.GetDatabaseMetadataFunc(q.ctx, q.gCtx.InstanceID, database)
	if err != nil {
		var resourceNotFound *parsererror.ResourceNotFoundError
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

// addPredicateColumn adds a column to the tracked predicate columns.
func (q *querySpanExtractor) addPredicateColumn(col base.ColumnResource) {
	q.predicateColumns[col] = true
}

// findTableSchema locates a table or view and returns its metadata.
func (q *querySpanExtractor) findTableSchema(db, schema, name string) (*model.TableMetadata, error) {
	// Get database metadata
	metadata, err := q.getDatabaseMetadata(db)
	if err != nil {
		return nil, err
	}

	// Get schema metadata
	schemaMeta := metadata.GetSchema(schema)
	if schemaMeta == nil {
		return nil, &parsererror.ResourceNotFoundError{
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
	var viewMeta *model.ViewMetadata
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
	return nil, &parsererror.ResourceNotFoundError{
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
