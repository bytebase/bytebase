package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

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

// identityConsoleOnly ends each result of a create that stored the identity:
// the gate refuses its update and its delete over MCP.
const identityConsoleOnly = "MCP cannot change or remove a workload identity: change or remove it in the console."

// identityRole is the role the entry grants: the console's default, which the
// GitOps workflows need.
var (
	identityRole      = common.FormatRole(store.GitOpsServiceAgentRole)
	identityRoleTitle = roleTitle(identityRole)
)

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

// identityRepository is the one repository whose every job a subject pattern
// admits. ownerID and repoID are the GitHub IDs the pattern pins, if any.
type identityRepository struct {
	github                bool
	path, ownerID, repoID string
}

// readIdentityRepository reads a request's subject pattern with the handler's
// own grammar, through wif.
func readIdentityRepository(config *v1pb.WorkloadIdentityConfig) (identityRepository, bool) {
	switch config.GetProviderType() {
	case v1pb.WorkloadIdentityConfig_GITHUB:
		repository, ok := wif.ParseGitHubRepositoryPattern(config.GetSubjectPattern())
		return identityRepository{github: true, path: repository.Owner + "/" + repository.Name, ownerID: repository.OwnerID, repoID: repository.ID}, ok
	case v1pb.WorkloadIdentityConfig_GITLAB:
		path, ok := wif.ParseGitLabProjectPattern(config.GetSubjectPattern())
		return identityRepository{path: path}, ok
	default:
		return identityRepository{}, false
	}
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
	change.fill()
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
	if result := s.confirmAdminChange(ctx, req, change, digest, form, creationSubject{nothing: "no workload identity was created"}, "this request and its form"); result != nil {
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
	if why := s.grantIdentityRole(ctx, change); why != "" {
		membersLink := s.consoleLink(ctx, project+"/members")
		text := fmt.Sprintf("Created workload identity %q in project %q, but Bytebase did not grant it the %s role: %s. It can sign in, and it does not have that role until it is granted signed in to the console. %s\nMembers: %s\nWorkload identities: %s",
			change.email, change.projectID, identityRoleTitle, why, identityConsoleOnly, membersLink, link)
		return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: membersLink}, nil
	}
	gitopsLink := s.consoleLink(ctx, project+"/gitops")
	text := fmt.Sprintf("Created: workload identity %q in project %q, with the %s role there. A workflow signs in with that email as BYTEBASE_WORKLOAD_IDENTITY, which the GitOps page's workflows use. %s\nGitOps: %s\nWorkload identities: %s",
		change.email, change.projectID, identityRoleTitle, identityConsoleOnly, gitopsLink, link)
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
// fail and leave the identity without its role. The API refuses this setting's
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
		Message:    fmt.Sprintf("this workspace limits how long a role can be granted, and administer_workspace grants the %s role with no expiry, so no workload identity was created", identityRoleTitle),
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
			Suggestion: `set parent to the project's name, such as "projects/hr"`,
		}
	}
	return &identityCreation{
		request:   request,
		projectID: projectID,
		email:     common.BuildWorkloadIdentityEmail(request.GetWorkloadIdentityId(), projectID),
	}, nil
}

// check refuses what the form could not state. The create refuses the rest,
// an identity ID it cannot keep among them, after the user confirms.
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
	if _, ok := readIdentityRepository(config); !ok {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("the subject pattern %q does not name one repository with every job in it, which is what administer_workspace creates an identity for", config.GetSubjectPattern()),
			Suggestion: `send "repo:OWNER/REPO:*" for GitHub, with "@OWNER_ID" after the owner when you know it, or "project_path:GROUP/PROJECT:*" for GitLab. A pattern for a branch, an environment, pull requests or every repository of an owner is set up signed in to the Bytebase console: ` + link,
		}
	}
	return nil
}

// fill sets what the tool chooses, after check has refused another choice: the
// provider's issuer, the audience, and a title for a request that has none, as
// the console does. It runs before the form and the digest, which state the
// request as filled.
func (c *identityCreation) fill() {
	config := c.config()
	config.IssuerUrl = identityIssuers[config.GetProviderType()]
	config.AllowedAudiences = []string{identityAudience}
	if identity := c.request.GetWorkloadIdentity(); strings.TrimSpace(identity.GetTitle()) == "" {
		identity.Title = c.request.GetWorkloadIdentityId()
	}
}

// identityForm builds the form that asks to create the identity: who signs in
// as it, and the role it gets.
func identityForm(ctx context.Context, c *identityCreation, link string) (*adminForm, error) {
	question := func(identityID string) string {
		return fmt.Sprintf("Create workload identity %q in %q", identityID, c.projectID)
	}
	// Only the account's email may run past a line, so the IDs must fit.
	if lineColumns(question(c.request.GetWorkloadIdentityId())+"?") > messageLineWidth {
		suggestion := "create it signed in to the Bytebase console: " + link
		if lineColumns(question("a")+"?") <= messageLineWidth {
			suggestion = "use a shorter identity ID, or " + suggestion
		}
		return nil, &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("the question naming identity %q and project %q is wider than one line of the confirmation form, so no workload identity was created", c.request.GetWorkloadIdentityId(), c.projectID),
			Suggestion: suggestion,
		}
	}
	repository, _ := readIdentityRepository(c.config())
	jobs := "GitLab CI"
	if repository.github {
		jobs = "GitHub Actions"
	}
	changes := []formChange{
		{text: fmt.Sprintf("Any %s job in %q signs in as it, and the MCP access policy does not limit it.", jobs, repository.path)},
		{text: fmt.Sprintf("Role: %s. It can roll out without approval.", identityRoleTitle)},
	}
	return fitForm(ctx, question(c.request.GetWorkloadIdentityId()), changes, "", "", identityDescription(c, repository), func(needed int, _ string) error {
		return &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("this workload identity needs %d lines, more than the %d one confirmation form shows, so no workload identity was created", needed, messageLines),
			Suggestion: "create it signed in to the Bytebase console: " + link,
		}
	})
}

// identityDescription is what the message leaves out. Its first line is what
// the pattern pins on GitHub, since fullscreen shows that line only.
func identityDescription(c *identityCreation, repository identityRepository) string {
	var sb strings.Builder
	if repository.github {
		var pins []string
		if repository.ownerID != "" {
			pins = append(pins, fmt.Sprintf("owner ID %q", repository.ownerID))
		}
		if repository.repoID != "" {
			pins = append(pins, fmt.Sprintf("repository ID %q", repository.repoID))
		}
		owner, _, _ := strings.Cut(repository.path, "/")
		if len(pins) == 0 {
			fmt.Fprintf(&sb, "The rule trusts the owner's name: if %q is renamed or deleted, whoever registers the name can sign in as it.\n", owner)
		} else {
			fmt.Fprintf(&sb, "The pattern pins %s, so a new account under a freed name cannot sign in as it.\n", strings.Join(pins, " and "))
		}
	}
	fmt.Fprintf(&sb, "It signs in as %s with tokens from %s for audience %q. It lasts until it is removed.", c.email, c.config().GetIssuerUrl(), identityAudience)
	// The role holds bb.taskRuns.create, which runs tasks in every environment,
	// and no issue permission, so its rollouts carry no issue; approval checks
	// apply only to a rollout with an issue (RolloutService.CreateRollout).
	fmt.Fprintf(&sb, "\n%s lets it create plans, releases and rollouts in this project and run their tasks in every environment. Its rollouts carry no issue, so no approval rule holds them.", identityRoleTitle)
	fmt.Fprintf(&sb, " Titled %q.", c.request.GetWorkloadIdentity().GetTitle())
	return sb.String()
}
