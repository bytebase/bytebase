package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/utils"
)

// DecideIssueInput is the input for the decide_issue tool.
type DecideIssueInput struct {
	Issue    string `json:"issue"`
	Decision string `json:"decision"`
	Comment  string `json:"comment,omitempty"`
}

// DecideIssueOutput is the output for the decide_issue tool.
type DecideIssueOutput struct {
	Issue          string `json:"issue"`
	Decision       string `json:"decision"`
	ApprovalStatus string `json:"approvalStatus"`
	Status         string `json:"status"`
	Link           string `json:"link"`
}

const decideIssueDescription = `Approve or reject a Bytebase issue at its current approval step. The user confirms the decision in this client first.

| Parameter | Required | Description |
|-----------|----------|-------------|
| issue     | Yes      | Issue name, e.g. "projects/hr/issues/42" |
| decision  | Yes      | "approve" or "reject" |
| comment   | No       | Note recorded with the decision |

The tool shows the user what the issue changes and asks them to confirm in a form. It decides only when they accept; declining or cancelling changes nothing. A client that cannot show the form gets the issue's link, to decide it in the Bytebase console instead. Do not ask in chat first: the form is the question.

The approval rules decide who may approve, as in the console: the user must hold the role of the current step, and approving their own issue, or a plan they last changed, is refused unless the project allows it.

This is the only way to decide an issue over MCP; call_api refuses IssueService/ApproveIssue and RejectIssue.`

// confirmDecisionRequestID is the ID the confirmation is filed under, and
// confirmDecisionField the form field it asks for. The client echoes both back,
// so the two halves of one tool call must agree on them.
const (
	confirmDecisionRequestID = "confirm"
	confirmDecisionField     = "confirm"
)

// decisionNote opens the comment recorded with every decision. It claims the
// client confirmed, not that a person did: a hook in the client can answer.
const decisionNote = "Decided over MCP; the decision was confirmed in the MCP client."

const (
	// maxDescribedSpecs bounds how many plan specs the confirmation lists.
	maxDescribedSpecs = 3
	// maxStatementPreview bounds a statement in the confirmation, in runes.
	maxStatementPreview = 300
)

func (s *Server) registerDecideIssueTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "decide_issue",
		Description: decideIssueDescription,
	}, s.handleDecideIssue)
}

// issueDecision is a validated decide_issue request.
type issueDecision struct {
	issue     string
	approve   bool
	procedure string
	comment   string
}

func (d *issueDecision) verb() string {
	if d.approve {
		return "approve"
	}
	return "reject"
}

func (d *issueDecision) pastTense() string {
	if d.approve {
		return "approved"
	}
	return "rejected"
}

func (s *Server) handleDecideIssue(ctx context.Context, req *mcp.CallToolRequest, input DecideIssueInput) (*mcp.CallToolResult, any, error) {
	decision, err := parseIssueDecision(input)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	link := s.issueLink(ctx, decision.issue)

	// Checked before any answer is read: a client that declared no form cannot
	// have shown one, so an answer it sends is not a confirmation.
	if req == nil || req.Session == nil || !clientSupportsElicitation(req.Session) {
		return formatToolError(&toolError{
			Code: "CONFIRMATION_UNAVAILABLE",
			Message: fmt.Sprintf("this MCP client cannot show the confirmation form decide_issue needs, so %s was not %s and nothing changed",
				decision.issue, decision.pastTense()),
			Suggestion: "decide the issue signed in to the Bytebase console: " + link,
		}), nil, nil
	}

	if req.Params != nil {
		if response, answered := req.Params.InputResponses[confirmDecisionRequestID]; answered {
			result, output := s.applyIssueDecision(ctx, decision, response, link)
			return result, output, nil
		}
	}

	message, err := s.describeIssueDecision(ctx, decision, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	// The same multi round-trip shape as elicitDatabaseChoice: the SDK elicits
	// within this tools/call and re-invokes the handler with the answer.
	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			confirmDecisionRequestID: &mcp.ElicitParams{
				Mode:    "form",
				Message: message,
				RequestedSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						confirmDecisionField: map[string]any{
							"type":        "boolean",
							"title":       fmt.Sprintf("%s this issue", capitalize(decision.verb())),
							"description": fmt.Sprintf("Check to %s the issue described above. Leaving it unchecked, declining or cancelling changes nothing.", decision.verb()),
						},
					},
					"required": []string{confirmDecisionField},
				},
			},
		},
	}, nil, nil
}

