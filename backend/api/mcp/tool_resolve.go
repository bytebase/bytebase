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

	"github.com/bytebase/bytebase/backend/common"
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
	resourceName string
	dataSourceID string
	engine       string
	project      string // "projects/{id}" from databaseEntry.Project
	ambiguous    bool
	candidates   []Candidate
	// Per-candidate lookups, filled for every match count. A unique match keeps
	// them so an elicited answer can be reconciled against what matches now.
	dataSourceIDs map[string]string // resourceName -> dataSourceID
	engines       map[string]string // resourceName -> engine
	projects      map[string]string // resourceName -> project
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
		filter += fmt.Sprintf(" && instance == %q", formatInstanceFilter(instance))
	}
	if project != "" {
		filter += fmt.Sprintf(" && project == %q", formatProjectFilter(project))
	}
	return filter
}

// formatInstanceFilter preserves a canonical instance resource name. A bare ID
// remains the workspace-instance shorthand for backwards compatibility; it
// must not be used to address a project instance.
func formatInstanceFilter(instance string) string {
	if _, _, err := common.GetInstanceResourceName(instance); err == nil {
		return instance
	}
	return common.FormatInstance(instance)
}

func formatProjectFilter(project string) string {
	if _, err := common.GetProjectID(project); err == nil {
		return project
	}
	return common.FormatProject(project)
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
			if message := parseError(resp.Body); IsPolicyRefusal(message) {
				return nil, &toolError{Code: "PERMISSION_DENIED", Message: message}
			}
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

	resolved := candidateView(matches)
	if len(matches) > 1 {
		resolved.ambiguous = true
		return resolved, nil
	}

	db := matches[0]
	resolved.resourceName = db.Name
	resolved.dataSourceID = selectDataSource(db.InstanceResource.DataSources)
	resolved.engine = db.InstanceResource.Engine
	resolved.project = db.Project
	return resolved, nil
}

// candidateView fills the candidate list and the per-candidate lookups for a
// match set. It is built for a unique match too: an answered elicitation is
// reconciled against it, and a stale answer has to be recognized as naming
// something else rather than silently replaced by what this resolve found.
func candidateView(matches []databaseEntry) *resolvedDatabase {
	candidates := make([]Candidate, 0, len(matches))
	dsIDs := make(map[string]string, len(matches))
	engines := make(map[string]string, len(matches))
	projects := make(map[string]string, len(matches))
	for _, db := range matches {
		candidates = append(candidates, Candidate{
			Database: db.Name,
			Instance: extractShortName(db.InstanceResource.Name),
			Project:  extractShortName(db.Project),
			Engine:   db.InstanceResource.Engine,
		})
		dsIDs[db.Name] = selectDataSource(db.InstanceResource.DataSources)
		engines[db.Name] = db.InstanceResource.Engine
		projects[db.Name] = db.Project
	}
	return &resolvedDatabase{
		candidates:    candidates,
		dataSourceIDs: dsIDs,
		engines:       engines,
		projects:      projects,
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

// resolveTarget resolves the database a tool was asked for and settles any
// elicitation it needs.
//
// Every tool that names a database goes through here. The answer has to be
// consulted even when this resolve is unique, and a call site that re-derived
// that condition and got it wrong would run against a database the caller never
// chose. Returns either a resolved database or the result to return unchanged.
func (s *Server) resolveTarget(ctx context.Context, req *mcp.CallToolRequest, database, instance, project string) (*resolvedDatabase, *mcp.CallToolResult) {
	resolveCtx, resolveCancel := context.WithTimeout(ctx, resolveTimeout)
	defer resolveCancel()

	resolved, err := s.resolveDatabase(resolveCtx, database, instance, project)
	if err != nil {
		return nil, formatToolError(err)
	}
	if _, answered := elicitedDatabaseChoice(req); !resolved.ambiguous && !answered {
		return resolved, nil
	}
	return s.elicitDatabaseChoice(req, resolved, database)
}

// databaseChoiceRequestID is the ID the ambiguous-database elicitation is filed
// under, and databaseChoiceField the form field it asks for. The client echoes
// both back, so the two halves of one tool call must agree on them.
const (
	databaseChoiceRequestID = "database"
	databaseChoiceField     = "database"
)

// elicitDatabaseChoice picks one database out of an ambiguous match.
//
// It returns either a resolved database or a result the caller must return
// unchanged, never both: an input request, or the AMBIGUOUS_TARGET listing when
// no one can be asked or the answer is unusable.
//
// One shape serves both client generations (SEP-2322). A client on the
// 2026-07-28 protocol answers the input request itself; for an older one the
// SDK's middleware answers it by eliciting and re-invoking this handler.
func (*Server) elicitDatabaseChoice(req *mcp.CallToolRequest, resolved *resolvedDatabase, database string) (*resolvedDatabase, *mcp.CallToolResult) {
	fallback := formatAmbiguousResult(database, resolved.candidates)
	if req == nil || req.Session == nil {
		return nil, fallback
	}
	enumValues, resourceByLabel := candidateLabels(resolved.candidates)

	// Read the answer before asking again: on the retry the SDK re-invokes this
	// handler, which resolves a second time, so resourceByLabel holds whatever
	// matches now. An answer naming something else finds no entry and falls back.
	// It is read even when this resolve is no longer ambiguous, because dropping
	// it there would run the statement against a database nobody chose.
	if response, answered := elicitedDatabaseChoice(req); answered {
		selected, accepted := acceptedDatabaseLabel(response)
		if !accepted {
			return nil, fallback
		}
		resourceName, current := resourceByLabel[selected]
		if !current {
			return nil, formatStaleChoiceResult(database, selected, resolved.candidates)
		}
		return &resolvedDatabase{
			resourceName: resourceName,
			dataSourceID: resolved.dataSourceIDs[resourceName],
			engine:       resolved.engines[resourceName],
			project:      resolved.projects[resourceName],
		}, nil
	}

	// For an older client the SDK answers the input request itself, and an error
	// there fails the whole tool call instead of returning something the model
	// can act on. So a client that cannot answer a form gets the listing.
	//
	// DEFER: only what the client declared up front is caught; an elicitation
	// that fails once under way still fails the call. Upgrade when the SDK lets
	// a handler see that error.
	if !clientSupportsElicitation(req.Session) {
		return nil, fallback
	}

	return nil, &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			databaseChoiceRequestID: &mcp.ElicitParams{
				Mode:    "form",
				Message: "Multiple databases match. Which one do you want to query?",
				RequestedSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						databaseChoiceField: map[string]any{
							"type":        "string",
							"enum":        enumValues,
							"description": "Select the target database",
						},
					},
					"required": []string{databaseChoiceField},
				},
			},
		},
	}
}

