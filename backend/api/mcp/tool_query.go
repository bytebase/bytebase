package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

const (
	defaultQueryLimit = 100
	maxQueryLimit     = 1000
	queryTimeout      = 30 * time.Second
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
	// Schema is the default schema for unqualified names.
	Schema string `json:"schema,omitempty"`
	// Container is the CosmosDB container to query.
	Container string `json:"container,omitempty"`
	// Limit is the maximum number of rows to return (default: 100, max: 1000).
	Limit int `json:"limit,omitempty"`
}

// QueryOutput is the output for the query_database tool.
type QueryOutput struct {
	Columns            []string       `json:"columns"`
	ColumnTypes        []string       `json:"columnTypes"`
	Rows               [][]any        `json:"rows"`
	RowCount           int            `json:"rowCount"`
	Truncated          bool           `json:"truncated"`
	LatencyMs          int64          `json:"latencyMs"`
	MaskedColumns      []MaskedColumn `json:"maskedColumns,omitempty"`
	AppliedAccessGrant string         `json:"appliedAccessGrant,omitempty"`
}

// MaskedColumn is a result column whose values masking replaced.
type MaskedColumn struct {
	// Column repeats columns[Index], so a model reading one entry need not
	// count positions.
	Column string `json:"column"`
	// Index is the column's position in columns and in each row, which tells
	// apart columns that share a name.
	Index        int    `json:"index"`
	SemanticType string `json:"semanticType,omitempty"`
	Algorithm    string `json:"algorithm,omitempty"`
	Context      string `json:"context,omitempty"`
}

// queryDatabaseDescription is the description for the query_database tool.
const queryDatabaseDescription = `Execute a SQL query against a Bytebase database. Resolves the database by name automatically.

| Parameter | Required | Description |
|-----------|----------|-------------|
| database  | Yes      | Database name or substring (e.g., "employee_db" or "employee") |
| statement | Yes      | SQL query to execute |
| instance  | No       | Workspace instance ID/name, or the full canonical name for a project instance |
| project   | No       | Project name or ID to narrow resolution |
| schema    | No       | Default schema for unqualified names (Postgres search path). An access grant applies only when this equals the grant's schema |
| container | No       | CosmosDB container (required for CosmosDB). An access grant applies only when this equals the grant's container |
| limit     | No       | Max rows (default: 100, max: 1000) |

**Examples:**
query_database(database="employee_db", statement="SELECT * FROM users LIMIT 10")
query_database(database="employee", instance="prod-pg", statement="SELECT count(*) FROM orders")
query_database(database="employee", instance="projects/hr/instances/prod-pg", statement="SELECT count(*) FROM orders")

**Notes:**
- Masked data: masking replaces a value fully (by default "******"), partly (e.g. "**rn**"), or with a hash. maskedColumns in the result names each masked column and why (not reported for MongoDB, CosmosDB or Elasticsearch). A masked value is a placeholder, not a value anything holds, so never write it back and never filter on it. Any statement containing "******" is refused. When showing a partly masked value, wrap it in backticks: "**rn**" renders as bold Markdown.
- Seeing masked values: the user can ask a project owner for a masking exemption, or, if the project allows just-in-time access, request an access grant with Unmask for this exact statement, schema and container. In a Read-write session, file the request with call_api(operationId="AccessGrantService/CreateAccessGrant"); otherwise the user requests it in the SQL Editor. Once it is approved, find the grant among the user's access grants, then rerun the granted statement unchanged with the grant's schema and container. appliedAccessGrant in the result names the grant that applied.
`

func (s *Server) registerQueryTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "query_database",
		Description: queryDatabaseDescription,
	}, s.handleQueryDatabase)
}

