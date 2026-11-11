package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/idp/wif"
	"github.com/bytebase/bytebase/backend/store"
)

// identityIssuers are the issuers the identity entry sets: the console's
// presets for GitHub Actions and GitLab CI. Whoever chooses the issuer chooses
// the keys a token is checked with, so the request cannot.
var identityIssuers = map[v1pb.WorkloadIdentityConfig_ProviderType]string{
	v1pb.WorkloadIdentityConfig_GITHUB: "https://token.actions.githubusercontent.com",
	v1pb.WorkloadIdentityConfig_GITLAB: "https://gitlab.com",
}

// identityAudience is the console's preset and the audience the generated
// GitOps workflows request.
const identityAudience = "bytebase"

// identityRole is the role the entry grants: the console's default, which the
// GitOps workflows need.
var identityRole = common.FormatRole(store.GitOpsServiceAgentRole)

// identityRequestFields are the CreateWorkloadIdentityRequest fields the
// identity entry takes. A request that sets any other field is refused before
// the form, since the form could not state it.
var identityRequestFields = map[string]bool{
	"parent":                  true,
	"workload_identity_id":    true,
	"workload_identity":       true,
	"workload_identity.title": true,
	"workload_identity.workload_identity_config":                   true,
	"workload_identity.workload_identity_config.provider_type":     true,
	"workload_identity.workload_identity_config.issuer_url":        true,
	"workload_identity.workload_identity_config.allowed_audiences": true,
	"workload_identity.workload_identity_config.subject_pattern":   true,
}

// identityCreation is a request to create a workload identity for CI in a
// project, which the entry then grants its role there.
type identityCreation struct {
	request   *v1pb.CreateWorkloadIdentityRequest
	projectID string
	email     string
	// path is the repository whose jobs sign in, as "owner/repo" or
	// "group/project".
	path string
	// pinned is whether an owner ID or a repository ID pins the GitHub
	// repository, so a new account under a freed name cannot sign in.
	pinned bool
}

func (*identityCreation) confirmedProcedure() string {
	return v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure
}

// confirmedResource is the new identity's name, so an answer covers that one
// identity, not every identity in its project.
func (c *identityCreation) confirmedResource() string {
	return common.FormatWorkloadIdentityEmail(c.email)
}

func (c *identityCreation) config() *v1pb.WorkloadIdentityConfig {
	return c.request.GetWorkloadIdentity().GetWorkloadIdentityConfig()
}

func (c *identityCreation) github() bool {
	return c.config().GetProviderType() == v1pb.WorkloadIdentityConfig_GITHUB
}

// identitySubject is a workload identity's creation in confirmAdminChange's
// words.
type identitySubject struct{}

func (identitySubject) unchanged() string { return "no workload identity was created" }

func (identitySubject) notChanged() string { return "no workload identity was created" }

func (identitySubject) revised() string { return "the revised request" }

func (identitySubject) reread() string {
	return "call administer_workspace again to ask the user about the request as it is now"
}

