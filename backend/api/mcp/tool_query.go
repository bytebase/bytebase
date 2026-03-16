package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

const (
	defaultQueryLimit = 100
	maxQueryLimit     = 1000
	queryTimeout      = 30 * time.Second
	resolveTimeout    = 30 * time.Second
)

// QueryInput is the input for the query_database tool.
type QueryInput struct {
	// Database is the database name or substring to match.
	Database string `json:"database"`
	// Statement is the SQL query to execute.
	Statement string `json:"statement"`
	// Instance narrows database resolution to a specific instance.
	Instance string `json:"instance,omitempty"`
	// Project narrows database resolution to a specific project.
	Project string `json:"project,omitempty"`
	// Limit is the maximum number of rows to return (default: 100, max: 1000).
	Limit int `json:"limit,omitempty"`
}

// QueryOutput is the output for the query_database tool.
type QueryOutput struct {
	Columns     []string `json:"columns"`
	ColumnTypes []string `json:"columnTypes"`
	Rows        [][]any  `json:"rows"`
	RowCount    int      `json:"rowCount"`
	Truncated   bool     `json:"truncated"`
	LatencyMs   int64    `json:"latencyMs"`
}

// Candidate represents a database match for ambiguous resolution.
type Candidate struct {
	Database string `json:"database"`
	Instance string `json:"instance"`
	Project  string `json:"project"`
	Engine   string `json:"engine"`
}

// resolvedDatabase holds the result of database resolution.
type resolvedDatabase struct {
	resourceName string
	dataSourceID string
	ambiguous    bool
	candidates   []Candidate
}

// queryDatabaseDescription is the description for the query_database tool.
const queryDatabaseDescription = `Execute a SQL query against a Bytebase database. Resolves the database by name automatically.

| Parameter | Required | Description |
|-----------|----------|-------------|
| database  | Yes      | Database name or substring (e.g., "employee_db" or "employee") |
| statement | Yes      | SQL query to execute |
| instance  | No       | Instance name to narrow resolution |
| project   | No       | Project name to narrow resolution |
| limit     | No       | Max rows (default: 100, max: 1000) |

**Examples:**
query_database(database="employee_db", statement="SELECT * FROM users LIMIT 10")
query_database(database="employee", instance="prod-pg", statement="SELECT count(*) FROM orders")`

func (s *Server) registerQueryTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "query_database",
		Description: queryDatabaseDescription,
	}, s.handleQueryDatabase)
}

func (s *Server) handleQueryDatabase(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
	if input.Database == "" {
		return nil, nil, errors.New("database is required")
	}
	if input.Statement == "" {
		return nil, nil, errors.New("statement is required")
	}

	// Normalize limit.
	limit := input.Limit
	if limit <= 0 {
		limit = defaultQueryLimit
	}
	if limit > maxQueryLimit {
		limit = maxQueryLimit
	}

	// Resolve database.
	resolveCtx, resolveCancel := context.WithTimeout(ctx, resolveTimeout)
	defer resolveCancel()

	resolved, err := s.resolveDatabase(resolveCtx, input)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if resolved.ambiguous {
		return formatAmbiguousResult(input.Database, resolved.candidates), nil, nil
	}

	// Execute query.
	queryCtx, queryCancel := context.WithTimeout(ctx, queryTimeout)
	defer queryCancel()

	output, err := s.executeQuery(queryCtx, resolved, input.Statement, limit)
	if err != nil {
		return formatToolError(err), nil, nil
	}

	text := formatQueryOutput(input.Statement, resolved.resourceName, output)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}

// listDatabasesResponse is the typed response from ListDatabases API.
type listDatabasesResponse struct {
	Databases []databaseEntry `json:"databases"`
}

// databaseEntry represents a database in the ListDatabases response.
type databaseEntry struct {
	Name             string           `json:"name"`
	Project          string           `json:"project"`
	InstanceResource instanceResource `json:"instanceResource"`
}

// instanceResource holds instance details including data sources.
type instanceResource struct {
	Name        string       `json:"name"`
	Engine      string       `json:"engine"`
	DataSources []dataSource `json:"dataSources"`
}

// dataSource represents a database data source.
type dataSource struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// buildDatabaseFilter builds a CEL filter expression for ListDatabases.
func buildDatabaseFilter(input QueryInput) string {
	// name.matches does substring matching server-side.
	filter := fmt.Sprintf("name.matches(%q)", input.Database)
	if input.Instance != "" {
		filter += fmt.Sprintf(" && instance == %q", "instances/"+input.Instance)
	}
	if input.Project != "" {
		filter += fmt.Sprintf(" && project == %q", "projects/"+input.Project)
	}
	return filter
}

