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

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const (
	administerCreateInstance = "InstanceService/CreateInstance"
	journeyOneInstance       = "projects/hr/instances/hr-staging"
	journeyOnePassword       = "stagingpass"
	callerRefused            = "MCP sessions in this workspace are read-only. Ask a workspace admin to raise the MCP ceiling"
	// cloudHost is a 60-character cloud host name, as a managed PostgreSQL
	// service names one.
	cloudHost = "staging-db1.cluster-abcdefghijkl.us-east-1.rds.amazonaws.com"
)

// journeyOneBody is journey 1's request as an agent writes it from search_api's
// schema: camelCase, enums by name, the port as a string.
const journeyOneBody = `{
  "parent": "projects/hr",
  "instanceId": "hr-staging",
  "instance": {
    "title": "HR Staging",
    "engine": "POSTGRES",
    "environment": "environments/test",
    "activation": true,
    "dataSources": [{
      "id": "9f8e7d6c-5b4a-4c3d-8e2f-1a0b9c8d7e6f",
      "type": "ADMIN",
      "host": "localhost",
      "port": "5433",
      "username": "postgres",
      "password": "stagingpass"
    }]
  }
}`

// instanceMock is the internal API behind the instance entry: the environments
// the tool reads, and the connection test and the create, each recorded with its
// confirmation.
type instanceMock struct {
	mu sync.Mutex
	// environmentStatus, when set, is the HTTP status the environment read
	// answers with.
	environmentStatus int
	environments      []string
	// connectionFails makes the connection test fail as a refused sign-in does.
	connectionFails bool
	// testRefusal and createRefusal, when set, are the HTTP status the test or
	// the create answers with: a 401 or 403 refuses the caller, as the gate does,
	// and another status refuses the request, as the handler does.
	testRefusal   int
	createRefusal int
	// inactive makes the create answer with an instance that is not activated
	// in effect, and unsynced with one whose instance sync failed.
	inactive  bool
	unsynced  bool
	creations []approvalCall
	// maskingSkips is what the license says of an instance that is not
	// activated: whether masking would skip it.
	maskingSkips bool
	// createFails, when set, is how the create fails after a passing test.
	createFails createFailure
	// release ends a create that runs past its caller's deadline, as a sync
	// that has not yet noticed the cancellation.
	release  chan struct{}
	creating bool
	// stored is the instance a read finds: the one the create stored, or one
	// that held the ID before. readStatus, when set, is the status a read
	// answers with instead.
	stored     *v1pb.Instance
	readStatus int
	// syncTimeWritten sets last_sync_time on what the create stores, as
	// SyncInstance writes it before it registers the databases.
	syncTimeWritten   bool
	reads             int
	readWhileCreating bool
}

// createFailure is how a create fails after its connection test passed: with a
// 5xx, or by running past the caller's deadline, before or after it stores the
// instance.
type createFailure int

const (
	createFails500BeforeStore createFailure = iota + 1
	createFails500AfterStore
	createTimesOutBeforeStore
	createTimesOutAfterStore
)

func newInstanceMock() *instanceMock {
	return &instanceMock{environments: []string{"test", "prod"}, maskingSkips: true, release: make(chan struct{})}
}

// storedInstance is the instance a create stores for body, before any sync, as
// a read returns it: the request's instance, active, with no password.
func storedInstance(body map[string]any) *v1pb.Instance {
	encoded, _ := json.Marshal(body)
	request := &v1pb.CreateInstanceRequest{}
	_ = common.ProtojsonUnmarshaler.Unmarshal(encoded, request)
	instance := request.GetInstance()
	instance.Name, instance.State = journeyOneInstance, v1pb.State_ACTIVE
	for _, ds := range instance.GetDataSources() {
		ds.Password = ""
	}
	return instance
}

// storedFor is what the create stores for body.
func (m *instanceMock) storedFor(body map[string]any) *v1pb.Instance {
	stored := storedInstance(body)
	if m.syncTimeWritten {
		stored.LastSyncTime = timestamppb.New(time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC))
	}
	return stored
}

// runPastTheDeadline keeps the create running, without the lock, until the test
// releases it.
func (m *instanceMock) runPastTheDeadline() {
	m.creating = true
	m.mu.Unlock()
	<-m.release
	m.mu.Lock()
	m.creating = false
}

