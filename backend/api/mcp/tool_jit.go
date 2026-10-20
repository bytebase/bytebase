package mcp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/utils"
)

// SetJustInTimeAccessInput is the input for the set_just_in_time_access tool.
type SetJustInTimeAccessInput struct {
	Project string `json:"project"`
	Enabled bool   `json:"enabled"`
}

// SetJustInTimeAccessOutput is the output for the set_just_in_time_access tool.
type SetJustInTimeAccessOutput struct {
	Project string `json:"project"`
	Enabled bool   `json:"enabled"`
	Changed bool   `json:"changed"`
	Link    string `json:"link"`
}

const setJustInTimeAccessDescription = `Turn just-in-time access on or off for one Bytebase project. The user confirms the change in this client first.

| Parameter | Required | Description |
|-----------|----------|-------------|
| project   | Yes      | Project name, e.g. "projects/hr" |
| enabled   | Yes      | true to turn it on, false to turn it off |

Just-in-time access lets members request access grants in the project: time-boxed permission to run one statement, for a one-off query. With it off, new requests are refused, and active grants stop applying until it is turned on again; they are not revoked.

The tool shows the user the change and asks them to confirm in a form. It changes the setting only when they accept; declining or cancelling changes nothing. A client that cannot show the form gets the project's settings link, to change it in the Bytebase console instead. Do not ask in chat first: the form is the question.

Changing it needs bb.projects.update, as in the console. This is the only way to change it over MCP; call_api refuses ProjectService/UpdateProject.`

func (s *Server) registerSetJustInTimeAccessTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "set_just_in_time_access",
		Description: setJustInTimeAccessDescription,
	}, s.handleSetJustInTimeAccess)
}

// justInTimeChange is a validated set_just_in_time_access request.
type justInTimeChange struct {
	project string
	enabled bool
}

func (*justInTimeChange) confirmedProcedure() string {
	return v1connect.ProjectServiceUpdateProjectProcedure
}

func (c *justInTimeChange) confirmedResource() string { return c.project }

func (c *justInTimeChange) state() string {
	if c.enabled {
		return "on"
	}
	return "off"
}

// body is the UpdateProject request the user is asked to confirm. Its digest
// binds the answer to these arguments.
func (c *justInTimeChange) body() map[string]any {
	return map[string]any{
		"project":    map[string]any{"name": c.project, "allowJustInTimeAccess": c.enabled},
		"updateMask": "allowJustInTimeAccess",
	}
}

func parseJustInTimeChange(input SetJustInTimeAccessInput) (*justInTimeChange, error) {
	if _, err := common.GetProjectID(input.Project); err != nil {
		return nil, &toolError{
			Code:       "INVALID_ARGUMENT",
			Message:    fmt.Sprintf("invalid project %q", input.Project),
			Suggestion: "pass the project name, e.g. projects/hr",
		}
	}
	return &justInTimeChange{project: input.Project, enabled: input.Enabled}, nil
}

// projectView is the part of a Project the tool reads.
type projectView struct {
	AllowJustInTimeAccess bool `json:"allowJustInTimeAccess"`
}