func parseIssueDecision(input DecideIssueInput) (*issueDecision, error) {
	if _, _, err := common.GetProjectIDIssueUID(input.Issue); err != nil {
		return nil, &toolError{
			Code:       "INVALID_ARGUMENT",
			Message:    fmt.Sprintf("invalid issue %q", input.Issue),
			Suggestion: "pass the issue name, e.g. projects/hr/issues/42",
		}
	}
	decision := &issueDecision{issue: input.Issue, comment: strings.TrimSpace(input.Comment)}
	switch strings.ToLower(strings.TrimSpace(input.Decision)) {
	case "approve":
		decision.approve = true
		decision.procedure = v1connect.IssueServiceApproveIssueProcedure
	case "reject":
		decision.procedure = v1connect.IssueServiceRejectIssueProcedure
	default:
		return nil, &toolError{
			Code:       "INVALID_ARGUMENT",
			Message:    fmt.Sprintf("invalid decision %q", input.Decision),
			Suggestion: `allowed values: "approve", "reject"`,
		}
	}
	return decision, nil
}

// applyIssueDecision acts on the user's answer. Only an accepted form with the
// box checked decides; every other answer changes nothing.
func (s *Server) applyIssueDecision(ctx context.Context, decision *issueDecision, response mcp.InputResponse, link string) (*mcp.CallToolResult, any) {
	if confirmed, why := confirmationOutcome(response); !confirmed {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
			"Not %s: %s, so nothing changed. Do not retry unless the user asks.\nIssue: %s",
			decision.pastTense(), why, link)}}}, nil
	}

	comment := decisionNote
	if decision.comment != "" {
		comment += "\n\n" + decision.comment
	}
	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: decision.procedure,
		Issue:     decision.issue,
	})
	resp, err := s.apiRequest(confirmedCtx, decision.procedure, map[string]any{
		"name":    decision.issue,
		"comment": comment,
	})
	if err != nil {
		return formatToolError(errors.Wrapf(err, "failed to %s %s", decision.verb(), decision.issue)), nil
	}
	if resp.Status >= 400 {
		return formatToolError(&toolError{
			Code:       "DECISION_REFUSED",
			Message:    fmt.Sprintf("Bytebase refused to %s %s, so nothing changed: %s", decision.verb(), decision.issue, parseError(resp.Body)),
			Suggestion: "the approval rules decide who may decide this issue; an approver can decide it in the Bytebase console: " + link,
		}), nil
	}

	var decided issueView
	if err := json.Unmarshal(resp.Body, &decided); err != nil {
		return formatToolError(errors.Wrapf(err, "%s %s, but its response could not be read", decision.pastTense(), decision.issue)), nil
	}
	output := &DecideIssueOutput{
		Issue:          decision.issue,
		Decision:       decision.verb(),
		ApprovalStatus: decided.ApprovalStatus,
		Status:         decided.Status,
		Link:           link,
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
		"%s %s. Approval: %s. Issue status: %s.\nIssue: %s",
		capitalize(decision.pastTense()), decision.issue, describeApprovalStep(decided), decided.Status, link)}}}, output
}

// confirmationOutcome reports whether the answer confirms the decision, and if
// not, why, in words the result can carry.
func confirmationOutcome(response mcp.InputResponse) (bool, string) {
	elicited, ok := response.(*mcp.ElicitResult)
	if !ok || elicited == nil {
		return false, "the confirmation answer could not be read"
	}
	switch elicited.Action {
	case "accept":
		if confirmed, ok := elicited.Content[confirmDecisionField].(bool); ok && confirmed {
			return true, ""
		}
		return false, "the form was submitted without the confirmation checked"
	case "decline":
		return false, "the user declined"
	case "cancel":
		return false, "the user cancelled"
	default:
		return false, fmt.Sprintf("the confirmation answer %q is not one Bytebase acts on", elicited.Action)
	}
}

