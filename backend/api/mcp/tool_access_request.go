package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// RequestAccessGrantInput is the input for the request_access_grant tool.
type RequestAccessGrantInput struct {
	Database  string `json:"database"`
	Statement string `json:"statement"`
	Schema    string `json:"schema,omitempty"`
	Container string `json:"container,omitempty"`
	Unmask    bool   `json:"unmask,omitempty"`
	Export    bool   `json:"export,omitempty"`
	Reason    string `json:"reason"`
	// An int32 gives the input schema a maximum, so the seconds computed from
	// it fit in an int64.
	DurationHours int32 `json:"durationHours"`
}

// RequestAccessGrantOutput is the output for the request_access_grant tool.
type RequestAccessGrantOutput struct {
	AccessGrant   string `json:"accessGrant"`
	Issue         string `json:"issue"`
	DurationHours int32  `json:"durationHours"`
	Link          string `json:"link"`
}

const requestAccessGrantDescription = `Request an access grant: time-boxed permission to run one read-only statement on one Bytebase database, optionally with results unmasked or exported. A person approves the request in Bytebase before it takes effect.

| Parameter     | Required | Description |
|---------------|----------|-------------|
| database      | Yes      | Full database name, as query_database prints it after "Database:", e.g. "projects/hr/instances/prod/databases/hr" |
| statement     | Yes      | One read-only statement; the grant applies to this exact statement |
| schema        | No       | Default schema for unqualified names; the rerun must pass the same one |
| container     | No       | CosmosDB container; the rerun must pass the same one |
| unmask        | No       | true to return masked columns unmasked |
| export        | No       | true to allow exporting the results |
| reason        | Yes      | Why the access is needed; the approver reads it |
| durationHours | Yes      | Hours of access, counted from approval. The SQL Editor offers 1, 4, 24 and 168; the workspace maximum applies |

The project must allow just-in-time access. Check allowJustInTimeAccess with call_api(operationId="ProjectService/GetProject") first; a missing field means off, and set_just_in_time_access turns it on.

Clients such as Claude Code ask the user to approve each call of this tool, so do not ask in chat first. Once the request is approved, rerun the statement unchanged with query_database, with the same schema and container; appliedAccessGrant in its result names the grant that applied.`

func (s *Server) registerRequestAccessGrantTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "request_access_grant",
		Description: requestAccessGrantDescription,
		Meta:        requiresUserInteraction(),
	}, s.handleRequestAccessGrant)
}

func (s *Server) handleRequestAccessGrant(ctx context.Context, _ *mcp.CallToolRequest, input RequestAccessGrantInput) (*mcp.CallToolResult, any, error) {
	if input.DurationHours < 1 {
		return formatToolError(&toolError{
			Code:       "INVALID_ARGUMENT",
			Message:    fmt.Sprintf("invalid durationHours %d: it must be at least 1", input.DurationHours),
			Suggestion: "pass the hours of access the user needs, counted from approval, e.g. 4",
		}), nil, nil
	}
	// The request's parent is the database's project, which the name of a
	// database on a workspace instance does not carry.
	var database struct {
		Project string `json:"project"`
	}
	if err := s.readResource(ctx, v1connect.DatabaseServiceGetDatabaseProcedure, input.Database, "get databases", "bb.databases.get", &database); err != nil {
		return formatToolError(err), nil, nil
	}
	resp, err := s.apiRequest(ctx, v1connect.AccessGrantServiceCreateAccessGrantProcedure, accessGrantBody(database.Project, input))
	if err != nil {
		return formatToolError(errors.Wrapf(err, "failed to request access to %s", input.Database)), nil, nil
	}
	if err := checkAPIResponse(resp, "request access grants", "bb.accessGrants.create"); err != nil {
		return formatToolError(err), nil, nil
	}
	var grant struct {
		Name  string `json:"name"`
		Issue string `json:"issue"`
	}
	if err := json.Unmarshal(resp.Body, &grant); err != nil {
		return formatToolError(errors.Wrapf(err, "requested access to %s, but the response could not be read", input.Database)), nil, nil
	}
	link := s.issueLink(ctx, grant.Issue)
	output := &RequestAccessGrantOutput{
		AccessGrant:   grant.Name,
		Issue:         grant.Issue,
		DurationHours: input.DurationHours,
		Link:          link,
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
		"Requested %s: %s for %s from approval. A person approves it in Bytebase before it takes effect.\nIssue: %s\nOnce it is approved, rerun the statement unchanged with query_database, with the same schema and container.",
		grant.Name, describeRequestedAccess(input), hours(input.DurationHours), link)}}}, output, nil
}

func hours(n int32) string {
	if n == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", n)
}

// accessGrantBody is the CreateAccessGrant request. A ttl, unlike an
// expire_time, starts when the request is approved.
func accessGrantBody(project string, input RequestAccessGrantInput) map[string]any {
	return map[string]any{
		"parent": project,
		"accessGrant": map[string]any{
			"targets":   []string{input.Database},
			"query":     input.Statement,
			"schema":    input.Schema,
			"container": input.Container,
			"unmask":    input.Unmask,
			"export":    input.Export,
			"reason":    input.Reason,
			"ttl":       fmt.Sprintf("%ds", int64(input.DurationHours)*3600),
		},
	}
}

func describeRequestedAccess(input RequestAccessGrantInput) string {
	access := "run the statement on " + input.Database
	if input.Unmask {
		access += ", unmasked"
	}
	if input.Export {
		access += ", with export"
	}
	return access
}
