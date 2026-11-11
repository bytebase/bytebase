package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const (
	administerCreateIdentity = "WorkloadIdentityService/CreateWorkloadIdentity"
	journeyFourEmail         = "hr-migrations-ci@hr.workload.bytebase.com"
	journeyFourMember        = "workloadIdentity:" + journeyFourEmail
	journeyFourEtag          = "etag-1"
)

// journeyFourBody is journey 4's request as an agent writes it from the
// search_api note: camelCase, the provider by name, no issuer.
const journeyFourBody = `{
  "parent": "projects/hr",
  "workloadIdentityId": "hr-migrations-ci",
  "workloadIdentity": {
    "title": "HR migrations CI",
    "workloadIdentityConfig": {
      "providerType": "GITHUB",
      "allowedAudiences": ["bytebase"],
      "subjectPattern": "repo:example-org/hr-migrations:*"
    }
  }
}`

// identityMock is the internal API behind the identity entry: the create, and
// the project's IAM policy the grant reads and writes, each write recorded with
// its confirmation.
type identityMock struct {
	mu sync.Mutex
	// createRefusal, grantRefusal and policyReadStatus, when set, are the HTTP
	// status the create, the grant or the policy read answers with.
	createRefusal    int
	grantRefusal     int
	policyReadStatus int
	policy           *v1pb.IamPolicy
	creations        []approvalCall
	grants           []approvalCall
	policyReads      int
}

func newIdentityMock() *identityMock {
	return &identityMock{policy: &v1pb.IamPolicy{
		Bindings: []*v1pb.Binding{{Role: "roles/projectOwner", Members: []string{"user:alice@example.com"}}},
		Etag:     journeyFourEtag,
	}}
}