// resolveDatabase resolves a database name to a unique resource using tiered matching.
func (s *Server) resolveDatabase(ctx context.Context, input QueryInput) (*resolvedDatabase, error) {
	body := map[string]any{
		"parent":   "workspaces/-",
		"filter":   buildDatabaseFilter(input),
		"pageSize": 1000,
	}
	resp, err := s.apiRequest(ctx, "/bytebase.v1.DatabaseService/ListDatabases", body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list databases")
	}
	if resp.Status >= 400 {
		return nil, errors.Errorf("failed to list databases: HTTP %d: %s", resp.Status, parseError(resp.Body))
	}

	var listResp listDatabasesResponse
	if err := json.Unmarshal(resp.Body, &listResp); err != nil {
		return nil, errors.Wrap(err, "failed to parse database list")
	}

	// Server-side filter already narrows by name substring and instance/project.
	// Apply client-side tiered matching to pick the best result.
	databases := listResp.Databases

	// Tiered matching: exact -> case-insensitive exact -> substring.
	matches := matchExact(databases, input.Database)
	if len(matches) == 0 {
		matches = matchCaseInsensitive(databases, input.Database)
	}
	if len(matches) == 0 {
		matches = matchSubstring(databases, input.Database)
	}

	if len(matches) == 0 {
		suggestion := "check the database name or use search_api to list available databases"
		if input.Instance != "" || input.Project != "" {
			suggestion = "try without instance/project filters, or use search_api to list available databases"
		}
		return nil, &toolError{
			Code:       "DATABASE_NOT_FOUND",
			Message:    fmt.Sprintf("no database matching %q", input.Database),
			Suggestion: suggestion,
		}
	}

	if len(matches) > 1 {
		candidates := make([]Candidate, 0, len(matches))
		for _, db := range matches {
			candidates = append(candidates, Candidate{
				Database: db.Name,
				Instance: extractShortName(db.InstanceResource.Name),
				Project:  extractShortName(db.Project),
				Engine:   db.InstanceResource.Engine,
			})
		}
		return &resolvedDatabase{ambiguous: true, candidates: candidates}, nil
	}

	// Single match.
	db := matches[0]
	return &resolvedDatabase{
		resourceName: db.Name,
		dataSourceID: selectDataSource(db.InstanceResource.DataSources),
	}, nil
}

// matchExact returns databases whose short name exactly matches the input.
func matchExact(databases []databaseEntry, name string) []databaseEntry {
	var result []databaseEntry
	for _, db := range databases {
		if extractDatabaseName(db.Name) == name {
			result = append(result, db)
		}
	}
	return result
}

// matchCaseInsensitive returns databases whose short name matches case-insensitively.
func matchCaseInsensitive(databases []databaseEntry, name string) []databaseEntry {
	var result []databaseEntry
	lower := strings.ToLower(name)
	for _, db := range databases {
		if strings.ToLower(extractDatabaseName(db.Name)) == lower {
			result = append(result, db)
		}
	}
	return result
}

// matchSubstring returns databases whose short name contains the input as a substring.
func matchSubstring(databases []databaseEntry, name string) []databaseEntry {
	var result []databaseEntry
	lower := strings.ToLower(name)
	for _, db := range databases {
		if strings.Contains(strings.ToLower(extractDatabaseName(db.Name)), lower) {
			result = append(result, db)
		}
	}
	return result
}

// extractDatabaseName extracts the database name from a resource name like "instances/prod-pg/databases/employee_db".
func extractDatabaseName(resourceName string) string {
	parts := strings.Split(resourceName, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return resourceName
}

// extractShortName extracts the last segment from a resource name like "instances/prod-pg" or "projects/hr-system".
func extractShortName(resourceName string) string {
	parts := strings.Split(resourceName, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return resourceName
}

// selectDataSource picks READ_ONLY if available, otherwise ADMIN.
func selectDataSource(dataSources []dataSource) string {
	var adminID string
	for _, ds := range dataSources {
		if ds.Type == "READ_ONLY" {
			return ds.ID
		}
		if ds.Type == "ADMIN" {
			adminID = ds.ID
		}
	}
	return adminID
}

// formatQueryOutput formats a successful query result as text header + JSON body.
func formatQueryOutput(statement, resourceName string, output *QueryOutput) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Query: %s\n", statement)
	fmt.Fprintf(&sb, "Database: %s\n", resourceName)

	columnList := strings.Join(output.Columns, ", ")
	if output.Truncated {
		fmt.Fprintf(&sb, "Result: Showing %d of unknown total rows, %d columns (%s) | %dms\n",
			output.RowCount, len(output.Columns), columnList, output.LatencyMs)
		sb.WriteString("Truncated: use limit param for more (max 1000).\n")
	} else {
		fmt.Fprintf(&sb, "Result: %d rows, %d columns (%s) | %dms\n",
			output.RowCount, len(output.Columns), columnList, output.LatencyMs)
	}

	sb.WriteString("\n")
	jsonBytes, _ := json.Marshal(output)
	sb.Write(jsonBytes)

	return sb.String()
}

// formatToolError converts an error into an MCP error result.
// If the error is a *toolError, it returns structured JSON with code/message/suggestion.
// Otherwise, it returns the error message as plain text.
func formatToolError(err error) *mcp.CallToolResult {
	var te *toolError
	if errors.As(err, &te) {
		jsonBytes, _ := json.MarshalIndent(te, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
			IsError: true,
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}
}

// formatAmbiguousResult returns an MCP result for ambiguous database matches.
func formatAmbiguousResult(database string, candidates []Candidate) *mcp.CallToolResult {
	result := struct {
		Code       string      `json:"code"`
		Message    string      `json:"message"`
		Candidates []Candidate `json:"candidates"`
	}{
		Code:       "AMBIGUOUS_TARGET",
		Message:    fmt.Sprintf("Multiple databases match %q. Specify instance or project to narrow.", database),
		Candidates: candidates,
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: true,
	}
}

// executeQuery executes a SQL query against the resolved database.
func (*Server) executeQuery(_ context.Context, _ *resolvedDatabase, _ string, _ int) (*QueryOutput, error) {
	return nil, errors.New("not implemented")
}
