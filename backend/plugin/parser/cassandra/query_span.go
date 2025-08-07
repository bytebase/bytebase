package cassandra

import (
	"context"
	"errors"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	// Register our GetQuerySpan function for Cassandra engine
	// This tells Bytebase to use our function when it needs to extract query spans for Cassandra
	base.RegisterGetQuerySpan(storepb.Engine_CASSANDRA, GetQuerySpan)
}

// GetQuerySpan returns the query span for the given CQL statement.
// This is the main entry point that Bytebase will call
func GetQuerySpan(
	ctx context.Context,
	gCtx base.GetQuerySpanContext,
	statement, keyspace, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	// Create an extractor with our context
	extractor := newQuerySpanExtractor(keyspace, gCtx, ignoreCaseSensitive)

	// Extract the query span from the statement
	return extractor.getQuerySpan(ctx, statement)
}

// querySpanExtractor handles the extraction of table and column information from CQL queries
type querySpanExtractor struct {
	keyspace            string // The current keyspace (database in Cassandra terms)
	gCtx                base.GetQuerySpanContext
	ignoreCaseSensitive bool
}

// newQuerySpanExtractor creates a new extractor instance
func newQuerySpanExtractor(keyspace string, gCtx base.GetQuerySpanContext, ignoreCaseSensitive bool) *querySpanExtractor {
	return &querySpanExtractor{
		keyspace:            keyspace,
		gCtx:                gCtx,
		ignoreCaseSensitive: ignoreCaseSensitive,
	}
}

// getQuerySpan extracts the query span from a CQL statement
func (e *querySpanExtractor) getQuerySpan(ctx context.Context, statement string) (*base.QuerySpan, error) {
	// Normalize the statement
	normalizedStmt := strings.TrimSpace(statement)
	upperStmt := strings.ToUpper(normalizedStmt)

	// Check if it's a SELECT statement
	if !strings.HasPrefix(upperStmt, "SELECT") {
		// Not a SELECT, so no masking needed
		return &base.QuerySpan{
			Type: base.QueryTypeUnknown,
		}, nil
	}

	// Parse the SELECT statement to extract table and columns
	tableName, columns, err := e.parseSelectStatement(ctx, normalizedStmt)
	if err != nil {
		return nil, err
	}

	// Build the query span with the extracted information
	span := &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: make(base.SourceColumnSet),
	}

	// Add each column to the source columns and results
	for _, col := range columns {
		columnResource := base.ColumnResource{
			Database: e.keyspace,
			Table:    tableName,
			Column:   col,
		}

		// Mark this column as being accessed
		span.SourceColumns[columnResource] = true

		// Add to the result set (what will be returned to the client)
		span.Results = append(span.Results, base.QuerySpanResult{
			Name:          col,
			SourceColumns: base.SourceColumnSet{columnResource: true},
		})
	}

	return span, nil
}

// parseSelectStatement extracts the table name and column list from a CQL SELECT statement
func (e *querySpanExtractor) parseSelectStatement(ctx context.Context, stmt string) (string, []string, error) {
	// Convert to uppercase for keyword searching, but keep original for extracting values
	upperStmt := strings.ToUpper(stmt)

	// 1. Find where SELECT ends and FROM begins
	selectIdx := strings.Index(upperStmt, "SELECT")
	fromIdx := strings.Index(upperStmt, "FROM")

	// Validate we have both keywords in the right order
	if selectIdx == -1 || fromIdx == -1 || fromIdx <= selectIdx {
		return "", nil, errors.New("invalid CQL SELECT statement: missing SELECT or FROM")
	}

	// 2. Extract the column list between SELECT and FROM
	// Start after "SELECT" (position + 6 for the length of "SELECT")
	columnsPart := strings.TrimSpace(stmt[selectIdx+6 : fromIdx])

	// 3. Extract the table name after FROM
	// Start after "FROM" (position + 4 for the length of "FROM")
	tablePartStart := fromIdx + 4
	tablePartEnd := len(stmt)

	// Find where the table name ends (at WHERE, LIMIT, ORDER BY, or end of statement)
	for _, keyword := range []string{"WHERE", "LIMIT", "ORDER", "ALLOW"} {
		if idx := strings.Index(upperStmt[tablePartStart:], keyword); idx != -1 {
			if idx < tablePartEnd-tablePartStart {
				tablePartEnd = tablePartStart + idx
			}
		}
	}

	tableName := strings.TrimSpace(stmt[tablePartStart:tablePartEnd])

	// Remove semicolon if present at the end
	tableName = strings.TrimSuffix(tableName, ";")
	tableName = strings.TrimSpace(tableName)

	// Handle keyspace.table format
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			// If keyspace is specified in the query, use just the table name
			tableName = parts[1]
		}
	}

	// Parse the columns from columnsPart
	var columns []string
	if strings.TrimSpace(columnsPart) == "*" {
		// SELECT * case - try to get actual columns from metadata
		if e.gCtx.GetDatabaseMetadataFunc != nil {
			columns = e.getTableColumns(ctx, tableName)
		}
		// If we couldn't get metadata, keep the wildcard
		if len(columns) == 0 {
			columns = []string{"*"}
		}
	} else {
		// Split by comma to get individual columns
		colParts := strings.Split(columnsPart, ",")
		for _, col := range colParts {
			col = strings.TrimSpace(col)
			// Handle column aliases (AS keyword or just space)
			// Examples: "name AS username" or "name username"
			if asIdx := strings.Index(strings.ToUpper(col), " AS "); asIdx != -1 {
				col = strings.TrimSpace(col[:asIdx])
			} else if spaceIdx := strings.Index(col, " "); spaceIdx != -1 {
				// Make sure it's not a function call like "COUNT(*)"
				if !strings.Contains(col[:spaceIdx], "(") {
					col = strings.TrimSpace(col[:spaceIdx])
				}
			}
			columns = append(columns, col)
		}
	}

	return tableName, columns, nil
}

// getTableColumns retrieves the column names for a table from metadata
func (e *querySpanExtractor) getTableColumns(ctx context.Context, tableName string) []string {
	if e.gCtx.GetDatabaseMetadataFunc == nil {
		return nil
	}

	// Get database metadata - for Cassandra, the keyspace is the "database"
	// GetDatabaseMetadataFunc expects (instanceID, databaseName)
	_, metadata, err := e.gCtx.GetDatabaseMetadataFunc(ctx, e.gCtx.InstanceID, e.keyspace)
	if err != nil || metadata == nil {
		return nil
	}

	// For Cassandra, there's a single schema with empty name
	// Try the empty schema first (which is what Cassandra uses)
	schema := metadata.GetSchema("")
	if schema == nil {
		// If empty schema doesn't exist, try the first available schema
		schemaNames := metadata.ListSchemaNames()
		for _, schemaName := range schemaNames {
			schema = metadata.GetSchema(schemaName)
			if schema != nil {
				break
			}
		}
	}

	if schema == nil {
		return nil
	}

	// Get the table from the schema
	table := schema.GetTable(tableName)
	if table != nil {
		var columns []string
		for _, col := range table.GetColumns() {
			columns = append(columns, col.Name)
		}
		return columns
	}

	return nil
}
