package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const (
	gitopsEmail  = "deploy-ci@shop.workload.bytebase.com"
	gitopsMember = "workloadIdentity:" + gitopsEmail
)

// gitopsMock is the internal API behind get_gitops_workflow: the identity, a
// database group, the databases, the environments and the project's IAM policy.
type gitopsMock struct {
	mu sync.Mutex
	// identity, when nil, is not found.
	identity     *v1pb.WorkloadIdentity
	group        *v1pb.DatabaseGroup
	databases    map[string]*v1pb.Database
	environments []string
	policy       *v1pb.IamPolicy
	// policyStatus, when set, is the status the policy read answers with.
	policyStatus int
	// environmentStatus, when set, is the status the environment read answers with.
	environmentStatus int
	// batchParents are the parents the database reads named.
	batchParents []string
}

func gitopsDatabase(name, environment string) *v1pb.Database {
	database := &v1pb.Database{Name: name, Project: "projects/shop"}
	if environment != "" {
		database.EffectiveEnvironment = proto.String("environments/" + environment)
	}
	return database
}

func newGitOpsMock() *gitopsMock {
	databases := map[string]*v1pb.Database{}
	for _, database := range []*v1pb.Database{
		gitopsDatabase("instances/pg-test/databases/shop_test", "test"),
		gitopsDatabase("instances/pg-prod/databases/shop_prod", "prod"),
		gitopsDatabase("instances/pg-prod/databases/shop_eu", "prod"),
		gitopsDatabase("instances/pg-new/databases/shop_new", ""),
		gitopsDatabase("projects/shop/instances/sample/databases/shop_sample", "test"),
	} {
		databases[database.GetName()] = database
	}
	return &gitopsMock{
		identity: &v1pb.WorkloadIdentity{
			Name:  common.FormatWorkloadIdentityEmail(gitopsEmail),
			Email: gitopsEmail,
			WorkloadIdentityConfig: &v1pb.WorkloadIdentityConfig{
				ProviderType:   v1pb.WorkloadIdentityConfig_GITHUB,
				SubjectPattern: "repo:example-org/shop:*",
			},
		},
		databases:    databases,
		environments: []string{"test", "prod"},
		policy: &v1pb.IamPolicy{Bindings: []*v1pb.Binding{
			{Role: "roles/projectOwner", Members: []string{"user:alice@example.com"}},
			{Role: "roles/gitopsServiceAgent", Members: []string{gitopsMember}},
		}},
	}
}

