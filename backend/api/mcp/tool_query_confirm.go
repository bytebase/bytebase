package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/text/width"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// The choice field of the form that asks before a statement runs, and its
// values.
const (
	runChoiceField  = "choice"
	runChoiceRun    = "run"
	runChoiceIssue  = "issue"
	runChoiceChange = "change"
)

// Claude Code shows four lines of a form's message and cuts each line six
// columns short of the terminal width. A request goes in the message only when
// it fits under the header in lines of at most 70 columns.
const (
	messageLines     = 4
	messageLineWidth = 70
)

// runConfirmation is a statement query_database asks the user to run.
type runConfirmation struct {
	database string
}

func (runConfirmation) confirmedProcedure() string { return v1connect.SQLServiceQueryProcedure }

func (c runConfirmation) confirmedResource() string { return c.database }

// confirmationRequired is Query's refusal of a statement that may write and that
// the user has not confirmed, with what the form shows.
type confirmationRequired struct {
	environment string
	schema      string
}

func (*confirmationRequired) Error() string {
	return "Bytebase refused this statement's confirmation, so nothing ran; asking the user again will not help"
}

// confirmationRequiredIn returns the refusal body carries, or nil when body is
// another error.
func confirmationRequiredIn(body json.RawMessage) *confirmationRequired {
	var wire struct {
		Details []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"details"`
	}
	// A body that is not JSON carries no details.
	_ = json.Unmarshal(body, &wire)
	for _, detail := range wire.Details {
		if detail.Type != "google.rpc.ErrorInfo" {
			continue
		}
		value, err := connect.DecodeBinaryHeader(detail.Value)
		if err != nil {
			continue
		}
		var info errdetails.ErrorInfo
		if proto.Unmarshal(value, &info) != nil || info.GetDomain() != auth.MCPConfirmationDomain || info.GetReason() != auth.MCPConfirmationReason {
			continue
		}
		return &confirmationRequired{environment: info.GetMetadata()["environment"], schema: info.GetMetadata()["schema"]}
	}
	return nil
}

// askToRun returns the form that asks the user to run statement. A call that
// already asked which database to use gets a result that has the agent call
// again instead: an older client answers one form per call.
func (s *Server) askToRun(ctx context.Context, req *mcp.CallToolRequest, confirmation runConfirmation, digest []byte, statement, container string, required *confirmationRequired, offerIssue bool) *mcp.CallToolResult {
	project, instance, database, err := common.GetDatabaseResourceName(confirmation.database)
	if err != nil {
		return formatToolError(err)
	}
	if _, answered := elicitedDatabaseChoice(req); answered {
		instanceName := instance
		if project != nil {
			instanceName = common.FormatProjectInstance(*project, instance)
		}
		return textResult(fmt.Sprintf(
			"Not run: this statement may write, so the user confirms it in a form, and this call already asked which database to use. "+
				"Call query_database again with database %q and instance %q to ask the user.", database, instanceName))
	}
	header := headerLines(required, database, instance, container)
	const runsAtOnce = "Run now runs it at once, with no issue or approval."
	request := strings.Split(strings.TrimSpace(statement), "\n")
	description := runsAtOnce
	if len(header)+len(request) > messageLines || !fitsMessageLines(request) {
		notes := []string{fmt.Sprintf("The request is under the choice (%d characters).", utf8.RuneCountInString(statement)), runsAtOnce}
		request = notes[:min(len(notes), messageLines-len(header))]
		description = statement
		if len(request) < len(notes) {
			description += "\n\n" + runsAtOnce
		}
	}
	message := strings.Join(append(header, request...), "\n")
	choices := []any{map[string]any{"const": runChoiceRun, "title": "Run now"}}
	if offerIssue {
		choices = append(choices, map[string]any{"const": runChoiceIssue, "title": "Create an issue"})
	}
	choices = append(choices, map[string]any{"const": runChoiceChange, "title": "Change it in chat"})
	return s.askWithForm(ctx, confirmation, digest, message, map[string]any{
		"type": "object",
		"properties": map[string]any{
			runChoiceField: map[string]any{
				"type":        "string",
				"title":       "Run this request?",
				"description": description,
				"oneOf":       choices,
				"default":     runChoiceRun,
			},
		},
		"required": []string{runChoiceField},
	})
}

// headerLines names where the request runs, the environment before the
// database so a long database name cannot hide it. Names too long for two lines
// go on up to four, whose labels leave room for a 63-character ID or PostgreSQL
// database name in the 74 columns an 80-column terminal shows.
func headerLines(required *confirmationRequired, database, instance, container string) []string {
	where := fmt.Sprintf("Run on database %q,", database)
	if required.environment != "" {
		where = fmt.Sprintf("Run in environment %q on database %q,", required.environment, database)
	}
	scope := fmt.Sprintf("instance %q", instance)
	if required.schema != "" {
		scope += fmt.Sprintf(", schema %q", required.schema)
	}
	if container != "" {
		scope += fmt.Sprintf(", container %q", container)
	}
	if header := []string{where, scope + ":"}; fitsMessageLines(header) {
		return header
	}
	var lines []string
	if required.environment != "" {
		lines = append(lines, fmt.Sprintf("Run in %q:", required.environment))
	}
	lines = append(lines, fmt.Sprintf("Database %q", database), fmt.Sprintf("Instance %q", instance))
	// The schema and the container share a line, so the names never take more
	// than the four lines Claude Code shows.
	var scopes []string
	if required.schema != "" {
		scopes = append(scopes, fmt.Sprintf("schema %q", required.schema))
	}
	if container != "" {
		scopes = append(scopes, fmt.Sprintf("container %q", container))
	}
	if len(scopes) > 0 {
		lines = append(lines, capitalize(strings.Join(scopes, ", ")))
	}
	return lines
}

// fitsMessageLines reports whether every line is at most messageLineWidth
// columns wide.
func fitsMessageLines(lines []string) bool {
	for _, line := range lines {
		if lineColumns(line) > messageLineWidth {
			return false
		}
	}
	return true
}

// lineColumns counts the columns Claude Code gives line: two for a wide
// character, none for a combining mark.
func lineColumns(line string) int {
	columns := 0
	for _, r := range line {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		columns++
		if kind := width.LookupRune(r).Kind(); kind == width.EastAsianWide || kind == width.EastAsianFullwidth {
			columns++
		}
	}
	return columns
}

// answeredChoice returns the choice an answer acts on, Run now or Create an
// issue, or else the result for an answer that acts on nothing. It reads no
// target or signed state, so a request that changed while the user was asked
// cannot turn a decline into a form.
func answeredChoice(response mcp.InputResponse) (string, *mcp.CallToolResult) {
	content, accepted, why := acceptedForm(response)
	if !accepted {
		return "", textResult(fmt.Sprintf("Not run: %s, so nothing changed. Do not retry unless the user asks.", why))
	}
	switch content[runChoiceField] {
	case runChoiceRun:
		return runChoiceRun, nil
	case runChoiceIssue:
		return runChoiceIssue, nil
	case runChoiceChange:
		return "", textResult("Not run: the user wants to change this request. Ask them what to change, then call query_database with the revised request.")
	default:
		return "", noOfferedChoice()
	}
}

// noOfferedChoice reports an answer that chose none of the form's choices.
func noOfferedChoice() *mcp.CallToolResult {
	return textResult("Not run: the answer chose none of the form's choices, so nothing changed.")
}

// staleAnswer reports a Run now or Create an issue answer that no longer fits
// the request. It asks nothing: an older client answers one form per call.
func staleAnswer(why string) *mcp.CallToolResult {
	return formatToolError(&toolError{
		Code:       "STALE_CONFIRMATION",
		Message:    fmt.Sprintf("%s, so nothing ran and no issue was filed", why),
		Suggestion: "call query_database again to ask the user about the request as it is now",
	})
}

// fileAsIssue files statement against the database as a change issue, with
// propose_database_change's steps but no rollout: Bytebase creates the rollout
// once the issue is approved and its plan checks pass.
func (s *Server) fileAsIssue(ctx context.Context, resolved *resolvedDatabase, statement string) *mcp.CallToolResult {
	_, _, database, err := common.GetDatabaseResourceName(resolved.resourceName)
	if err != nil {
		return formatToolError(err)
	}
	var project struct {
		EnforceIssueTitle bool `json:"enforceIssueTitle"`
	}
	if err := s.readResource(ctx, "/bytebase.v1.ProjectService/GetProject", resolved.project, "get projects", "bb.projects.get", &project); err != nil {
		return formatToolError(err)
	}
	// The server refuses only an empty title, so a generated one would pass in a
	// project that requires a manual title; the console leaves it to the user
	// there (frontend/src/lib/plan/title.ts).
	if project.EnforceIssueTitle {
		return textResult(fmt.Sprintf("Not run: the user chose Create an issue, and project %s requires a manual title, so Bytebase files nothing until the user writes one. "+
			"Ask the user for the issue title, then call propose_database_change with this statement and that title as written.", resolved.project))
	}
	// The title the console's AI agent gives a change it drafts
	// (agent.plan-draft.default-title).
	filed, failed := s.fileChange(ctx, resolved.project, resolved.resourceName, statement, fmt.Sprintf("[%s] Proposed database change", database), "")
	if failed != nil {
		return failed
	}
	return textResult(fmt.Sprintf("Not run: the user chose Create an issue, so the request is filed as issue %s (%s). The issue page shows its approval status.%s",
		filed.issue, s.consoleLink(ctx, filed.issue), planCheckNote(filed.planChecks)))
}

// planCheckNote reports plan checks that may hold the rollout: Bytebase creates
// it on its own only once they are done with no error.
func planCheckNote(checks *PlanCheckInfo) string {
	if checks != nil && checks.Status == planCheckRunning {
		return " Its plan checks are still running; the issue page shows their result."
	}
	if !planChecksHaveErrors(checks) {
		return ""
	}
	var errs []string
	for _, result := range checks.Results {
		if result.Type == "ERROR" {
			errs = append(errs, result.Message)
		}
	}
	note := " Its plan checks did not pass, so Bytebase will not create its rollout on its own"
	if len(errs) > 0 {
		note += ": " + strings.Join(errs, "; ")
	}
	return note + "."
}

// textResult carries text alone, which a client shows the model whole.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