func (m *instanceMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	m.mu.Lock()
	defer m.mu.Unlock()
	refuse := func(status int, code, message string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": message})
	}
	switch r.URL.Path {
	case v1connect.SettingServiceGetSettingProcedure:
		if m.environmentStatus != 0 {
			refuse(m.environmentStatus, "denied", "setting read refused")
			return
		}
		environments := &v1pb.EnvironmentSetting{}
		for _, id := range m.environments {
			environments.Environments = append(environments.Environments, &v1pb.EnvironmentSetting_Environment{Id: id, Title: strings.ToUpper(id[:1]) + id[1:]})
		}
		body, _ := protojson.Marshal(&v1pb.Setting{Name: "settings/ENVIRONMENT", Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Environment{Environment: environments}}})
		_, _ = w.Write(body)
	case v1connect.InstanceServiceCreateInstanceProcedure:
		call := approvalCall{}
		_ = json.NewDecoder(r.Body).Decode(&call.body)
		if confirmed, ok := auth.MCPConfirmedDecisionFromContext(r.Context()); ok {
			call.confirmed = &confirmed
		}
		m.creations = append(m.creations, call)
		if validateOnly, ok := call.body["validateOnly"].(bool); ok && validateOnly {
			switch {
			case m.connectionFails:
				refuse(http.StatusBadRequest, "invalid_argument", `failed to connect to database: password authentication failed for user "postgres"`)
			case m.testRefusal == http.StatusUnauthorized || m.testRefusal == http.StatusForbidden:
				refuse(m.testRefusal, "permission_denied", callerRefused)
			case m.testRefusal != 0:
				refuse(m.testRefusal, "not_found", `project "projects/hr" not found`)
			default:
				_, _ = w.Write([]byte(`{"name":"` + journeyOneInstance + `"}`))
			}
			return
		}
		if m.createRefusal != 0 {
			refuse(m.createRefusal, "already_exists", `instance ID "hr-staging" already exists`)
			return
		}
		switch m.createFails {
		case createFails500BeforeStore:
			refuse(http.StatusInternalServerError, "internal", "failed to count the instances")
			return
		case createTimesOutBeforeStore:
			m.runPastTheDeadline()
			return
		case createFails500AfterStore:
			m.stored = m.storedFor(call.body)
			refuse(http.StatusInternalServerError, "internal", "the instance sync panicked")
			return
		case createTimesOutAfterStore:
			m.stored = m.storedFor(call.body)
			m.runPastTheDeadline()
			return
		}
		created := &v1pb.Instance{Name: journeyOneInstance, Title: "HR Staging", Engine: v1pb.Engine_POSTGRES, Activation: !m.inactive}
		if !m.unsynced {
			created.LastSyncTime = timestamppb.New(time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC))
		}
		body, _ := protojson.Marshal(created)
		_, _ = w.Write(body)
	case v1connect.InstanceServiceGetInstanceProcedure:
		m.reads++
		m.readWhileCreating = m.readWhileCreating || m.creating
		var read struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&read)
		switch {
		case m.readStatus != 0:
			refuse(m.readStatus, "internal", "the instance read failed")
		case m.stored == nil || read.Name != journeyOneInstance:
			refuse(http.StatusNotFound, "not_found", fmt.Sprintf("instance %q not found", read.Name))
		default:
			body, _ := protojson.Marshal(m.stored)
			_, _ = w.Write(body)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *instanceMock) readCount() (reads int, whileCreating bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reads, m.readWhileCreating
}

func (m *instanceMock) calls() []approvalCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]approvalCall(nil), m.creations...)
}

func connectToInstanceServer(t *testing.T, mock *instanceMock, opts *mcp.ClientOptions) (*Server, *mcp.ClientSession) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, mock)
	require.NoError(t, err)
	s.maskingSkipsUnactivated = func(context.Context, string) bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return mock.maskingSkips
	}
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	token := mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
	return s, connectLegacy(t, ts.URL+mcpResourcePath, token, opts)
}

// instanceBody is journey 1's request with edit applied to its instance and
// data source, as decoded JSON.
func instanceBody(t *testing.T, edit func(request, instance, dataSource map[string]any)) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(journeyOneBody), &body))
	if edit != nil {
		instance, ok := body["instance"].(map[string]any)
		require.True(t, ok)
		dataSources, ok := instance["dataSources"].([]any)
		require.True(t, ok)
		dataSource, ok := dataSources[0].(map[string]any)
		require.True(t, ok)
		edit(body, instance, dataSource)
	}
	return body
}

func instanceParams(body map[string]any) *mcp.CallToolParams {
	return &mcp.CallToolParams{Arguments: map[string]any{"operationId": administerCreateInstance, "body": body}}
}

// bodyDataSource is the request body's one data source.
func bodyDataSource(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	instance, ok := body["instance"].(map[string]any)
	require.True(t, ok)
	dataSources, ok := instance["dataSources"].([]any)
	require.True(t, ok)
	require.Len(t, dataSources, 1)
	dataSource, ok := dataSources[0].(map[string]any)
	require.True(t, ok)
	return dataSource
}

// signedInstanceState is the request state the instance entry signs when it
// asks about body, as alice, at askedAt.
func signedInstanceState(t *testing.T, s *Server, body map[string]any, askedAt time.Time) string {
	t.Helper()
	change, err := parseInstanceCreation(body, "ws-test")
	require.NoError(t, err)
	alice := withOAuth2ClientID(withUserEmail(withWorkspaceID(context.Background(), "ws-test"), "alice@example.com"), "client-A")
	form, err := instanceForm(alice, change, s.maskingSkipsUnactivated(alice, "ws-test"), "")
	require.NoError(t, err)
	return s.signConfirmationState(alice, change, adminFormDigest(form, change.request), askedAt)
}