func (m *gitopsMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.Lock()
	defer m.mu.Unlock()
	refuse := func(status int, code, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": message})
	}
	answer := func(message proto.Message) {
		body, _ := protojson.Marshal(message)
		_, _ = w.Write(body)
	}
	var body struct {
		Parent string   `json:"parent"`
		Names  []string `json:"names"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch r.URL.Path {
	case v1connect.WorkloadIdentityServiceGetWorkloadIdentityProcedure:
		if m.identity == nil {
			refuse(http.StatusNotFound, "not_found", "workload identity not found")
			return
		}
		answer(m.identity)
	case v1connect.DatabaseGroupServiceGetDatabaseGroupProcedure:
		if m.group == nil {
			refuse(http.StatusNotFound, "not_found", "database group not found")
			return
		}
		answer(m.group)
	case v1connect.DatabaseServiceBatchGetDatabasesProcedure:
		m.batchParents = append(m.batchParents, body.Parent)
		response := &v1pb.BatchGetDatabasesResponse{}
		for _, name := range body.Names {
			database, ok := m.databases[name]
			if !ok {
				refuse(http.StatusNotFound, "not_found", fmt.Sprintf("database %q not found", name))
				return
			}
			response.Databases = append(response.Databases, database)
		}
		answer(response)
	case v1connect.SettingServiceGetSettingProcedure:
		if m.environmentStatus != 0 {
			refuse(m.environmentStatus, "permission_denied", "permission denied: bb.settings.getEnvironment")
			return
		}
		environments := &v1pb.EnvironmentSetting{}
		for _, id := range m.environments {
			environments.Environments = append(environments.Environments, &v1pb.EnvironmentSetting_Environment{Id: id})
		}
		answer(&v1pb.Setting{Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Environment{Environment: environments}}})
	case v1connect.ProjectServiceGetIamPolicyProcedure:
		if m.policyStatus != 0 {
			refuse(m.policyStatus, "permission_denied", "permission denied")
			return
		}
		answer(m.policy)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// gitopsProfile is a self-hosted server at 3.24.0, whose External URL ends
// with a slash the files must not carry.
func gitopsProfile() *config.Profile {
	return &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com/", Version: "3.24.0"}
}

func newGitOpsServer(t *testing.T, mock *gitopsMock, profile *config.Profile) (*Server, context.Context) {
	t.Helper()
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	return s, withWorkspaceID(context.Background(), "ws-test")
}

func gitopsInput(edit func(*GitOpsWorkflowInput)) GitOpsWorkflowInput {
	input := GitOpsWorkflowInput{
		Project: "shop",
		Targets: []string{
			"instances/pg-prod/databases/shop_prod",
			"instances/pg-test/databases/shop_test",
			"instances/pg-prod/databases/shop_eu",
		},
		WorkloadIdentity: gitopsEmail,
	}
	if edit != nil {
		edit(&input)
	}
	return input
}

func TestGitOpsWorkflowReturnsTheFilesTheGitOpsPageBuilds(t *testing.T) {
	mock := newGitOpsMock()
	s, ctx := newGitOpsServer(t, mock, gitopsProfile())
	input := gitopsInput(nil)

	output, err := s.gitopsWorkflow(ctx, input)
	require.NoError(t, err)
	require.Equal(t, generateWorkflowFiles(workflowParams{
		Provider:              workflowProviderGitHub,
		BytebaseURL:           "https://bb.example.com",
		ActionImage:           "bytebase/bytebase-action:3.24.0",
		WorkloadIdentityEmail: gitopsEmail,
		ProjectID:             "shop",
		Targets:               input.Targets,
		Environments:          []string{"test", "prod"},
		Folder:                "migrations",
	}), output.Files)
	require.Empty(t, output.Include)
	require.Equal(t, []GitOpsDeployStage{
		{Environment: "test", GitOpsDatabases: GitOpsDatabases{DatabaseCount: 1, Databases: []string{"instances/pg-test/databases/shop_test"}}},
		{Environment: "prod", GitOpsDatabases: GitOpsDatabases{DatabaseCount: 2, Databases: []string{
			"instances/pg-prod/databases/shop_prod", "instances/pg-prod/databases/shop_eu",
		}}},
	}, output.DeployOrder)
	require.Nil(t, output.Skipped)
	require.Equal(t, GitOpsIdentity{
		Email:          gitopsEmail,
		Provider:       "GitHub Actions",
		SubjectPattern: "repo:example-org/shop:*",
		Roles:          []string{"GitOps Service Agent"},
	}, output.Identity)
	require.Equal(t, strings.Join([]string{
		`GitOps workflow files for project "shop", for GitHub Actions.`,
		`Deploy order: test (1 database), then prod (2 databases).`,
		`The jobs sign in as ` + gitopsEmail + `, whose subject pattern is "repo:example-org/shop:*": check that it names this repository.`,
		`Its roles in the project: GitOps Service Agent.`,
		`With the GitOps Service Agent role its rollouts carry no issue, so no approval rule holds them: tell the owner that merging deploys.`,
		``,
		`Next steps:`,
		`1. Write each file at its path in the repository. Ask the owner before you replace a file that exists.`,
		`2. Open a pull request into the default branch. The repository needs no secret: the jobs sign in with their own ID token.`,
		`3. The workflows run only when a file in migrations/ changes, so this pull request runs neither, and the first pull request with a migration file is the test. File names start with a version number, and the workflow skips a file without one.`,
		``,
		`You may fit the files to the repository. Keep BYTEBASE_URL, BYTEBASE_WORKLOAD_IDENTITY, the job permissions, the ID token settings, the action image, --project, --targets and the order of the deploy jobs. The triggers alone decide which branch deploys: tell the owner before a change that lets another branch deploy.`,
	}, "\n"), output.Message)
	require.Equal(t, []string{"projects/shop"}, mock.batchParents, "the databases are read under the project, which refuses another project's")
}

func TestGitOpsWorkflowExpandsAGroupAndKeepsItAsTheTarget(t *testing.T) {
	mock := newGitOpsMock()
	mock.identity.WorkloadIdentityConfig.ProviderType = v1pb.WorkloadIdentityConfig_GITLAB
	mock.identity.WorkloadIdentityConfig.SubjectPattern = "project_path:example-group/shop:*"
	mock.group = &v1pb.DatabaseGroup{Name: "projects/shop/databaseGroups/all-shops", MatchedDatabases: []*v1pb.DatabaseGroup_Database{
		{Name: "instances/pg-prod/databases/shop_prod"},
		{Name: "instances/pg-new/databases/shop_new"},
	}}
	s, ctx := newGitOpsServer(t, mock, gitopsProfile())

	output, err := s.gitopsWorkflow(ctx, gitopsInput(func(input *GitOpsWorkflowInput) {
		input.Project = "projects/shop"
		input.Targets = []string{"projects/shop/databaseGroups/all-shops"}
		input.Folder = "db/migrations"
	}))
	require.NoError(t, err)
	require.Len(t, output.Files, 1)
	require.Equal(t, gitlabWorkflowPath, output.Files[0].Path)
	require.Contains(t, output.Files[0].Content, "BYTEBASE_TARGETS: projects/shop/databaseGroups/all-shops\n")
	require.Contains(t, output.Files[0].Content, `BYTEBASE_FILE_PATTERN: "db/migrations/*.sql"`)
	require.Equal(t, "include:\n  - local: bytebase.gitlab-ci.yml", output.Include)
	require.Equal(t, []GitOpsDeployStage{
		{Environment: "prod", GitOpsDatabases: GitOpsDatabases{DatabaseCount: 1, Databases: []string{"instances/pg-prod/databases/shop_prod"}}},
	}, output.DeployOrder)
	require.Equal(t, &GitOpsDatabases{DatabaseCount: 1, Databases: []string{"instances/pg-new/databases/shop_new"}}, output.Skipped)
	for _, line := range []string{
		`GitOps workflow files for project "shop", for GitLab CI.`,
		`Not deployed, because they have no environment: instances/pg-new/databases/shop_new. Assign an environment in the Bytebase console to include them.`,
		`2. Add the include line to the project's .gitlab-ci.yml, and create that file if it is missing.`,
		`3. Open a merge request into the default branch.`,
		`4. The workflows run only when a file in db/migrations/ changes, so this merge request runs neither, and the first merge request with a migration file is the test.`,
	} {
		require.Contains(t, output.Message, line)
	}
}

