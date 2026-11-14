package mcp

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/utils"
)

// GitOpsWorkflowInput is the input for the get_gitops_workflow tool.
type GitOpsWorkflowInput struct {
	// Project is the project ID or name.
	Project string `json:"project"`
	// Targets is one database group, or one or more databases, as resource names.
	Targets []string `json:"targets"`
	// WorkloadIdentity is the email of the identity the jobs sign in as.
	WorkloadIdentity string `json:"workloadIdentity"`
	// Folder holds the migration files. Defaults to "migrations".
	Folder string `json:"folder,omitempty"`
}

// GitOpsWorkflowOutput is the output for the get_gitops_workflow tool. Message
// carries everything the result's text says but the files: Claude Code hands
// the model only the structured result when a result has one (BOT-165).
type GitOpsWorkflowOutput struct {
	Files []WorkflowFile `json:"files"`
	// Include is the line to add to the project's own .gitlab-ci.yml.
	Include     string              `json:"include,omitempty"`
	DeployOrder []GitOpsDeployStage `json:"deployOrder"`
	// Skipped are picked databases with no environment. The rollout skips them.
	Skipped  *GitOpsDatabases `json:"skipped,omitempty"`
	Identity GitOpsIdentity   `json:"identity"`
	Message  string           `json:"message"`
}

// GitOpsDeployStage is one environment the workflow deploys to.
type GitOpsDeployStage struct {
	Environment string `json:"environment"`
	GitOpsDatabases
}

// GitOpsDatabases counts databases and names the first few.
type GitOpsDatabases struct {
	DatabaseCount int      `json:"databaseCount"`
	Databases     []string `json:"databases"`
}

// GitOpsIdentity is what the result shows of the workload identity. Roles are
// the roles the project's IAM policy binds to it.
type GitOpsIdentity struct {
	Email          string   `json:"email"`
	Provider       string   `json:"provider"`
	SubjectPattern string   `json:"subjectPattern"`
	Roles          []string `json:"roles"`
}

// namedDatabases is how many databases a stage or the skipped list names.
const namedDatabases = 10

const findIdentitySuggestion = "list the project's workload identities with call_api WorkloadIdentityService/ListWorkloadIdentities, or create one with administer_workspace"

const gitopsWorkflowDescription = `Return the workflow files that set up GitOps for a project. With them in the repository, every pull request that changes the migration folder gets a Bytebase SQL review, and merging it deploys. The tool reads only; it changes nothing in Bytebase.

| Parameter        | Required | Description |
|------------------|----------|-------------|
| project          | Yes      | The project ID, e.g. "hr" |
| targets          | Yes      | One database group ("projects/hr/databaseGroups/all"), or one or more databases by the name ListDatabases gives ("instances/prod/databases/hr") |
| workloadIdentity | Yes      | The email of the workload identity the jobs sign in as |
| folder           | No       | The folder that holds the migration files. Default "migrations" |

To set up GitOps from the team's repository:
1. Read the repository. The git remote gives the provider (GitHub or GitLab), the owner and the repository. Look for the folder that holds the SQL migration files.
2. Ask the owner which databases to deploy to. List the project's database groups and databases with call_api (DatabaseGroupService/ListDatabaseGroups, DatabaseService/ListDatabases). Offer a group first: a database that joins it later is deployed with no change to the files, when its environment is one the files already deploy to.
3. Find the workload identity with call_api WorkloadIdentityService/ListWorkloadIdentities on the project. Use one whose subject pattern names this repository. If there is none, create one with administer_workspace.
4. Call this tool. Write each file at its path, and on GitLab add the include line the result gives to .gitlab-ci.yml. Ask before you replace a file that exists. Open a pull request into the default branch.

The jobs sign in with their own ID token, so the repository needs no secret. The workflows run only when a file in the migration folder changes, so the setup pull request runs neither, and the first pull request with a migration file is the test. With the GitOps Service Agent role, the identity's rollouts carry no issue, so no approval rule holds them: tell the owner that merging deploys.

You may fit the files to the repository, for example the runner, the stages, or how they join an existing pipeline. Keep BYTEBASE_URL, BYTEBASE_WORKLOAD_IDENTITY, the job permissions, the ID token settings, the action image, --project, --targets and the order of the deploy jobs. The identity admits every job in the repository, so the triggers alone decide which branch deploys: tell the owner before a change that lets another branch deploy.`