func TestAdministerInstanceAddsJourneyOnesServer(t *testing.T) {
	mock := newInstanceMock()
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	body := instanceBody(t, nil)

	res := callAdminister(t, session, instanceParams(body))
	require.False(t, res.IsError, resultText(t, res))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Add activated instance "hr-staging" to "hr", as alice@example.com?`,
		`Host "localhost", port 5433, user "postgres", TLS not required`,
		`Environment "test". Its databases join the project for good.`,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n") {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, "PostgreSQL instance \"hr-staging\", titled \"HR Staging\".\n"+
		"After you apply, Bytebase tests the connection and adds nothing if it fails. "+
		`It signs in as "postgres" with the password in the request; this form does not show it.`, choiceDescription(t, asked))
	schema, err := json.Marshal(asked.RequestedSchema)
	require.NoError(t, err)
	require.NotContains(t, asked.Message+string(schema), journeyOnePassword, "the form never shows the password")

	confirmed := &auth.MCPConfirmedDecision{Procedure: v1connect.InstanceServiceCreateInstanceProcedure, Resource: journeyOneInstance}
	calls := mock.calls()
	require.Len(t, calls, 2, "a connection test, then the create")
	require.Equal(t, true, calls[0].body["validateOnly"])
	require.Equal(t, confirmed, calls[0].confirmed)
	sent := bodyDataSource(t, calls[1].body)
	require.Equal(t, bodyDataSource(t, calls[0].body)["id"], sent["id"], "the test and the create name the same data source")
	id, ok := sent["id"].(string)
	require.True(t, ok)
	_, err = uuid.Parse(id)
	require.NoError(t, err, "the tool names the data source with a new UUID")
	require.NotEqual(t, bodyDataSource(t, body)["id"], id)
	sent["id"] = bodyDataSource(t, body)["id"]
	require.Equal(t, body, calls[1].body, "apart from that ID, the create is the request the user confirmed")
	require.Equal(t, confirmed, calls[1].confirmed)

	text := "Added: instance \"hr-staging\" in project \"hr\", environment \"test\". Bytebase synced its databases into the project.\n" +
		"Instance: https://bb.example.com/" + journeyOneInstance
	require.Equal(t, text, resultText(t, res))
	output, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	require.NotContains(t, string(output), journeyOnePassword)
	require.JSONEq(t, `{"operationId":"InstanceService/CreateInstance","changed":true,
		"message":`+jsonString(t, text)+`,"link":"https://bb.example.com/`+journeyOneInstance+`"}`, string(output))
}

func jsonString(t *testing.T, s string) string {
	t.Helper()
	encoded, err := json.Marshal(s)
	require.NoError(t, err)
	return string(encoded)
}

// resultToolError reads the tool error a result carries, which formatToolError
// writes as JSON.
func resultToolError(t *testing.T, res *mcp.CallToolResult) *toolError {
	t.Helper()
	require.True(t, res.IsError, resultText(t, res))
	te := &toolError{}
	require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), te), resultText(t, res))
	return te
}

func TestAdministerInstanceFitsACloudHostInFourLines(t *testing.T) {
	require.Len(t, cloudHost, 60)
	mock := newInstanceMock()
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	callAdminister(t, session, instanceParams(instanceBody(t, func(_, _, dataSource map[string]any) {
		dataSource["host"] = cloudHost
		dataSource["port"] = "5432"
		dataSource["useSsl"] = true
		dataSource["verifyTlsCertificate"] = true
	})))
	asked := user.lastParams(t)
	require.Equal(t, strings.Join([]string{
		`Add activated instance "hr-staging" to "hr", as alice@example.com?`,
		`Host "` + cloudHost + `",`,
		`> port 5432, user "postgres", TLS required and verified`,
		`Environment "test". Its databases join the project for good.`,
	}, "\n"), asked.Message)
	for _, line := range strings.Split(asked.Message, "\n") {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
}

func TestAdministerInstanceFitsLongerNamesInFourLines(t *testing.T) {
	mock := newInstanceMock()
	mock.environments = append(mock.environments, "staging")
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})

	callAdminister(t, session, instanceParams(instanceBody(t, func(request, instance, dataSource map[string]any) {
		request["parent"] = "projects/data-platform"
		request["instanceId"] = "analytics-staging-pg"
		instance["environment"] = "environments/staging"
		dataSource["host"] = "10.0.12.34"
		dataSource["port"] = "5432"
		dataSource["username"] = "bytebase_admin"
		dataSource["useSsl"] = true
		dataSource["verifyTlsCertificate"] = true
	})))
	lines := strings.Split(user.lastParams(t).Message, "\n")
	require.Len(t, lines, messageLines)
	require.Equal(t, `Add activated instance "analytics-staging-pg" to "data-platform", as alice@example.com?`, lines[0],
		"only the question may run past a line, and only by the email")
	for _, line := range lines[1:] {
		require.LessOrEqual(t, lineColumns(line), messageLineWidth, line)
	}
	require.Equal(t, `Environment "staging". Its databases join the project for good.`, lines[3])
}

func TestAdministerInstanceStatesTheTLSMode(t *testing.T) {
	for name, tc := range map[string]struct {
		useSSL, verify bool
		want           string
	}{
		"verified":                       {true, true, "TLS required and verified"},
		"required only":                  {true, false, "TLS required, not verified"},
		"not required":                   {false, false, "TLS not required"},
		"verification without TLS stays": {false, true, "TLS not required"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
			_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			callAdminister(t, session, instanceParams(instanceBody(t, func(_, _, dataSource map[string]any) {
				dataSource["host"] = "db"
				dataSource["useSsl"] = tc.useSSL
				dataSource["verifyTlsCertificate"] = tc.verify
			})))
			lines := strings.Split(user.lastParams(t).Message, "\n")
			require.Equal(t, `Host "db", port 5433, user "postgres", `+tc.want, lines[1])
		})
	}
}

func TestAdministerInstanceKeepsTheMaskingWarningInTheMessage(t *testing.T) {
	notActivated := func(_, instance, _ map[string]any) { instance["activation"] = false }

	t.Run("journey 1's server, not activated", func(t *testing.T) {
		mock := newInstanceMock()
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		callAdminister(t, session, instanceParams(instanceBody(t, notActivated)))
		asked := user.lastParams(t)
		lines := strings.Split(asked.Message, "\n")
		require.Len(t, lines, messageLines)
		require.Equal(t, instanceNotActivated, lines[3])
		require.NotContains(t, choiceDescription(t, asked), "ctivated")
	})

	t.Run("a cloud host, not activated, needs a fifth line and goes to the console", func(t *testing.T) {
		mock := newInstanceMock()
		user := &asker{}
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		res := callAdminister(t, session, instanceParams(instanceBody(t, func(request, instance, dataSource map[string]any) {
			notActivated(request, instance, dataSource)
			dataSource["host"] = cloudHost
		})))
		refusal := resultToolError(t, res)
		require.Equal(t, "CHANGE_TOO_LONG", refusal.Code)
		require.Equal(t, "this instance needs 5 lines, more than the 4 one confirmation form shows, so no instance was added", refusal.Message)
		require.Contains(t, refusal.Suggestion, "https://bb.example.com/projects/hr/instances/new")
		require.Zero(t, user.asked.Load())
		require.Empty(t, mock.calls())
	})

	t.Run("no warning where masking would not skip it, so a cloud host fits", func(t *testing.T) {
		mock := newInstanceMock()
		mock.maskingSkips = false
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		callAdminister(t, session, instanceParams(instanceBody(t, func(request, instance, dataSource map[string]any) {
			notActivated(request, instance, dataSource)
			dataSource["host"] = cloudHost
		})))
		asked := user.lastParams(t)
		require.Len(t, strings.Split(asked.Message, "\n"), messageLines)
		require.NotContains(t, asked.Message+choiceDescription(t, asked), "ctivated")
	})

	t.Run("no warning for an activated instance", func(t *testing.T) {
		mock := newInstanceMock()
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		callAdminister(t, session, instanceParams(instanceBody(t, nil)))
		require.NotContains(t, user.lastParams(t).Message, instanceNotActivated)
	})
}

func TestAdministerInstanceNamesTheActivationInTheQuestion(t *testing.T) {
	ask := func(t *testing.T, edit func(request, instance, dataSource map[string]any)) (*mcp.ElicitParams, *mcp.CallToolResult) {
		t.Helper()
		mock := newInstanceMock()
		user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
		res := callAdminister(t, session, instanceParams(instanceBody(t, edit)))
		if user.asked.Load() == 0 {
			return nil, res
		}
		return user.lastParams(t), res
	}
	notActivated := func(_, instance, _ map[string]any) { instance["activation"] = false }

	t.Run("activated", func(t *testing.T) {
		asked, _ := ask(t, nil)
		require.Equal(t, `Add activated instance "hr-staging" to "hr", as alice@example.com?`, strings.Split(asked.Message, "\n")[0])
	})

	t.Run("not activated", func(t *testing.T) {
		asked, _ := ask(t, notActivated)
		require.Equal(t, `Add instance "hr-staging" to "hr", as alice@example.com?`, strings.Split(asked.Message, "\n")[0])
	})

	// An instance ID and a project ID of 38 characters together fill the
	// activated question's 70 columns.
	ids := func(instanceID string) func(request, instance, dataSource map[string]any) {
		return func(request, _, _ map[string]any) {
			request["parent"] = "projects/data-platform"
			request["instanceId"] = instanceID
		}
	}
	fills := "analytics-staging-pg-0001"
	require.Len(t, fills+"data-platform", 38)

	t.Run("an activated question of 70 columns names both IDs", func(t *testing.T) {
		asked, res := ask(t, ids(fills))
		require.NotNil(t, asked, resultText(t, res))
		question := strings.Split(asked.Message, "\n")[0]
		require.Equal(t, `Add activated instance "analytics-staging-pg-0001" to "data-platform", as alice@example.com?`, question)
		require.Equal(t, messageLineWidth, lineColumns(strings.TrimSuffix(question, ", as alice@example.com?")+"?"))
	})

	t.Run("one column more is refused with the shorter-ID advice", func(t *testing.T) {
		asked, res := ask(t, ids(fills+"x"))
		require.Nil(t, asked)
		refusal := resultToolError(t, res)
		require.Equal(t, "CHANGE_TOO_LONG", refusal.Code)
		require.Contains(t, refusal.Suggestion, "use a shorter instance ID, or add the instance signed in to the Bytebase console")
	})

	t.Run("the same IDs fit when not activated", func(t *testing.T) {
		asked, res := ask(t, func(request, instance, dataSource map[string]any) {
			ids(fills+"x")(request, instance, dataSource)
			notActivated(request, instance, dataSource)
		})
		require.NotNil(t, asked, resultText(t, res))
		require.Equal(t, `Add instance "analytics-staging-pg-0001x" to "data-platform", as alice@example.com?`, strings.Split(asked.Message, "\n")[0])
	})
}

func TestAdministerInstanceStatesNoEnvironment(t *testing.T) {
	mock := newInstanceMock()
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	callAdminister(t, session, instanceParams(instanceBody(t, func(_, instance, _ map[string]any) {
		delete(instance, "environment")
	})))
	lines := strings.Split(user.lastParams(t).Message, "\n")
	require.Equal(t, "No environment. Its databases join the project for good.", lines[2])
}

func TestAdministerInstanceStatesTheConnectionDatabaseInTheDescription(t *testing.T) {
	mock := newInstanceMock()
	user := &asker{answer: &mcp.ElicitResult{Action: "decline"}}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	callAdminister(t, session, instanceParams(instanceBody(t, func(_, _, dataSource map[string]any) {
		dataSource["database"] = "hr_staging"
		delete(dataSource, "password")
	})))
	asked := user.lastParams(t)
	require.NotContains(t, asked.Message, "hr_staging")
	description := choiceDescription(t, asked)
	require.Contains(t, description, `It signs in as "postgres" with no password. It connects to database "hr_staging", and syncs every database on the server.`)
}

func TestAdministerInstanceRefusesBeforeAsking(t *testing.T) {
	for name, tc := range map[string]struct {
		edit func(request, instance, dataSource map[string]any)
		want string
	}{
		"no parent":                {func(request, _, _ map[string]any) { delete(request, "parent") }, "UNSUPPORTED_CHANGE"},
		"every project":            {func(request, _, _ map[string]any) { request["parent"] = "projects/-" }, "UNSUPPORTED_CHANGE"},
		"a parent below a project": {func(request, _, _ map[string]any) { request["parent"] = "projects/hr/instances" }, "UNSUPPORTED_CHANGE"},
		"the default project":      {func(request, _, _ map[string]any) { request["parent"] = "projects/default-ws-test" }, "cannot own an instance"},
		"the old default project":  {func(request, _, _ map[string]any) { request["parent"] = "projects/default" }, "cannot own an instance"},
		"another engine":           {func(_, instance, _ map[string]any) { instance["engine"] = "MYSQL" }, "adds PostgreSQL instances"},
		"no data source":           {func(_, instance, _ map[string]any) { instance["dataSources"] = []any{} }, "has 0 data sources"},
		"two data sources": {func(_, instance, dataSource map[string]any) {
			instance["dataSources"] = []any{dataSource, map[string]any{"id": "ro", "type": "READ_ONLY", "host": "localhost", "port": "5433", "username": "reader"}}
		}, "has 2 data sources"},
		"cloud IAM sign-in":       {func(_, _, dataSource map[string]any) { dataSource["authenticationType"] = "AWS_RDS_IAM" }, "signs in with AWS_RDS_IAM"},
		"an invalid instance ID":  {func(request, _, _ map[string]any) { request["instanceId"] = "HR_Staging" }, `the instance ID "HR_Staging" is not valid`},
		"no title":                {func(_, instance, _ map[string]any) { instance["title"] = "  " }, "the instance has no title"},
		"an environment ID alone": {func(_, instance, _ map[string]any) { instance["environment"] = "test" }, `the environment "test" is not an environment's name`},
		"a read-only data source": {func(_, _, dataSource map[string]any) { dataSource["type"] = "READ_ONLY" }, "the data source's type is READ_ONLY"},
		"a host list":             {func(_, _, dataSource map[string]any) { dataSource["host"] = "db1,db2" }, `the host "db1,db2" is not one host name or IP address`},
		"a trailing comma":        {func(_, _, dataSource map[string]any) { dataSource["host"] = "db," }, `the host "db," is not one host name or IP address`},
		"a Unix socket":           {func(_, _, dataSource map[string]any) { dataSource["host"] = "/var/run/postgresql" }, `the host "/var/run/postgresql" is not one host name or IP address`},
		"a Windows socket path":   {func(_, _, dataSource map[string]any) { dataSource["host"] = `C:\pg` }, `the host "C:\\pg" is not one host name or IP address`},
		"no host":                 {func(_, _, dataSource map[string]any) { dataSource["host"] = " " }, "the data source has no host"},
		"no port":                 {func(_, _, dataSource map[string]any) { delete(dataSource, "port") }, `the port "" is not a number`},
		"options after the port":  {func(_, _, dataSource map[string]any) { dataSource["port"] = "5432 host=other.example" }, "is not a number from 1 to 65535"},
		"port zero":               {func(_, _, dataSource map[string]any) { dataSource["port"] = "0" }, `the port "0" is not a number`},
		"a port past 65535":       {func(_, _, dataSource map[string]any) { dataSource["port"] = "65536" }, `the port "65536" is not a number`},
		"a signed port":           {func(_, _, dataSource map[string]any) { dataSource["port"] = "+5432" }, `the port "+5432" is not a number`},
		"a question too wide for any instance ID": {func(request, _, _ map[string]any) {
			request["parent"] = "projects/" + strings.Repeat("p", 40)
		}, "is wider than one line of the confirmation form, so no instance was added (add the instance signed in"},
		"no username":                 {func(_, _, dataSource map[string]any) { delete(dataSource, "username") }, "the data source has no username"},
		"a connection test requested": {func(request, _, _ map[string]any) { request["validateOnly"] = true }, "the request sets validate_only"},
		"the instance's name":         {func(_, instance, _ map[string]any) { instance["name"] = "instances/hr-staging" }, "the request sets instance.name"},
		"labels":                      {func(_, instance, _ map[string]any) { instance["labels"] = map[string]any{"team": "hr"} }, "the request sets instance.labels"},
		"a CA certificate":            {func(_, _, dataSource map[string]any) { dataSource["sslCa"] = "-----BEGIN CERTIFICATE-----" }, "the request sets instance.data_sources.ssl_ca"},
		"an SSH tunnel":               {func(_, _, dataSource map[string]any) { dataSource["sshHost"] = "bastion" }, "the request sets instance.data_sources.ssh_host"},
		"extra connection parameters": {func(_, _, dataSource map[string]any) {
			dataSource["extraConnectionParameters"] = map[string]any{"sslrootcert": "/etc/passwd"}
		}, "the request sets instance.data_sources.extra_connection_parameters"},
		"a question too long for one line": {func(request, _, _ map[string]any) {
			request["instanceId"] = "a" + strings.Repeat("b", 61) + "c"
		}, "is wider than one line of the confirmation form, so no instance was added (use a shorter instance ID, or add"},
		"a host too long for any form": {func(_, _, dataSource map[string]any) {
			dataSource["host"] = strings.Repeat("h", 200) + ".example.com"
		}, "CHANGE_TOO_LONG"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			user := &asker{}
			_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			res := callAdminister(t, session, instanceParams(instanceBody(t, tc.edit)))
			refusal := resultToolError(t, res).Error()
			require.Contains(t, refusal, tc.want)
			require.NotContains(t, refusal, journeyOnePassword)
			require.Zero(t, user.asked.Load(), "nothing is asked about an instance that cannot be added")
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerInstanceChecksTheEnvironment(t *testing.T) {
	for name, tc := range map[string]struct {
		setup      func(m *instanceMock)
		edit       func(request, instance, dataSource map[string]any)
		message    string
		suggestion string
	}{
		"a misspelled environment": {edit: func(_, instance, _ map[string]any) { instance["environment"] = "environments/tset" },
			message:    `the environment "environments/tset" does not exist, so no instance was added`,
			suggestion: `the workspace's environments are "environments/test", "environments/prod"; use one of them, or leave instance.environment out for no environment`},
		"a workspace with no environments": {setup: func(m *instanceMock) { m.environmentStatus = http.StatusNotFound },
			message: `the environment "environments/test" does not exist, so no instance was added`, suggestion: "the workspace has no environments"},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			if tc.setup != nil {
				tc.setup(mock)
			}
			user := &asker{}
			_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
			refusal := resultToolError(t, callAdminister(t, session, instanceParams(instanceBody(t, tc.edit))))
			require.Equal(t, tc.message, refusal.Message)
			require.Contains(t, refusal.Suggestion, tc.suggestion)
			require.Zero(t, user.asked.Load())
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerInstanceRefusesAnAnswerOverAnEnvironmentRemoved(t *testing.T) {
	mock := newInstanceMock()
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	user.whileAsked = func() {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		mock.environments = []string{"prod"}
	}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	refusal := resultToolError(t, callAdminister(t, session, instanceParams(instanceBody(t, nil))))
	require.Equal(t, `the environment "environments/test" does not exist, so no instance was added`, refusal.Message)
	require.Empty(t, mock.calls())
}

func TestAdministerInstanceSkipsTheEnvironmentCheckForARoleThatCannotRead(t *testing.T) {
	mock := newInstanceMock()
	mock.environmentStatus = http.StatusForbidden
	user := &asker{answer: confirmAnswer(confirmChoiceApply)}
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: user.handle})
	res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), user.asked.Load())
	require.Len(t, mock.calls(), 2)
}