func (m *identityMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.Lock()
	defer m.mu.Unlock()
	refuse := func(status int, code, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": message})
	}
	record := func() approvalCall {
		call := approvalCall{}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		return call
	}
	switch r.URL.Path {
	case v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure:
		m.creations = append(m.creations, record())
		if m.createRefusal != 0 {
			refuse(m.createRefusal, "already_exists", `workload identity with email "`+journeyFourEmail+`" already exists`)
			return
		}
		body, _ := protojson.Marshal(&v1pb.WorkloadIdentity{Name: common.FormatWorkloadIdentityEmail(journeyFourEmail), Email: journeyFourEmail})
		_, _ = w.Write(body)
	case v1connect.ProjectServiceGetIamPolicyProcedure:
		m.policyReads++
		if m.policyReadStatus != 0 {
			refuse(m.policyReadStatus, "permission_denied", "permission denied")
			return
		}
		body, _ := protojson.Marshal(m.policy)
		_, _ = w.Write(body)
	case v1connect.ProjectServiceSetIamPolicyProcedure:
		m.grants = append(m.grants, record())
		if m.grantRefusal != 0 {
			refuse(m.grantRefusal, "invalid_argument", "failed to validate expiration for binding roles/gitopsServiceAgent: request.time is required")
			return
		}
		_, _ = w.Write([]byte(`{}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *identityMock) calls() (creations, grants []approvalCall, policyReads int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]approvalCall(nil), m.creations...), append([]approvalCall(nil), m.grants...), m.policyReads
}

func connectToIdentityServer(t *testing.T, mock *identityMock, stores *testServerStore, opts *mcp.ClientOptions) (*Server, *mcp.ClientSession) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(stores, profile, revalidationSecret, mock)
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	return s, connectLegacy(t, ts.URL+mcpResourcePath, token, opts)
}

// identityBody is journey 4's request with edit applied to it and to its
// config, as decoded JSON.
func identityBody(t *testing.T, edit func(request, config map[string]any)) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(journeyFourBody), &body))
	if edit != nil {
		identity, ok := body["workloadIdentity"].(map[string]any)
		require.True(t, ok)
		config, ok := identity["workloadIdentityConfig"].(map[string]any)
		require.True(t, ok)
		edit(body, config)
	}
	return body
}

func identityParams(body map[string]any) *mcp.CallToolParams {
	return &mcp.CallToolParams{Arguments: map[string]any{"operationId": administerCreateIdentity, "body": body}}
}

// signedIdentityState is the request state the identity entry signs when it
// asks about body, as alice, at askedAt.
func signedIdentityState(t *testing.T, s *Server, body map[string]any, askedAt time.Time) string {
	t.Helper()
	change, err := parseIdentityCreation(body)
	require.NoError(t, err)
	require.NoError(t, change.check(""))
	change.fill()
	alice := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	form, err := identityForm(alice, change, "")
	require.NoError(t, err)
	return s.signConfirmationState(alice, change, adminFormDigest(form, change.request), askedAt)
}

func TestAdministerIdentityCreatesJourneyFoursIdentityWithItsRole(t *testing.T) {
	mock := newIdentityMock()
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: user.handle})

	res := callAdminister(t, session, identityParams(identityBody(t, nil)))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Create workload identity "hr-migrations-ci" in "hr", as alice@example.com?`,
		`Any GitHub Actions job in "example-org/hr-migrations" signs in as it,`,
		`> and the MCP access policy does not limit it.`,
		`Role: GitOps Service Agent. It can roll out without approval.`,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n")[1:] {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, strings.Join([]string{
		`The rule trusts the owner's name: if "example-org" is renamed or deleted, whoever registers the name can sign in as it.`,
		`It signs in as hr-migrations-ci@hr.workload.bytebase.com with tokens from https://token.actions.githubusercontent.com for audience "bytebase". It lasts until it is removed.`,
		`GitOps Service Agent lets it create plans, releases and rollouts in this project and run their tasks in every environment. Its rollouts carry no issue, so no approval rule holds them. Titled "HR migrations CI".`,
	}, "\n"), choiceDescription(t, asked))

	creations, grants, _ := mock.calls()
	require.Len(t, creations, 1)
	require.JSONEq(t, `{
		"parent": "projects/hr",
		"workloadIdentityId": "hr-migrations-ci",
		"workloadIdentity": {
			"title": "HR migrations CI",
			"workloadIdentityConfig": {
				"providerType": "GITHUB",
				"issuerUrl": "https://token.actions.githubusercontent.com",
				"allowedAudiences": ["bytebase"],
				"subjectPattern": "repo:example-org/hr-migrations:*"
			}
		}
	}`, jsonString(t, creations[0].body), "the tool sets the issuer")
	require.Equal(t, &auth.MCPConfirmedDecision{
		Procedure: v1connect.WorkloadIdentityServiceCreateWorkloadIdentityProcedure,
		Resource:  "workloadIdentities/" + journeyFourEmail,
	}, creations[0].confirmed)

	require.Len(t, grants, 1)
	require.JSONEq(t, `{
		"resource": "projects/hr",
		"policy": {
			"bindings": [
				{"role": "roles/projectOwner", "members": ["user:alice@example.com"]},
				{"role": "roles/gitopsServiceAgent", "members": ["`+journeyFourMember+`"]}
			],
			"etag": "`+journeyFourEtag+`"
		}
	}`, jsonString(t, grants[0].body), "the grant adds the identity and keeps every stored binding, with the etag it read")
	require.Equal(t, &auth.MCPConfirmedDecision{Procedure: v1connect.ProjectServiceSetIamPolicyProcedure, Resource: "projects/hr"}, grants[0].confirmed)

	text := `Created: workload identity "` + journeyFourEmail + `" in project "hr", with the GitOps Service Agent role there. ` +
		"A workflow signs in with that email as BYTEBASE_WORKLOAD_IDENTITY, which the GitOps page's workflows use.\n" +
		"GitOps: https://bb.example.com/projects/hr/gitops"
	require.Equal(t, text, resultText(t, res))
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.JSONEq(t, `{"operationId":"`+administerCreateIdentity+`","changed":true,
		"message":`+jsonString(t, text)+`,"link":"https://bb.example.com/projects/hr/gitops"}`, string(output))
}

