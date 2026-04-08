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
	resolveCtx, resolveCancel := context.WithTimeout(ctx, resolveTimeout)
	defer resolveCancel()

	resolved, err := s.resolveDatabase(resolveCtx, input.Database, input.Instance, input.Project)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	if resolved.ambiguous {
		picked, elicitErr := s.elicitDatabaseChoice(ctx, req, resolved)
		if elicitErr != nil {
			// Elicitation unsupported or user cancelled — fall back to AMBIGUOUS_TARGET.
			return formatAmbiguousResult(input.Database, resolved.candidates), nil, nil
		}
		resolved = picked
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

// queryResponse is the typed response from the SQL Query API.
type queryResponse struct {
	Results []queryResult `json:"results"`
}

// queryResult represents a single result set from a query.
type queryResult struct {
	ColumnNames     []string    `json:"columnNames"`
	ColumnTypeNames []string    `json:"columnTypeNames"`
	Rows            []queryRow  `json:"rows"`
	RowsCount       json.Number `json:"rowsCount"`
	Latency         string      `json:"latency"`
	Error           string      `json:"error"`
	Statement       string      `json:"statement"`
}

// queryRow represents a row of values from a query result.
type queryRow struct {
	Values []json.RawMessage `json:"values"`
}

// executeQuery executes a SQL query against the resolved database.
func (s *Server) executeQuery(ctx context.Context, resolved *resolvedDatabase, statement string, limit int) (*QueryOutput, error) {
	body := map[string]any{
		"name":         resolved.resourceName,
		"dataSourceId": resolved.dataSourceID,
		"statement":    statement,
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
			suggestion = "you may not have permission to query this database — request the SQL Editor role on the project"
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
			Columns:     []string{},
			ColumnTypes: []string{},
			Rows:        [][]any{},
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
		Columns:     result.ColumnNames,
		ColumnTypes: result.ColumnTypeNames,
		Rows:        rows,
		RowCount:    len(rows),
		Truncated:   truncated,
		LatencyMs:   parseLatencyMs(result.Latency),
	}, nil
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