func TestAdministerInstanceChangesNothingUnlessApplied(t *testing.T) {
	for name, tc := range map[string]struct {
		answer *mcp.ElicitResult
		want   string
	}{
		"declined":        {&mcp.ElicitResult{Action: "decline"}, "Not changed: the user declined, so no instance was added. Do not retry unless the user asks."},
		"cancelled":       {&mcp.ElicitResult{Action: "cancel"}, "Not changed: the user cancelled, so no instance was added. Do not retry unless the user asks."},
		"changed in chat": {confirmAnswer(confirmChoiceChange), "Not changed: the user wants to change this request. Ask them what to change, then call administer_workspace with the revised request."},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: tc.answer}).handle})
			res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
			require.Equal(t, tc.want, resultText(t, res))
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerInstanceChangesNothingForAnAnswerWithNoChoice(t *testing.T) {
	for name, answer := range map[string]*mcp.ElicitResult{
		"an unknown choice": confirmAnswer("skip"),
		"no choice":         {Action: "accept", Content: map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			s, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
			body := instanceBody(t, nil)
			params := instanceParams(body)
			params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: answer}
			params.RequestState = signedInstanceState(t, s, body, time.Now())
			res := callAdminister(t, session, params)
			require.Equal(t, "Not changed: the answer chose none of the form's choices, so no instance was added. Do not retry unless the user asks.", resultText(t, res))
			require.Empty(t, mock.calls())
		})
	}
}

