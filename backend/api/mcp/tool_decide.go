package mcp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/audit"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

It decides database changes and access requests. Role requests stay in the Bytebase console, because approving one grants a project role.

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

// confirmationStatePurpose names this use of the server secret, so the key
// that signs a confirmation's request state signs nothing else.
const confirmationStatePurpose = "bb.mcp.decide_issue.request_state"

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
	project   string
	issue     string
	procedure string
	comment   string
}

func (d *issueDecision) verb() string {
	if d.procedure == v1connect.IssueServiceApproveIssueProcedure {
		return "approve"
	}
	return "reject"
}

func (d *issueDecision) pastTense() string {
	if d.procedure == v1connect.IssueServiceApproveIssueProcedure {
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

	// Read on both rounds, so a type this tool does not decide is refused before
	// any answer is acted on, and an answer is acted on only while the issue
	// reads as it did when the user was asked.
	var issue issueView
	if err := s.readResource(ctx, "/bytebase.v1.IssueService/GetIssue", decision.issue, "get issues", "bb.issues.get", &issue); err != nil {
		return formatToolError(err), nil, nil
	}
	if refusal := refuseIssueType(issue, decision, link); refusal != nil {
		s.recordRefusedDecision(ctx, decision, refusal.Message)
		return formatToolError(refusal), nil, nil
	}
	view, err := s.readConfirmationView(ctx, decision, issue)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	digest, err := view.digest()
	if err != nil {
		return formatToolError(err), nil, nil
	}

	// An answer counts only with the request state this server signed when it
	// asked. A client can send an answer on a call that was never asked, so
	// without that state the call is asked like a first one.
	if response, state, answered := confirmationAnswer(req); answered {
		if asked, signed := s.verifyConfirmationState(ctx, decision, state); signed {
			result, output := s.decideOnAnswer(ctx, decision, response, bytes.Equal(asked, digest), issue.UpdateTime, link)
			return result, output, nil
		}
	}

	message, err := s.describeIssueDecision(ctx, decision, view)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	// The same multi round-trip shape as elicitDatabaseChoice: the SDK elicits
	// within this tools/call and re-invokes the handler with the answer and the
	// request state. A client on the 2026-07-28 protocol answers and echoes the
	// state itself.
	return &mcp.CallToolResult{
		RequestState: s.signConfirmationState(ctx, decision, digest),
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
	project, _, err := common.GetProjectIDIssueUID(input.Issue)
	if err != nil {
		return nil, &toolError{
			Code:       "INVALID_ARGUMENT",
			Message:    fmt.Sprintf("invalid issue %q", input.Issue),
			Suggestion: "pass the issue name, e.g. projects/hr/issues/42",
		}
	}
	decision := &issueDecision{project: project, issue: input.Issue, comment: strings.TrimSpace(input.Comment)}
	switch strings.ToLower(strings.TrimSpace(input.Decision)) {
	case "approve":
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

// refuseIssueType serves the issue types this tool decides. It is an
// allow-list, so a type added later is refused until someone decides about it.
// A role request stays in the console: approving one grants a project role.
func refuseIssueType(issue issueView, decision *issueDecision, link string) *toolError {
	switch issue.Type {
	case "DATABASE_CHANGE", "ACCESS_GRANT":
		return nil
	case "ROLE_GRANT":
		return &toolError{
			Code: "ROLE_GRANT_NOT_SUPPORTED",
			Message: fmt.Sprintf("approving a role request grants a project role, which MCP does not do yet, so %s was not %s and nothing changed",
				decision.issue, decision.pastTense()),
			Suggestion: "decide it signed in to the Bytebase console: " + link,
		}
	default:
		return &toolError{
			Code: "ISSUE_TYPE_NOT_SUPPORTED",
			Message: fmt.Sprintf("decide_issue does not decide %q issues, so %s was not %s and nothing changed",
				issue.Type, decision.issue, decision.pastTense()),
			Suggestion: "decide it signed in to the Bytebase console: " + link,
		}
	}
}

// recordRefusedDecision writes the audit row for a decision this tool refuses
// itself. The refusal comes before any decision request, so no connect
// interceptor sees it.
func (s *Server) recordRefusedDecision(ctx context.Context, decision *issueDecision, reason string) {
	delegated, _ := getDelegatedIdentity(ctx)
	row := &storepb.AuditLog{
		Parent:          common.FormatProject(decision.project),
		Method:          decision.procedure,
		Resource:        decision.issue,
		User:            common.FormatUserEmail(getUserEmail(ctx)),
		Status:          &spb.Status{Code: int32(codes.PermissionDenied), Message: reason},
		RequestMetadata: &storepb.RequestMetadata{CallerIp: getCallerIP(ctx)},
		McpDelegation: &storepb.MCPDelegation{
			Scope:         delegated.Scope,
			Resource:      delegated.Resource,
			ClientId:      delegated.ClientID,
			CorrelationId: delegated.CorrelationID,
		},
	}
	audit.RecordOutOfBandAudit(ctx, s.store, s.profile.RuntimeEnableAuditLogStdout.Load(), getWorkspaceID(ctx), row)
}

// confirmationAnswer returns the answer and the request state this call
// carries, if it carries an answer.
func confirmationAnswer(req *mcp.CallToolRequest) (mcp.InputResponse, string, bool) {
	if req.Params == nil {
		return nil, "", false
	}
	response, answered := req.Params.InputResponses[confirmDecisionRequestID]
	return response, req.Params.RequestState, answered
}

// decideOnAnswer acts on the answer to a form this server sent. Only an
// accepted form with the box checked decides, and only while what the user
// was asked about still matches the issue. A decline is reported before a
// change, so a user who said no is not asked again.
func (s *Server) decideOnAnswer(ctx context.Context, decision *issueDecision, response mcp.InputResponse, unchanged bool, issueUpdateTime, link string) (*mcp.CallToolResult, any) {
	if confirmed, why := confirmationOutcome(response); !confirmed {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
			"Not %s: %s, so nothing changed. Do not retry unless the user asks.\nIssue: %s",
			decision.pastTense(), why, link)}}}, nil
	}
	if !unchanged {
		return formatToolError(&toolError{
			Code: "STALE_CONFIRMATION",
			Message: fmt.Sprintf("what the user confirmed no longer matches %s as it is now, so it was not %s and nothing changed",
				decision.issue, decision.pastTense()),
			Suggestion: "call decide_issue again to ask the user about the issue as it is now",
		}), nil
	}
	return s.applyIssueDecision(ctx, decision, issueUpdateTime, link)
}

