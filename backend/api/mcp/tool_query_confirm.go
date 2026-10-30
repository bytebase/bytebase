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
func (s *Server) askToRun(ctx context.Context, req *mcp.CallToolRequest, confirmation runConfirmation, digest []byte, statement, container string, required *confirmationRequired) *mcp.CallToolResult {
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
	return s.askWithForm(ctx, confirmation, digest, message, map[string]any{
		"type": "object",
		"properties": map[string]any{
			runChoiceField: map[string]any{
				"type":        "string",
				"title":       "Run this request?",
				"description": description,
				"oneOf": []any{
					map[string]any{"const": runChoiceRun, "title": "Run now"},
					map[string]any{"const": runChoiceChange, "title": "Change it in chat"},
				},
				"default": runChoiceRun,
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

// unrunResult returns the result for an answer that runs nothing, or nil when
// the user chose Run now. Only Run now is checked for staleness, so a user who
// said no or asked for a change is not asked again.
func unrunResult(reply *confirmationReply) *mcp.CallToolResult {
	content, accepted, why := acceptedForm(reply.response)
	if !accepted {
		return textResult(fmt.Sprintf("Not run: %s, so nothing changed. Do not retry unless the user asks.", why))
	}
	switch content[runChoiceField] {
	case runChoiceRun:
		if why := reply.staleReason("the request changed after the user was asked"); why != "" {
			return formatToolError(&toolError{
				Code:       "STALE_CONFIRMATION",
				Message:    fmt.Sprintf("%s, so nothing ran", why),
				Suggestion: "call query_database again to ask the user about the request as it is now",
			})
		}
		return nil
	case runChoiceChange:
		return textResult("Not run: the user wants to change this request. Ask them what to change, then call query_database with the revised request.")
	default:
		return textResult("Not run: the answer chose neither option, so nothing changed.")
	}
}

// textResult carries text alone, which a client shows the model whole.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