func (s *Server) administerIdentity(ctx context.Context, req *mcp.CallToolRequest, input AdministerWorkspaceInput) (*mcp.CallToolResult, any, error) {
	change, err := parseIdentityCreation(input.Body)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	project := common.FormatProject(change.projectID)
	link := s.consoleLink(ctx, project+"/workload-identities")
	if err := change.check(link); err != nil {
		return formatToolError(err), nil, nil
	}
	if !canShowConfirmation(req) {
		return formatToolError(&toolError{
			Code:       "CONFIRMATION_UNAVAILABLE",
			Message:    "this MCP client cannot show the confirmation form administer_workspace needs, so no workload identity was created",
			Suggestion: "create it signed in to the Bytebase console: " + link,
		}), nil, nil
	}
	// Read on both rounds: an answer is acted on only while the grant the form
	// promises can be made.
	if err := s.checkRoleExpiryLimit(ctx, link); err != nil {
		return formatToolError(err), nil, nil
	}
	form, err := identityForm(ctx, change, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	digest := adminFormDigest(form, change.request)
	if result := s.confirmAdminChange(ctx, req, change, digest, form, identitySubject{}, "this request and its form"); result != nil {
		return result, nil, nil
	}

	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: change.confirmedProcedure(),
		Resource:  change.confirmedResource(),
	})
	body, err := protojson.Marshal(change.request)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to build the request")), nil, nil
	}
	resp, err := s.apiRequest(confirmedCtx, change.confirmedProcedure(), json.RawMessage(body))
	if why := refusal(resp, err); why != "" {
		return formatToolError(errors.Errorf("failed to create the workload identity: %s", why)), nil, nil
	}
	role := roleTitle(identityRole)
	if why := s.grantIdentityRole(ctx, change); why != "" {
		membersLink := s.consoleLink(ctx, project+"/members")
		text := fmt.Sprintf("Created workload identity %q in project %q, but Bytebase did not grant it the %s role: %s. It can sign in, and has no role in the project until one is granted signed in to the console.\nMembers: %s", change.email, change.projectID, role, why, membersLink)
		return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: membersLink}, nil
	}
	gitopsLink := s.consoleLink(ctx, project+"/gitops")
	text := fmt.Sprintf("Created: workload identity %q in project %q, with the %s role there. A workflow signs in with that email as BYTEBASE_WORKLOAD_IDENTITY, which the GitOps page's workflows use.\nGitOps: %s", change.email, change.projectID, role, gitopsLink)
	return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: gitopsLink}, nil
}

// grantIdentityRole adds the new identity to the role's binding that has no
// condition, or appends that binding, with the etag of the policy it read. It
// returns why the grant was not made, or "".
func (s *Server) grantIdentityRole(ctx context.Context, c *identityCreation) string {
	project := common.FormatProject(c.projectID)
	resp, err := s.apiRequest(ctx, v1connect.ProjectServiceGetIamPolicyProcedure, map[string]any{"resource": project})
	if why := refusal(resp, err); why != "" {
		return "the project's IAM policy could not be read: " + why
	}
	policy := &v1pb.IamPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, policy); err != nil {
		return "the project's IAM policy could not be read: " + err.Error()
	}
	addToRole(policy, identityRole, common.WorkloadIdentityBindingPrefix+c.email)
	body, err := protojson.Marshal(&v1pb.SetIamPolicyRequest{Resource: project, Policy: policy})
	if err != nil {
		return "the grant could not be built: " + err.Error()
	}
	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: v1connect.ProjectServiceSetIamPolicyProcedure,
		Resource:  project,
	})
	resp, err = s.apiRequest(confirmedCtx, v1connect.ProjectServiceSetIamPolicyProcedure, json.RawMessage(body))
	return refusal(resp, err)
}

// addToRole adds member to the role's binding with no condition, or appends
// that binding. A binding with a condition would give the member its expiry or
// its database scope.
func addToRole(policy *v1pb.IamPolicy, role, member string) {
	for _, binding := range policy.GetBindings() {
		if binding.GetRole() != role || binding.GetCondition().GetExpression() != "" {
			continue
		}
		if !slices.Contains(binding.GetMembers(), member) {
			binding.Members = append(binding.Members, member)
		}
		return
	}
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{Role: role, Members: []string{member}})
}

// checkRoleExpiryLimit refuses when the workspace limits how long a role is
// granted. The IAM policy then refuses a binding with no expiry for every role
// but Project Owner (validateBindings), so the grant after the create would
// fail and leave the identity with no role. The API refuses this setting's
// read over MCP, so it is read from the store.
func (s *Server) checkRoleExpiryLimit(ctx context.Context, link string) error {
	profile, err := s.store.GetWorkspaceProfileSetting(ctx, getWorkspaceID(ctx))
	if err != nil {
		return errors.Wrap(err, "failed to read the workspace's role expiry limit")
	}
	if profile.GetMaximumRoleExpiration() == nil {
		return nil
	}
	return &toolError{
		Code:       "ROLE_EXPIRY_REQUIRED",
		Message:    fmt.Sprintf("this workspace limits how long a role can be granted, and administer_workspace grants the %s role with no expiry, so no workload identity was created", roleTitle(identityRole)),
		Suggestion: "create the identity signed in to the Bytebase console, and grant its role there with an expiry the limit allows: " + link,
	}
}

