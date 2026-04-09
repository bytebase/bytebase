package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

const (
	schemaFetchTimeout = 30 * time.Second
	// schemaTableLimit is the user-facing cap on tables returned per schema in
	// bulk modes (columns/details). The server is asked for limit+1 so that
	// returning exactly `limit` vs. truncated-at-limit can be disambiguated.
	schemaTableLimit = 200
	// schemaCandidateFetchLimit bounds the second (no-filter) fetch used to build
	// candidate suggestions on TABLE_NOT_FOUND. Large enough to surface any reasonable
	// typo, small enough to cap wire payload.
	schemaCandidateFetchLimit = 500
	// schemaCandidateResultCap is the max number of candidate names returned in the
	// TABLE_NOT_FOUND error body.
	schemaCandidateResultCap = 10
)

// Detail levels for the `include` parameter.
const (
	schemaIncludeSummary = "summary"
	schemaIncludeColumns = "columns"
	schemaIncludeDetails = "details"
)

// SchemaInput is the input for the get_schema tool.
type SchemaInput struct {
	// Database is the database name or substring to match (same rules as query_database).
	Database string `json:"database"`
	// Instance narrows database resolution to a specific instance.
	Instance string `json:"instance,omitempty"`
	// Project narrows database resolution to a specific project.
	Project string `json:"project,omitempty"`
	// Schema filters to a single PostgreSQL/MSSQL schema; ignored on engines without schemas.
	Schema string `json:"schema,omitempty"`
	// Table filters to a single table; implies include="details" when Include is empty.
	Table string `json:"table,omitempty"`
	// Include controls output detail level: "summary" (default), "columns", or "details".
	Include string `json:"include,omitempty"`
}

// SchemaOutput is the output for the get_schema tool.
type SchemaOutput struct {
	Database string          `json:"database"` // full resource name
	Engine   string          `json:"engine"`
	Schemas  []SchemaSection `json:"schemas,omitempty"` // used when Table is empty
	Table    *TableDetail    `json:"table,omitempty"`   // used when Table is set
}

// SchemaSection holds per-schema metadata.
type SchemaSection struct {
	Name           string       `json:"name"` // "public" or "" for MySQL
	Tables         []TableEntry `json:"tables"`
	Views          []string     `json:"views,omitempty"`
	FunctionCount  int          `json:"functionCount,omitempty"`
	ProcedureCount int          `json:"procedureCount,omitempty"`
	Truncated      bool         `json:"truncated,omitempty"`   // true when len(Tables) hit the per-schema limit
	TablesShown    int          `json:"tablesShown,omitempty"` // == len(Tables) when Truncated; omitted otherwise
}

// TableEntry describes a table. Which fields are populated depends on the include level.
type TableEntry struct {
	Name        string        `json:"name"`
	RowCount    int64         `json:"rowCount"`
	ColumnCount int           `json:"columnCount,omitempty"` // summary only; in columns/details modes, len(Columns) is the count
	Comment     string        `json:"comment,omitempty"`
	Columns     []ColumnEntry `json:"columns,omitempty"`     // columns+
	Indexes     []IndexEntry  `json:"indexes,omitempty"`     // details+
	ForeignKeys []FKEntry     `json:"foreignKeys,omitempty"` // details+
}

// ColumnEntry describes a single column.
type ColumnEntry struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	PrimaryKey bool   `json:"primaryKey,omitempty"`
	Default    string `json:"default,omitempty"` // details+
	Comment    string `json:"comment,omitempty"` // details+
}

// IndexEntry describes an index on a table.
type IndexEntry struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Unique  bool     `json:"unique"`
	Columns []string `json:"columns"`
}

// FKEntry describes a foreign key constraint.
type FKEntry struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedTable   string   `json:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns"`
}

// TableDetail is an alias for a details-level TableEntry used when the caller drills
// into a specific table.
type TableDetail = TableEntry