func (s *Server) registerGitOpsWorkflowTool() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_gitops_workflow",
		Description: gitopsWorkflowDescription,
	}, s.handleGetGitOpsWorkflow)
}

func (s *Server) handleGetGitOpsWorkflow(ctx context.Context, _ *mcp.CallToolRequest, input GitOpsWorkflowInput) (*mcp.CallToolResult, any, error) {
	output, err := s.gitopsWorkflow(ctx, input)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	var text strings.Builder
	text.WriteString(output.Message)
	for _, file := range output.Files {
		fmt.Fprintf(&text, "\n\n=== %s ===\n%s", file.Path, file.Content)
	}
	return textResult(text.String()), output, nil
}

func (s *Server) gitopsWorkflow(ctx context.Context, input GitOpsWorkflowInput) (*GitOpsWorkflowOutput, error) {
	projectID, err := common.GetProjectID(formatProjectName(strings.TrimSpace(input.Project)))
	if err != nil || projectID == "-" {
		return nil, &toolError{
			Code:       "INVALID_PROJECT",
			Message:    fmt.Sprintf("get_gitops_workflow returns the files for one project, and the request's project is %q", input.Project),
			Suggestion: `set project to the project's ID, such as "hr"`,
		}
	}
	project := common.FormatProject(projectID)
	email := strings.TrimPrefix(strings.TrimSpace(input.WorkloadIdentity), common.WorkloadIdentityNamePrefix)
	if email == "" {
		return nil, &toolError{
			Code:       "INVALID_IDENTITY",
			Message:    "workloadIdentity is required: the email of the workload identity the jobs sign in as",
			Suggestion: findIdentitySuggestion,
		}
	}
	folder := input.Folder
	if folder == "" {
		folder = defaultMigrationFolder
	}
	if !isValidMigrationFolder(folder) {
		return nil, &toolError{
			Code:       "INVALID_FOLDER",
			Message:    fmt.Sprintf("the folder %q is not a relative folder path", folder),
			Suggestion: `send a path from the repository root, such as "migrations" or "db/migrations": segments of letters, digits, ".", "_" and "-" joined by "/"`,
		}
	}
	group, err := groupTarget(input.Targets, project)
	if err != nil {
		return nil, err
	}

	bytebaseURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile, getWorkspaceID(ctx))
	if err != nil {
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			return nil, errors.Wrap(err, "failed to read the External URL")
		}
		return nil, &toolError{
			Code:       "EXTERNAL_URL_NOT_SET",
			Message:    "this workspace has no External URL, the address the jobs reach Bytebase at, so the files would not work",
			Suggestion: "a workspace admin sets it in the Bytebase console under Settings > General; then call get_gitops_workflow again",
		}
	}

	identity, err := s.readGitOpsIdentity(ctx, email, project)
	if err != nil {
		return nil, err
	}
	databaseNames := input.Targets
	if group != "" {
		if databaseNames, err = s.readGroupDatabases(ctx, group); err != nil {
			return nil, err
		}
	}
	databases, err := s.readTargetDatabases(ctx, project, databaseNames)
	if err != nil {
		return nil, err
	}
	environmentOrder, err := s.readEnvironmentOrder(ctx)
	if err != nil {
		return nil, apiReadError(err, "PERMISSION_DENIED", "the workspace's environments, which set the deploy order, could not be read", "")
	}
	stages, skipped := deployStages(databases, environmentOrder)
	if len(stages) == 0 {
		return nil, &toolError{
			Code:       "NO_DEPLOY_TARGET",
			Message:    "none of the picked databases has an environment, and a rollout deploys by environment, so the files would deploy nothing",
			Suggestion: "assign an environment to the databases in the Bytebase console, or pick databases that have one: " + buildResourceURL(bytebaseURL, project+"/databases"),
		}
	}

	params := workflowParams{
		Provider:              identity.provider,
		BytebaseURL:           strings.TrimSuffix(bytebaseURL, "/"),
		ActionImage:           bytebaseActionImage(s.profile.Version),
		WorkloadIdentityEmail: email,
		ProjectID:             projectID,
		Targets:               input.Targets,
		Folder:                folder,
	}
	output := &GitOpsWorkflowOutput{DeployOrder: stages, Skipped: skipped, Identity: identity.shown}
	for _, stage := range stages {
		params.Environments = append(params.Environments, stage.Environment)
	}
	output.Files = generateWorkflowFiles(params)
	if identity.provider == workflowProviderGitLab {
		output.Include = gitlabWorkflowInclude
	}
	output.Message = gitopsMessage(output, identity, projectID, folder, buildResourceURL(bytebaseURL, project+"/members"))
	return output, nil
}