func TestAdministerIdentityFitsTheOtherShapesInFourLines(t *testing.T) {
	for name, tc := range map[string]struct {
		edit func(request, config map[string]any)
		want []string
	}{
		"a longer path wraps once": {
			func(request, config map[string]any) {
				request["parent"], request["workloadIdentityId"] = "projects/data-platform", "analytics-pipeline-ci"
				config["subjectPattern"] = "repo:data-platform-org/analytics-migrations:*"
			},
			[]string{
				`Create workload identity "analytics-pipeline-ci" in "data-platform", as alice@example.com?`,
				`Any GitHub Actions job in "data-platform-org/analytics-migrations"`,
				`> signs in as it, and the MCP access policy does not limit it.`,
				`Role: GitOps Service Agent. It can roll out without approval.`,
			},
		},
		"a GitLab project in a subgroup": {
			func(_, config map[string]any) {
				config["providerType"], config["subjectPattern"] = "GITLAB", "project_path:acme/team/deploy:*"
			},
			[]string{
				`Create workload identity "hr-migrations-ci" in "hr", as alice@example.com?`,
				`Any GitLab CI job in "acme/team/deploy" signs in as it, and the MCP`,
				`> access policy does not limit it.`,
				`Role: GitOps Service Agent. It can roll out without approval.`,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			_, session := connectToIdentityServer(t, newIdentityMock(), newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: user.handle})
			callAdminister(t, session, identityParams(identityBody(t, tc.edit)))
			require.Equal(t, strings.Join(tc.want, "\n"), user.lastParams(t).Message)
		})
	}
}

func TestAdministerIdentityWarnsOfTheOwnersNameOnlyWithNeitherID(t *testing.T) {
	for pattern, firstLine := range map[string]string{
		"repo:example-org/hr-migrations:*":         `The rule trusts the owner's name: if "example-org" is renamed or deleted, whoever registers the name can sign in as it.`,
		"repo:example-org@111/hr-migrations:*":     `The pattern pins owner ID "111", so a new account under a freed name cannot sign in as it.`,
		"repo:example-org/hr-migrations@456:*":     `The pattern pins repository ID "456", so a new account under a freed name cannot sign in as it.`,
		"repo:example-org@111/hr-migrations@456:*": `The pattern pins owner ID "111" and repository ID "456", so a new account under a freed name cannot sign in as it.`,
		"project_path:example-org/hr-migrations:*": `It signs in as hr-migrations-ci@hr.workload.bytebase.com with tokens from https://gitlab.com for audience "bytebase". It lasts until it is removed.`,
	} {
		t.Run(pattern, func(t *testing.T) {
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			_, session := connectToIdentityServer(t, newIdentityMock(), newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: user.handle})
			callAdminister(t, session, identityParams(identityBody(t, func(_, config map[string]any) {
				config["subjectPattern"] = pattern
				if strings.HasPrefix(pattern, "project_path:") {
					config["providerType"] = "GITLAB"
				}
			})))
			first, _, _ := strings.Cut(choiceDescription(t, user.lastParams(t)), "\n")
			require.Equal(t, firstLine, first)
		})
	}
}

func TestReadIdentityRepositoryNamesTheRepositoryAndItsIDs(t *testing.T) {
	github := v1pb.WorkloadIdentityConfig_GITHUB
	for pattern, want := range map[string]identityRepository{
		"repo:example-org/hr-migrations:*":         {github: true, path: "example-org/hr-migrations"},
		"repo:example-org@111/hr-migrations:*":     {github: true, path: "example-org/hr-migrations", ownerID: "111"},
		"repo:example-org/hr-migrations@456:*":     {github: true, path: "example-org/hr-migrations", repoID: "456"},
		"repo:example-org@111/hr-migrations@456:*": {github: true, path: "example-org/hr-migrations", ownerID: "111", repoID: "456"},
	} {
		got, ok := readIdentityRepository(&v1pb.WorkloadIdentityConfig{ProviderType: github, SubjectPattern: pattern})
		require.True(t, ok, pattern)
		require.Equal(t, want, got, pattern)
	}
	got, ok := readIdentityRepository(&v1pb.WorkloadIdentityConfig{ProviderType: v1pb.WorkloadIdentityConfig_GITLAB, SubjectPattern: "project_path:acme/team/deploy:*"})
	require.True(t, ok)
	require.Equal(t, identityRepository{path: "acme/team/deploy"}, got)
	for provider, pattern := range map[v1pb.WorkloadIdentityConfig_ProviderType]string{
		github:                             "project_path:acme/deploy:*",
		v1pb.WorkloadIdentityConfig_GITLAB: "repo:acme/deploy:*",
		v1pb.WorkloadIdentityConfig_OIDC:   "repo:example-org/hr-migrations:*",
		v1pb.WorkloadIdentityConfig_PROVIDER_TYPE_UNSPECIFIED: "repo:example-org/hr-migrations:*",
	} {
		_, ok := readIdentityRepository(&v1pb.WorkloadIdentityConfig{ProviderType: provider, SubjectPattern: pattern})
		require.False(t, ok, "%s %q", provider, pattern)
	}
}