func (s *Server) handleSetJustInTimeAccess(ctx context.Context, req *mcp.CallToolRequest, input SetJustInTimeAccessInput) (*mcp.CallToolResult, any, error) {
	change, err := parseJustInTimeChange(input)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	link := s.projectSettingsLink(ctx, change.project)

	// Checked before any answer is read: a client that declared no form cannot
	// have shown one, so an answer it sends is not a confirmation.
	if req == nil || req.Session == nil || !clientSupportsElicitation(req.Session) {
		return formatToolError(&toolError{
			Code: "CONFIRMATION_UNAVAILABLE",
			Message: fmt.Sprintf("this MCP client cannot show the confirmation form set_just_in_time_access needs, so just-in-time access in %s was not turned %s and nothing changed",
				change.project, change.state()),
			Suggestion: "change it signed in to the Bytebase console: " + link,
		}), nil, nil
	}

	// Read on both rounds: a project already in the requested state, before
	// asking or by the time the answer comes back, needs no write.
	var project projectView
	if err := s.readResource(ctx, "/bytebase.v1.ProjectService/GetProject", change.project, "get projects", "bb.projects.get", &project); err != nil {
		return formatToolError(err), nil, nil
	}
	if project.AllowJustInTimeAccess == change.enabled {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
				"Just-in-time access is already %s in %s, so nothing changed.\nProject settings: %s", change.state(), change.project, link)}}},
			&SetJustInTimeAccessOutput{Project: change.project, Enabled: change.enabled, Link: link}, nil
	}
	body, err := json.Marshal(change.body())
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	digest := sha256.Sum256(body)

	// An answer counts only with the request state this server signed when it
	// asked; without one the call is asked like a first one.
	if response, state, answered := confirmationAnswer(req); answered {
		if asked, signed := s.verifyConfirmationState(ctx, change, state); signed {
			result, output := s.setJustInTimeAccessOnAnswer(ctx, change, response, bytes.Equal(asked, digest[:]), link)
			return result, output, nil
		}
	}

	verb := "Turn " + change.state()
	return s.askToConfirm(ctx, change, digest[:], describeJustInTimeChange(ctx, change),
		verb+" just-in-time access",
		fmt.Sprintf("Check to turn %s just-in-time access in the project above. Leaving it unchecked, declining or cancelling changes nothing.", change.state()),
	), nil, nil
}

// describeJustInTimeChange builds the confirmation message. The project name
// enters through %q, so it cannot pose as a line of the message.
func describeJustInTimeChange(ctx context.Context, change *justInTimeChange) string {
	question := fmt.Sprintf("Turn %s just-in-time access in project %q", change.state(), change.project)
	if email := getUserEmail(ctx); email != "" {
		question += " as " + email
	}
	effect := "Active grants that have not expired apply again at once."
	if !change.enabled {
		effect = "Active grants here stop applying (not revoked); new requests are refused."
	}
	return question + "?\n" + effect
}

// setJustInTimeAccessOnAnswer acts on the answer to a form this server sent. A
// decline is reported before a change, so a user who said no is not asked again.
func (s *Server) setJustInTimeAccessOnAnswer(ctx context.Context, change *justInTimeChange, response mcp.InputResponse, unchanged bool, link string) (*mcp.CallToolResult, any) {
	if confirmed, why := confirmationOutcome(response); !confirmed {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
			"Just-in-time access in %s was not turned %s: %s, so nothing changed. Do not retry unless the user asks.\nProject settings: %s",
			change.project, change.state(), why, link)}}}, nil
	}
	if !unchanged {
		return formatToolError(&toolError{
			Code: "STALE_CONFIRMATION",
			Message: fmt.Sprintf("what the user confirmed does not match this call, so just-in-time access in %s was not turned %s and nothing changed",
				change.project, change.state()),
			Suggestion: "call set_just_in_time_access again to ask the user about this change",
		}), nil
	}
	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: change.confirmedProcedure(),
		Resource:  change.project,
	})
	resp, err := s.apiRequest(confirmedCtx, change.confirmedProcedure(), change.body())
	if err != nil {
		return formatToolError(errors.Wrapf(err, "failed to turn %s just-in-time access in %s", change.state(), change.project)), nil
	}
	if resp.Status >= 400 {
		return formatToolError(errors.Errorf("failed to turn %s just-in-time access in %s: HTTP %d: %s",
			change.state(), change.project, resp.Status, parseError(resp.Body))), nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
			"Turned %s just-in-time access in %s.\nProject settings: %s", change.state(), change.project, link)}}},
		&SetJustInTimeAccessOutput{Project: change.project, Enabled: change.enabled, Changed: true, Link: link}
}

func (s *Server) projectSettingsLink(ctx context.Context, project string) string {
	externalURL, _ := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, getWorkspaceID(ctx))
	return buildResourceURL(externalURL, project) + "/settings"
}