// --- Proto response types (decoded from GetDatabaseMetadata) ---
//
// These mirror the fields we consume from the DatabaseMetadata protobuf. We only
// decode what the tool actually needs; extra fields are ignored by json.Unmarshal.

type databaseMetadata struct {
	Name    string           `json:"name"`
	Schemas []schemaMetadata `json:"schemas"`
}

type schemaMetadata struct {
	Name       string          `json:"name"`
	Tables     []tableMetadata `json:"tables"`
	Views      []viewMetadata  `json:"views"`
	Functions  []namedMetadata `json:"functions"`
	Procedures []namedMetadata `json:"procedures"`
}

type tableMetadata struct {
	Name        string               `json:"name"`
	Columns     []columnMetadata     `json:"columns"`
	Indexes     []indexMetadata      `json:"indexes"`
	RowCount    json.Number          `json:"rowCount"`
	Comment     string               `json:"comment"`
	ForeignKeys []foreignKeyMetadata `json:"foreignKeys"`
}

type columnMetadata struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
	Comment  string `json:"comment"`
}

type indexMetadata struct {
	Name        string   `json:"name"`
	Expressions []string `json:"expressions"`
	Type        string   `json:"type"`
	Unique      bool     `json:"unique"`
	Primary     bool     `json:"primary"`
}

type foreignKeyMetadata struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedTable   string   `json:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns"`
}

type viewMetadata struct {
	Name string `json:"name"`
}

type namedMetadata struct {
	Name string `json:"name"`
}

// getSchemaDescription is the description for the get_schema tool.
const getSchemaDescription = `Inspect a Bytebase database's schema.

| Parameter | Required | Description |
|-----------|----------|-------------|
| database  | Yes      | Database name or substring (e.g., "employee_db" or "employee") |
| instance  | No       | Instance name to narrow resolution |
| project   | No       | Project name to narrow resolution |
| schema    | No       | PostgreSQL/MSSQL schema (e.g., "public") |
| table     | No       | Drill into a single table; implies include="details" |
| include   | No       | Detail level: "summary" (default), "columns", "details" |

**Usage:**
- First call: get_schema(database="employee") → compact overview
- Drill down: get_schema(database="employee", table="orders") → full table detail
- Write a query: get_schema(database="employee", include="columns") → all tables with columns

**Notes:**
- Metadata is auto-synced on access. The first call to a stale database may take longer.
- For databases with many tables, columns/details modes return up to 200 tables PER SCHEMA.
  Use schema= or table= to narrow further.
- Column masking metadata is only returned when table= is set.
- Requires bb.databases.getSchema permission.`

func (s *Server) registerSchemaTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_schema",
		Description: getSchemaDescription,
	}, s.handleGetSchema)
}