func (s *Server) handleQueryDatabase(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
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
	resolved, resolveResult := s.resolveTarget(ctx, req, input.Database, input.Instance, input.Project)
	if resolveResult != nil {
		return resolveResult, nil, nil
	}

	// Execute query.
	queryCtx, queryCancel := context.WithTimeout(ctx, queryTimeout)
	defer queryCancel()

	output, err := s.executeQuery(queryCtx, resolved, input, limit)
	if err != nil {
		return formatToolError(err), nil, nil
	}

	text := formatQueryOutput(input.Statement, resolved.resourceName, output)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
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
	if len(output.MaskedColumns) > 0 {
		names := make([]string, 0, len(output.MaskedColumns))
		for _, column := range output.MaskedColumns {
			names = append(names, column.Column)
		}
		fmt.Fprintf(&sb, "Masked columns: %s.\n", strings.Join(names, ", "))
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

// queryResponse is the typed response from the SQL Query API.
type queryResponse struct {
	Results            []queryResult `json:"results"`
	AppliedAccessGrant string        `json:"appliedAccessGrant"`
}

// queryResult represents a single result set from a query.
type queryResult struct {
	ColumnNames     []string        `json:"columnNames"`
	ColumnTypeNames []string        `json:"columnTypeNames"`
	Rows            []queryRow      `json:"rows"`
	RowsCount       json.Number     `json:"rowsCount"`
	Latency         string          `json:"latency"`
	Error           string          `json:"error"`
	Statement       string          `json:"statement"`
	Masked          []maskingReason `json:"masked"`
}

// maskingReason holds the MaskingReason fields the tool reports.
type maskingReason struct {
	SemanticTypeTitle string `json:"semanticTypeTitle"`
	Algorithm         string `json:"algorithm"`
	Context           string `json:"context"`
}

// queryRow represents a row of values from a query result.
type queryRow struct {
	Values []json.RawMessage `json:"values"`
}

// executeQuery executes a SQL query against the resolved database.
func (s *Server) executeQuery(ctx context.Context, resolved *resolvedDatabase, input QueryInput, limit int) (*QueryOutput, error) {
	body := map[string]any{
		"name":         resolved.resourceName,
		"dataSourceId": resolved.dataSourceID,
		"statement":    input.Statement,
	}
	if input.Schema != "" {
		body["schema"] = input.Schema
	}
	if input.Container != "" {
		body["container"] = input.Container
	}
	resp, err := s.apiRequest(ctx, "/bytebase.v1.SQLService/Query", body)
	if err != nil {
		return nil, &toolError{
			Code:       "QUERY_ERROR",
			Message:    fmt.Sprintf("query request failed: %s", err.Error()),
			Suggestion: "check network connectivity and try again",
		}
	}
	if resp.Status >= 400 {
		errMsg := parseError(resp.Body)
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.Status)
		}
		suggestion := "check your SQL syntax and try again"
		if resp.Status == http.StatusForbidden || resp.Status == http.StatusUnauthorized {
			// A refusal that already names its own way out gets none added.
			// Telling an agent to request a project role for one sends the
			// person it acts for after a grant that cannot lift a workspace
			// setting.
			suggestion = ""
			if !IsPolicyRefusal(errMsg) {
				suggestion = "you may not have permission to query this database — request the SQL Editor role on the project"
			}
		}
		return nil, &toolError{
			Code:       "QUERY_ERROR",
			Message:    errMsg,
			Suggestion: suggestion,
		}
	}

	var qr queryResponse
	if err := json.Unmarshal(resp.Body, &qr); err != nil {
		return nil, errors.Wrap(err, "failed to parse query response")
	}
	if len(qr.Results) == 0 {
		return &QueryOutput{
			Columns:            []string{},
			ColumnTypes:        []string{},
			Rows:               [][]any{},
			AppliedAccessGrant: qr.AppliedAccessGrant,
		}, nil
	}

	result := qr.Results[0]
	if result.Error != "" {
		return nil, &toolError{
			Code:       "QUERY_ERROR",
			Message:    result.Error,
			Suggestion: "check your SQL syntax and try again",
		}
	}

	// Flatten rows.
	rows := make([][]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		flat := make([]any, 0, len(row.Values))
		for _, v := range row.Values {
			flat = append(flat, flattenRowValue(v))
		}
		rows = append(rows, flat)
	}

	// Truncation.
	truncated := len(rows) > limit
	if truncated {
		rows = rows[:limit]
	}

	return &QueryOutput{
		Columns:            result.ColumnNames,
		ColumnTypes:        result.ColumnTypeNames,
		Rows:               rows,
		RowCount:           len(rows),
		Truncated:          truncated,
		LatencyMs:          parseLatencyMs(result.Latency),
		MaskedColumns:      maskedColumns(result.ColumnNames, result.Masked),
		AppliedAccessGrant: qr.AppliedAccessGrant,
	}, nil
}