// A database on a project's own instance is named under the project.
func TestGitOpsWorkflowTakesADatabaseOfAProjectInstance(t *testing.T) {
	s, ctx := newGitOpsServer(t, newGitOpsMock(), gitopsProfile())
	target := "projects/shop/instances/sample/databases/shop_sample"
	output, err := s.gitopsWorkflow(ctx, gitopsInput(func(input *GitOpsWorkflowInput) { input.Targets = []string{target} }))
	require.NoError(t, err)
	require.Contains(t, output.Files[0].Content, "--targets="+target+" \\\n")
	require.Equal(t, []GitOpsDeployStage{
		{Environment: "test", GitOpsDatabases: GitOpsDatabases{DatabaseCount: 1, Databases: []string{target}}},
	}, output.DeployOrder)
}

func TestDeployStagesFollowTheRollout(t *testing.T) {
	var databases []*v1pb.Database
	for i := range 12 {
		databases = append(databases, gitopsDatabase(fmt.Sprintf("instances/pg/databases/new_%d", i), ""))
	}
	databases = append(databases,
		gitopsDatabase("instances/pg/databases/prod", "prod"),
		gitopsDatabase("instances/pg/databases/canary", "canary"),
		gitopsDatabase("instances/pg/databases/test", "test"),
	)

	stages, skipped := deployStages(databases, []string{"environments/test", "environments/staging", "environments/prod"})
	var order []string
	for _, stage := range stages {
		order = append(order, stage.Environment)
	}
	require.Equal(t, []string{"canary", "test", "prod"}, order, "an environment missing from the workspace order comes first")
	require.Equal(t, 12, skipped.DatabaseCount)
	require.Len(t, skipped.Databases, namedDatabases)
	require.True(t, strings.HasSuffix(namedList(skipped), "instances/pg/databases/new_9 and 2 more"), namedList(skipped))
}