func (s *Server) handleGetSchema(ctx context.Context, req *mcp.CallToolRequest, input SchemaInput) (*mcp.CallToolResult, any, error) {
	if input.Database == "" {
		return nil, nil, errors.New("database is required")
	}

	// Default include based on whether a specific table was requested.
	include := input.Include
	if include == "" {
		if input.Table != "" {
			include = schemaIncludeDetails
		} else {
			include = schemaIncludeSummary
		}
	}
	if include != schemaIncludeSummary && include != schemaIncludeColumns && include != schemaIncludeDetails {
		return nil, nil, errors.Errorf("invalid include value %q (must be summary|columns|details)", input.Include)
	}

	// Resolve database (shared resolver with query_database).
	resolveCtx, resolveCancel := context.WithTimeout(ctx, resolveTimeout)
	defer resolveCancel()

	resolved, err := s.resolveDatabase(resolveCtx, input.Database, input.Instance, input.Project)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if resolved.ambiguous {
		picked, elicitErr := s.elicitDatabaseChoice(ctx, req, resolved)
		if elicitErr != nil {
			return formatAmbiguousResult(input.Database, resolved.candidates), nil, nil
		}
		resolved = picked
	}

	// Fetch metadata with server-side filter + limit.
	fetchCtx, fetchCancel := context.WithTimeout(ctx, schemaFetchTimeout)
	defer fetchCancel()

	metadata, err := s.fetchMetadata(fetchCtx, resolved.resourceName, buildMetadataFilter(input.Schema, input.Table), limitForInclude(include, input.Table))
	if err != nil {
		return formatToolError(err), nil, nil
	}

	// Transform → SchemaOutput.
	if input.Table != "" {
		matches := findTableMatches(metadata, input.Table)
		switch {
		case len(matches) == 0:
			// Re-fetch without table filter to surface candidates.
			candidates := s.lookupTableCandidates(fetchCtx, resolved.resourceName, input.Schema, input.Table)
			return formatTableNotFound(input.Database, input.Table, candidates), nil, nil
		case len(matches) > 1:
			// Same table name in multiple schemas. Don't silently pick one.
			schemas := make([]string, 0, len(matches))
			for _, m := range matches {
				schemas = append(schemas, m.schema)
			}
			return formatAmbiguousTable(input.Database, input.Table, schemas), nil, nil
		}
		match := matches[0]
		entry := buildTableEntry(match.table, schemaIncludeDetails)
		output := &SchemaOutput{
			Database: resolved.resourceName,
			Engine:   resolved.engine,
			Table:    &entry,
		}
		text := formatSchemaOutput(output, include)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, output, nil
	}

	sections := transformSchemas(metadata, include)
	output := &SchemaOutput{
		Database: resolved.resourceName,
		Engine:   resolved.engine,
		Schemas:  sections,
	}
	text := formatSchemaOutput(output, include)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}

// buildMetadataFilter builds a CEL filter string for GetDatabaseMetadataRequest.
// Returns the empty string when no narrowing is needed.
func buildMetadataFilter(schema, table string) string {
	var parts []string
	if schema != "" {
		parts = append(parts, fmt.Sprintf("schema == %q", schema))
	}
	if table != "" {
		parts = append(parts, fmt.Sprintf("table == %q", table))
	}
	return strings.Join(parts, " && ")
}

// limitForInclude returns the per-schema `limit` value to send to the server.
// Summary and single-table modes don't need a limit. Bulk modes request
// schemaTableLimit+1 so the client can disambiguate exactly-200 from
// truncated-at-200.
func limitForInclude(include, table string) int32 {
	if table != "" {
		return 0
	}
	if include == schemaIncludeColumns || include == schemaIncludeDetails {
		return int32(schemaTableLimit + 1)
	}
	return 0
}

// fetchMetadata calls DatabaseService/GetDatabaseMetadata and decodes the response.
func (s *Server) fetchMetadata(ctx context.Context, resourceName, filter string, limit int32) (*databaseMetadata, error) {
	body := map[string]any{
		"name": resourceName + "/metadata",
	}
	if filter != "" {
		body["filter"] = filter
	}
	if limit > 0 {
		body["limit"] = limit
	}

	resp, err := s.apiRequest(ctx, "/bytebase.v1.DatabaseService/GetDatabaseMetadata", body)
	if err != nil {
		return nil, &toolError{
			Code:       "SCHEMA_FETCH_ERROR",
			Message:    fmt.Sprintf("metadata request failed: %s", err.Error()),
			Suggestion: "check network connectivity and try again",
		}
	}

	if resp.Status >= 400 {
		return nil, translateMetadataError(resp)
	}

	var metadata databaseMetadata
	if err := json.Unmarshal(resp.Body, &metadata); err != nil {
		return nil, errors.Wrap(err, "failed to parse database metadata")
	}
	return &metadata, nil
}

