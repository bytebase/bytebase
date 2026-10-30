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

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
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
	// IssueTitle titles the issue when the user files a possible write as one.
	IssueTitle string `json:"issueTitle,omitempty"`
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
| issueTitle | No      | For a statement that may write: the issue's title if the user chooses Create an issue. The form shows it in that choice. Say what the change does, without data values. Without it, the form does not offer Create an issue |

**Examples:**
query_database(database="employee_db", statement="SELECT * FROM users LIMIT 10")
query_database(database="employee", instance="prod-pg", statement="SELECT count(*) FROM orders")
query_database(database="employee", instance="projects/hr/instances/prod-pg", statement="SELECT count(*) FROM orders")
query_database(database="employee_db", statement="UPDATE employee SET active = false WHERE id = 42", issueTitle="Deactivate an employee")

**Notes:**
- Show the user the statement each answer came from, so they can check it read what they asked about.
- When the user asks for an issue or a review of a change, call propose_database_change instead. It takes no schema, so when the request named one, qualify each name with it.
- Masked data: masking replaces a value fully (by default "******"), partly (e.g. "**rn**"), or with a hash. maskedColumns in the result names each masked column and why (not reported for MongoDB, CosmosDB or Elasticsearch). A masked value is a placeholder, not a value anything holds, so never write it back and never filter on it. Any statement containing "******" is refused. When showing a partly masked value, wrap it in backticks: "**rn**" renders as bold Markdown.
- Seeing masked values: the user can ask a project owner for a masking exemption, or, if the project allows just-in-time access, request an access grant with Unmask for this exact statement, schema and container. In a Read-write session, file the request with request_access_grant; otherwise the user requests it in the SQL Editor. Once it is approved, find the grant among the user's access grants, then rerun the granted statement unchanged with the grant's schema and container. appliedAccessGrant in the result names the grant that applied.
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

	// An answer that acts on nothing is reported before the database is
	// resolved again, which may have changed while the user was asked.
	choice, answerState := "", ""
	if canShowConfirmation(req) {
		if response, state, answered := confirmationAnswer(req); answered {
			var result *mcp.CallToolResult
			if choice, result = answeredChoice(response); result != nil {
				return result, nil, nil
			}
			answerState = state
		}
	}

	// Resolve database.
	resolved, resolveResult := s.resolveTarget(ctx, req, input.Database, input.Instance, input.Project)
	if resolveResult != nil {
		if choice != "" && len(resolveResult.InputRequests) > 0 {
			return staleAnswer("the databases that match changed after the user was asked"), nil, nil
		}
		return resolveResult, nil, nil
	}

	// A statement Query cannot show is a read runs once the user chooses Run
	// now, or at once when the client cannot show the form.
	confirmation := runConfirmation{database: resolved.resourceName}
	digest := auth.MCPStatementDigest(input.Statement, resolved.resourceName, input.Schema, input.Container)
	// Create an issue files under the agent's title: one Bytebase generated would
	// pass in a project that requires a manual title, since the server refuses
	// only an empty one. A plan carries no schema, so an issue's rollout would
	// run with the database's default one.
	var issueTitle string
	if input.Schema == "" {
		issueTitle = strings.TrimSpace(input.IssueTitle)
	}
	asked := askedDigest(digest, issueTitle)
	var confirmed []byte
	if !canShowConfirmation(req) {
		confirmed = digest
	} else if reply, answered := s.signedReply(ctx, req, confirmation, asked); answered {
		if why := reply.staleReason("the request changed after the user was asked"); why != "" {
			return staleAnswer(why), nil, nil
		}
		if choice == runChoiceIssue {
			if issueTitle == "" {
				return noOfferedChoice(), nil, nil
			}
			return s.fileAsIssue(ctx, resolved, input.Statement, issueTitle), nil, nil
		}
		confirmed = digest
	} else if choice != "" && answerState != "" {
		// The answer was signed for another database.
		return staleAnswer("the database that matches changed after the user was asked"), nil, nil
	}

	// Execute query.
	queryCtx, queryCancel := context.WithTimeout(ctx, queryTimeout)
	defer queryCancel()

	output, err := s.executeQuery(queryCtx, resolved, input, limit, confirmed)
	var required *confirmationRequired
	// A confirmed call that Query still refuses reports the refusal: asking
	// again in the same call would send a second form.
	if errors.As(err, &required) && confirmed == nil {
		return s.askToRun(ctx, req, confirmation, asked, input.Statement, input.Container, required, issueTitle), nil, nil
	}
	if err != nil {
		var qe *queryError
		if errors.As(err, &qe) {
			return formatQueryError(qe), nil, nil
		}
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
	// PermissionDenied is set when the SQL access check refused the statement.
	PermissionDenied *struct {
		RequiredPermissions []permission.Permission `json:"requiredPermissions"`
		Resources           []string                `json:"resources"`
	} `json:"permissionDenied"`
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

// executeQuery executes a SQL query against the resolved database. confirmed is
// the digest the confirmation marker carries, or nil for an unconfirmed request.
func (s *Server) executeQuery(ctx context.Context, resolved *resolvedDatabase, input QueryInput, limit int, confirmed []byte) (*QueryOutput, error) {
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
	if confirmed != nil {
		ctx = auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
			Procedure: v1connect.SQLServiceQueryProcedure,
			Resource:  resolved.resourceName,
			Digest:    confirmed,
		})
	}
	resp, err := s.apiRequest(ctx, v1connect.SQLServiceQueryProcedure, body)
	if err != nil {
		// The call can end partway through a batch, as when queryTimeout passes.
		return nil, &toolError{
			Code:       "QUERY_ERROR",
			Message:    fmt.Sprintf("query request failed: %s", err.Error()),
			Suggestion: whatRanUnknown,
		}
	}
	if resp.Status >= 400 {
		if required := confirmationRequiredIn(resp.Body); required != nil {
			return nil, required
		}
		errMsg := parseError(resp.Body)
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.Status)
		}
		suggestion := ""
		switch {
		case resp.Status >= http.StatusInternalServerError:
			// A 5xx can follow statements that ran, as when a SQL Server batch
			// fails partway: its driver then returns no result.
			suggestion = whatRanUnknown
		case resp.Status != http.StatusForbidden && resp.Status != http.StatusUnauthorized:
			suggestion = "check your SQL syntax and try again"
		default:
			// A refusal at this level gets no advice: no access grant lifts a
			// missing bb.databases.get, which the ACL checks first, or an MCP policy.
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

	if failed := failedResult(qr.Results); failed >= 0 {
		return nil, queryFailure(qr.Results, failed, resolved.resourceName)
	}
	result := qr.Results[0]

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

const whatRanUnknown = "Bytebase cannot always tell which statements of a failed request ran; if it held a write, check the data before you retry"

// queryError is a failed query_database request.
type queryError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	// FailedStatement is what Bytebase reported as failed: one statement, or
	// the whole request where Query runs it as one unit, as on SQL Server.
	FailedStatement string `json:"failedStatement,omitempty"`
	// Ran can name FailedStatement too, when its error came after it ran. It
	// names a SQL Server procedure call once per result set the call returned:
	// the driver gives each later set the call's statement.
	Ran []string `json:"ran,omitempty"`
}