// signConfirmationState returns the request state the form is sent with: a
// digest of what the user is asked about, signed together with who asks,
// through which client, and which decision on which issue.
func (s *Server) signConfirmationState(ctx context.Context, decision *issueDecision, digest []byte) string {
	return base64.RawURLEncoding.EncodeToString(digest) + "." +
		base64.RawURLEncoding.EncodeToString(s.confirmationMAC(ctx, decision, digest))
}

// verifyConfirmationState returns the digest a request state carries, and
// whether this server signed it for this caller and this decision.
func (s *Server) verifyConfirmationState(ctx context.Context, decision *issueDecision, state string) ([]byte, bool) {
	encodedDigest, encodedMAC, found := strings.Cut(state, ".")
	if !found {
		return nil, false
	}
	digest, err := base64.RawURLEncoding.DecodeString(encodedDigest)
	if err != nil {
		return nil, false
	}
	mac, err := base64.RawURLEncoding.DecodeString(encodedMAC)
	if err != nil {
		return nil, false
	}
	return digest, hmac.Equal(mac, s.confirmationMAC(ctx, decision, digest))
}

func (s *Server) confirmationMAC(ctx context.Context, decision *issueDecision, digest []byte) []byte {
	key := hmac.New(sha256.New, []byte(s.secret))
	_, _ = key.Write([]byte(confirmationStatePurpose))
	signed := strings.Join([]string{getWorkspaceID(ctx), getUserEmail(ctx), getOAuth2ClientID(ctx), decision.procedure, decision.issue, ""}, "\x00")
	mac := hmac.New(sha256.New, key.Sum(nil))
	_, _ = mac.Write(append([]byte(signed), digest...))
	return mac.Sum(nil)
}