func TestAdministerIdentityRefusesBeforeAsking(t *testing.T) {
	for name, tc := range map[string]struct {
		edit func(request, config map[string]any)
		code string
		// words are the refusal's own, for the cases whose code another check
		// shares.
		words string
	}{
		"another provider": {func(_, config map[string]any) { config["providerType"] = "OIDC" }, "UNSUPPORTED_CHANGE", "creates identities for GitHub Actions and GitLab CI"},
		"no provider":      {func(_, config map[string]any) { delete(config, "providerType") }, "UNSUPPORTED_CHANGE", "creates identities for GitHub Actions and GitLab CI"},
		"another issuer": {func(_, config map[string]any) {
			config["issuerUrl"] = "https://ghe.example.com/_services/token"
		}, "UNSUPPORTED_CHANGE", "the issuer administer_workspace sets"},
		"GitHub's issuer on GitLab": {func(_, config map[string]any) {
			config["providerType"], config["subjectPattern"], config["issuerUrl"] = "GITLAB", "project_path:acme/deploy:*", "https://token.actions.githubusercontent.com"
		}, "UNSUPPORTED_CHANGE", "the issuer administer_workspace sets for GITLAB"},
		"a key URL": {func(_, config map[string]any) { config["jwksUrl"] = "https://keys.example.com/jwks" }, "FIELD_NOT_SUPPORTED", ""},
		"an email": {func(request, _ map[string]any) {
			request["workloadIdentity"].(map[string]any)["email"] = journeyFourEmail
		}, "FIELD_NOT_SUPPORTED", ""},
		"another audience":     {func(_, config map[string]any) { config["allowedAudiences"] = []any{"sts.amazonaws.com"} }, "UNSUPPORTED_CHANGE", "sets the audience"},
		"a second audience":    {func(_, config map[string]any) { config["allowedAudiences"] = []any{"bytebase", "sts.amazonaws.com"} }, "UNSUPPORTED_CHANGE", "sets the audience"},
		"a workspace identity": {func(request, _ map[string]any) { request["parent"] = "workspaces/ws-test" }, "UNSUPPORTED_CHANGE", "creates a workload identity in a project"},
		"every project":        {func(request, _ map[string]any) { request["parent"] = "projects/-" }, "UNSUPPORTED_CHANGE", "creates a workload identity in a project"},
		"a branch pattern": {func(_, config map[string]any) {
			config["subjectPattern"] = "repo:example-org/hr-migrations:ref:refs/heads/main"
		}, "UNSUPPORTED_CHANGE", "does not name one repository with every job in it"},
		"every repository":    {func(_, config map[string]any) { config["subjectPattern"] = "repo:example-org/*" }, "UNSUPPORTED_CHANGE", "does not name one repository with every job in it"},
		"a question too wide": {func(request, _ map[string]any) { request["workloadIdentityId"] = strings.Repeat("a", 35) }, "CHANGE_TOO_LONG", "the question naming identity"},
		"a path too long to fit": {func(_, config map[string]any) {
			config["subjectPattern"] = "repo:" + strings.Repeat("o", 20) + "/" + strings.Repeat("r", 22) + ":*"
		}, "CHANGE_TOO_LONG", "this workload identity needs 5 lines"},
		"an unknown request body": {func(request, _ map[string]any) { request["parent"] = 7 }, "INVALID_REQUEST", ""},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newIdentityMock()
			user := &asker{answer: confirmAnswer(confirmChoiceApply)}
			_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: user.handle})
			refusal := resultToolError(t, callAdminister(t, session, identityParams(identityBody(t, tc.edit))))
			require.Equal(t, tc.code, refusal.Code, refusal.Message)
			if tc.words != "" {
				require.Contains(t, refusal.Message, tc.words)
			}
			require.Zero(t, user.asked.Load(), "no form may be sent")
			creations, grants, _ := mock.calls()
			require.Empty(t, creations)
			require.Empty(t, grants)
		})
	}
}

