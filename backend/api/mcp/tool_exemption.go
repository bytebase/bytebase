package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/utils"
)

// GrantMaskingExemptionInput is the input for the grant_masking_exemption tool.
type GrantMaskingExemptionInput struct {
	Project      string `json:"project"`
	Member       string `json:"member"`
	Database     string `json:"database,omitempty"`
	AllDatabases bool   `json:"allDatabases,omitempty"`
	Schema       string `json:"schema,omitempty"`
	Table        string `json:"table,omitempty"`
	Column       string `json:"column,omitempty"`
	ExpireTime   string `json:"expireTime,omitempty"`
	Reason       string `json:"reason,omitempty"`
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

const grantMaskingExemptionDescription = `Exempt one member from data masking in a Bytebase project, for one database (optionally one schema, table or column) or for every database in the project. The user confirms the exemption in this client first.

| Parameter    | Required | Description |
|--------------|----------|-------------|
| project      | Yes      | Project name, e.g. "projects/hr" |
| member       | Yes      | "user:{email}", "group:{email}", "serviceAccount:{email}" or "workloadIdentity:{email}" |
| database     | One of   | Database name, e.g. "instances/prod/databases/hr"; it must belong to the project |
| allDatabases | One of   | true for every database in the project, current and future |
| schema       | No       | Narrow to one schema (needs database) |
| table        | No       | Narrow to one table (needs database) |
| column       | No       | Narrow to one column (needs table) |
| expireTime   | No       | RFC 3339 time the exemption ends, e.g. "2026-10-27T00:00:00Z"; without it, it does not expire |
| reason       | No       | Why, recorded with the exemption |

An exemption is a standing policy change: the member then sees the covered data unmasked in query and export results. For a one-off query, request an access grant instead (AccessGrantService/CreateAccessGrant).

The tool adds one exemption and leaves every other entry as it is. It shows the user the member, the scope, the expiry and the effect, and asks them to confirm in a form; it writes only when they accept. A client that cannot show the form gets the project's masking exemption page, to grant it in the Bytebase console instead. Do not ask in chat first: the form is the question.

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
	project, member                         string
	database, instanceID, databaseName      string
	schema, table, column, expireTime, note string
}

func parseExemptionGrant(input GrantMaskingExemptionInput) (*exemptionGrant, error) {
	invalid := func(message, suggestion string) error {
		return &toolError{Code: "INVALID_ARGUMENT", Message: message, Suggestion: suggestion}
	}
	if err := checkProjectName(input.Project); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Member) == "" {
		return nil, invalid("member is required", `pass the member, e.g. "user:bob@example.com"`)
	}
	g := &exemptionGrant{
		project: input.Project, member: input.Member,
		schema: input.Schema, table: input.Table, column: input.Column, note: input.Reason,
	}
	switch {
	case input.Database != "" && input.AllDatabases:
		return nil, invalid("pass database or allDatabases, not both", "")
	case input.Database == "" && !input.AllDatabases:
		return nil, invalid("the exemption needs a scope: database, or allDatabases set to true",
			"pass the database, or allDatabases: true for every database in the project, current and future")
	case input.Database != "":
		instanceID, databaseName, err := common.GetInstanceDatabaseID(input.Database)
		if err != nil {
			return nil, invalid(fmt.Sprintf("invalid database %q", input.Database), "pass the database name, e.g. instances/prod/databases/hr")
		}
		g.database, g.instanceID, g.databaseName = input.Database, instanceID, databaseName
	default:
		if input.Schema != "" || input.Table != "" || input.Column != "" {
			return nil, invalid("schema, table and column narrow one database, so they need database", "")
		}
	}
	if input.Column != "" && input.Table == "" {
		return nil, invalid("column needs table", "pass the table the column belongs to")
	}
	if input.ExpireTime != "" {
		expireTime, err := time.Parse(time.RFC3339, input.ExpireTime)
		if err != nil {
			return nil, invalid(fmt.Sprintf("invalid expireTime %q", input.ExpireTime), `pass an RFC 3339 time, e.g. "2026-10-27T00:00:00Z"`)
		}
		g.expireTime = expireTime.UTC().Format(time.RFC3339)
	}
	return g, nil
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
			{"resource.column_name", g.column},
		} {
			if narrowing.value != "" {
				scope = append(scope, narrowing.attribute+" == "+strconv.Quote(narrowing.value))
			}
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
	if exemptions == nil {
		exemptions = []map[string]any{}
	}
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
	if grant.database != "" {
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
	exemptions = append(exemptions, map[string]any{"members": []string{grant.member}, "condition": condition})
	body := exemptionUpdateBody(write.policy, exemptions)
	output := &MaskingExemptionOutput{Project: grant.project, Member: grant.member, Condition: grant.condition(), Link: link}
	return s.confirmExemptionWrite(ctx, req, write, body, describeExemptionGrant(ctx, grant), output,
		"Exempt this member from masking",
		fmt.Sprintf("%s was not exempted", grant.member),
		fmt.Sprintf("Exempted %s from masking in %s.", grant.member, grant.project),
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
	return s.confirmExemptionWrite(ctx, req, write, body, describeExemptionRevoke(ctx, input, matched), output,
		"Remove this masking exemption",
		fmt.Sprintf("%s's exemption was not removed", input.Member),
		fmt.Sprintf("Removed %s's masking exemption in %s.", input.Member, input.Project),
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
		kept := make(map[string]any, len(entry))
		for k, v := range entry {
			kept[k] = v
		}
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
// The digest is the exact request, built from a fresh read of the whole list,
// so an answer is acted on only while the arguments and the list read as they
// did when the user was asked.
func (s *Server) confirmExemptionWrite(ctx context.Context, req *mcp.CallToolRequest, write *exemptionWrite, body map[string]any, message string, output *MaskingExemptionOutput, title, notDone, done string) (*mcp.CallToolResult, any, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the update")), nil, nil
	}
	digest := sha256.Sum256(encoded)
	reply, answered := s.signedReply(ctx, req, write, digest[:])
	if !answered {
		return s.askToConfirm(ctx, write, digest[:], message, title,
			"Check to make the change described above. Leaving it unchecked, declining or cancelling changes nothing."), nil, nil
	}
	if confirmed, why := confirmationOutcome(reply.response); !confirmed {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
			"%s: %s, so nothing changed. Do not retry unless the user asks.\nMasking exemptions: %s", capitalize(notDone), why, output.Link)}}}, nil, nil
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

// describeExemptionGrant builds the grant's confirmation message. On each line
// the value a cut would most mislead about comes first, and the rest is ordered
// so a cut makes the scope look broader, never narrower: the database name
// before its narrowing. Labels drop the colon where it would cost the longest
// value a column: `Instance "` and `Database "` leave 63 columns, a resource
// ID's maximum, within the 74 Claude Code shows of an 80-column line (73 and
// "…" when the line goes on). Values enter through %q, so none can pose as a
// line of the message.
func describeExemptionGrant(ctx context.Context, g *exemptionGrant) string {
	question := "Exempt from masking: " + g.member
	if email := getUserEmail(ctx); email != "" {
		question += ", as " + email
	}
	lines := []string{question + "?"}
	if g.database == "" {
		projectID, _ := common.GetProjectID(g.project)
		lines = append(lines, fmt.Sprintf("Project %q", projectID), "Scope: all databases in the project, current and future")
	} else {
		lines = append(lines, fmt.Sprintf("Instance %q", g.instanceID), fmt.Sprintf("Database %q, %s", g.databaseName, describeNarrowing(g)))
	}
	effect := "Unmasked in query and export results, with no expiry."
	if g.expireTime != "" {
		effect = fmt.Sprintf("Unmasked in query and export results until %q.", g.expireTime)
	}
	if g.note != "" {
		effect += fmt.Sprintf(" Reason: %q", g.note)
	}
	return strings.Join(append(lines, effect), "\n")
}

func describeNarrowing(g *exemptionGrant) string {
	table := strconv.Quote(g.table)
	if g.schema != "" {
		table = strconv.Quote(g.schema) + "." + table
	}
	switch {
	case g.column != "":
		return fmt.Sprintf("column %q in %s", g.column, table)
	case g.table != "":
		return "table " + table
	case g.schema != "":
		return fmt.Sprintf("schema %q", g.schema)
	default:
		return "all tables"
	}
}

// describeExemptionRevoke builds the revoke's confirmation message. The
// condition is unbounded and is cut at its tail; a revoke only removes access,
// and a wrong one is undone by a grant.
func describeExemptionRevoke(ctx context.Context, input RevokeMaskingExemptionInput, matched int) string {
	question := "Revoke masking exemption: " + input.Member
	if email := getUserEmail(ctx); email != "" {
		question += ", as " + email
	}
	projectID, _ := common.GetProjectID(input.Project)
	condition := "Condition"
	if matched > 1 {
		condition += fmt.Sprintf(" (%d entries)", matched)
	}
	condition += " " + strconv.Quote(input.Condition)
	if input.Condition == "" {
		condition += " (all databases, no expiry)"
	}
	return strings.Join([]string{
		question + "?",
		fmt.Sprintf("Project %q", projectID),
		condition,
		"Masked again in their results unless another exemption covers the data.",
	}, "\n")
}

func (s *Server) maskingExemptionLink(ctx context.Context, project string) string {
	externalURL, _ := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, getWorkspaceID(ctx))
	return buildResourceURL(externalURL, project) + "/masking-exemption"
}
