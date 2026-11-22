package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// GrantMaskingExemptionInput is the input for the grant_masking_exemption tool.
type GrantMaskingExemptionInput struct {
	Project      string   `json:"project"`
	Member       string   `json:"member"`
	Database     string   `json:"database,omitempty"`
	AllDatabases bool     `json:"allDatabases,omitempty"`
	Schema       string   `json:"schema,omitempty"`
	Table        string   `json:"table,omitempty"`
	Columns      []string `json:"columns,omitempty"`
	ExpireTime   string   `json:"expireTime,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

// RevokeMaskingExemptionInput is the input for the revoke_masking_exemption tool.
type RevokeMaskingExemptionInput struct {
	Project   string `json:"project"`
	Member    string `json:"member"`
	Condition string `json:"condition"`
}

// MaskingExemptionOutput is the output for both masking exemption tools.
type MaskingExemptionOutput struct {
	Project   string `json:"project"`
	Member    string `json:"member"`
	Condition string `json:"condition"`
	Link      string `json:"link"`
}

const grantMaskingExemptionDescription = `Exempt one member from data masking in a Bytebase project, for one database (optionally one schema, table, or columns of one table) or for every database in the project. The user confirms the exemption in this client first.

| Parameter    | Required | Description |
|--------------|----------|-------------|
| project      | Yes      | Project name, e.g. "projects/hr" |
| member       | Yes      | "user:{email}", "group:{email}", "serviceAccount:{email}" or "workloadIdentity:{email}" |
| database     | One of   | Database name, e.g. "instances/prod/databases/hr"; it must belong to the project |
| allDatabases | One of   | true for every database in the project, current and future |
| schema       | No       | Narrow to one schema (needs database) |
| table        | No       | Narrow to one table (needs database) |
| columns      | No       | Narrow to these columns of the table, e.g. ["first_name", "last_name"] (needs table) |
| expireTime   | No       | RFC 3339 time the exemption ends, e.g. "2026-10-27T00:00:00Z"; without it, it does not expire |
| reason       | No       | Why, recorded with the exemption |

An exemption is a standing policy change: the member bypasses the masking that applies to the covered data in query and export results, including masking set up later. It does not set up masking itself; administer_workspace changes the masking rules. For a one-off query, request an access grant instead with request_access_grant.

The tool adds one exemption and leaves every other entry as it is. For several columns of one table, pass them together in columns, so the user confirms once. It shows the user the member, the scope, the expiry and the effect, and asks them to confirm in a form; it writes only when they accept. A client that cannot show the form gets the project's masking exemption page, to grant it in the Bytebase console instead. Do not ask in chat first: the form is the question.

The same permissions apply as in the console. This and revoke_masking_exemption are the only ways to change exemptions over MCP; call_api refuses OrgPolicyService/UpdatePolicy.`

const revokeMaskingExemptionDescription = `Remove one member's masking exemption in a Bytebase project. The user confirms in this client first.

| Parameter | Required | Description |
|-----------|----------|-------------|
| project   | Yes      | Project name, e.g. "projects/hr" |
| member    | Yes      | The member as the policy lists it, e.g. "user:bob@example.com" |
| condition | Yes      | The exemption's condition expression, exactly as OrgPolicyService/GetPolicy returns it for "{project}/policies/masking_exemption"; "" for an exemption with no condition |

The member is removed from every exemption whose condition matches exactly; an exemption left with no member is removed. Other members and other exemptions stay as they are. The tool asks the user to confirm in a form, and writes only when they accept. A client that cannot show the form gets the project's masking exemption page.

The same permissions apply as in the console.`

func (s *Server) registerMaskingExemptionTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "grant_masking_exemption",
		Description: grantMaskingExemptionDescription,
		Meta:        requiresUserInteraction(),
	}, s.handleGrantMaskingExemption)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "revoke_masking_exemption",
		Description: revokeMaskingExemptionDescription,
	}, s.handleRevokeMaskingExemption)
}

// exemptionWrite is one confirmed change to a project's masking exemption
// policy.
type exemptionWrite struct {
	policy string
}

func (*exemptionWrite) confirmedProcedure() string {
	return v1connect.OrgPolicyServiceUpdatePolicyProcedure
}

func (w *exemptionWrite) confirmedResource() string { return w.policy }

func maskingExemptionPolicyName(project string) string {
	return project + "/policies/masking_exemption"
}

// exemptionGrant is a validated grant_masking_exemption request.
type exemptionGrant struct {
	project, member                    string
	database, instanceID, databaseName string
	schema, table                      string
	columns                            []string
	expireTime, note                   string
}

func parseExemptionGrant(input GrantMaskingExemptionInput) (*exemptionGrant, error) {
	if err := checkProjectName(input.Project); err != nil {
		return nil, err
	}
	if !isNamedMember(input.Member) {
		return nil, invalidArgument(fmt.Sprintf("invalid member %q: an exemption names one user, group, service account or workload identity", input.Member),
			`pass the member, e.g. "user:bob@example.com"; allUsers, which the policy API also accepts, would unmask the scope for everyone`)
	}
	g, err := parseExemptionScope(input)
	if err != nil {
		return nil, err
	}
	g.member, g.note = input.Member, input.Reason
	return g, nil
}

// parseExemptionScope checks what an exemption covers and until when, the
// parts its condition holds. A revoke rebuilds a grant from a stored condition
// through it, so it reads no member: the policy API stores members, such as
// allUsers, that a grant refuses.
func parseExemptionScope(input GrantMaskingExemptionInput) (*exemptionGrant, error) {
	g := &exemptionGrant{project: input.Project, schema: input.Schema, table: input.Table, columns: input.Columns}
	switch {
	case input.Database != "" && input.AllDatabases:
		return nil, invalidArgument("pass database or allDatabases, not both", "")
	case input.Database == "" && !input.AllDatabases:
		return nil, invalidArgument("the exemption needs a scope: database, or allDatabases set to true",
			"pass the database, or allDatabases: true for every database in the project, current and future")
	case input.Database != "":
		// A database on a project-owned instance is named under its project.
		_, instanceID, databaseName, err := common.GetDatabaseResourceName(input.Database)
		if err != nil {
			return nil, invalidArgument(fmt.Sprintf("invalid database %q", input.Database), "pass the database name, e.g. instances/prod/databases/hr")
		}
		g.database, g.instanceID, g.databaseName = input.Database, instanceID, databaseName
	default:
		if input.Schema != "" || input.Table != "" || len(input.Columns) > 0 {
			return nil, invalidArgument("schema, table and columns narrow one database, so they need database", "")
		}
	}
	if len(input.Columns) > 0 && input.Table == "" {
		return nil, invalidArgument("columns need table", "pass the table the columns belong to")
	}
	if input.ExpireTime != "" {
		expireTime, err := time.Parse(time.RFC3339, input.ExpireTime)
		if err != nil {
			return nil, invalidArgument(fmt.Sprintf("invalid expireTime %q", input.ExpireTime), `pass an RFC 3339 time, e.g. "2026-10-27T00:00:00Z"`)
		}
		// The console writes the expiry with JavaScript's toISOString.
		g.expireTime = expireTime.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	return g, nil
}

func invalidArgument(message, suggestion string) error {
	return &toolError{Code: "INVALID_ARGUMENT", Message: message, Suggestion: suggestion}
}

// isNamedMember reports whether member names one principal, as the tool's
// contract and the console's exemption picker do.
func isNamedMember(member string) bool {
	for _, prefix := range []string{common.UserBindingPrefix, common.GroupBindingPrefix, common.ServiceAccountBindingPrefix, common.WorkloadIdentityBindingPrefix} {
		if name, found := strings.CutPrefix(member, prefix); found && strings.TrimSpace(name) != "" {
			return true
		}
	}
	return false
}

func checkProjectName(project string) error {
	if _, err := common.GetProjectID(project); err != nil {
		return &toolError{Code: "INVALID_ARGUMENT", Message: fmt.Sprintf("invalid project %q", project), Suggestion: "pass the project name, e.g. projects/hr"}
	}
	return nil
}

// condition builds the exemption's CEL expression in the form the console
// writes, so a condition the tool writes matches one the console writes
// character for character when a revoke compares them.
func (g *exemptionGrant) condition() string {
	var parts []string
	if g.expireTime != "" {
		parts = append(parts, fmt.Sprintf("request.time < timestamp(%s)", strconv.Quote(g.expireTime)))
	}
	if g.database != "" {
		scope := []string{
			"resource.instance_id == " + strconv.Quote(g.instanceID),
			"resource.database_name == " + strconv.Quote(g.databaseName),
		}
		for _, narrowing := range []struct{ attribute, value string }{
			{"resource.schema_name", g.schema},
			{"resource.table_name", g.table},
		} {
			if narrowing.value != "" {
				scope = append(scope, narrowing.attribute+" == "+strconv.Quote(narrowing.value))
			}
		}
		if len(g.columns) == 1 {
			scope = append(scope, "resource.column_name == "+strconv.Quote(g.columns[0]))
		} else if len(g.columns) > 1 {
			scope = append(scope, "resource.column_name in ["+quoteAll(g.columns)+"]")
		}
		parts = append(parts, "("+strings.Join(scope, " && ")+")")
	}
	return strings.Join(parts, " && ")
}

// readMaskingExemptions returns the project's exemption entries as the API
// returns them, so an entry the tool does not change is written back with every
// field it read. A project with no exemption policy has none.
func (s *Server) readMaskingExemptions(ctx context.Context, policy string) ([]map[string]any, error) {
	resp, err := s.apiRequest(ctx, "/bytebase.v1.OrgPolicyService/GetPolicy", map[string]any{"name": policy})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", policy)
	}
	if resp.Status == http.StatusNotFound {
		return nil, nil
	}
	if err := checkAPIResponse(resp, "get policies", "bb.policies.getMaskingExemptionPolicy"); err != nil {
		return nil, err
	}
	var view struct {
		MaskingExemptionPolicy struct {
			Exemptions []map[string]any `json:"exemptions"`
		} `json:"maskingExemptionPolicy"`
	}
	if err := json.Unmarshal(resp.Body, &view); err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", policy)
	}
	return view.MaskingExemptionPolicy.Exemptions, nil
}

// exemptionUpdateBody is the UpdatePolicy request the user is asked to
// confirm. allowMissing creates the policy for a project's first exemption, as
// the console does.
func exemptionUpdateBody(policy string, exemptions []map[string]any) map[string]any {
	return map[string]any{
		"policy": map[string]any{
			"name":                   policy,
			"type":                   "MASKING_EXEMPTION",
			"maskingExemptionPolicy": map[string]any{"exemptions": exemptions},
		},
		"updateMask":   "maskingExemptionPolicy",
		"allowMissing": true,
	}
}

func (s *Server) handleGrantMaskingExemption(ctx context.Context, req *mcp.CallToolRequest, input GrantMaskingExemptionInput) (*mcp.CallToolResult, any, error) {
	grant, err := parseExemptionGrant(input)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	link := s.maskingExemptionLink(ctx, grant.project)
	if grant.database == "" {
		// An exemption written under a project ID that does not exist would
		// apply to a project created later with that ID.
		var project struct{}
		if err := s.readResource(ctx, "/bytebase.v1.ProjectService/GetProject", grant.project, "get projects", "bb.projects.get", &project); err != nil {
			return formatToolError(err), nil, nil
		}
	} else {
		var database struct {
			Project string `json:"project"`
		}
		if err := s.readResource(ctx, "/bytebase.v1.DatabaseService/GetDatabase", grant.database, "get databases", "bb.databases.get", &database); err != nil {
			return formatToolError(err), nil, nil
		}
		if database.Project != grant.project {
			return formatToolError(&toolError{
				Code:       "INVALID_ARGUMENT",
				Message:    fmt.Sprintf("%s belongs to %s, not %s, so an exemption in %s would not cover it", grant.database, database.Project, grant.project, grant.project),
				Suggestion: "grant it in the project the database belongs to",
			}), nil, nil
		}
	}
	if !canShowConfirmation(req) {
		return formatToolError(&toolError{
			Code: "CONFIRMATION_UNAVAILABLE",
			Message: fmt.Sprintf("this MCP client cannot show the confirmation form grant_masking_exemption needs, so %s was not exempted and nothing changed",
				grant.member),
			Suggestion: "grant it signed in to the Bytebase console: " + link,
		}), nil, nil
	}

	write := &exemptionWrite{policy: maskingExemptionPolicyName(grant.project)}
	exemptions, err := s.readMaskingExemptions(ctx, write.policy)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	condition := map[string]any{"expression": grant.condition()}
	if grant.note != "" {
		condition["description"] = grant.note
	}
	written := append(slices.Clone(exemptions), map[string]any{"members": []string{grant.member}, "condition": condition})
	body := exemptionUpdateBody(write.policy, written)
	output := &MaskingExemptionOutput{Project: grant.project, Member: grant.member, Condition: grant.condition(), Link: link}
	message, tooLong := describeExemptionGrant(ctx, grant)
	if tooLong != "" {
		next := "grant it signed in to the Bytebase console: " + link
		if eachColumnFits(ctx, grant) {
			next = "exempt fewer columns at a time, or " + next
		}
		return formatToolError(&toolError{Code: "CHANGE_TOO_LONG", Message: tooLong, Suggestion: next}), nil, nil
	}
	choice := confirmChoice{question: "Exempt this member from masking?", action: "Exempt"}
	if grant.note != "" {
		choice.description = fmt.Sprintf("Reason: %q", grant.note)
	}
	return s.confirmExemptionWrite(ctx, req, write, exemptions, body, message, choice, output,
		fmt.Sprintf("the exemption for %s was not granted", grant.member),
		fmt.Sprintf("Exempted %s from masking in %s.", grant.member, grant.project),
		"such as the member, the scope, the expiry or the reason, then call grant_masking_exemption again",
	)
}

func (s *Server) handleRevokeMaskingExemption(ctx context.Context, req *mcp.CallToolRequest, input RevokeMaskingExemptionInput) (*mcp.CallToolResult, any, error) {
	if err := checkProjectName(input.Project); err != nil {
		return formatToolError(err), nil, nil
	}
	link := s.maskingExemptionLink(ctx, input.Project)
	write := &exemptionWrite{policy: maskingExemptionPolicyName(input.Project)}
	exemptions, err := s.readMaskingExemptions(ctx, write.policy)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	remaining, matched, held := removeExemptionMember(exemptions, input.Member, input.Condition)
	if matched == 0 {
		suggestion := fmt.Sprintf("%s holds no exemption in %s", input.Member, input.Project)
		if len(held) > 0 {
			suggestion = fmt.Sprintf("%s holds exemptions with these conditions: %s", input.Member, quoteAll(held))
		}
		return formatToolError(&toolError{
			Code:       "NOT_FOUND",
			Message:    fmt.Sprintf("%s holds no exemption in %s with the condition %q, so nothing changed", input.Member, input.Project, input.Condition),
			Suggestion: suggestion,
		}), nil, nil
	}
	if !canShowConfirmation(req) {
		return formatToolError(&toolError{
			Code: "CONFIRMATION_UNAVAILABLE",
			Message: fmt.Sprintf("this MCP client cannot show the confirmation form revoke_masking_exemption needs, so %s's exemption was not removed and nothing changed",
				input.Member),
			Suggestion: "remove it signed in to the Bytebase console: " + link,
		}), nil, nil
	}
	body := exemptionUpdateBody(write.policy, remaining)
	output := &MaskingExemptionOutput{Project: input.Project, Member: input.Member, Condition: input.Condition, Link: link}
	message, expiry, tooLong := describeExemptionRevoke(ctx, input)
	if tooLong != "" {
		return formatToolError(&toolError{Code: "CHANGE_TOO_LONG", Message: tooLong, Suggestion: "remove it signed in to the Bytebase console: " + link}), nil, nil
	}
	var notes []string
	if matched > 1 {
		notes = append(notes, fmt.Sprintf("Removes the member from all %d exemptions with this condition.", matched))
	}
	if expiry != "" {
		notes = append(notes, expiry)
	}
	choice := confirmChoice{question: "Revoke this masking exemption?", action: "Revoke", description: strings.Join(notes, "\n")}
	return s.confirmExemptionWrite(ctx, req, write, exemptions, body, message, choice, output,
		fmt.Sprintf("the exemption for %s was not removed", input.Member),
		fmt.Sprintf("Removed %s's masking exemption in %s.", input.Member, input.Project),
		"such as the member or the exemption's condition, then call revoke_masking_exemption again",
	)
}

// removeExemptionMember removes member from every entry whose condition is
// exactly condition, and drops an entry left with no member. The console can
// add the same member and condition twice, and removing only one would leave
// the access in place. It returns the entries to write, how many matched, and
// the conditions member holds, for a refusal.
func removeExemptionMember(exemptions []map[string]any, member, condition string) ([]map[string]any, int, []string) {
	var remaining []map[string]any
	var held []string
	matched := 0
	for _, entry := range exemptions {
		members := stringList(entry["members"])
		expression := ""
		if c, ok := entry["condition"].(map[string]any); ok {
			if e, ok := c["expression"].(string); ok {
				expression = e
			}
		}
		if !slices.Contains(members, member) {
			remaining = append(remaining, entry)
			continue
		}
		held = append(held, expression)
		if expression != condition {
			remaining = append(remaining, entry)
			continue
		}
		matched++
		members = slices.DeleteFunc(members, func(m string) bool { return m == member })
		if len(members) == 0 {
			continue
		}
		kept := maps.Clone(entry)
		kept["members"] = members
		remaining = append(remaining, kept)
	}
	return remaining, matched, held
}

func stringList(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// confirmExemptionWrite asks the user to confirm body, or acts on their answer.
// The digest covers the list as read and the exact request built from it, so an
// answer is acted on only while the arguments and every entry, including one a
// revoke would drop, read as they did when the user was asked.
func (s *Server) confirmExemptionWrite(ctx context.Context, req *mcp.CallToolRequest, write *exemptionWrite, read []map[string]any, body map[string]any, message string, choice confirmChoice, output *MaskingExemptionOutput, notDone, done, toChange string) (*mcp.CallToolResult, any, error) {
	encoded, err := json.Marshal(map[string]any{"read": read, "write": body})
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	digest := sha256.Sum256(encoded)
	reply, answered := s.signedReply(ctx, req, write, digest[:])
	if !answered {
		return s.askToConfirm(ctx, write, digest[:], message, choice), nil, nil
	}
	switch choice, why := confirmationOutcome(reply.response); choice {
	case confirmChoiceApply:
	case confirmChoiceChange:
		return textResult(fmt.Sprintf("%s: %s. Ask them what to change, %s.\nMasking exemptions: %s", capitalize(notDone), why, toChange, output.Link)), nil, nil
	default:
		return textResult(fmt.Sprintf("%s: %s, so nothing changed. Do not retry unless the user asks.\nMasking exemptions: %s", capitalize(notDone), why, output.Link)), nil, nil
	}
	if why := reply.staleReason("what the user confirmed no longer matches this call and the project's exemptions as they are now"); why != "" {
		return formatToolError(&toolError{
			Code:       "STALE_CONFIRMATION",
			Message:    fmt.Sprintf("%s, so %s and nothing changed", why, notDone),
			Suggestion: "call the tool again to ask the user about the exemptions as they are now",
		}), nil, nil
	}
	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: write.confirmedProcedure(),
		Resource:  write.policy,
	})
	resp, err := s.apiRequest(confirmedCtx, write.confirmedProcedure(), body)
	if err != nil {
		return formatToolError(errors.Wrapf(err, "failed to update %s", write.policy)), nil, nil
	}
	if resp.Status >= 400 {
		return formatToolError(errors.Errorf("failed to update %s: HTTP %d: %s", write.policy, resp.Status, parseError(resp.Body))), nil, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: done + "\nMasking exemptions: " + output.Link}}}, output, nil
}

// describeExemptionGrant builds the grant's confirmation message. tooLong says
// why it does not fit a form, or is empty when it does.
func describeExemptionGrant(ctx context.Context, g *exemptionGrant) (message, tooLong string) {
	effect := "Bypasses masking here, with no expiry; adds no masking."
	if g.expireTime != "" {
		effect = fmt.Sprintf("Bypasses masking until %q; adds no masking.", g.expireTime)
	}
	return fitExemption(ctx, fmt.Sprintf("Exempt %q", g.member), " from masking", exemptionScope(g), effect)
}

// eachColumnFits reports whether every column of a grant naming several fits a
// form alone, so granting fewer columns at a time can help.
func eachColumnFits(ctx context.Context, g *exemptionGrant) bool {
	if len(g.columns) < 2 {
		return false
	}
	for _, column := range g.columns {
		one := *g
		one.columns = []string{column}
		if _, tooLong := describeExemptionGrant(ctx, &one); tooLong != "" {
			return false
		}
	}
	return true
}

// exemptionScope names what the grant covers.
func exemptionScope(g *exemptionGrant) string {
	if g.database == "" {
		projectID, _ := common.GetProjectID(g.project)
		return fmt.Sprintf("Project %q: all databases, current and future", projectID)
	}
	return fmt.Sprintf("Database %q on instance %q, %s", g.databaseName, g.instanceID, describeNarrowing(g))
}

// fitExemption puts an exemption form's question, its change and the change's
// effect into the lines a form shows. The question starts with head, which
// names the member and must fit one line: only the text after it, which ends
// with the account's email, may run past the line. tooLong says why the form
// does not fit, or is empty when it does. Values enter through %q or
// conditionText, so none can pose as a line of the message.
func fitExemption(ctx context.Context, head, rest, change, effect string) (message, tooLong string) {
	question := head + rest
	if email := getUserEmail(ctx); email != "" {
		question += ", as " + email
	}
	lines, _ := fitMessage(question+"?", []formChange{{text: change, effect: true}}, effect)
	message = strings.Join(lines, "\n")
	switch {
	case lineColumns(head) > messageLineWidth:
		return message, fmt.Sprintf("the member does not fit the %d columns one line of the confirmation form shows, so nothing changed", messageLineWidth)
	case len(lines) > messageLines:
		return message, fmt.Sprintf("this exemption needs %d lines, more than the %d one confirmation form shows, so nothing changed", len(lines), messageLines)
	default:
		return message, ""
	}
}

// describeNarrowing names what the grant covers within the database. A table
// named without a schema matches that table in every schema, so the text says
// "any schema".
func describeNarrowing(g *exemptionGrant) string {
	switch {
	case g.schema == "" && len(g.columns) > 0:
		return "any schema, " + describeColumns(g.columns, strconv.Quote(g.table))
	case g.schema == "" && g.table != "":
		return fmt.Sprintf("any schema, table %q", g.table)
	case len(g.columns) > 0:
		return describeColumns(g.columns, fmt.Sprintf("%q.%q", g.schema, g.table))
	case g.table != "":
		return fmt.Sprintf("table %q.%q", g.schema, g.table)
	case g.schema != "":
		return fmt.Sprintf("schema %q", g.schema)
	default:
		return "all tables"
	}
}

func describeColumns(columns []string, table string) string {
	if len(columns) == 1 {
		return fmt.Sprintf("column %q in %s", columns[0], table)
	}
	return fmt.Sprintf("columns %s in %s", quoteAll(columns), table)
}

// describeExemptionRevoke builds the revoke's confirmation message. The
// exemption's expiry ends the scope's line when the message has room, and
// otherwise moves under the choice, returned as expiry. tooLong says why the
// message does not fit a form, or is empty when it does.
func describeExemptionRevoke(ctx context.Context, input RevokeMaskingExemptionInput) (message, expiry, tooLong string) {
	const effect = "Ends their bypass of masking here, unless another exemption covers it."
	head, rest := fmt.Sprintf("Revoke %q", input.Member), "'s masking exemption"
	scope, until := revokeScope(input)
	if until != "" {
		if message, tooLong := fitExemption(ctx, head, rest, fmt.Sprintf("%s, until %q", scope, until), effect); tooLong == "" {
			return message, "", ""
		}
		expiry = fmt.Sprintf("It would expire %q.", until)
	}
	message, tooLong = fitExemption(ctx, head, rest, scope, effect)
	return message, expiry, tooLong
}

// revokeScope names the exemption the revoke ends, and its expiry apart. A
// condition in the form a grant writes reads in the grant's words. Any other
// shows through conditionText, with its expiry apart when splitExemptionExpiry
// can split one off.
func revokeScope(input RevokeMaskingExemptionInput) (scope, expiry string) {
	// An all-databases grant with no expiry writes no condition, which CEL
	// cannot parse.
	if input.Condition == "" {
		return exemptionScope(&exemptionGrant{project: input.Project}), ""
	}
	projectID, _ := common.GetProjectID(input.Project)
	parsed, ok := parseExemptionCondition(input.Condition)
	if !ok {
		return fmt.Sprintf("Project %q: %s", projectID, conditionText(input.Condition)), ""
	}
	if g, ok := exemptionGrantOf(input, parsed); ok {
		return exemptionScope(g), g.expireTime
	}
	condition, expiry := splitExemptionExpiry(input.Condition, parsed)
	if condition == "" {
		return fmt.Sprintf("Project %q: all databases", projectID), expiry
	}
	return fmt.Sprintf("Project %q: %s", projectID, conditionText(condition)), expiry
}

// parseExemptionCondition parses an exemption's condition, or reports false for
// one that is not CEL.
func parseExemptionCondition(condition string) (*celast.AST, bool) {
	env, err := cel.NewEnv(common.MaskingExemptionPolicyCELAttributes...)
	if err != nil {
		return nil, false
	}
	parsed, issues := env.Parse(condition)
	if issues != nil && issues.Err() != nil {
		return nil, false
	}
	return parsed.NativeRep(), true
}

// splitExemptionExpiry splits a condition whose top level is an expiry and'ed
// with a scope, as the console writes it, into the scope and the expiry, so the
// form names the expiry apart from the scope. It reads the parsed condition, so
// CEL's precedence and literal forms decide: the expiry of "T && A || B" does
// not cover B, and that condition is returned whole, with no expiry, as is any
// other shape.
func splitExemptionExpiry(condition string, native *celast.AST) (string, string) {
	root := native.Expr()
	if expiry, ok := exemptionExpiry(root); ok {
		return "", expiry
	}
	if root.Kind() != celast.CallKind || root.AsCall().FunctionName() != operators.LogicalAnd || len(root.AsCall().Args()) != 2 {
		return condition, ""
	}
	expiry, ok := exemptionExpiry(root.AsCall().Args()[0])
	if !ok {
		return condition, ""
	}
	scope, err := parser.Unparse(root.AsCall().Args()[1], native.SourceInfo())
	if err != nil {
		return condition, ""
	}
	return scope, expiry
}

// exemptionExpiry returns the time of `request.time < timestamp("…")`.
func exemptionExpiry(e celast.Expr) (string, bool) {
	if e.Kind() != celast.CallKind || e.AsCall().FunctionName() != operators.Less || len(e.AsCall().Args()) != 2 {
		return "", false
	}
	left, right := e.AsCall().Args()[0], e.AsCall().Args()[1]
	if left.Kind() != celast.SelectKind || left.AsSelect().FieldName() != "time" {
		return "", false
	}
	if operand := left.AsSelect().Operand(); operand.Kind() != celast.IdentKind || operand.AsIdent() != "request" {
		return "", false
	}
	if right.Kind() != celast.CallKind || right.AsCall().FunctionName() != "timestamp" || len(right.AsCall().Args()) != 1 {
		return "", false
	}
	return stringLiteral(right.AsCall().Args()[0])
}

// exemptionGrantOf returns the grant whose condition is exactly the revoke's,
// so a revoke can name what it ends in the grant's words. The terms go through
// parseExemptionScope, the grant's own checks on a condition, so a condition
// the grant would not write, such as columns with no table, returns false, as
// does a condition in any other form.
func exemptionGrantOf(input RevokeMaskingExemptionInput, native *celast.AST) (*exemptionGrant, bool) {
	grant := GrantMaskingExemptionInput{Project: input.Project}
	var instanceID, databaseName string
	for _, term := range conjuncts(native.Expr()) {
		if expiry, ok := exemptionExpiry(term); ok {
			grant.ExpireTime = expiry
			continue
		}
		attribute, values, ok := resourceValues(term)
		if !ok {
			return nil, false
		}
		switch attribute {
		case "instance_id":
			instanceID = values[0]
		case "database_name":
			databaseName = values[0]
		case "schema_name":
			grant.Schema = values[0]
		case "table_name":
			grant.Table = values[0]
		case "column_name":
			grant.Columns = values
		default:
			return nil, false
		}
	}
	if instanceID != "" || databaseName != "" {
		grant.Database = fmt.Sprintf("instances/%s/databases/%s", instanceID, databaseName)
	} else {
		grant.AllDatabases = true
	}
	g, err := parseExemptionScope(grant)
	if err != nil || g.condition() != input.Condition {
		return nil, false
	}
	return g, true
}

// conjuncts returns the terms of a chain of &&.
func conjuncts(e celast.Expr) []celast.Expr {
	if e.Kind() != celast.CallKind || e.AsCall().FunctionName() != operators.LogicalAnd {
		return []celast.Expr{e}
	}
	var terms []celast.Expr
	for _, arg := range e.AsCall().Args() {
		terms = append(terms, conjuncts(arg)...)
	}
	return terms
}

// resourceValues returns the attribute and value of `resource.x == "v"`, or the
// values of `resource.column_name in ["v", …]`. Only column_name may hold a
// list: exemptionGrantOf reads one value of every other attribute.
func resourceValues(e celast.Expr) (string, []string, bool) {
	if e.Kind() != celast.CallKind || len(e.AsCall().Args()) != 2 {
		return "", nil, false
	}
	left, right := e.AsCall().Args()[0], e.AsCall().Args()[1]
	if left.Kind() != celast.SelectKind {
		return "", nil, false
	}
	if operand := left.AsSelect().Operand(); operand.Kind() != celast.IdentKind || operand.AsIdent() != "resource" {
		return "", nil, false
	}
	attribute := left.AsSelect().FieldName()
	switch e.AsCall().FunctionName() {
	case operators.Equals:
		value, ok := stringLiteral(right)
		return attribute, []string{value}, ok
	case operators.In:
		if attribute != "column_name" || right.Kind() != celast.ListKind {
			return "", nil, false
		}
		var values []string
		for _, element := range right.AsList().Elements() {
			value, ok := stringLiteral(element)
			if !ok {
				return "", nil, false
			}
			values = append(values, value)
		}
		return attribute, values, true
	default:
		return "", nil, false
	}
}

func stringLiteral(e celast.Expr) (string, bool) {
	if e.Kind() != celast.LiteralKind {
		return "", false
	}
	value, ok := e.AsLiteral().(types.String)
	return string(value), ok
}

func (s *Server) maskingExemptionLink(ctx context.Context, project string) string {
	return s.consoleLink(ctx, project) + "/masking-exemption"
}