// applyIssueDecision records the confirmed decision.
func (s *Server) applyIssueDecision(ctx context.Context, decision *issueDecision, issueUpdateTime, link string) (*mcp.CallToolResult, any) {
	confirmedAt, err := time.Parse(time.RFC3339Nano, issueUpdateTime)
	if err != nil {
		return formatToolError(errors.Wrapf(err, "failed to read the update time of %s, so it was not %s", decision.issue, decision.pastTense())), nil
	}
	comment := decisionNote
	if decision.comment != "" {
		comment += "\n\n" + decision.comment
	}
	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure:       decision.procedure,
		Issue:           decision.issue,
		IssueUpdateTime: confirmedAt,
	})
	resp, err := s.apiRequest(confirmedCtx, decision.procedure, map[string]any{
		"name":    decision.issue,
		"comment": comment,
	})
	if err != nil {
		return formatToolError(errors.Wrapf(err, "failed to %s %s", decision.verb(), decision.issue)), nil
	}
	if resp.Status >= 400 {
		return formatToolError(errors.Errorf("failed to %s %s: HTTP %d: %s; check the issue's current state: %s",
			decision.verb(), decision.issue, resp.Status, parseError(resp.Body), link)), nil
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
	UpdateTime string `json:"updateTime"`
}

type planView struct {
	UpdateTime string `json:"updateTime"`
	Specs      []struct {
		ChangeDatabaseConfig *struct {
			Targets []string `json:"targets"`
			Sheet   string   `json:"sheet"`
			Release string   `json:"release"`
		} `json:"changeDatabaseConfig"`
		CreateDatabaseConfig *createDatabaseView `json:"createDatabaseConfig"`
	} `json:"specs"`
}

type createDatabaseView struct {
	Target       string `json:"target"`
	Database     string `json:"database"`
	Cluster      string `json:"cluster"`
	Owner        string `json:"owner"`
	Table        string `json:"table"`
	CharacterSet string `json:"characterSet"`
	Collation    string `json:"collation"`
}

type sheetView struct {
	Content     string `json:"content"`
	ContentSize int64  `json:"contentSize,string"`
}

type accessGrantView struct {
	Targets    []string `json:"targets"`
	Schema     string   `json:"schema"`
	Container  string   `json:"container"`
	Query      string   `json:"query"`
	Unmask     bool     `json:"unmask"`
	Export     bool     `json:"export"`
	ExpireTime string   `json:"expireTime"`
	TTL        string   `json:"ttl"`
	UpdateTime string   `json:"updateTime"`
}

// confirmationView is what the user is asked to confirm: the issue, the plan or
// access grant behind it, and the comment the decision records. The request
// state carries a digest of all of it. A change to the issue's approval, the
// plan's specs or the grant also changes that resource's update time, so the
// digest changes even when the changed field is not shown. Statements are left
// out: a sheet's name is its content hash (GetProjectResourceIDSheetSha256), so
// the plan's specs already pin every statement, shown or not.
type confirmationView struct {
	Issue   issueView        `json:"issue"`
	Plan    *planView        `json:"plan,omitempty"`
	Grant   *accessGrantView `json:"grant,omitempty"`
	Comment string           `json:"comment,omitempty"`
}

func (s *Server) readConfirmationView(ctx context.Context, decision *issueDecision, issue issueView) (*confirmationView, error) {
	view := &confirmationView{Issue: issue, Comment: decision.comment}
	switch {
	case issue.AccessGrant != "":
		grant, err := s.readAccessGrant(ctx, decision, issue)
		if err != nil {
			return nil, err
		}
		view.Grant = grant
	case issue.Plan != "":
		view.Plan = &planView{}
		if err := s.readResource(ctx, "/bytebase.v1.PlanService/GetPlan", issue.Plan, "get plans", "bb.plans.get", view.Plan); err != nil {
			return nil, err
		}
	default:
	}
	return view, nil
}