// parseIdentityCreation reads the request and the project it creates the
// identity in.
func parseIdentityCreation(body map[string]any) (*identityCreation, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body could not be read: " + err.Error()}
	}
	request := &v1pb.CreateWorkloadIdentityRequest{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(encoded, request); err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body is not a CreateWorkloadIdentityRequest: " + err.Error()}
	}
	parent := request.GetParent()
	projectID, err := common.GetProjectID(parent)
	if err != nil || projectID == "-" {
		return nil, &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace creates a workload identity in a project, and the request's parent is %q", parent),
			Suggestion: `set parent to the project's name, such as "projects/hr"; create a workspace identity signed in to the Bytebase console`,
		}
	}
	return &identityCreation{
		request:   request,
		projectID: projectID,
		email:     common.BuildWorkloadIdentityEmail(request.GetWorkloadIdentityId(), projectID),
	}, nil
}

// check refuses what the form could not state or the create would refuse, and
// then sets what the tool chooses: the provider's issuer, the audience, and a
// title for a request that has none, as the console does.
func (c *identityCreation) check(link string) error {
	if field := unlistedField(c.request.ProtoReflect(), "", identityRequestFields); field != "" {
		return &toolError{
			Code:       "FIELD_NOT_SUPPORTED",
			Message:    fmt.Sprintf("the request sets %s, which administer_workspace does not take", field),
			Suggestion: "send parent, workloadIdentityId and workloadIdentity with title and workloadIdentityConfig: providerType and subjectPattern only; the tool sets the issuer and the audience. Set up anything else signed in to the Bytebase console: " + link,
		}
	}
	config := c.config()
	issuer, ok := identityIssuers[config.GetProviderType()]
	if !ok {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace creates identities for GitHub Actions and GitLab CI, and the request's provider is %s", config.GetProviderType()),
			Suggestion: "set workloadIdentityConfig.providerType to GITHUB or GITLAB, or set up another provider signed in to the Bytebase console: " + link,
		}
	}
	if sent := config.GetIssuerUrl(); sent != "" && sent != issuer {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("the issuer %q is not %s, the issuer administer_workspace sets for %s", sent, issuer, config.GetProviderType()),
			Suggestion: "leave issuerUrl out; GitHub Enterprise Server and self-managed GitLab are set up signed in to the Bytebase console: " + link,
		}
	}
	if sent := config.GetAllowedAudiences(); len(sent) > 0 && !slices.Equal(sent, []string{identityAudience}) {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace sets the audience %q, the one the GitOps workflows request, and the request sends %s", identityAudience, quoteAll(sent)),
			Suggestion: "leave allowedAudiences out, or set up another audience signed in to the Bytebase console: " + link,
		}
	}
	path, pinned, ok := identityRepository(config)
	if !ok {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("the subject pattern %q does not name one repository with every job in it, which is what administer_workspace creates an identity for", config.GetSubjectPattern()),
			Suggestion: `send "repo:OWNER/REPO:*" for GitHub, with "@OWNER_ID" after the owner when you know it, or "project_path:GROUP/PROJECT:*" for GitLab. A pattern for a branch, an environment, pull requests or every repository of an owner is set up signed in to the Bytebase console: ` + link,
		}
	}
	if err := common.ValidateEmail(c.email); err != nil || !common.IsWorkloadIdentityEmail(c.email) {
		return &toolError{
			Code:       "INVALID_REQUEST",
			Message:    fmt.Sprintf("the identity ID %q does not make a valid email, %q", c.request.GetWorkloadIdentityId(), c.email),
			Suggestion: `use letters, digits, ".", "_" and "-", starting and ending with a letter or digit, such as "hr-ci"`,
		}
	}
	config.IssuerUrl = issuer
	config.AllowedAudiences = []string{identityAudience}
	if identity := c.request.GetWorkloadIdentity(); strings.TrimSpace(identity.GetTitle()) == "" {
		identity.Title = c.request.GetWorkloadIdentityId()
	}
	c.path, c.pinned = path, pinned
	return nil
}