// groupTarget returns the database group the targets name, or "" when they name
// databases. A workflow deploys to one group or to a list of databases, as the
// GitOps page offers.
func groupTarget(targets []string, project string) (string, error) {
	invalid := func(message string) error {
		return &toolError{
			Code:       "INVALID_TARGETS",
			Message:    message,
			Suggestion: `send one database group, such as "` + project + `/databaseGroups/all", or one or more databases by the name call_api DatabaseService/ListDatabases gives, such as "instances/prod/databases/hr"`,
		}
	}
	if len(targets) == 0 {
		return "", invalid("targets is required: the database group or the databases GitOps deploys to")
	}
	var groups int
	for _, target := range targets {
		if groupProject, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			if common.FormatProject(groupProject) != project {
				return "", invalid(fmt.Sprintf("the database group %q is not in %s", target, project))
			}
			groups++
			continue
		}
		if _, _, _, err := common.GetDatabaseResourceName(target); err != nil {
			return "", invalid(fmt.Sprintf("the target %q is neither a database group nor a database", target))
		}
	}
	switch {
	case groups == 0:
		return "", nil
	case len(targets) > 1:
		return "", invalid("the targets name a database group with other targets, and a workflow deploys to one group or to a list of databases")
	default:
		return targets[0], nil
	}
}

// gitopsIdentity is the workload identity the files sign in as.
type gitopsIdentity struct {
	provider string
	shown    GitOpsIdentity
	// rolesUnread is why the project's IAM policy could not be read, or "".
	rolesUnread string
}

func (s *Server) readGitOpsIdentity(ctx context.Context, email, project string) (*gitopsIdentity, error) {
	identity := &v1pb.WorkloadIdentity{}
	if err := s.readAPI(ctx, v1connect.WorkloadIdentityServiceGetWorkloadIdentityProcedure,
		map[string]any{"name": common.FormatWorkloadIdentityEmail(email)}, identity); err != nil {
		return nil, apiReadError(err, "INVALID_IDENTITY", fmt.Sprintf("the workload identity %q could not be read", email),
			findIdentitySuggestion)
	}
	if identity.GetState() == v1pb.State_DELETED {
		return nil, &toolError{
			Code:       "INVALID_IDENTITY",
			Message:    fmt.Sprintf("the workload identity %q is deleted, and a deleted identity cannot sign in", email),
			Suggestion: "restore it signed in to the Bytebase console, or use another: " + findIdentitySuggestion,
		}
	}
	config := identity.GetWorkloadIdentityConfig()
	result := &gitopsIdentity{shown: GitOpsIdentity{Email: email, SubjectPattern: config.GetSubjectPattern(), Roles: []string{}}}
	switch config.GetProviderType() {
	case v1pb.WorkloadIdentityConfig_GITHUB:
		result.provider, result.shown.Provider = workflowProviderGitHub, "GitHub Actions"
	case v1pb.WorkloadIdentityConfig_GITLAB:
		result.provider, result.shown.Provider = workflowProviderGitLab, "GitLab CI"
	default:
		return nil, &toolError{
			Code:       "UNSUPPORTED_PROVIDER",
			Message:    fmt.Sprintf("the workload identity %q is for %s, and Bytebase has workflow files for GitHub Actions and GitLab CI", email, config.GetProviderType()),
			Suggestion: "use a workload identity for GitHub Actions or GitLab CI, or create one with administer_workspace",
		}
	}

	policy, why := s.readIamPolicy(ctx, project)
	if why != "" {
		result.rolesUnread = why
		return result, nil
	}
	member := common.WorkloadIdentityBindingPrefix + email
	now := time.Now()
	for _, binding := range policy.GetBindings() {
		if !slices.Contains(binding.GetMembers(), member) {
			continue
		}
		// An expired binding grants nothing, as the IAM manager reads it.
		if holds, err := common.EvalBindingCondition(binding.GetCondition().GetExpression(), now); err == nil && !holds {
			continue
		}
		if title := roleTitle(binding.GetRole()); !slices.Contains(result.shown.Roles, title) {
			result.shown.Roles = append(result.shown.Roles, title)
		}
	}
	return result, nil
}