func TestAdministerIdentityOffersAShorterIDOnlyWhenOneFits(t *testing.T) {
	for name, tc := range map[string]struct {
		parent, id string
		shorter    bool
	}{
		"an ID too long for its project":   {"projects/hr", strings.Repeat("a", 35), true},
		"a project ID that leaves no room": {"projects/" + strings.Repeat("p", 36), "a", false},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newIdentityMock()
			_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
			refusal := resultToolError(t, callAdminister(t, session, identityParams(identityBody(t, func(request, _ map[string]any) {
				request["parent"], request["workloadIdentityId"] = tc.parent, tc.id
			}))))
			require.Equal(t, "CHANGE_TOO_LONG", refusal.Code)
			require.Equal(t, tc.shorter, strings.HasPrefix(refusal.Suggestion, "use a shorter identity ID"), refusal.Suggestion)
		})
	}
}

func TestAdministerIdentityFitsTheLongestQuestionAndPath(t *testing.T) {
	for name, edit := range map[string]func(request, config map[string]any){
		"a question of 70 columns": func(request, _ map[string]any) { request["workloadIdentityId"] = strings.Repeat("a", 34) },
		"a GitHub path of 41": func(_, config map[string]any) {
			config["subjectPattern"] = "repo:" + strings.Repeat("o", 20) + "/" + strings.Repeat("r", 20) + ":*"
		},
	} {
		t.Run(name, func(t *testing.T) {
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			_, session := connectToIdentityServer(t, newIdentityMock(), newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callAdminister(t, session, identityParams(identityBody(t, edit)))
			require.False(t, res.IsError, resultText(t, res))
			require.Len(t, strings.Split(user.lastParams(t).Message, "\n"), messageLines)
		})
	}
}

func TestAdministerIdentityRefusesAWorkspaceThatLimitsRoleExpiry(t *testing.T) {
	mock := newIdentityMock()
	stores := newTestServerStore()
	stores.workspaceProfile.MaximumRoleExpiration = durationpb.New(90 * 24 * time.Hour)
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToIdentityServer(t, mock, stores, &mcp.ClientOptions{ElicitationHandler: user.handle})
	refusal := resultToolError(t, callAdminister(t, session, identityParams(identityBody(t, nil))))
	require.Equal(t, "ROLE_EXPIRY_REQUIRED", refusal.Code)
	require.Contains(t, refusal.Suggestion, "https://bb.example.com/projects/hr/workload-identities")
	require.Zero(t, user.asked.Load(), "no form may promise a role the grant cannot make")
	creations, grants, _ := mock.calls()
	require.Empty(t, creations)
	require.Empty(t, grants)
}

func TestAdministerIdentityChangesNothingUnlessApplied(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":        {&mcp.ElicitResult{Action: "decline"}, "Not changed: the user declined, so no workload identity was created. Do not retry unless the user asks."},
		"cancelled":       {&mcp.ElicitResult{Action: "cancel"}, "Not changed: the user cancelled, so no workload identity was created. Do not retry unless the user asks."},
		"changed in chat": {confirmAnswer(confirmChoiceChange), "Not changed: the user wants to change this request. Ask them what to change, then call administer_workspace with the revised request."},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newIdentityMock()
			_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{answer: tc.answer}).handle})
			res := callAdminister(t, session, identityParams(identityBody(t, nil)))
			require.Equal(t, tc.want, resultText(t, res))
			creations, grants, _ := mock.calls()
			require.Empty(t, creations)
			require.Empty(t, grants)
		})
	}
}

func TestAdministerIdentityChangesNothingForAnAnswerWithNoChoice(t *testing.T) {
	mock := newIdentityMock()
	s, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := identityBody(t, nil)
	params := identityParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: &mcp.ElicitResult{Action: "accept", Content: map[string]any{}}}
	params.RequestState = signedIdentityState(t, s, body, time.Now())
	res := callAdminister(t, session, params)
	require.Equal(t, "Not changed: the answer chose none of the form's choices, so no workload identity was created. Do not retry unless the user asks.", resultText(t, res))
	creations, _, _ := mock.calls()
	require.Empty(t, creations)
}

func TestAdministerIdentityRefusesAnAnswerToAnotherRequest(t *testing.T) {
	mock := newIdentityMock()
	s, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	state := signedIdentityState(t, s, identityBody(t, nil), time.Now())
	// The answer comes back with another repository.
	params := identityParams(identityBody(t, func(_, config map[string]any) { config["subjectPattern"] = "repo:example-org/payroll:*" }))
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = state
	refusal := resultToolError(t, callAdminister(t, session, params))
	require.Equal(t, "STALE_CONFIRMATION", refusal.Code)
	require.Contains(t, refusal.Message, "so no workload identity was created")
	require.Equal(t, creationSubject{}.reread(), refusal.Suggestion)
	creations, _, _ := mock.calls()
	require.Empty(t, creations)
}

