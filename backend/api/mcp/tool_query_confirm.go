package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// Claude Code shows four lines of a form's message, so a request that wraps
// into at most three 70-character lines goes under the line that names the
// database; an 80-column terminal shows those whole.
const (
	messageRequestLines = 3
	messageLineWidth    = 70
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
	return "the user has not confirmed this statement, so nothing ran"
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
		// Connect writes the value unpadded, and the protocol lets a peer pad it.
		value, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(detail.Value, "="))
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
func (s *Server) askToRun(ctx context.Context, req *mcp.CallToolRequest, confirmation runConfirmation, digest []byte, statement string, required *confirmationRequired) *mcp.CallToolResult {
	instance, database, err := common.GetInstanceDatabaseID(confirmation.database)
	if err != nil {
		instance, database = "", confirmation.database
	}
	if _, answered := elicitedDatabaseChoice(req); answered {
		return textResult(fmt.Sprintf(
			"Not run: this statement may write, so the user confirms it in a form, and this call already asked which database to use. "+
				"Call query_database again with database %q and instance %q to ask the user.", database, instance))
	}
	names := []string{fmt.Sprintf("instance %q", instance)}
	if required.environment != "" {
		names = append(names, fmt.Sprintf("environment %q", required.environment))
	}
	if required.schema != "" {
		names = append(names, fmt.Sprintf("schema %q", required.schema))
	}
	message := fmt.Sprintf("Run on database %q (%s):", database, strings.Join(names, ", "))
	const runsAtOnce = "Run now runs it at once, with no issue or approval."
	description := runsAtOnce
	if lines := wrappedLines(statement); lines <= messageRequestLines {
		message += "\n" + strings.TrimSpace(statement)
	} else {
		message += fmt.Sprintf("\nThe request runs to %d lines, shown in full below.\n%s", lines, runsAtOnce)
		description = statement
	}
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

// wrappedLines counts the lines statement takes at messageLineWidth.
func wrappedLines(statement string) int {
	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(statement), "\n") {
		lines += max(1, (utf8.RuneCountInString(line)+messageLineWidth-1)/messageLineWidth)
	}
	return lines
}

// unrunResult returns the result for an answer that runs nothing, or nil when
// the user chose Run now. A decline is reported before a stale answer, so a
// user who said no is not asked again.
func unrunResult(reply *confirmationReply) *mcp.CallToolResult {
	content, accepted, why := acceptedForm(reply.response)
	if !accepted {
		return textResult(fmt.Sprintf("Not run: %s, so nothing changed. Do not retry unless the user asks.", why))
	}
	if why := reply.staleReason("the request changed after the user was asked"); why != "" {
		return formatToolError(&toolError{
			Code:       "STALE_CONFIRMATION",
			Message:    fmt.Sprintf("%s, so nothing ran", why),
			Suggestion: "call query_database again to ask the user about the request as it is now",
		})
	}
	switch content[runChoiceField] {
	case runChoiceRun:
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