func TestAdministerInstanceRefusesAnAnswerToAnotherRequest(t *testing.T) {
	mock := newInstanceMock()
	s, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	state := signedInstanceState(t, s, instanceBody(t, nil), time.Now())
	// The answer comes back with the host changed.
	params := instanceParams(instanceBody(t, func(_, _, dataSource map[string]any) { dataSource["host"] = "other.example.com" }))
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Contains(t, resultText(t, res), "so no instance was added")
	require.Contains(t, resultText(t, res), instanceSubject{}.reread())
	require.Empty(t, mock.calls())
}

// The form never shows the password, so only the request in the digest ties an
// answer to the password the user's agent sent.
func TestAdministerInstanceRefusesAnAnswerOverAnActivationTurnedOn(t *testing.T) {
	mock := newInstanceMock()
	s, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	state := signedInstanceState(t, s, instanceBody(t, func(_, instance, _ map[string]any) { instance["activation"] = false }), time.Now())
	params := instanceParams(instanceBody(t, nil))
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "STALE_CONFIRMATION")
	require.Empty(t, mock.calls())
}

func TestAdministerInstanceRefusesAnAnswerOverAnotherPassword(t *testing.T) {
	mock := newInstanceMock()
	s, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	state := signedInstanceState(t, s, instanceBody(t, nil), time.Now())
	params := instanceParams(instanceBody(t, func(_, _, dataSource map[string]any) { dataSource["password"] = "another-password" }))
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = state
	res := callAdminister(t, session, params)
	require.Equal(t, "STALE_CONFIRMATION", resultToolError(t, res).Code)
	require.Empty(t, mock.calls())
}