// issueView is the part of an Issue the confirmation and the result read.
type issueView struct {
	Name             string `json:"name"`
	Title            string `json:"title"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	Creator          string `json:"creator"`
	Plan             string `json:"plan"`
	AccessGrant      string `json:"accessGrant"`
	ApprovalStatus   string `json:"approvalStatus"`
	ApprovalTemplate struct {
		Flow struct {
			Roles []string `json:"roles"`
		} `json:"flow"`
	} `json:"approvalTemplate"`
	Approvers []struct {
		Status string `json:"status"`
	} `json:"approvers"`
	RoleGrant *struct {
		Role      string `json:"role"`
		User      string `json:"user"`
		Condition *struct {
			Expression string `json:"expression"`
		} `json:"condition"`
		Expiration string `json:"expiration"`
	} `json:"roleGrant"`
}

type planView struct {
	Specs []struct {
		ChangeDatabaseConfig *struct {
			Targets []string `json:"targets"`
			Sheet   string   `json:"sheet"`
			Release string   `json:"release"`
		} `json:"changeDatabaseConfig"`
		CreateDatabaseConfig *struct {
			Target   string `json:"target"`
			Database string `json:"database"`
		} `json:"createDatabaseConfig"`
	} `json:"specs"`
}

type sheetView struct {
	Content string `json:"content"`
}

type accessGrantView struct {
	Creator    string   `json:"creator"`
	Targets    []string `json:"targets"`
	Query      string   `json:"query"`
	Unmask     bool     `json:"unmask"`
	Export     bool     `json:"export"`
	ExpireTime string   `json:"expireTime"`
	TTL        string   `json:"ttl"`
}

// describeIssueDecision builds the confirmation message: the decision, the
// issue, what approving it would let happen, and the step it decides. Free
// text from the issue is quoted or folded to one line, so it cannot pose as
// one of the message's own lines.
//
// It fails closed: if any part cannot be read, nothing is asked.
func (s *Server) describeIssueDecision(ctx context.Context, decision *issueDecision, link string) (string, error) {
	var issue issueView
	if err := s.readResource(ctx, "/bytebase.v1.IssueService/GetIssue", decision.issue, "get issues", "bb.issues.get", &issue); err != nil {
		return "", err
	}
	question := fmt.Sprintf("%s issue %q", capitalize(decision.verb()), issue.Title)
	if email := getUserEmail(ctx); email != "" {
		question += " as " + email
	}
	lines := []string{
		question + "?",
		"",
		fmt.Sprintf("Issue: %s (%s, requested by %s)", issue.Name, issue.Type, strings.TrimPrefix(issue.Creator, common.UserNamePrefix)),
	}
	var details []string
	var err error
	switch {
	case issue.AccessGrant != "":
		details, err = s.describeAccessGrant(ctx, issue.AccessGrant)
	case issue.Plan != "":
		details, err = s.describePlan(ctx, issue.Plan)
	case issue.RoleGrant != nil:
		details = []string{describeRoleGrant(issue)}
	default:
	}
	if err != nil {
		return "", err
	}
	lines = append(lines, details...)
	lines = append(lines, "Approval: "+describeApprovalStep(issue), "Details: "+link)
	return strings.Join(lines, "\n"), nil
}

func (s *Server) describePlan(ctx context.Context, plan string) ([]string, error) {
	var view planView
	if err := s.readResource(ctx, "/bytebase.v1.PlanService/GetPlan", plan, "get plans", "bb.plans.get", &view); err != nil {
		return nil, err
	}
	var lines []string
	for i, spec := range view.Specs {
		if i == maxDescribedSpecs {
			lines = append(lines, fmt.Sprintf("...and %d more changes", len(view.Specs)-maxDescribedSpecs))
			break
		}
		switch {
		case spec.ChangeDatabaseConfig != nil:
			config := spec.ChangeDatabaseConfig
			change := "release " + config.Release
			if config.Sheet != "" {
				statement, err := s.statementPreview(ctx, config.Sheet)
				if err != nil {
					return nil, err
				}
				change = statement
			}
			lines = append(lines, fmt.Sprintf("Change on %s: %s", strings.Join(config.Targets, ", "), change))
		case spec.CreateDatabaseConfig != nil:
			lines = append(lines, fmt.Sprintf("Create database %s on %s", spec.CreateDatabaseConfig.Database, spec.CreateDatabaseConfig.Target))
		default:
		}
	}
	return lines, nil
}

func (s *Server) statementPreview(ctx context.Context, sheet string) (string, error) {
	var view sheetView
	if err := s.readResource(ctx, "/bytebase.v1.SheetService/GetSheet", sheet, "get sheets", "bb.sheets.get", &view); err != nil {
		return "", err
	}
	content, err := base64.StdEncoding.DecodeString(view.Content)
	if err != nil {
		return "", errors.Wrapf(err, "failed to decode sheet %s", sheet)
	}
	return oneLine(string(content), maxStatementPreview), nil
}

func (s *Server) describeAccessGrant(ctx context.Context, name string) ([]string, error) {
	var grant accessGrantView
	if err := s.readResource(ctx, "/bytebase.v1.AccessGrantService/GetAccessGrant", name, "get access grants", "bb.accessGrants.get", &grant); err != nil {
		return nil, err
	}
	lines := []string{
		fmt.Sprintf("Access: run %s on %s", oneLine(grant.Query, maxStatementPreview), strings.Join(grant.Targets, ", ")),
		fmt.Sprintf("Unmasked: %s. Export: %s. Requested by %s", yesNo(grant.Unmask), yesNo(grant.Export), strings.TrimPrefix(grant.Creator, common.UserNamePrefix)),
	}
	switch {
	case grant.ExpireTime != "":
		lines = append(lines, "Expires: "+grant.ExpireTime)
	case grant.TTL != "":
		lines = append(lines, fmt.Sprintf("Expires: %s after approval", readableDuration(grant.TTL)))
	default:
	}
	return lines, nil
}

func describeRoleGrant(issue issueView) string {
	grant := issue.RoleGrant
	line := fmt.Sprintf("Grants %s to %s", grant.Role, strings.TrimPrefix(grant.User, common.UserNamePrefix))
	if grant.Condition != nil && grant.Condition.Expression != "" {
		line += " when " + oneLine(grant.Condition.Expression, maxStatementPreview)
	}
	if grant.Expiration != "" {
		line += ", for " + readableDuration(grant.Expiration)
	}
	return line
}

// readResource reads one resource by name from the internal API.
func (s *Server) readResource(ctx context.Context, path, name, operation, permission string, out any) error {
	resp, err := s.apiRequest(ctx, path, map[string]any{"name": name})
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", name)
	}
	if err := checkAPIResponse(resp, operation, permission); err != nil {
		return err
	}
	if err := json.Unmarshal(resp.Body, out); err != nil {
		return errors.Wrapf(err, "failed to parse %s", name)
	}
	return nil
}

func (s *Server) issueLink(ctx context.Context, issue string) string {
	externalURL, _ := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, getWorkspaceID(ctx))
	return buildResourceURL(externalURL, issue)
}

// describeApprovalStep names the step a decision would record. It mirrors
// utils.FindNextPendingRole: one approver per step, in flow order.
func describeApprovalStep(issue issueView) string {
	roles := issue.ApprovalTemplate.Flow.Roles
	if issue.ApprovalStatus == "PENDING" && len(issue.Approvers) < len(roles) {
		return fmt.Sprintf("step %d of %d, role %s", len(issue.Approvers)+1, len(roles), roles[len(issue.Approvers)])
	}
	if issue.ApprovalStatus == "" {
		return "not started"
	}
	return issue.ApprovalStatus
}

// oneLine folds whitespace, newlines included, to single spaces and cuts the
// result to limit runes.
func oneLine(text string, limit int) string {
	folded := strings.Join(strings.Fields(text), " ")
	if runes := []rune(folded); len(runes) > limit {
		return string(runes[:limit]) + "..."
	}
	return folded
}

// readableDuration renders a protojson duration such as "3600s" as "1h0m0s".
func readableDuration(protoDuration string) string {
	if d, err := time.ParseDuration(protoDuration); err == nil {
		return d.String()
	}
	return protoDuration
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func capitalize(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}