// translateMetadataError maps GetDatabaseMetadata HTTP errors into toolError codes.
func translateMetadataError(resp *apiResponse) error {
	errMsg := parseError(resp.Body)
	switch resp.Status {
	case http.StatusNotFound:
		return &toolError{
			Code:       "DATABASE_NOT_FOUND",
			Message:    "database not found",
			Suggestion: "check the database name or use search_api to list available databases",
		}
	case http.StatusForbidden, http.StatusUnauthorized:
		return &toolError{
			Code:       "PERMISSION_DENIED",
			Message:    "you don't have permission to read this database's schema",
			Suggestion: "ask your workspace admin to grant you the bb.databases.getSchema permission",
		}
	case http.StatusInternalServerError:
		suggestion := "check instance connectivity in the Bytebase UI"
		if errMsg != "" {
			suggestion = fmt.Sprintf("%s; backend error: %s", suggestion, errMsg)
		}
		return &toolError{
			Code:       "SCHEMA_SYNC_FAILED",
			Message:    "the database is reachable but the schema sync failed",
			Suggestion: suggestion,
		}
	default:
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.Status)
		}
		return &toolError{
			Code:       "SCHEMA_FETCH_ERROR",
			Message:    errMsg,
			Suggestion: "check the database name and try again",
		}
	}
}

// tableMatch holds a single (schema, table) match found during lookup.
type tableMatch struct {
	schema string
	table  *tableMetadata
}

// findTableMatches returns every (schema, table) pair whose table name equals
// the requested name. The caller decides how to handle 0, 1, or multiple matches.
func findTableMatches(metadata *databaseMetadata, table string) []tableMatch {
	var matches []tableMatch
	for i := range metadata.Schemas {
		schema := &metadata.Schemas[i]
		for j := range schema.Tables {
			t := &schema.Tables[j]
			if t.Name == table {
				matches = append(matches, tableMatch{schema: schema.Name, table: t})
			}
		}
	}
	return matches
}

// lookupTableCandidates issues a second GetDatabaseMetadata call (no table filter)
// to build a list of candidate table names for a TABLE_NOT_FOUND error.
// Bounded by schemaCandidateFetchLimit per schema on the wire and
// schemaCandidateResultCap on the returned slice.
func (s *Server) lookupTableCandidates(ctx context.Context, resourceName, schemaFilter, missing string) []string {
	filter := buildMetadataFilter(schemaFilter, "")
	metadata, err := s.fetchMetadata(ctx, resourceName, filter, int32(schemaCandidateFetchLimit))
	if err != nil || metadata == nil {
		return nil
	}
	var candidates []string
	for _, schema := range metadata.Schemas {
		for _, t := range schema.Tables {
			if tableNameMatches(t.Name, missing) {
				candidates = append(candidates, t.Name)
				if len(candidates) >= schemaCandidateResultCap {
					return candidates
				}
			}
		}
	}
	return candidates
}

// tableNameMatches returns true if `name` is a plausible candidate for a typo of
// `missing`. A name matches if it contains the input as a (case-insensitive)
// substring, or if it shares a sufficiently long common prefix (useful for typos
// like "orderz" → "orders", "order_items").
func tableNameMatches(name, missing string) bool {
	n := strings.ToLower(name)
	m := strings.ToLower(missing)
	if m == "" {
		return false
	}
	if strings.Contains(n, m) {
		return true
	}
	// Require a shared prefix of at least 4 chars (or the full input if shorter).
	prefixLen := min(4, len(m))
	if len(n) < prefixLen {
		return false
	}
	return n[:prefixLen] == m[:prefixLen]
}