func TestAdministerInstanceRefusesAnExpiredAnswer(t *testing.T) {
	mock := newInstanceMock()
	s, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{}).handle})
	body := instanceBody(t, nil)
	params := instanceParams(body)
	params.InputResponses = mcp.InputResponseMap{confirmDecisionRequestID: confirmAnswer(confirmChoiceApply)}
	params.RequestState = signedInstanceState(t, s, body, time.Now().Add(-confirmationLifetime-time.Minute))
	res := callAdminister(t, session, params)
	require.True(t, res.IsError, resultText(t, res))
	require.Contains(t, resultText(t, res), "the user was asked more than 10 minutes ago")
	require.Empty(t, mock.calls())
}

func TestAdministerInstanceRefusesAClientThatCannotConfirm(t *testing.T) {
	mock := newInstanceMock()
	_, session := connectToInstanceServer(t, mock, nil)
	res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "CONFIRMATION_UNAVAILABLE")
	require.Contains(t, resultText(t, res), "https://bb.example.com/projects/hr/instances/new")
	require.Empty(t, mock.calls())
}

func TestAdministerInstanceAddsNothingWhenTheConnectionFails(t *testing.T) {
	mock := newInstanceMock()
	mock.connectionFails = true
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
	refusal := resultToolError(t, callAdminister(t, session, instanceParams(instanceBody(t, nil))))
	require.Equal(t, "CONNECTION_TEST_FAILED", refusal.Code)
	require.Equal(t, "Bytebase's connection test refused the request, so no instance was added: "+
		`HTTP 400: failed to connect to database: password authentication failed for user "postgres"`, refusal.Message)
	require.Contains(t, refusal.Suggestion, "correct the request with the user as the message says")
	require.NotContains(t, refusal.Error(), journeyOnePassword)
	calls := mock.calls()
	require.Len(t, calls, 1, "the create is never sent after a failed test")
	require.Equal(t, true, calls[0].body["validateOnly"])
}

