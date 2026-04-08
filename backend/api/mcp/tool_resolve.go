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

const resolveTimeout = 30 * time.Second

// Candidate represents a database match for ambiguous resolution.
type Candidate struct {
	Database string `json:"database"`
	Instance string `json:"instance"`
	Project  string `json:"project"`
	Engine   string `json:"engine"`
}

// resolvedDatabase holds the result of database resolution.
type resolvedDatabase struct {
	resourceName  string
	dataSourceID  string
	engine        string
	ambiguous     bool
	candidates    []Candidate
	dataSourceIDs map[string]string // resourceName -> dataSourceID (populated when ambiguous)
	engines       map[string]string // resourceName -> engine (populated when ambiguous)
}

// listDatabasesResponse is the typed response from ListDatabases API.
type listDatabasesResponse struct {
	Databases     []databaseEntry `json:"databases"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
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
func buildDatabaseFilter(database, instance, project string) string {
	// name.contains does substring matching server-side.
	filter := fmt.Sprintf("name.contains(%q)", database)
	if instance != "" {
		filter += fmt.Sprintf(" && instance == %q", "instances/"+instance)
	}
	if project != "" {
		filter += fmt.Sprintf(" && project == %q", "projects/"+project)
	}
	return filter
}

// listDatabases lists databases matching the filter in the user's workspace.
// Uses the workspace ID from the JWT token stored in context.
func (s *Server) listDatabases(ctx context.Context, filter string) ([]databaseEntry, error) {
	workspaceID := getWorkspaceID(ctx)
	if workspaceID == "" {
		return nil, &toolError{
			Code:       "AUTH_ERROR",
			Message:    "workspace ID not found in token",
			Suggestion: "re-authenticate with Bytebase",
		}
	}

	var databases []databaseEntry
	pageToken := ""
	for {
		body := map[string]any{
			"parent":   fmt.Sprintf("workspaces/%s", workspaceID),
			"filter":   filter,
			"pageSize": 1000,
		}
		if pageToken != "" {
			body["pageToken"] = pageToken
		}
		resp, err := s.apiRequest(ctx, "/bytebase.v1.DatabaseService/ListDatabases", body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list databases")
		}
		if resp.Status == http.StatusForbidden {
			return nil, &toolError{
				Code:       "PERMISSION_DENIED",
				Message:    "you don't have permission to list databases in this workspace",
				Suggestion: "ask your workspace admin to grant you the bb.databases.list permission",
			}
		}
		if resp.Status >= 400 {
			return nil, errors.Errorf("failed to list databases: HTTP %d: %s", resp.Status, parseError(resp.Body))
		}

		var listResp listDatabasesResponse
		if err := json.Unmarshal(resp.Body, &listResp); err != nil {
			return nil, errors.Wrap(err, "failed to parse database list")
		}
		databases = append(databases, listResp.Databases...)

		if listResp.NextPageToken == "" {
			break
		}
		pageToken = listResp.NextPageToken
	}
	return databases, nil
}

// matchDatabases applies tiered matching (exact > case-insensitive > substring) and
// returns the resolved result, an ambiguous result, or a not-found error.
func matchDatabases(databases []databaseEntry, database, instance, project string) (*resolvedDatabase, error) {
	matches := matchExact(databases, database)
	if len(matches) == 0 {
		matches = matchCaseInsensitive(databases, database)
	}
	if len(matches) == 0 {
		matches = matchSubstring(databases, database)
	}

	if len(matches) == 0 {
		suggestion := "check the database name or use search_api to list available databases"
		if instance != "" || project != "" {
			suggestion = "try without instance/project filters, or use search_api to list available databases"
		}
		return nil, &toolError{
			Code:       "DATABASE_NOT_FOUND",
			Message:    fmt.Sprintf("no database matching %q", database),
			Suggestion: suggestion,
		}
	}

	if len(matches) > 1 {
		return buildAmbiguousResult(matches), nil
	}

	db := matches[0]
	return &resolvedDatabase{
		resourceName: db.Name,
		dataSourceID: selectDataSource(db.InstanceResource.DataSources),
		engine:       db.InstanceResource.Engine,
	}, nil
}

// buildAmbiguousResult constructs a resolvedDatabase with multiple candidates.
func buildAmbiguousResult(matches []databaseEntry) *resolvedDatabase {
	candidates := make([]Candidate, 0, len(matches))
	dsIDs := make(map[string]string, len(matches))
	engines := make(map[string]string, len(matches))
	for _, db := range matches {
		candidates = append(candidates, Candidate{
			Database: db.Name,
			Instance: extractShortName(db.InstanceResource.Name),
			Project:  extractShortName(db.Project),
			Engine:   db.InstanceResource.Engine,
		})
		dsIDs[db.Name] = selectDataSource(db.InstanceResource.DataSources)
		engines[db.Name] = db.InstanceResource.Engine
	}
	return &resolvedDatabase{
		ambiguous:     true,
		candidates:    candidates,
		dataSourceIDs: dsIDs,
		engines:       engines,
	}
}

// resolveDatabase resolves a database name to a unique resource using tiered matching.
func (s *Server) resolveDatabase(ctx context.Context, database, instance, project string) (*resolvedDatabase, error) {
	databases, err := s.listDatabases(ctx, buildDatabaseFilter(database, instance, project))
	if err != nil {
		return nil, err
	}
	return matchDatabases(databases, database, instance, project)
}

// elicitDatabaseChoice prompts the user to pick from ambiguous database matches
// using MCP elicitation. Returns an error if elicitation is unsupported, the user
// cancels/declines, or the selection is invalid.
func (*Server) elicitDatabaseChoice(ctx context.Context, req *mcp.CallToolRequest, resolved *resolvedDatabase) (*resolvedDatabase, error) {
	if req == nil || req.Session == nil {
		return nil, errors.New("elicitation unsupported: no session")
	}

	// Build enum values and a lookup map from display label to resource name.
	enumValues := make([]any, 0, len(resolved.candidates))
	resourceByLabel := make(map[string]string, len(resolved.candidates))
	for _, c := range resolved.candidates {
		label := fmt.Sprintf("%s (%s, %s)", c.Database, c.Instance, c.Engine)
		enumValues = append(enumValues, label)
		resourceByLabel[label] = c.Database
	}

	result, err := req.Session.Elicit(ctx, &mcp.ElicitParams{
		Mode:    "form",
		Message: "Multiple databases match. Which one do you want to query?",
		RequestedSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"database": map[string]any{
					"type":        "string",
					"enum":        enumValues,
					"description": "Select the target database",
				},
			},
			"required": []string{"database"},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Action != "accept" {
		return nil, errors.Errorf("user %sd database selection", result.Action)
	}

	selected, ok := result.Content["database"].(string)
	if !ok {
		return nil, errors.New("invalid database selection")
	}

	resourceName, ok := resourceByLabel[selected]
	if !ok {
		return nil, errors.Errorf("unknown database selection: %s", selected)
	}

	return &resolvedDatabase{
		resourceName: resourceName,
		dataSourceID: resolved.dataSourceIDs[resourceName],
		engine:       resolved.engines[resourceName],
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