func (s *Server) readGroupDatabases(ctx context.Context, group string) ([]string, error) {
	databaseGroup := &v1pb.DatabaseGroup{}
	if err := s.readAPI(ctx, v1connect.DatabaseGroupServiceGetDatabaseGroupProcedure,
		map[string]any{"name": group, "view": v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL.String()}, databaseGroup); err != nil {
		return nil, apiReadError(err, "INVALID_TARGETS", fmt.Sprintf("the database group %q could not be read", group),
			"list the project's database groups with call_api DatabaseGroupService/ListDatabaseGroups")
	}
	names := make([]string, 0, len(databaseGroup.GetMatchedDatabases()))
	for _, database := range databaseGroup.GetMatchedDatabases() {
		names = append(names, database.GetName())
	}
	return names, nil
}

// readTargetDatabases reads the databases under the project, which refuses a
// database of another project.
func (s *Server) readTargetDatabases(ctx context.Context, project string, names []string) ([]*v1pb.Database, error) {
	if len(names) == 0 {
		return nil, nil
	}
	response := &v1pb.BatchGetDatabasesResponse{}
	if err := s.readAPI(ctx, v1connect.DatabaseServiceBatchGetDatabasesProcedure,
		map[string]any{"parent": project, "names": names}, response); err != nil {
		return nil, apiReadError(err, "INVALID_TARGETS", fmt.Sprintf("the databases could not be read in %s", project),
			"list the project's databases with call_api DatabaseService/ListDatabases")
	}
	return response.GetDatabases(), nil
}

// readEnvironmentOrder returns the workspace's environments in order, as
// resource names. Every workspace stores its environments when it is created,
// so one without the setting has none, as store.GetEnvironment reads it.
func (s *Server) readEnvironmentOrder(ctx context.Context) ([]string, error) {
	setting := &v1pb.Setting{}
	err := s.readAPI(ctx, v1connect.SettingServiceGetSettingProcedure,
		map[string]any{"name": common.SettingNamePrefix + v1pb.Setting_ENVIRONMENT.String()}, setting)
	var refused *apiRefusal
	switch {
	case errors.As(err, &refused) && refused.status == http.StatusNotFound:
		return nil, nil
	case err != nil:
		return nil, err
	}
	var order []string
	for _, environment := range setting.GetValue().GetEnvironment().GetEnvironments() {
		order = append(order, common.FormatEnvironment(environment.GetId()))
	}
	return order, nil
}

// apiReadError turns a read the API refused into a tool error the agent can act
// on, and passes on any other failure.
func apiReadError(err error, code, message, suggestion string) error {
	var refused *apiRefusal
	switch {
	case !errors.As(err, &refused) || refused.status >= http.StatusInternalServerError:
		return errors.Wrap(err, message)
	// As checkAPIResponse: the server's reason says what is missing.
	case refused.status == http.StatusUnauthorized || refused.status == http.StatusForbidden:
		return &toolError{Code: "PERMISSION_DENIED", Message: message + ": " + refused.reason}
	}
	return &toolError{Code: code, Message: message + ": " + refused.reason, Suggestion: suggestion}
}

// deployStages groups databases the way the rollout does: environments in the
// workspace order, an environment missing from that order first, and databases
// with no environment left out.
func deployStages(databases []*v1pb.Database, environmentOrder []string) ([]GitOpsDeployStage, *GitOpsDatabases) {
	byEnvironment := map[string]*GitOpsDatabases{}
	var unknown []string
	skipped := &GitOpsDatabases{}
	add := func(list *GitOpsDatabases, name string) {
		list.DatabaseCount++
		if len(list.Databases) < namedDatabases {
			list.Databases = append(list.Databases, name)
		}
	}
	for _, database := range databases {
		environment := database.GetEffectiveEnvironment()
		if environment == "" {
			add(skipped, database.GetName())
			continue
		}
		if byEnvironment[environment] == nil {
			byEnvironment[environment] = &GitOpsDatabases{}
			if !slices.Contains(environmentOrder, environment) {
				unknown = append(unknown, environment)
			}
		}
		add(byEnvironment[environment], database.GetName())
	}
	var stages []GitOpsDeployStage
	for _, environment := range append(unknown, environmentOrder...) {
		if list := byEnvironment[environment]; list != nil {
			stages = append(stages, GitOpsDeployStage{
				Environment:     strings.TrimPrefix(environment, common.EnvironmentNamePrefix),
				GitOpsDatabases: *list,
			})
		}
	}
	if skipped.DatabaseCount == 0 {
		skipped = nil
	}
	return stages, skipped
}