// readAccessGrant reads the grant behind an access request. A requester may
// lack bb.accessGrants.get, as project developers do, so the caller's own
// request is read through SearchMyAccessGrants, as the console's issue page
// reads it. The grant and its issue share a creator (CreateAccessGrant).
func (s *Server) readAccessGrant(ctx context.Context, decision *issueDecision, issue issueView) (*accessGrantView, error) {
	if issue.Creator != common.FormatUserEmail(getUserEmail(ctx)) {
		grant := &accessGrantView{}
		if err := s.readResource(ctx, "/bytebase.v1.AccessGrantService/GetAccessGrant", issue.AccessGrant, "get access grants", "bb.accessGrants.get", grant); err != nil {
			return nil, err
		}
		return grant, nil
	}
	resp, err := s.apiRequest(ctx, "/bytebase.v1.AccessGrantService/SearchMyAccessGrants", map[string]any{
		"parent": common.FormatProject(decision.project),
		"filter": fmt.Sprintf("name == %q", issue.AccessGrant),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", issue.AccessGrant)
	}
	if err := checkAPIResponse(resp, "search your access grants", "bb.accessGrants.get"); err != nil {
		return nil, err
	}
	var mine struct {
		AccessGrants []accessGrantView `json:"accessGrants"`
	}
	if err := json.Unmarshal(resp.Body, &mine); err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", issue.AccessGrant)
	}
	if len(mine.AccessGrants) != 1 {
		return nil, errors.Errorf("%s is not among the access grants you requested", issue.AccessGrant)
	}
	return &mine.AccessGrants[0], nil
}

func (v *confirmationView) digest() ([]byte, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to digest the confirmation")
	}
	sum := sha256.Sum256(payload)
	return sum[:], nil
}

// describeIssueDecision builds the confirmation message: the decision, the
// issue, what approving it would let happen, the step it decides, and the
// comment it records. Every value taken from the issue or from the agent enters
// through %q, which escapes line breaks, so no value can pose as a line of the
// message. The message carries no link: the MCP specification asks servers not
// to put clickable URLs in a form.
//
// Claude Code 2.1.302 shows four lines of a message, or three and "… (+N more
// lines)", and cuts each line six columns short of the terminal width. So each
// line does one job, its bounded part first: the decision; the targets, counted
// before they are named; the statement; then the approval step, requester and
// comment. A plan with several changes shows its first change and the total.
//
// It fails closed: if any part cannot be read, nothing is asked.
func (s *Server) describeIssueDecision(ctx context.Context, decision *issueDecision, view *confirmationView) (string, error) {
	issue := view.Issue
	question := fmt.Sprintf("%s issue %q titled %q", capitalize(decision.verb()), issue.Name, issue.Title)
	if email := getUserEmail(ctx); email != "" {
		question += " as " + email
	}
	var details []string
	last := fmt.Sprintf("Approval: %s. Requested by %q.",
		describeApprovalStep(issue), strings.TrimPrefix(issue.Creator, common.UserNamePrefix))
	switch {
	case view.Grant != nil:
		var terms string
		details, terms = describeAccessGrant(view.Grant)
		last = terms + " " + last
	case view.Plan != nil:
		var err error
		if details, err = s.describePlan(ctx, view.Plan); err != nil {
			return "", err
		}
	default:
	}
	if view.Comment != "" {
		last += fmt.Sprintf(" Comment: %q", view.Comment)
	}
	lines := append([]string{question + "?"}, details...)
	return strings.Join(append(lines, last), "\n"), nil
}

func (s *Server) describePlan(ctx context.Context, plan *planView) ([]string, error) {
	var lines []string
	for i, spec := range plan.Specs {
		if i == maxDescribedSpecs {
			lines = append(lines, fmt.Sprintf("(%d of %d changes shown)", maxDescribedSpecs, len(plan.Specs)))
			break
		}
		change, create := "Change", "Create database"
		if len(plan.Specs) > 1 {
			change = fmt.Sprintf("Change %d of %d", i+1, len(plan.Specs))
			create = change + ": create database"
		}
		switch {
		case spec.ChangeDatabaseConfig != nil:
			config := spec.ChangeDatabaseConfig
			what := fmt.Sprintf("Release %q (statements not shown)", config.Release)
			if config.Sheet != "" {
				statement, err := s.statementPreview(ctx, config.Sheet)
				if err != nil {
					return nil, err
				}
				what = statement
			}
			lines = append(lines, fmt.Sprintf("%s on %s", change, describeTargets(config.Targets, "targets")), what)
		case spec.CreateDatabaseConfig != nil:
			lines = append(lines, describeNewDatabase(create, spec.CreateDatabaseConfig)...)
		default:
		}
	}
	return lines, nil
}