func (e *queryError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// formatQueryError formats a queryError into an MCP error result.
func formatQueryError(err *queryError) *mcp.CallToolResult {
	jsonBytes, _ := json.MarshalIndent(err, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: true,
	}
}

// ran reports whether the result's statement ran. A result with column names
// came from a result set, so its statement ran even when an error follows, as
// when the size limit cuts it short or masking withholds its rows.
func (r queryResult) ran() bool {
	return r.Error == "" || len(r.ColumnNames) > 0
}

// failedResult returns the index of the result to report as failed: the first
// whose statement did not run, else the first with an error, or -1.
func failedResult(results []queryResult) int {
	if i := slices.IndexFunc(results, func(r queryResult) bool { return !r.ran() }); i >= 0 {
		return i
	}
	return slices.IndexFunc(results, func(r queryResult) bool { return r.Error != "" })
}

// queryFailure reports a request whose failed result is results[failed].
func queryFailure(results []queryResult, failed int, database string) *queryError {
	qe := &queryError{
		Code:            "QUERY_ERROR",
		Message:         results[failed].Error,
		FailedStatement: strings.TrimSpace(results[failed].Statement),
	}
	for _, r := range results {
		if r.ran() {
			qe.Ran = append(qe.Ran, strings.TrimSpace(r.Statement))
		}
	}
	denied := results[failed].PermissionDenied
	// The access check refuses a statement before it runs, so a denial with
	// nothing in ran changed nothing and gets no caution.
	var advice []string
	if len(qe.Ran) > 0 {
		advice = append(advice, "Bytebase ran the statements in ran, in a session that has since closed; check the data before you retry")
	} else if denied == nil {
		advice = append(advice, whatRanUnknown)
	}
	// MCP sessions cannot file role requests, so a grant is the only access
	// advice a denial gets.
	if denied != nil && grantLifts(denied.RequiredPermissions) && withinDatabase(denied.Resources, database) {
		advice = append(advice, grantAdvice)
	}
	qe.Suggestion = strings.Join(advice, "; ")
	return qe
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

// grantLifts reports whether an approved access grant lifts a refusal that
// required these permissions. Writes and DDL cannot be granted, and a granted
// target still needs bb.sql.explain (SQLService.accessCheckWithGrantedTargets).
func grantLifts(required []permission.Permission) bool {
	if len(required) == 0 {
		return false
	}
	for _, p := range required {
		if p != permission.SQLSelect && p != permission.SQLInfo {
			return false
		}
	}
	return true
}

// withinDatabase reports whether every denied resource is the database or lies
// inside it. request_access_grant files a grant for that one database, and a
// grant exempts only its own targets.
func withinDatabase(resources []string, database string) bool {
	if len(resources) == 0 {
		return false
	}
	for _, resource := range resources {
		if resource != database && !strings.HasPrefix(resource, database+"/") {
			return false
		}
	}
	return true
}

// grantAdvice advises on a refusal an access grant lifts. A grant applies only
// to the exact read-only statement it names (hashAccessGrantQuery).
const grantAdvice = "in a Read-write session, request an access grant for failedStatement with request_access_grant and, " +
	"once approved, run it by itself (the project must allow just-in-time access); otherwise the user requests access in the SQL Editor"