// elicitedDatabaseChoice returns the answer the client sent for the
// ambiguous-database elicitation, if this call carries one.
func elicitedDatabaseChoice(req *mcp.CallToolRequest) (mcp.InputResponse, bool) {
	if req == nil || req.Params == nil {
		return nil, false
	}
	response, answered := req.Params.InputResponses[databaseChoiceRequestID]
	return response, answered
}

// candidateLabels returns the enum values in candidate order and a lookup from
// label back to resource name.
func candidateLabels(candidates []Candidate) ([]any, map[string]string) {
	enumValues := make([]any, 0, len(candidates))
	resourceByLabel := make(map[string]string, len(candidates))
	for _, c := range candidates {
		label := fmt.Sprintf("%s (%s, %s)", c.Database, c.Instance, c.Engine)
		enumValues = append(enumValues, label)
		resourceByLabel[label] = c.Database
	}
	return enumValues, resourceByLabel
}

// acceptedDatabaseLabel returns the label the client picked. It reports false
// when the answer carries no pick: declined, cancelled, or malformed.
func acceptedDatabaseLabel(response mcp.InputResponse) (string, bool) {
	elicited, ok := response.(*mcp.ElicitResult)
	if !ok || elicited.Action != "accept" {
		return "", false
	}
	selected, ok := elicited.Content[databaseChoiceField].(string)
	return selected, ok
}

func clientSupportsElicitation(session *mcp.ServerSession) bool {
	params := session.InitializeParams()
	if params == nil || params.Capabilities == nil || params.Capabilities.Elicitation == nil {
		return false
	}
	// The input request asks for a form, and the SDK refuses form mode to a
	// client that advertised URL elicitation without it. A client advertising
	// neither is assumed to support form, for clients older than the split.
	elicitation := params.Capabilities.Elicitation
	return elicitation.Form != nil || elicitation.URL == nil
}

// formatStaleChoiceResult answers a pick that no longer matches.
//
// It cannot reuse formatAmbiguousResult: that one says to narrow by instance,
// and one candidate is often all that is left, so following it would re-issue
// the call against the database the caller did not pick.
func formatStaleChoiceResult(database, selected string, candidates []Candidate) *mcp.CallToolResult {
	result := struct {
		Code       string      `json:"code"`
		Message    string      `json:"message"`
		Candidates []Candidate `json:"candidates"`
	}{
		Code: "STALE_TARGET",
		Message: fmt.Sprintf("The database chosen for %q (%s) no longer matches. "+
			"Choose again from the candidates below, or name one explicitly.", database, selected),
		Candidates: candidates,
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: true,
	}
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

// extractDatabaseName extracts the database ID from a canonical database resource name.
func extractDatabaseName(resourceName string) string {
	_, _, databaseName, err := common.GetDatabaseResourceName(resourceName)
	if err == nil {
		return databaseName
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