// gitopsMessage is what the result says besides the files: what they deploy
// to, who signs in, and what the agent does next.
func gitopsMessage(output *GitOpsWorkflowOutput, identity *gitopsIdentity, projectID, folder, membersLink string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "GitOps workflow files for project %q, for %s.\n", projectID, identity.shown.Provider)

	order := make([]string, 0, len(output.DeployOrder))
	for _, stage := range output.DeployOrder {
		order = append(order, fmt.Sprintf("%s (%s)", stage.Environment, databaseCount(stage.DatabaseCount)))
	}
	fmt.Fprintf(&sb, "Deploy order: %s.\n", strings.Join(order, ", then "))
	if output.Skipped != nil {
		fmt.Fprintf(&sb, "Not deployed, because they have no environment: %s. Assign an environment in the Bytebase console to include them.\n",
			namedList(output.Skipped))
	}

	fmt.Fprintf(&sb, "The jobs sign in as %s, whose subject pattern is %q: check that it names this repository.\n",
		identity.shown.Email, identity.shown.SubjectPattern)
	switch {
	case identity.rolesUnread != "":
		fmt.Fprintf(&sb, "Its roles in the project could not be read (%s). Without a role its jobs are refused.\n", identity.rolesUnread)
	case len(identity.shown.Roles) == 0:
		fmt.Fprintf(&sb, "No role in project %q is bound to it. Unless it holds a role through the workspace, its jobs are refused: "+
			"grant it the %s role, signed in to the Bytebase console: %s\n", projectID, identityRoleTitle, membersLink)
	default:
		fmt.Fprintf(&sb, "Its roles in the project: %s.\n", strings.Join(identity.shown.Roles, ", "))
	}
	if slices.Contains(identity.shown.Roles, identityRoleTitle) {
		fmt.Fprintf(&sb, "With the %s role its rollouts carry no issue, so no approval rule holds them: tell the owner that merging deploys.\n", identityRoleTitle)
	}

	gitlab := identity.provider == workflowProviderGitLab
	request := "pull request"
	if gitlab {
		request = "merge request"
	}
	sb.WriteString("\nNext steps:\n")
	sb.WriteString("1. Write each file at its path in the repository. Ask the owner before you replace a file that exists.\n")
	step := 2
	if gitlab {
		fmt.Fprintf(&sb, "%d. Add the include line to the project's .gitlab-ci.yml, and create that file if it is missing.\n", step)
		step++
	}
	fmt.Fprintf(&sb, "%d. Open a %s into the default branch. The repository needs no secret: the jobs sign in with their own ID token.\n", step, request)
	fmt.Fprintf(&sb, "%d. The workflows run only when a file in %s/ changes, so this %s runs neither, and the first %s with a migration file is the test. "+
		"File names start with a version number, and the workflow skips a file without one.\n", step+1, folder, request, request)
	sb.WriteString("\nYou may fit the files to the repository. Keep BYTEBASE_URL, BYTEBASE_WORKLOAD_IDENTITY, the job permissions, the ID token settings, the action image, " +
		"--project, --targets and the order of the deploy jobs. The triggers alone decide which branch deploys: " +
		"tell the owner before a change that lets another branch deploy.")
	return sb.String()
}

func databaseCount(n int) string {
	if n == 1 {
		return "1 database"
	}
	return fmt.Sprintf("%d databases", n)
}

// namedList names the databases a list holds, and counts the ones it does not.
func namedList(list *GitOpsDatabases) string {
	named := strings.Join(list.Databases, ", ")
	if more := list.DatabaseCount - len(list.Databases); more > 0 {
		return fmt.Sprintf("%s and %d more", named, more)
	}
	return named
}
