// Package tests provides integration tests for backend services.
package tests

import (
	"context"
	"crypto/tls"
	"database/sql"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bytebase/bytebase/backend/common"
	component "github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/server"
)

// authInterceptor implements connect.Interceptor to add authentication headers
type authInterceptor struct {
	token string
}

func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().IsClient && a.token != "" {
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
		}
		return next(ctx, req)
	})
}

func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if a.token != "" {
			conn.RequestHeader().Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
		}
		return conn
	})
}

func (*authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, conn)
	})
}

var (
	migrationStatement1 = `
	CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);`

	//go:embed test-data/book_schema.result
	wantBookSchema string

	dataUpdateStatement = `
	INSERT INTO book(name) VALUES
		("byte"),
		(NULL);
	`
	dumpedSchema = `CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
`
)

type controller struct {
	server                       *server.Server
	profile                      *component.Profile
	client                       *http.Client
	authInterceptor              *authInterceptor
	issueServiceClient           v1connect.IssueServiceClient
	rolloutServiceClient         v1connect.RolloutServiceClient
	planServiceClient            v1connect.PlanServiceClient
	orgPolicyServiceClient       v1connect.OrgPolicyServiceClient
	reviewConfigServiceClient    v1connect.ReviewConfigServiceClient
	projectServiceClient         v1connect.ProjectServiceClient
	databaseGroupServiceClient   v1connect.DatabaseGroupServiceClient
	authServiceClient            v1connect.AuthServiceClient
	userServiceClient            v1connect.UserServiceClient
	settingServiceClient         v1connect.SettingServiceClient
	instanceServiceClient        v1connect.InstanceServiceClient
	databaseServiceClient        v1connect.DatabaseServiceClient
	databaseCatalogServiceClient v1connect.DatabaseCatalogServiceClient
	sheetServiceClient           v1connect.SheetServiceClient
	sqlServiceClient             v1connect.SQLServiceClient
	subscriptionServiceClient    v1connect.SubscriptionServiceClient
	actuatorServiceClient        v1connect.ActuatorServiceClient
	workspaceServiceClient       v1connect.WorkspaceServiceClient
	releaseServiceClient         v1connect.ReleaseServiceClient
	revisionServiceClient        v1connect.RevisionServiceClient
	groupServiceClient           v1connect.GroupServiceClient
	auditLogServiceClient        v1connect.AuditLogServiceClient

	project *v1pb.Project

	rootURL       string
	apiURL        string
	v1APIURL      string
	principalName string
}

var (
	mu       sync.Mutex
	nextPort = time.Now().Second()*200 + 5010

	externalPgHost string
	externalPgPort string

	nextDatabaseNumber = 20210113
)

func getTestPort() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 2
	return p
}

// getTestDatabaseString returns a unique database name in external pg database server.
func getTestDatabaseString() string {
	mu.Lock()
	defer mu.Unlock()
	p := nextDatabaseNumber
	nextDatabaseNumber++
	return fmt.Sprintf("bbtest%d", p)
}

// StartServerWithExternalPg starts the main server with external Postgres.
func (ctl *controller) StartServerWithExternalPg(ctx context.Context) (context.Context, error) {
	pgMainURL := fmt.Sprintf("postgresql://postgres:root-password@%s:%s/postgres", externalPgHost, externalPgPort)
	db, err := sql.Open("pgx", pgMainURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	databaseName := getTestDatabaseString()
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName)); err != nil {
		return nil, err
	}

	pgURL := fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s", externalPgHost, externalPgPort, databaseName)
	serverPort := getTestPort()
	profile := getTestProfileWithExternalPg(serverPort, pgURL)
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return nil, err
	}
	ctl.server = server
	ctl.profile = profile

	metaCtx, err := ctl.start(ctx, serverPort)
	if err != nil {
		return nil, err
	}
	if err := ctl.setLicense(metaCtx); err != nil {
		return nil, err
	}
	if err := ctl.initWorkspaceProfile(metaCtx); err != nil {
		return nil, err
	}
	for _, environment := range []string{"test", "prod"} {
		if _, err := ctl.orgPolicyServiceClient.CreatePolicy(metaCtx, connect.NewRequest(&v1pb.CreatePolicyRequest{
			Parent: fmt.Sprintf("environments/%s", environment),
			Policy: &v1pb.Policy{
				Type: v1pb.PolicyType_ROLLOUT_POLICY,
				Policy: &v1pb.Policy_RolloutPolicy{
					RolloutPolicy: &v1pb.RolloutPolicy{
						Automatic: false,
					},
				},
			},
		})); err != nil {
			return nil, err
		}
	}

	projectID := "test-project"
	resp, err := ctl.projectServiceClient.CreateProject(metaCtx, connect.NewRequest(&v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title: projectID,
		},
		ProjectId: projectID,
	}))
	if err != nil {
		return nil, err
	}
	ctl.project = resp.Msg

	return metaCtx, nil
}

func (ctl *controller) initWorkspaceProfile(ctx context.Context) error {
	// Don't set external_url in the database if it's set via profile (command-line flag),
	// as our validation will reject it. The profile value takes precedence at runtime.
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceProfile{
					WorkspaceProfile: &v1pb.WorkspaceProfileSetting{
						DisallowSignup: false,
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{
				"value.workspace_profile.disallow_signup",
			},
		},
	}))
	return err
}

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func getTestProfileWithExternalPg(port int, pgURL string) *component.Profile {
	return &component.Profile{
		Mode:        common.ReleaseModeDev,
		ExternalURL: fmt.Sprintf("http://localhost:%d", port),
		Port:        port,
		PgURL:       pgURL,
	}
}