func TestAdministerInstanceReportsEachRefusalAsTheHandlerStatesIt(t *testing.T) {
	t.Run("at the test", func(t *testing.T) {
		mock := newInstanceMock()
		mock.testRefusal = http.StatusNotFound
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
		res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
		require.True(t, res.IsError)
		refusal := resultToolError(t, res)
		require.Equal(t, "CONNECTION_TEST_FAILED", refusal.Code)
		require.Equal(t, `Bytebase's connection test refused the request, so no instance was added: HTTP 404: project "projects/hr" not found`, refusal.Message)
		require.Len(t, mock.calls(), 1)
	})

	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprintf("a refusal of the caller at the test, HTTP %d", status), func(t *testing.T) {
			mock := newInstanceMock()
			mock.testRefusal = status
			_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
			res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
			require.True(t, res.IsError)
			require.Equal(t, fmt.Sprintf("failed to add the instance: HTTP %d: %s", status, callerRefused), resultText(t, res))
			require.Len(t, mock.calls(), 1)
		})
	}

	t.Run("at the create", func(t *testing.T) {
		mock := newInstanceMock()
		mock.createRefusal = http.StatusConflict
		_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
		res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
		require.True(t, res.IsError)
		require.Equal(t, `failed to add the instance: HTTP 409: instance ID "hr-staging" already exists`, resultText(t, res))
		require.Len(t, mock.calls(), 2)
		reads, _ := mock.readCount()
		require.Zero(t, reads, "a refusal of the request is no uncertain outcome")
	})
}

func TestAdministerInstanceNamesTheDataSourceItself(t *testing.T) {
	mock := newInstanceMock()
	_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
	res := callAdminister(t, session, instanceParams(instanceBody(t, func(_, _, dataSource map[string]any) { delete(dataSource, "id") })))
	require.False(t, res.IsError, resultText(t, res))
	calls := mock.calls()
	require.Len(t, calls, 2)
	id, ok := bodyDataSource(t, calls[1].body)["id"].(string)
	require.True(t, ok)
	_, err := uuid.Parse(id)
	require.NoError(t, err)
}

func TestAdministerInstanceReadsBackACreateThatDidNotFinish(t *testing.T) {
	added := "Added: instance \"hr-staging\" in project \"hr\", environment \"test\". " +
		"Bytebase did not finish answering the create, so some of its databases may not be in the project yet; the instance page shows the sync.\n" +
		"Instance: https://bb.example.com/" + journeyOneInstance
	held := func(edit func(*v1pb.Instance)) *v1pb.Instance {
		instance := storedInstance(instanceBody(t, nil))
		edit(instance)
		return instance
	}
	for name, tc := range map[string]struct {
		fails           createFailure
		syncTimeWritten bool
		heldBefore      *v1pb.Instance
		readStatus      int
		// want is the result text when the instance was added, and wantError
		// the error otherwise.
		want, wantError string
	}{
		"a timeout after the store, with the create still running": {fails: createTimesOutAfterStore, want: added},
		"a 5xx after the store": {fails: createFails500AfterStore, want: added},
		"a timeout after the sync time was written, before the databases": {
			fails: createTimesOutAfterStore, syncTimeWritten: true, want: added,
		},
		"a 5xx before the store":     {fails: createFails500BeforeStore, wantError: "ADD_UNCONFIRMED"},
		"a timeout before the store": {fails: createTimesOutBeforeStore, wantError: "ADD_UNCONFIRMED"},
		"a read that fails":          {fails: createFails500AfterStore, readStatus: http.StatusInternalServerError, wantError: "ADD_UNCONFIRMED"},
		"an ID that another instance holds": {
			fails:      createFails500BeforeStore,
			heldBefore: held(func(instance *v1pb.Instance) { instance.DataSources[0].Id = "admin" }),
			wantError:  `failed to add the instance: instance ID "hr-staging" already belongs to another instance, so no instance was added`,
		},
		"an ID that a deleted instance holds": {
			fails:      createFails500BeforeStore,
			heldBefore: held(func(instance *v1pb.Instance) { instance.State = v1pb.State_DELETED }),
			wantError:  `failed to add the instance: instance ID "hr-staging" already belongs to another instance, so no instance was added`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			mock.createFails = tc.fails
			mock.stored = tc.heldBefore
			mock.syncTimeWritten = tc.syncTimeWritten
			mock.readStatus = tc.readStatus
			s, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
			s.internalAPITimeoutOverride = 200 * time.Millisecond
			// Registered last, so it runs first and ends a create still running.
			t.Cleanup(func() { close(mock.release) })

			res := callAdminister(t, session, instanceParams(instanceBody(t, nil)))
			reads, whileCreating := mock.readCount()
			require.Equal(t, 1, reads)
			if tc.fails == createTimesOutAfterStore {
				require.True(t, whileCreating, "the read must not wait for the create")
			}
			if tc.want != "" {
				require.False(t, res.IsError, resultText(t, res))
				require.Equal(t, tc.want, resultText(t, res))
				return
			}
			require.True(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), tc.wantError)
			if tc.wantError == "ADD_UNCONFIRMED" {
				refusal := resultToolError(t, res)
				require.Contains(t, refusal.Message, `could not confirm whether instance "hr-staging" was added`)
				require.Contains(t, refusal.Suggestion, "https://bb.example.com/"+journeyOneInstance)
			}
		})
	}
}