// An answer is acted on only for the request it was asked about, so a request
// signed with another owner ID is refused, and the one asked about is created.
func TestAdministerIdentityRefusesAnAnswerOverAnotherOwnerID(t *testing.T) {
	pinned := func(id string) map[string]any {
		return identityBody(t, func(_, config map[string]any) {
			config["subjectPattern"] = "repo:example-org@" + id + "/hr-migrations:*"
		})
	}
	t.Run("another owner ID", func(t *testing.T) {
		mock := newIdentityMock()
		s, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
		params := identityParams(pinned("999"))
		params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
		params.RequestState = signedIdentityState(t, s, pinned("111"), time.Now())
		require.Equal(t, "STALE_CONFIRMATION", resultToolError(t, callAdminister(t, session, params)).Code)
		creations, _, _ := mock.calls()
		require.Empty(t, creations)
	})
	t.Run("the same request", func(t *testing.T) {
		mock := newIdentityMock()
		s, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
		params := identityParams(pinned("111"))
		params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
		params.RequestState = signedIdentityState(t, s, pinned("111"), time.Now())
		res := callAdminister(t, session, params)
		require.False(t, res.IsError, resultText(t, res))
		creations, _, _ := mock.calls()
		require.Len(t, creations, 1)
	})
}

func TestAdministerIdentityCreatesAGitLabIdentityWithGitLabsIssuer(t *testing.T) {
	mock := newIdentityMock()
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, identityParams(identityBody(t, func(_, config map[string]any) {
		config["providerType"], config["subjectPattern"] = "GITLAB", "project_path:acme/team/deploy:*"
	})))
	require.False(t, res.IsError, resultText(t, res))
	require.Contains(t, choiceDescription(t, user.lastParams(t)), "with tokens from https://gitlab.com for audience \"bytebase\".")
	creations, grants, _ := mock.calls()
	require.Len(t, creations, 1)
	require.Len(t, grants, 1)
	identity, ok := creations[0].body["workloadIdentity"].(map[string]any)
	require.True(t, ok)
	config, ok := identity["workloadIdentityConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://gitlab.com", config["issuerUrl"])
	require.Equal(t, []any{"bytebase"}, config["allowedAudiences"])
	require.Equal(t, "project_path:acme/team/deploy:*", config["subjectPattern"])
}

func TestAdministerIdentityRefusesAnExpiredAnswer(t *testing.T) {
	mock := newIdentityMock()
	s, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := identityBody(t, nil)
	params := identityParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = signedIdentityState(t, s, body, time.Now().Add(-confirmationLifetime-time.Minute))
	res := callAdminister(t, session, params)
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "the user was asked more than 10 minutes ago")
	creations, _, _ := mock.calls()
	require.Empty(t, creations)
}

func TestAdministerIdentityRefusesAClientThatCannotConfirm(t *testing.T) {
	mock := newIdentityMock()
	_, session := connectToIdentityServer(t, mock, newTestServerStore(), nil)
	refusal := resultToolError(t, callAdminister(t, session, identityParams(identityBody(t, nil))))
	require.Equal(t, "CONFIRMATION_UNAVAILABLE", refusal.Code)
	require.Contains(t, refusal.Suggestion, "https://bb.example.com/projects/hr/workload-identities")
	creations, _, _ := mock.calls()
	require.Empty(t, creations)
}