// maskedColumns pairs each masked column with its reason. Query aligns masked
// with the columns by index and sends {} for an unmasked column, and the list
// can be shorter than the columns, leaving the rest unmasked (doMaskResult in
// backend/api/v1/query_result_masker.go).
func maskedColumns(columns []string, reasons []maskingReason) []MaskedColumn {
	var masked []MaskedColumn
	for i := range min(len(columns), len(reasons)) {
		reason := reasons[i]
		if reason == (maskingReason{}) {
			continue
		}
		masked = append(masked, MaskedColumn{
			Column:       columns[i],
			Index:        i,
			SemanticType: reason.SemanticTypeTitle,
			Algorithm:    reason.Algorithm,
			Context:      reason.Context,
		})
	}
	return masked
}

// flattenRowValue extracts a plain Go value from a protojson RowValue oneof.
// Each value is a JSON object with exactly one key like {"stringValue": "x"}.
func flattenRowValue(raw json.RawMessage) any {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return string(raw)
	}
	for key, val := range m {
		if v, ok := unmarshalRowField(key, val); ok {
			return v
		}
	}
	return string(raw)
}

// unmarshalRowField decodes a single protojson RowValue oneof field.
func unmarshalRowField(key string, val json.RawMessage) (any, bool) {
	switch key {
	case "nullValue":
		return nil, true
	case "boolValue":
		return unmarshalAs[bool](val)
	case "stringValue":
		return unmarshalAs[string](val)
	case "int32Value":
		return unmarshalAs[int32](val)
	case "int64Value":
		// protojson encodes int64 as string.
		if v, ok := unmarshalAs[string](val); ok {
			return v, true
		}
		return unmarshalAs[int64](val)
	case "doubleValue", "floatValue":
		return unmarshalAs[float64](val)
	case "timestampValue", "timestampTzValue":
		return unmarshalTimestamp(val)
	default:
		if v, ok := unmarshalAs[string](val); ok {
			return v, true
		}
		return string(val), true
	}
}

// unmarshalAs attempts to decode JSON into the given type.
func unmarshalAs[T any](val json.RawMessage) (T, bool) {
	var v T
	if json.Unmarshal(val, &v) == nil {
		return v, true
	}
	return v, false
}

// unmarshalTimestamp extracts the string value from a timestamp object.
func unmarshalTimestamp(val json.RawMessage) (any, bool) {
	var ts map[string]string
	if json.Unmarshal(val, &ts) == nil {
		if v, ok := ts["value"]; ok {
			return v, true
		}
	}
	return nil, false
}

// parseLatencyMs parses a duration string like "0.012s" into milliseconds.
func parseLatencyMs(latency string) int64 {
	latency = strings.TrimSpace(latency)
	if latency == "" {
		return 0
	}
	latency = strings.TrimSuffix(latency, "s")
	d, err := time.ParseDuration(latency + "s")
	if err != nil {
		return 0
	}
	return d.Milliseconds()
}

// IsPolicyRefusal reports whether a 403 came from the MCP enforcement chain
// rather than from the caller's own RBAC. It matches wording because the verdict
// crosses a process boundary as an error body and nothing else survives the
// trip; the producers call this, not a copy of it, in
// TestMCPRefusalsNameThemselvesToTheQueryTool (backend/api/v1).
func IsPolicyRefusal(message string) bool {
	for _, phrase := range []string{
		"MCP capability ceiling",
		// Singular, so it also matches the plural the gate uses and the
		// refusals taken in handlers.
		"MCP session",
		"MCP classification",
	} {
		if strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}