// transformSchemas converts decoded metadata into the response shape for the
// given include level. Applies sorting and per-schema truncation.
func transformSchemas(metadata *databaseMetadata, include string) []SchemaSection {
	// Only bulk-detail modes ask the server for limit+1 and need the "trim to
	// schemaTableLimit and flag Truncated" dance. Summary mode is designed to
	// return every table because the per-table payload is tiny — applying the
	// cap there would silently drop tables from agents' view.
	applyTruncation := include == schemaIncludeColumns || include == schemaIncludeDetails

	sections := make([]SchemaSection, 0, len(metadata.Schemas))
	for i := range metadata.Schemas {
		schema := &metadata.Schemas[i]

		// Sort tables alphabetically by name.
		slices.SortStableFunc(schema.Tables, func(a, b tableMetadata) int {
			return strings.Compare(a.Name, b.Name)
		})

		truncated := false
		tablesShown := 0
		if applyTruncation && len(schema.Tables) > schemaTableLimit {
			// Drop the sentinel 201st entry.
			schema.Tables = schema.Tables[:schemaTableLimit]
			truncated = true
			tablesShown = schemaTableLimit
		}

		tables := make([]TableEntry, 0, len(schema.Tables))
		for j := range schema.Tables {
			tables = append(tables, buildTableEntry(&schema.Tables[j], include))
		}

		views := make([]string, 0, len(schema.Views))
		for _, v := range schema.Views {
			views = append(views, v.Name)
		}
		slices.Sort(views)

		sections = append(sections, SchemaSection{
			Name:           schema.Name,
			Tables:         tables,
			Views:          views,
			FunctionCount:  len(schema.Functions),
			ProcedureCount: len(schema.Procedures),
			Truncated:      truncated,
			TablesShown:    tablesShown,
		})
	}

	// Sort schemas alphabetically by name (empty string sorts first → MySQL friendly).
	slices.SortStableFunc(sections, func(a, b SchemaSection) int {
		return strings.Compare(a.Name, b.Name)
	})
	return sections
}