func TestGitOpsWorkflowRefuses(t *testing.T) {
	for _, row := range []struct {
		name     string
		input    func(*GitOpsWorkflowInput)
		mock     func(*gitopsMock)
		profile  func(*config.Profile)
		code     string
		contains string
	}{
		{name: "no project", input: func(i *GitOpsWorkflowInput) { i.Project = "" }, code: "INVALID_PROJECT", contains: `the request's project is ""`},
		{name: "every project", input: func(i *GitOpsWorkflowInput) { i.Project = "projects/-" }, code: "INVALID_PROJECT", contains: `"projects/-"`},
		{name: "no identity", input: func(i *GitOpsWorkflowInput) { i.WorkloadIdentity = " " }, code: "INVALID_IDENTITY", contains: "workloadIdentity is required"},
		{name: "a folder outside the repository", input: func(i *GitOpsWorkflowInput) { i.Folder = "../migrations" }, code: "INVALID_FOLDER", contains: `"../migrations"`},
		{name: "no targets", input: func(i *GitOpsWorkflowInput) { i.Targets = nil }, code: "INVALID_TARGETS", contains: "targets is required"},
		{
			name:  "a group with a database",
			input: func(i *GitOpsWorkflowInput) { i.Targets = append(i.Targets, "projects/shop/databaseGroups/all-shops") },
			code:  "INVALID_TARGETS", contains: "one group or to a list of databases",
		},
		{
			name:  "another project's group",
			input: func(i *GitOpsWorkflowInput) { i.Targets = []string{"projects/hr/databaseGroups/all"} },
			code:  "INVALID_TARGETS", contains: `"projects/hr/databaseGroups/all" is not in projects/shop`,
		},
		{
			name:  "a target that is neither",
			input: func(i *GitOpsWorkflowInput) { i.Targets = []string{"shop_prod"} },
			code:  "INVALID_TARGETS", contains: `"shop_prod" is neither a database group nor a database`,
		},
		{
			name:    "no External URL",
			profile: func(p *config.Profile) { p.ExternalURL = "" },
			code:    "EXTERNAL_URL_NOT_SET", contains: "no External URL",
		},
		{
			name: "an identity that does not exist",
			mock: func(m *gitopsMock) { m.identity = nil },
			code: "INVALID_IDENTITY", contains: "could not be read: HTTP 404: workload identity not found",
		},
		{
			name: "a deleted identity",
			mock: func(m *gitopsMock) { m.identity.State = v1pb.State_DELETED },
			code: "INVALID_IDENTITY", contains: "is deleted, and a deleted identity cannot sign in",
		},
		{
			name: "an identity for another OIDC provider",
			mock: func(m *gitopsMock) { m.identity.WorkloadIdentityConfig.ProviderType = v1pb.WorkloadIdentityConfig_OIDC },
			code: "UNSUPPORTED_PROVIDER", contains: "is for OIDC",
		},
		{
			name:  "a group that does not exist",
			input: func(i *GitOpsWorkflowInput) { i.Targets = []string{"projects/shop/databaseGroups/gone"} },
			code:  "INVALID_TARGETS", contains: `"projects/shop/databaseGroups/gone" could not be read: HTTP 404`,
		},
		{
			name:  "a database the project does not hold",
			input: func(i *GitOpsWorkflowInput) { i.Targets = []string{"instances/pg-prod/databases/hr"} },
			code:  "INVALID_TARGETS", contains: `could not be read in projects/shop: HTTP 404: database "instances/pg-prod/databases/hr" not found`,
		},
		{
			name:  "databases with no environment alone",
			input: func(i *GitOpsWorkflowInput) { i.Targets = []string{"instances/pg-new/databases/shop_new"} },
			code:  "NO_DEPLOY_TARGET", contains: "none of the picked databases has an environment",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			mock, profile := newGitOpsMock(), gitopsProfile()
			if row.mock != nil {
				row.mock(mock)
			}
			if row.profile != nil {
				row.profile(profile)
			}
			s, ctx := newGitOpsServer(t, mock, profile)
			_, err := s.gitopsWorkflow(ctx, gitopsInput(row.input))
			var refused *toolError
			require.ErrorAs(t, err, &refused)
			require.Equal(t, row.code, refused.Code)
			require.Contains(t, refused.Message, row.contains)
			require.NotEmpty(t, refused.Suggestion)
		})
	}
}

