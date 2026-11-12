package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

const testWorkloadIdentity = "ci@workload.bytebase.com"

// signInTestServer stands in for Bytebase. It issues a numbered token on each
// exchange and accepts only the latest one, until expire is called.
type signInTestServer struct {
	v1connect.UnimplementedAuthServiceHandler
	v1connect.UnimplementedActuatorServiceHandler

	url string

	mu sync.Mutex
	// refuse, when set, is the error every exchange returns.
	refuse           error
	exchanges        []*v1pb.ExchangeTokenRequest
	exchangeCookies  []string
	validToken       string
	actuatorRequests int
}

func newSignInTestServer(t *testing.T) *signInTestServer {
	t.Helper()

	s := &signInTestServer{}
	mux := http.NewServeMux()
	mux.Handle(v1connect.NewAuthServiceHandler(s))
	mux.Handle(v1connect.NewActuatorServiceHandler(s))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	s.url = server.URL
	return s
}

func (s *signInTestServer) ExchangeToken(_ context.Context, req *connect.Request[v1pb.ExchangeTokenRequest]) (*connect.Response[v1pb.ExchangeTokenResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.refuse != nil {
		return nil, s.refuse
	}
	s.exchanges = append(s.exchanges, req.Msg)
	s.exchangeCookies = append(s.exchangeCookies, req.Header().Get("Cookie"))
	s.validToken = fmt.Sprintf("bytebase-token-%d", len(s.exchanges))
	return connect.NewResponse(&v1pb.ExchangeTokenResponse{AccessToken: s.validToken}), nil
}

func (s *signInTestServer) GetActuatorInfo(_ context.Context, req *connect.Request[v1pb.GetActuatorInfoRequest]) (*connect.Response[v1pb.ActuatorInfo], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actuatorRequests++
	if s.validToken == "" || req.Header().Get("Authorization") != "Bearer "+s.validToken {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
	}
	return connect.NewResponse(&v1pb.ActuatorInfo{Version: "3.15.0"}), nil
}

func (s *signInTestServer) expire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validToken = ""
}

// gitHubTestIssuer stands in for GitHub's ID token endpoint. It issues a
// numbered token on each request.
type gitHubTestIssuer struct {
	env func(string) string

	mu       sync.Mutex
	requests int
	cookies  []string
}

func newGitHubTestIssuer(t *testing.T) *gitHubTestIssuer {
	t.Helper()

	issuer := &gitHubTestIssuer{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issuer.mu.Lock()
		defer issuer.mu.Unlock()
		issuer.requests++
		issuer.cookies = append(issuer.cookies, r.Header.Get("Cookie"))
		_, _ = fmt.Fprintf(w, `{"value":"id-token-%d"}`, issuer.requests)
	}))
	t.Cleanup(server.Close)
	issuer.env = envOf(map[string]string{
		"ACTIONS_ID_TOKEN_REQUEST_URL":   server.URL + "/token?api-version=2.0",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "request-token",
	})
	return issuer
}

func newWorkloadIdentityTestClient(t *testing.T, server *signInTestServer, env func(string) string, platform world.JobPlatform) *client {
	t.Helper()

	opts := defaultClientOptions()
	opts.retryConfig = &retryConfig{maxAttempts: 3, initialInterval: time.Millisecond, maxInterval: time.Millisecond}
	opts.customHeaders = http.Header{"Cookie": []string{"CF_Authorization=proxy"}}
	client, err := newClient(server.url, credential{
		workloadIdentity: testWorkloadIdentity,
		idToken:          newIDTokenSource(env, platform, "bytebase", 5*time.Second),
	}, opts)
	require.NoError(t, err)
	t.Cleanup(client.close)
	return client
}

func TestWorkloadIdentitySignIn(t *testing.T) {
	t.Parallel()

	server := newSignInTestServer(t)
	issuer := newGitHubTestIssuer(t)
	client := newWorkloadIdentityTestClient(t, server, issuer.env, world.GitHub)

	require.NoError(t, client.signIn(t.Context()))
	_, err := client.getActuatorInfo(t.Context())
	require.NoError(t, err)

	require.Len(t, server.exchanges, 1)
	require.Equal(t, "id-token-1", server.exchanges[0].Token)
	require.Equal(t, testWorkloadIdentity, server.exchanges[0].Email)
	require.Equal(t, 1, server.actuatorRequests, "the first request carries the token")
	require.Equal(t, []string{"CF_Authorization=proxy"}, server.exchangeCookies, "custom headers reach Bytebase")
	require.Equal(t, []string{""}, issuer.cookies, "custom headers do not reach GitHub")
}