// identityRepository reads the one repository a subject pattern admits every
// job of: GitHub's through the handler's own parser, and GitLab's by its path.
// No name holds white space, so a pattern with any matches no job. pinned is
// whether an owner ID or a repository ID pins a GitHub repository.
func identityRepository(config *v1pb.WorkloadIdentityConfig) (path string, pinned bool, ok bool) {
	pattern := config.GetSubjectPattern()
	if strings.ContainsFunc(pattern, unicode.IsSpace) {
		return "", false, false
	}
	switch config.GetProviderType() {
	case v1pb.WorkloadIdentityConfig_GITHUB:
		repository, parsed := wif.ParseGitHubRepositoryPattern(pattern)
		if !parsed {
			return "", false, false
		}
		return repository.Owner + "/" + repository.Name, repository.OwnerID != "" || repository.ID != "", true
	case v1pb.WorkloadIdentityConfig_GITLAB:
		projectPath, parsed := gitlabProjectPath(pattern)
		return projectPath, false, parsed
	default:
		return "", false, false
	}
}

// gitlabProjectPath reads "project_path:GROUP/PROJECT:*", where PROJECT may
// name subgroups. The handler has no GitLab parser and matches the pattern by
// prefix alone; ":" ends the path in GitLab's subject, so the pattern admits
// that project and no other.
func gitlabProjectPath(pattern string) (string, bool) {
	body, ok := strings.CutPrefix(pattern, "project_path:")
	if !ok {
		return "", false
	}
	path, ok := strings.CutSuffix(body, ":*")
	if !ok {
		return "", false
	}
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return "", false
	}
	for _, segment := range segments {
		if segment == "" || strings.ContainsAny(segment, "*:@") {
			return "", false
		}
	}
	return path, true
}

// identityForm builds the form that asks to create the identity: who signs in
// as it, and the role it gets.
func identityForm(ctx context.Context, c *identityCreation, link string) (*adminForm, error) {
	question := fmt.Sprintf("Create workload identity %q in %q", c.request.GetWorkloadIdentityId(), c.projectID)
	// Only the account's email may run past a line, so the IDs must fit.
	if lineColumns(question+"?") > messageLineWidth {
		return nil, &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("the question naming identity %q and project %q is wider than one line of the confirmation form, so no workload identity was created", c.request.GetWorkloadIdentityId(), c.projectID),
			Suggestion: "use a shorter identity ID, or create it signed in to the Bytebase console: " + link,
		}
	}
	jobs := "GitLab CI"
	if c.github() {
		jobs = "GitHub Actions"
	}
	changes := []formChange{
		{text: fmt.Sprintf("Any %s job in %q signs in as it, and the MCP access policy does not limit it.", jobs, c.path)},
		{text: fmt.Sprintf("Role: %s in this project. It lasts until removed.", roleTitle(identityRole))},
	}
	return fitForm(ctx, question, changes, "", "", identityDescription(c), func(needed int, _ string) error {
		return &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("this workload identity needs %d lines, more than the %d one confirmation form shows, so no workload identity was created", needed, messageLines),
			Suggestion: "create it signed in to the Bytebase console: " + link,
		}
	})
}

// identityDescription is what the message leaves out. The owner warning comes
// first, since fullscreen shows the description's first line only.
func identityDescription(c *identityCreation) string {
	var sb strings.Builder
	if c.github() && !c.pinned {
		owner, _, _ := strings.Cut(c.path, "/")
		fmt.Fprintf(&sb, "The rule trusts the owner's name: if %q is renamed or deleted, whoever registers the name can sign in as it.\n", owner)
	}
	config := c.config()
	fmt.Fprintf(&sb, "It signs in as %s with tokens from %s for audience %q.", c.email, config.GetIssuerUrl(), identityAudience)
	fmt.Fprintf(&sb, "\n%s lets it create plans, releases and rollouts in this project and run their tasks; the project's settings decide whether a change waits for approval.", roleTitle(identityRole))
	fmt.Fprintf(&sb, " Titled %q.", c.request.GetWorkloadIdentity().GetTitle())
	return sb.String()
}

// roleTitle names a role by its display name when it is predefined, as the
// console does.
func roleTitle(role string) string {
	if roleID, err := common.GetRoleID(role); err == nil {
		if predefined := store.GetPredefinedRole(roleID); predefined != nil {
			return predefined.Name
		}
	}
	return role
}