func describeNewDatabase(create string, config *createDatabaseView) []string {
	lines := []string{fmt.Sprintf("%s %q on %q", create, config.Database, config.Target)}
	var settings []string
	for _, setting := range []struct{ name, value string }{
		{"cluster", config.Cluster},
		{"owner", config.Owner},
		{"table", config.Table},
		{"character set", config.CharacterSet},
		{"collation", config.Collation},
	} {
		if setting.value != "" {
			settings = append(settings, fmt.Sprintf("%s %q", setting.name, setting.value))
		}
	}
	if len(settings) > 0 {
		lines = append(lines, capitalize(strings.Join(settings, ", ")))
	}
	return lines
}

// statementPreview quotes the start of a change's statement. When it leaves the
// rest out, "(partial)" comes first and the statement's full size last. GetSheet
// cuts content it returns at common.MaxSheetSize, so the size comes from
// contentSize.
func (s *Server) statementPreview(ctx context.Context, sheet string) (string, error) {
	var view sheetView
	if err := s.readResource(ctx, "/bytebase.v1.SheetService/GetSheet", sheet, "get sheets", "bb.sheets.get", &view); err != nil {
		return "", err
	}
	content, err := base64.StdEncoding.DecodeString(view.Content)
	if err != nil {
		return "", errors.Wrapf(err, "failed to decode sheet %s", sheet)
	}
	shown, cut := common.TruncateString(string(content), maxStatementPreview)
	if !cut {
		return fmt.Sprintf("SQL: %q", shown), nil
	}
	return fmt.Sprintf("SQL (partial): %q (first %d characters of %d bytes)", shown, maxStatementPreview, view.ContentSize), nil
}

// describeAccessGrant returns the grant's scope and query lines, and its terms
// for the approval line. The schema and container go on the scope line: the
// requester can set them to any length, and nothing may push the query out.
func describeAccessGrant(grant *accessGrantView) ([]string, string) {
	var qualifiers []string
	if grant.Schema != "" {
		qualifiers = append(qualifiers, fmt.Sprintf("schema %q", grant.Schema))
	}
	if grant.Container != "" {
		qualifiers = append(qualifiers, fmt.Sprintf("container %q", grant.Container))
	}
	scope := "Access on " + describeTargets(grant.Targets, "databases", qualifiers...)
	query := fmt.Sprintf("Query: %q", grant.Query)
	if shown, cut := common.TruncateString(grant.Query, maxStatementPreview); cut {
		query = fmt.Sprintf("Query (partial): %q (first %d of %d characters)", shown, maxStatementPreview, utf8.RuneCountInString(grant.Query))
	}
	terms := fmt.Sprintf("Unmasked: %s. Export: %s.", yesNo(grant.Unmask), yesNo(grant.Export))
	switch {
	case grant.ExpireTime != "":
		terms += fmt.Sprintf(" Expires: %q.", grant.ExpireTime)
	case grant.TTL != "":
		terms += fmt.Sprintf(" Expires: %q after approval.", readableDuration(grant.TTL))
	default:
	}
	return []string{scope, query}, terms
}

// describeTargets names targets and their qualifiers. One target's name comes
// first, so a long name cuts its qualifiers, not itself. With several targets,
// the count and the qualifiers come before the names, so a line cut short keeps
// them.
func describeTargets(targets []string, noun string, qualifiers ...string) string {
	if len(targets) == 1 {
		return strings.Join(append([]string{strconv.Quote(targets[0])}, qualifiers...), ", ")
	}
	head := append([]string{fmt.Sprintf("%d %s", len(targets), noun)}, qualifiers...)
	return strings.Join(head, ", ") + ": " + quoteAll(targets)
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
		return fmt.Sprintf("step %d of %d, role %q", len(issue.Approvers)+1, len(roles), roles[len(issue.Approvers)])
	}
	if issue.ApprovalStatus == "" {
		return "not started"
	}
	return strconv.Quote(issue.ApprovalStatus)
}

func quoteAll(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return strings.Join(quoted, ", ")
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