func TestWorkloadIdentityRenewsOnUnauthenticated(t *testing.T) {
	t.Parallel()

	server := newSignInTestServer(t)
	issuer := newGitHubTestIssuer(t)
	client := newWorkloadIdentityTestClient(t, server, issuer.env, world.GitHub)
	require.NoError(t, client.signIn(t.Context()))

	server.expire()
	_, err := client.getActuatorInfo(t.Context())
	require.NoError(t, err)

	require.Len(t, server.exchanges, 2)
	require.Equal(t, "id-token-2", server.exchanges[1].Token, "the renewal asks GitHub for a new ID token")
}

func TestSignInErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		refuse error
		env    map[string]string
		want   []string
	}{
		{
			name:   "the server's reason is shown",
			refuse: connect.NewError(connect.CodeUnauthenticated, errors.New(`token validation failed: subject mismatch: expected pattern "repo:my-org/*", got "repo:other/app:ref:refs/heads/main"`)),
			env:    map[string]string{"BYTEBASE_ID_TOKEN": "job-id-token"},
			want:   []string{`sign in as ci@workload.bytebase.com: unauthenticated: token validation failed: subject mismatch: expected pattern "repo:my-org/*"`},
		},
		{
			name:   "an audience mismatch names the override",
			refuse: connect.NewError(connect.CodeUnauthenticated, errors.New("token validation failed: audience mismatch: token has [other], allowed [bytebase]")),
			env:    map[string]string{"BYTEBASE_ID_TOKEN": "job-id-token"},
			want:   []string{"sign in as ci@workload.bytebase.com: ", "--workload-identity-audience", "audience mismatch: token has [other], allowed [bytebase]"},
		},
		{
			name: "no ID token",
			want: []string{"sign in as ci@workload.bytebase.com: the job has no ID token: set BYTEBASE_ID_TOKEN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newSignInTestServer(t)
			server.refuse = tt.refuse
			client := newWorkloadIdentityTestClient(t, server, envOf(tt.env), world.GitLab)

			err := client.signIn(t.Context())
			var signInErr *SignInError
			require.ErrorAs(t, err, &signInErr)
			for _, want := range tt.want {
				require.Contains(t, err.Error(), want)
			}
			require.NotContains(t, err.Error(), "job-id-token", "the ID token is not in the error")
		})
	}
}

func TestRejectedAccessTokenStopsAtTheVersionCheck(t *testing.T) {
	t.Parallel()

	server := newSignInTestServer(t)
	opts := defaultClientOptions()
	opts.retryConfig = &retryConfig{maxAttempts: 3, initialInterval: time.Millisecond, maxInterval: time.Millisecond}
	client, err := newClient(server.url, credential{accessToken: "stale"}, opts)
	require.NoError(t, err)
	t.Cleanup(client.close)
	require.NoError(t, client.signIn(t.Context()))

	w := &world.World{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	err = checkVersionCompatibility(w, client, "3.15.0")
	var signInErr *SignInError
	require.ErrorAs(t, err, &signInErr)
	require.Equal(t, "sign in with the access token: unauthenticated: invalid token", err.Error())
}

// Not parallel: the command reads the ID token from the process environment.
func TestRefusedSignInEndsTheCommandBeforeAnyOtherRequest(t *testing.T) {
	t.Setenv("BYTEBASE_ID_TOKEN", "id-token")
	t.Setenv("BYTEBASE_ACCESS_TOKEN", "")

	server := newSignInTestServer(t)
	server.refuse = connect.NewError(connect.CodeNotFound, errors.Errorf("workload identity %q not found", testWorkloadIdentity))

	w := world.NewWorld()
	w.Platform = world.LocalPlatform
	cmd := NewRootCommand(w)
	var stderr bytes.Buffer
	cmd.SetOut(io.Discard)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{
		"check",
		"--url", server.url,
		"--workload-identity", testWorkloadIdentity,
		"--project", "projects/example",
		"--targets", "instances/example/databases/example",
	})

	err := cmd.ExecuteContext(t.Context())
	var signInErr *SignInError
	require.ErrorAs(t, err, &signInErr)
	require.Contains(t, stderr.String(), `Error: sign in as ci@workload.bytebase.com: not_found: workload identity "ci@workload.bytebase.com" not found`)
	require.NotContains(t, stderr.String(), "Usage:")
	require.Zero(t, server.actuatorRequests)
}