func TestAdministerIdentityGrantsOnTheBindingWithNoCondition(t *testing.T) {
	expiring := &expr.Expr{Expression: `request.time < timestamp("2027-01-01T00:00:00Z")`}
	for name, tc := range map[string]struct {
		stored, want []*v1pb.Binding
	}{
		"joins the role's binding with no condition": {
			stored: []*v1pb.Binding{{Role: identityRole, Members: []string{"user:bot@example.com"}}},
			want:   []*v1pb.Binding{{Role: identityRole, Members: []string{"user:bot@example.com", journeyFourMember}}},
		},
		"leaves a binding with a condition alone": {
			stored: []*v1pb.Binding{{Role: identityRole, Members: []string{"user:bot@example.com"}, Condition: expiring}},
			want: []*v1pb.Binding{
				{Role: identityRole, Members: []string{"user:bot@example.com"}, Condition: expiring},
				{Role: identityRole, Members: []string{journeyFourMember}},
			},
		},
		"adds no member twice": {
			stored: []*v1pb.Binding{{Role: identityRole, Members: []string{journeyFourMember}}},
			want:   []*v1pb.Binding{{Role: identityRole, Members: []string{journeyFourMember}}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			policy := &v1pb.IamPolicy{Bindings: tc.stored}
			addToRole(policy, identityRole, journeyFourMember)
			require.JSONEq(t, mustProtoJSON(t, &v1pb.IamPolicy{Bindings: tc.want}), mustProtoJSON(t, policy))
		})
	}
}

func mustProtoJSON(t *testing.T, policy *v1pb.IamPolicy) string {
	t.Helper()
	encoded, err := protojson.Marshal(policy)
	require.NoError(t, err)
	return string(encoded)
}

func TestAdministerIdentityGrantsNothingWhenTheCreateIsRefused(t *testing.T) {
	mock := newIdentityMock()
	mock.createRefusal = http.StatusConflict
	_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
	res := callAdminister(t, session, identityParams(identityBody(t, nil)))
	require.True(t, res.IsError)
	require.Equal(t, `failed to create the workload identity: HTTP 409: workload identity with email "`+journeyFourEmail+`" already exists`, resultText(t, res))
	_, grants, policyReads := mock.calls()
	require.Empty(t, grants)
	require.Zero(t, policyReads)
}

func TestAdministerIdentityReportsARoleItCouldNotGrant(t *testing.T) {
	for name, tc := range map[string]struct {
		refuse func(*identityMock)
		reason string
		grants int
	}{
		"the grant is refused": {func(m *identityMock) { m.grantRefusal = http.StatusBadRequest }, "request.time is required", 1},
		"the policy cannot be read": {func(m *identityMock) { m.policyReadStatus = http.StatusForbidden },
			"the project's IAM policy could not be read: HTTP 403: permission denied", 0},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newIdentityMock()
			tc.refuse(mock)
			_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
			res := callAdminister(t, session, identityParams(identityBody(t, nil)))
			require.False(t, res.IsError, "the identity was created, so the result is not a refusal")
			text := resultText(t, res)
			require.True(t, strings.HasPrefix(text, `Created workload identity "`+journeyFourEmail+`" in project "hr", but Bytebase did not grant it the GitOps Service Agent role: `), text)
			require.Contains(t, text, tc.reason)
			require.Contains(t, text, "It can sign in, and it does not have that role until it is granted signed in to the console.")
			require.True(t, strings.HasSuffix(text, "\nMembers: https://bb.example.com/projects/hr/members"), text)
			output, err := json.Marshal(res.StructuredContent)
			require.NoError(t, err)
			require.Contains(t, string(output), `"changed":true`)
			_, grants, _ := mock.calls()
			require.Len(t, grants, tc.grants)
		})
	}
}

func TestAdministerIdentityClassifiesEveryRequestField(t *testing.T) {
	refused := map[string]bool{
		"workload_identity.name":                              true,
		"workload_identity.state":                             true,
		"workload_identity.email":                             true,
		"workload_identity.create_time":                       true,
		"workload_identity.workload_identity_config.jwks_url": true,
	}
	requireEveryFieldClassified(t, (&v1pb.CreateWorkloadIdentityRequest{}).ProtoReflect().Descriptor(), "", identityRequestFields, refused)
}

func TestAdministerIdentitySetsTheAudienceAndATitleItWasNotSent(t *testing.T) {
	mock := newIdentityMock()
	_, session := connectToIdentityServer(t, mock, newTestServerStore(), &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
	res := callAdminister(t, session, identityParams(identityBody(t, func(request, config map[string]any) {
		delete(config, "allowedAudiences")
		request["workloadIdentity"].(map[string]any)["title"] = " "
	})))
	require.False(t, res.IsError, resultText(t, res))
	creations, _, _ := mock.calls()
	require.Len(t, creations, 1)
	identity, ok := creations[0].body["workloadIdentity"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "hr-migrations-ci", identity["title"], "a blank title takes the ID, as the console does")
	config, ok := identity["workloadIdentityConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []any{"bytebase"}, config["allowedAudiences"])
}