// start only called by StartServer() and StartServerWithExternalPg().
func (ctl *controller) start(ctx context.Context, port int) (context.Context, error) {
	ctl.rootURL = fmt.Sprintf("http://localhost:%d", port)
	ctl.apiURL = fmt.Sprintf("http://localhost:%d/api", port)
	ctl.v1APIURL = fmt.Sprintf("http://localhost:%d/v1", port)

	errChan := make(chan error, 1)

	go func() {
		if err := ctl.server.Run(ctx, port); err != nil {
			errChan <- errors.Wrap(err, "failed to run main server")
		}
	}()

	// initialize controller clients.
	ctl.client = &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(_ context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	ctl.authInterceptor = &authInterceptor{}
	interceptors := connect.WithInterceptors(ctl.authInterceptor)

	baseURL := "http://localhost:" + fmt.Sprintf("%d", port)
	ctl.issueServiceClient = v1connect.NewIssueServiceClient(ctl.client, baseURL, interceptors)
	ctl.rolloutServiceClient = v1connect.NewRolloutServiceClient(ctl.client, baseURL, interceptors)
	ctl.planServiceClient = v1connect.NewPlanServiceClient(ctl.client, baseURL, interceptors)
	ctl.orgPolicyServiceClient = v1connect.NewOrgPolicyServiceClient(ctl.client, baseURL, interceptors)
	ctl.reviewConfigServiceClient = v1connect.NewReviewConfigServiceClient(ctl.client, baseURL, interceptors)
	ctl.projectServiceClient = v1connect.NewProjectServiceClient(ctl.client, baseURL, interceptors)
	ctl.databaseGroupServiceClient = v1connect.NewDatabaseGroupServiceClient(ctl.client, baseURL, interceptors)
	ctl.authServiceClient = v1connect.NewAuthServiceClient(ctl.client, baseURL, interceptors)
	ctl.userServiceClient = v1connect.NewUserServiceClient(ctl.client, baseURL, interceptors)
	ctl.settingServiceClient = v1connect.NewSettingServiceClient(ctl.client, baseURL, interceptors)
	ctl.instanceServiceClient = v1connect.NewInstanceServiceClient(ctl.client, baseURL, interceptors)
	ctl.databaseServiceClient = v1connect.NewDatabaseServiceClient(ctl.client, baseURL, interceptors)
	ctl.databaseCatalogServiceClient = v1connect.NewDatabaseCatalogServiceClient(ctl.client, baseURL, interceptors)
	ctl.sheetServiceClient = v1connect.NewSheetServiceClient(ctl.client, baseURL, interceptors)
	ctl.sqlServiceClient = v1connect.NewSQLServiceClient(ctl.client, baseURL, interceptors)
	ctl.subscriptionServiceClient = v1connect.NewSubscriptionServiceClient(ctl.client, baseURL, interceptors)
	ctl.actuatorServiceClient = v1connect.NewActuatorServiceClient(ctl.client, baseURL, interceptors)
	ctl.workspaceServiceClient = v1connect.NewWorkspaceServiceClient(ctl.client, baseURL, interceptors)
	ctl.releaseServiceClient = v1connect.NewReleaseServiceClient(ctl.client, baseURL, interceptors)
	ctl.revisionServiceClient = v1connect.NewRevisionServiceClient(ctl.client, baseURL, interceptors)
	ctl.groupServiceClient = v1connect.NewGroupServiceClient(ctl.client, baseURL, interceptors)
	ctl.auditLogServiceClient = v1connect.NewAuditLogServiceClient(ctl.client, baseURL, interceptors)

	if err := ctl.waitForHealthz(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to wait for healthz")
	}
	authToken, err := ctl.signupAndLogin(ctx)
	if err != nil {
		return nil, err
	}
	ctl.authInterceptor.token = authToken

	return ctx, nil
}

func (ctl *controller) waitForHealthz(ctx context.Context) error {
	begin := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	timer := time.NewTimer(10 * time.Second)
	defer ticker.Stop()
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			_, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, &connect.Request[v1pb.GetActuatorInfoRequest]{})
			if err != nil && status.Code(err) == codes.Unavailable {
				continue
			}
			if err != nil {
				return err
			}
			return nil
		case end := <-timer.C:
			return errors.Errorf("cannot wait for healthz in duration: %v", end.Sub(begin).Seconds())
		}
	}
}

// Close closes long running resources.
func (ctl *controller) Close(ctx context.Context) error {
	var e error
	if ctl.server != nil {
		if err := ctl.server.Shutdown(ctx); err != nil {
			e = multierr.Append(e, err)
		}
	}
	return e
}

// provisionSQLiteInstance provisions a SQLite instance (a directory).
func (*controller) provisionSQLiteInstance(rootDir, name string) (string, error) {
	p := path.Join(rootDir, name)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return "", errors.Wrapf(err, "failed to make directory %q", p)
	}

	return p, nil
}

// signupAndLogin will signup and login as user demo@example.com.
func (ctl *controller) signupAndLogin(ctx context.Context) (string, error) {
	userResp, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    "demo@example.com",
			Password: "1024bytebase",
			Title:    "demo",
			UserType: v1pb.UserType_USER,
		},
	}))
	if err != nil && !strings.Contains(err.Error(), "exist") {
		return "", err
	}
	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024bytebase",
	}))
	if err != nil {
		return "", err
	}
	ctl.principalName = userResp.Msg.Name
	return loginResp.Msg.Token, nil
}