func TestGitOpsWorkflowNeedsTheEnvironmentOrder(t *testing.T) {
	t.Run("a refused read is a permission refusal", func(t *testing.T) {
		mock := newGitOpsMock()
		mock.environmentStatus = http.StatusForbidden
		s, ctx := newGitOpsServer(t, mock, gitopsProfile())
		_, err := s.gitopsWorkflow(ctx, gitopsInput(nil))
		var refused *toolError
		require.ErrorAs(t, err, &refused)
		require.Equal(t, "PERMISSION_DENIED", refused.Code)
		require.Equal(t, "the workspace's environments, which set the deploy order, could not be read: HTTP 403: permission denied: bb.settings.getEnvironment", refused.Message)
	})
	t.Run("a workspace with no setting has no order", func(t *testing.T) {
		mock := newGitOpsMock()
		mock.environmentStatus = http.StatusNotFound
		s, ctx := newGitOpsServer(t, mock, gitopsProfile())
		output, err := s.gitopsWorkflow(ctx, gitopsInput(nil))
		require.NoError(t, err)
		require.Len(t, output.DeployOrder, 2)
	})
}

func TestGitOpsWorkflowReadsTheExternalURLFromTheWorkspace(t *testing.T) {
	stores := newTestServerStore()
	stores.workspaceProfile = &storepb.WorkspaceProfileSetting{ExternalUrl: "https://bytebase.example.com"}
	profile := gitopsProfile()
	profile.ExternalURL = ""
	profile.Version = "cloud-20261023"
	s, err := newServerWithStore(stores, profile, revalidationSecret, newGitOpsMock())
	require.NoError(t, err)

	output, err := s.gitopsWorkflow(withWorkspaceID(context.Background(), "ws-test"), gitopsInput(nil))
	require.NoError(t, err)
	require.Contains(t, output.Files[0].Content, `BYTEBASE_URL: "https://bytebase.example.com"`)
	require.Contains(t, output.Files[0].Content, "image: bytebase/bytebase-action:cloud\n")
}