// buildTableEntry converts a single tableMetadata into the response shape at the
// requested detail level.
func buildTableEntry(t *tableMetadata, include string) TableEntry {
	rowCount, _ := t.RowCount.Int64()
	entry := TableEntry{
		Name:     t.Name,
		RowCount: rowCount,
		Comment:  t.Comment,
	}

	if include == schemaIncludeSummary {
		entry.ColumnCount = len(t.Columns)
		return entry
	}

	// Derive primary-key columns from indexes (ColumnMetadata has no primary_key field).
	pkColumns := primaryKeyColumns(t.Indexes)

	entry.Columns = make([]ColumnEntry, 0, len(t.Columns))
	for _, c := range t.Columns {
		col := ColumnEntry{
			Name:       c.Name,
			Type:       c.Type,
			Nullable:   c.Nullable,
			PrimaryKey: pkColumns[c.Name],
		}
		if include == schemaIncludeDetails {
			col.Default = c.Default
			col.Comment = c.Comment
		}
		entry.Columns = append(entry.Columns, col)
	}

	if include == schemaIncludeDetails {
		entry.Indexes = make([]IndexEntry, 0, len(t.Indexes))
		for _, idx := range t.Indexes {
			entry.Indexes = append(entry.Indexes, IndexEntry{
				Name:    idx.Name,
				Type:    idx.Type,
				Unique:  idx.Unique,
				Columns: append([]string{}, idx.Expressions...),
			})
		}
		slices.SortStableFunc(entry.Indexes, func(a, b IndexEntry) int {
			return strings.Compare(a.Name, b.Name)
		})

		entry.ForeignKeys = make([]FKEntry, 0, len(t.ForeignKeys))
		for _, fk := range t.ForeignKeys {
			entry.ForeignKeys = append(entry.ForeignKeys, FKEntry{
				Name:              fk.Name,
				Columns:           append([]string{}, fk.Columns...),
				ReferencedTable:   fk.ReferencedTable,
				ReferencedColumns: append([]string{}, fk.ReferencedColumns...),
			})
		}
		slices.SortStableFunc(entry.ForeignKeys, func(a, b FKEntry) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	return entry
}

// primaryKeyColumns scans indexes for the primary-key index and returns the set
// of column names it covers.
//
// IndexMetadata.expressions may contain expressions rather than column names for
// expression indexes (e.g. PostgreSQL CREATE INDEX ON users (LOWER(email))). PostgreSQL
// does not currently allow expression indexes as primary keys (PK columns must be real
// NOT NULL columns), so matching expression text against column name is correct today.
// Silent misses are preferred over flagging the wrong column.
// TODO: revisit if PG ever allows expression PKs.
func primaryKeyColumns(indexes []indexMetadata) map[string]bool {
	result := map[string]bool{}
	for _, idx := range indexes {
		if !idx.Primary {
			continue
		}
		for _, expr := range idx.Expressions {
			result[expr] = true
		}
	}
	return result
}

// formatSchemaOutput produces a text header + JSON body (same pattern as query_database).
func formatSchemaOutput(output *SchemaOutput, include string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Database: %s (%s)\n", output.Database, output.Engine)

	if output.Table != nil {
		t := output.Table
		fmt.Fprintf(&sb, "Table: %s (%d rows, %d columns, %d indexes, %d foreign keys)\n",
			t.Name, t.RowCount, len(t.Columns), len(t.Indexes), len(t.ForeignKeys))
	} else {
		schemaNames := make([]string, 0, len(output.Schemas))
		totalTables := 0
		totalViews := 0
		totalFunctions := 0
		truncatedAny := false
		for _, section := range output.Schemas {
			if section.Name != "" {
				schemaNames = append(schemaNames, section.Name)
			}
			totalTables += len(section.Tables)
			totalViews += len(section.Views)
			totalFunctions += section.FunctionCount
			if section.Truncated {
				truncatedAny = true
			}
		}
		if len(schemaNames) > 0 {
			fmt.Fprintf(&sb, "Schemas: %d (%s)\n", len(output.Schemas), strings.Join(schemaNames, ", "))
		}
		switch include {
		case schemaIncludeColumns:
			fmt.Fprintf(&sb, "Tables: %d (showing columns) | Views: %d\n", totalTables, totalViews)
		case schemaIncludeDetails:
			fmt.Fprintf(&sb, "Tables: %d (showing details) | Views: %d\n", totalTables, totalViews)
		default:
			fmt.Fprintf(&sb, "Tables: %d | Views: %d | Functions: %d\n", totalTables, totalViews, totalFunctions)
		}
		if truncatedAny {
			fmt.Fprintf(&sb, "Truncated: some schemas hit the %d-table limit per schema. Use schema= or table= to narrow.\n", schemaTableLimit)
		}
	}

	sb.WriteString("\n")
	jsonBytes, _ := json.Marshal(output)
	sb.Write(jsonBytes)

	return sb.String()
}

// formatTableNotFound returns an MCP result for the TABLE_NOT_FOUND error path.
func formatTableNotFound(database, table string, candidates []string) *mcp.CallToolResult {
	result := struct {
		Code       string   `json:"code"`
		Message    string   `json:"message"`
		Suggestion string   `json:"suggestion"`
		Candidates []string `json:"candidates,omitempty"`
	}{
		Code:       "TABLE_NOT_FOUND",
		Message:    fmt.Sprintf("no table matching %q in %s", table, database),
		Suggestion: "call get_schema without table= to see available tables",
		Candidates: candidates,
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: true,
	}
}

// formatAmbiguousTable returns an MCP result when the requested table name
// exists in multiple schemas. The caller must pick a schema.
func formatAmbiguousTable(database, table string, schemas []string) *mcp.CallToolResult {
	result := struct {
		Code       string   `json:"code"`
		Message    string   `json:"message"`
		Suggestion string   `json:"suggestion"`
		Schemas    []string `json:"schemas"`
	}{
		Code:       "AMBIGUOUS_TABLE",
		Message:    fmt.Sprintf("table %q exists in multiple schemas of %s", table, database),
		Suggestion: fmt.Sprintf("re-run with schema= set to one of: %s", strings.Join(schemas, ", ")),
		Schemas:    schemas,
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: true,
	}
}