func TestAdministerInstanceReportsWhatTheCreateLeft(t *testing.T) {
	for name, tc := range map[string]struct {
		setup func(m *instanceMock)
		edit  func(request, instance, dataSource map[string]any)
		want  string
	}{
		"an instance sync that failed": {setup: func(m *instanceMock) { m.unsynced = true },
			want: `Added: instance "hr-staging" in project "hr", environment "test". Bytebase has not synced its databases yet; the instance page shows the sync.` + "\n"},
		"not activated in effect": {setup: func(m *instanceMock) { m.inactive = true },
			want: `Bytebase synced its databases into the project. It is not activated, so masking skips it.` + "\n"},
		"not activated, with no masking to skip it": {setup: func(m *instanceMock) { m.inactive = true; m.maskingSkips = false },
			want: `Bytebase synced its databases into the project.` + "\n"},
		"no environment": {edit: func(_, instance, _ map[string]any) { delete(instance, "environment") },
			want: `Added: instance "hr-staging" in project "hr". Bytebase synced`},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newInstanceMock()
			if tc.setup != nil {
				tc.setup(mock)
			}
			_, session := connectToInstanceServer(t, mock, &mcp.ClientOptions{ElicitationHandler: (&asker{answer: confirmAnswer(confirmChoiceApply)}).handle})
			res := callAdminister(t, session, instanceParams(instanceBody(t, tc.edit)))
			require.False(t, res.IsError, resultText(t, res))
			require.Contains(t, resultText(t, res), tc.want)
			require.Contains(t, resultText(t, res), "\nInstance: https://bb.example.com/"+journeyOneInstance)
		})
	}
}

func TestAdministerInstanceClassifiesEveryRequestField(t *testing.T) {
	refused := map[string]bool{"validate_only": true}
	for _, field := range []string{"name", "state", "engine_version", "external_link", "roles", "sync_interval", "sync_databases", "last_sync_time", "labels"} {
		refused["instance."+field] = true
	}
	for _, field := range []string{
		"ssl_ca", "ssl_cert", "ssl_key", "ssl_ca_path", "ssl_cert_path", "ssl_key_path",
		"ssl_ca_set", "ssl_cert_set", "ssl_key_set", "ssl_ca_path_set", "ssl_cert_path_set", "ssl_key_path_set",
		"srv", "authentication_database", "replica_set", "sid", "service_name",
		"ssh_host", "ssh_port", "ssh_user", "ssh_password", "ssh_private_key",
		"authentication_private_key", "authentication_private_key_passphrase", "external_secret",
		"cloud_sql_ip_type", "sasl_config", "additional_addresses", "direct_connection", "region", "warehouse_id",
		"master_name", "master_username", "master_password", "redis_type", "extra_connection_parameters", "project_id", "instance_id",
	} {
		refused["instance.data_sources."+field] = true
	}
	requireEveryFieldClassified(t, (&v1pb.CreateInstanceRequest{}).ProtoReflect().Descriptor(), "", instanceRequestFields, refused)
}

func TestValidPortTakesWhatTheDriverReads(t *testing.T) {
	for _, port := range []string{"5432", "05432", "1", "65535"} {
		require.True(t, validPort(port), port)
	}
	for _, port := range []string{"", "0", "65536", "+5432", "-5432", "0x1538", "5_432", " 5432", "5432 host=other.example"} {
		require.False(t, validPort(port), port)
	}
}

func TestRefusalStatesTheTransportErrorOrTheHandlersMessage(t *testing.T) {
	require.Equal(t, "context deadline exceeded", refusal(nil, context.DeadlineExceeded))
	require.Equal(t, "HTTP 403: permission denied", refusal(&apiResponse{Status: http.StatusForbidden, Body: []byte(`{"code":"permission_denied","message":"permission denied"}`)}, nil))
	require.Empty(t, refusal(&apiResponse{Status: http.StatusOK}, nil))
}

func TestConfirmAdminChangeNamesEachEntrysChange(t *testing.T) {
	for name, tc := range map[string]struct {
		subject                                adminSubject
		unchanged, notChanged, revised, reread string
	}{
		"approval rules": {approvalRules, "the approval rules stay as they are", "the approval rules were not changed", "the revised rules",
			"read the rules again with call_api SettingService/GetSetting, make the user's change to them, and call administer_workspace with the result"},
		"masking rules": {maskingRules, "the masking rules stay as they are", "the masking rules were not changed", "the revised rules",
			"read the rules again with call_api OrgPolicyService/GetPolicy, make the user's change to them, and call administer_workspace with the result"},
		"an instance": {instanceSubject{}, "no instance was added", "no instance was added", "the revised request",
			"call administer_workspace again to ask the user about the request as it is now"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.unchanged, tc.subject.unchanged())
			require.Equal(t, tc.notChanged, tc.subject.notChanged())
			require.Equal(t, tc.revised, tc.subject.revised())
			require.Equal(t, tc.reread, tc.subject.reread())
		})
	}
}

type licenseFacts struct {
	masking, unified bool
}

func (l licenseFacts) IsFeatureEnabled(context.Context, string, v1pb.PlanFeature) error {
	if !l.masking {
		return errors.New("feature not in plan")
	}
	return nil
}

func (l licenseFacts) IsUnifiedInstanceLicense(context.Context, string) bool { return l.unified }

func TestMaskingSkipsAnUnactivatedInstanceOnlyWithMaskingAndASeatLicense(t *testing.T) {
	for _, tc := range []struct {
		license licenseFacts
		want    bool
	}{
		{licenseFacts{masking: true}, true},
		{licenseFacts{masking: true, unified: true}, false},
		{licenseFacts{}, false},
		{licenseFacts{unified: true}, false},
	} {
		require.Equal(t, tc.want, maskingSkipsUnactivatedUnder(tc.license)(context.Background(), "ws-test"), "%+v", tc.license)
	}
}