func TestGitOpsWorkflowSaysWhenTheIdentityShowsNoRole(t *testing.T) {
	t.Run("no binding names it", func(t *testing.T) {
		mock := newGitOpsMock()
		mock.policy.Bindings = mock.policy.Bindings[:1]
		s, ctx := newGitOpsServer(t, mock, gitopsProfile())
		output, err := s.gitopsWorkflow(ctx, gitopsInput(nil))
		require.NoError(t, err)
		require.Empty(t, output.Identity.Roles)
		require.NotContains(t, output.Message, "no approval rule holds them")
		require.Contains(t, output.Message, `No role in project "shop" is bound to it. Unless it holds a role through the workspace, its jobs are refused: `+
			`grant it the GitOps Service Agent role, signed in to the Bytebase console: https://bb.example.com/projects/shop/members`)
	})
	t.Run("its only binding has expired", func(t *testing.T) {
		mock := newGitOpsMock()
		mock.policy.Bindings[1].Condition = &expr.Expr{Expression: `request.time < timestamp("2020-01-01T00:00:00Z")`}
		s, ctx := newGitOpsServer(t, mock, gitopsProfile())
		output, err := s.gitopsWorkflow(ctx, gitopsInput(nil))
		require.NoError(t, err)
		require.Empty(t, output.Identity.Roles)
		require.Contains(t, output.Message, `No role in project "shop" is bound to it.`)
		require.NotContains(t, output.Message, "no approval rule holds them")
	})
	t.Run("a binding that has not expired counts", func(t *testing.T) {
		mock := newGitOpsMock()
		mock.policy.Bindings[1].Condition = &expr.Expr{Expression: `request.time < timestamp("2999-01-01T00:00:00Z")`}
		s, ctx := newGitOpsServer(t, mock, gitopsProfile())
		output, err := s.gitopsWorkflow(ctx, gitopsInput(nil))
		require.NoError(t, err)
		require.Equal(t, []string{"GitOps Service Agent"}, output.Identity.Roles)
	})
	t.Run("the policy cannot be read", func(t *testing.T) {
		mock := newGitOpsMock()
		mock.policyStatus = http.StatusForbidden
		s, ctx := newGitOpsServer(t, mock, gitopsProfile())
		output, err := s.gitopsWorkflow(ctx, gitopsInput(nil))
		require.NoError(t, err)
		require.Contains(t, output.Message, "Its roles in the project could not be read (HTTP 403: permission denied). Without a role its jobs are refused.")
	})
}

// The text carries the files for a client that shows text alone, and the
// structured result carries them for one that hands the model that alone.
func TestGetGitOpsWorkflowAnswersInTextAndStructure(t *testing.T) {
	// The token's audience is the External URL as mintMCPToken writes it.
	profile := gitopsProfile()
	profile.ExternalURL = "https://bb.example.com"
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, newGitOpsMock())
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	session := connectLegacy(t, ts.URL+mcpResourcePath, mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	arguments, err := json.Marshal(gitopsInput(nil))
	require.NoError(t, err)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_gitops_workflow", Arguments: json.RawMessage(arguments)})
	require.NoError(t, err)
	require.False(t, res.IsError, resultText(t, res))

	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var output GitOpsWorkflowOutput
	require.NoError(t, json.Unmarshal(raw, &output))
	require.Len(t, output.Files, 2)
	text := resultText(t, res)
	require.True(t, strings.HasPrefix(text, output.Message))
	for _, file := range output.Files {
		require.Contains(t, text, "\n\n=== "+file.Path+" ===\n"+file.Content)
	}

	refused, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_gitops_workflow", Arguments: map[string]any{
		"project": "shop", "targets": []string{"shop_prod"}, "workloadIdentity": gitopsEmail,
	}})
	require.NoError(t, err)
	require.True(t, refused.IsError)
	require.Contains(t, resultText(t, refused), `"code": "INVALID_TARGETS"`)
}

func TestGitOpsWorkflowDescriptionFitsClaudeCode(t *testing.T) {
	require.LessOrEqual(t, utf8.RuneCountInString(gitopsWorkflowDescription), 4096)
}
