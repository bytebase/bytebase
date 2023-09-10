// Package tests provides integration tests for backend services.
package tests

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Import pg driver.
	// init() in pgx/v4/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	componentConfig "github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/server"
	"github.com/bytebase/bytebase/backend/tests/fake"
)

var (
	migrationStatement = `
	CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);`
	migrationStatement2 = `
	CREATE TABLE book2 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);`
	migrationStatement3 = `
	CREATE TABLE book3 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);`
	migrationStatement4 = `
	CREATE TABLE book4 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);`

	//go:embed test-data/book_schema.result
	wantBookSchema string

	//go:embed test-data/book_3_schema.result
	want3BookSchema string

	dataUpdateStatementWrong = "INSERT INTO book(name) xxx"
	dataUpdateStatement      = `
	INSERT INTO book(name) VALUES
		("byte"),
		(NULL);
	`
	dumpedSchema = `CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
`
	dumpedSchema2 = `CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
CREATE TABLE book2 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
`
	dumpedSchema3 = `CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
CREATE TABLE book2 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
CREATE TABLE book3 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
`
	dumpedSchema4 = `CREATE TABLE book (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
CREATE TABLE book2 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
CREATE TABLE book3 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
CREATE TABLE book4 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NULL
	);
`
	deploySchedule = &v1pb.Schedule{
		Deployments: []*v1pb.ScheduleDeployment{
			{
				Title: "Test stage",
				Spec: &v1pb.DeploymentSpec{
					LabelSelector: &v1pb.LabelSelector{
						MatchExpressions: []*v1pb.LabelSelectorRequirement{
							{
								Key:      api.EnvironmentLabelKey,
								Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
								Values:   []string{"test"},
							},
							{
								Key:      api.TenantLabelKey,
								Operator: v1pb.OperatorType_OPERATOR_TYPE_EXISTS,
							},
						},
					},
				},
			},
			{
				Title: "Prod stage",
				Spec: &v1pb.DeploymentSpec{
					LabelSelector: &v1pb.LabelSelector{
						MatchExpressions: []*v1pb.LabelSelectorRequirement{
							{
								Key:      api.EnvironmentLabelKey,
								Operator: v1pb.OperatorType_OPERATOR_TYPE_IN,
								Values:   []string{"prod"},
							},
							{
								Key:      api.TenantLabelKey,
								Operator: v1pb.OperatorType_OPERATOR_TYPE_EXISTS,
							},
						},
					},
				},
			},
		},
	}
)

type controller struct {
	server                    *server.Server
	profile                   componentConfig.Profile
	client                    *http.Client
	grpcConn                  *grpc.ClientConn
	issueServiceClient        v1pb.IssueServiceClient
	rolloutServiceClient      v1pb.RolloutServiceClient
	orgPolicyServiceClient    v1pb.OrgPolicyServiceClient
	projectServiceClient      v1pb.ProjectServiceClient
	authServiceClient         v1pb.AuthServiceClient
	settingServiceClient      v1pb.SettingServiceClient
	environmentServiceClient  v1pb.EnvironmentServiceClient
	instanceServiceClient     v1pb.InstanceServiceClient
	databaseServiceClient     v1pb.DatabaseServiceClient
	sheetServiceClient        v1pb.SheetServiceClient
	evcsClient                v1pb.ExternalVersionControlServiceClient
	sqlServiceClient          v1pb.SQLServiceClient
	subscriptionServiceClient v1pb.SubscriptionServiceClient
	actuatorServiceClient     v1pb.ActuatorServiceClient

	cookie    string
	authToken string
	project   *v1pb.Project

	vcsProvider fake.VCSProvider

	rootURL  string
	apiURL   string
	vcsURL   string
	v1APIURL string
}

type config struct {
	dataDir            string
	vcsProviderCreator fake.VCSProviderCreator
	readOnly           bool
	skipOnboardingData bool
}

var (
	mu       sync.Mutex
	nextPort = time.Now().Second()*200 + 5010

	// Shared external PG server variables.
	externalPgUser     = "bbexternal"
	externalPgPort     = time.Now().Second()*200 + 5000
	externalPgBinDir   string
	externalPgDataDir  string
	nextDatabaseNumber = 20210113

	// resourceDir is the shared resource directory.
	resourceDir string
)

// getTestPort reserves 3 ports, 1 for server, 2 for sample pg instance.
func getTestPort() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 4
	return p
}

// getTestPortForEmbeddedPg reserves 4 ports, 1 for server, 2 for sample pg instance, 1 for embedded postgres server.
func getTestPortForEmbeddedPg() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 5
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
func (ctl *controller) StartServerWithExternalPg(ctx context.Context, config *config) (context.Context, error) {
	log.GLogLevel.Set(slog.LevelDebug)
	if err := ctl.startMockServers(config.vcsProviderCreator); err != nil {
		return nil, err
	}

	pgMainURL := fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", externalPgUser, externalPgPort, "postgres", common.GetPostgresSocketDir())
	db, err := sql.Open("pgx", pgMainURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	databaseName := getTestDatabaseString()
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName)); err != nil {
		return nil, err
	}

	pgURL := fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", externalPgUser, externalPgPort, databaseName, common.GetPostgresSocketDir())
	serverPort := getTestPort()
	profile := getTestProfileWithExternalPg(config.dataDir, resourceDir, serverPort, externalPgUser, pgURL, config.skipOnboardingData)
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
	if err := ctl.initWorkspaceProfile(metaCtx); err != nil {
		return nil, err
	}
	if err := ctl.setLicense(metaCtx); err != nil {
		return nil, err
	}
	for _, environment := range []string{"test", "prod"} {
		if _, err := ctl.orgPolicyServiceClient.CreatePolicy(metaCtx, &v1pb.CreatePolicyRequest{
			Parent: fmt.Sprintf("environments/%s", environment),
			Policy: &v1pb.Policy{
				Type: v1pb.PolicyType_DEPLOYMENT_APPROVAL,
				Policy: &v1pb.Policy_DeploymentApprovalPolicy{
					DeploymentApprovalPolicy: &v1pb.DeploymentApprovalPolicy{
						DefaultStrategy: v1pb.ApprovalStrategy_MANUAL,
					},
				},
			},
		}); err != nil {
			return nil, err
		}
	}

	projectID := "test-project"
	project, err := ctl.projectServiceClient.CreateProject(metaCtx, &v1pb.CreateProjectRequest{
		Project: &v1pb.Project{
			Title: projectID,
			Key:   projectID,
		},
		ProjectId: projectID,
	})
	if err != nil {
		return nil, err
	}
	ctl.project = project

	return metaCtx, nil
}

// StartServer starts the main server with embed Postgres.
func (ctl *controller) StartServer(ctx context.Context, config *config) (context.Context, error) {
	log.GLogLevel.Set(slog.LevelDebug)
	if err := ctl.startMockServers(config.vcsProviderCreator); err != nil {
		return nil, err
	}
	serverPort := getTestPortForEmbeddedPg()
	profile := getTestProfile(config.dataDir, resourceDir, serverPort, config.readOnly)
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return nil, err
	}
	ctl.server = server
	ctl.profile = profile

	return ctl.start(ctx, serverPort)
}

func (ctl *controller) initWorkspaceProfile(ctx context.Context) error {
	_, err := ctl.settingServiceClient.SetSetting(ctx, &v1pb.SetSettingRequest{
		Setting: &v1pb.Setting{
			Name: fmt.Sprintf("settings/%s", api.SettingWorkspaceProfile),
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceProfileSettingValue{
					WorkspaceProfileSettingValue: &v1pb.WorkspaceProfileSetting{
						ExternalUrl:    ctl.profile.ExternalURL,
						DisallowSignup: false,
					},
				},
			},
		},
	})
	return err
}

// GetTestProfile will return a profile for testing.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports.
func getTestProfile(dataDir, resourceDir string, port int, readOnly bool) componentConfig.Profile {
	return componentConfig.Profile{
		Mode:                 testReleaseMode,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		GrpcPort:             port + 1,
		DatastorePort:        port + 2,
		SampleDatabasePort:   0,
		PgUser:               "bbtest",
		Readonly:             readOnly,
		DataDir:              dataDir,
		ResourceDir:          resourceDir,
		AppRunnerInterval:    1 * time.Second,
		BackupRunnerInterval: 10 * time.Second,
		BackupStorageBackend: api.BackupStorageBackendLocal,
	}
}

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func getTestProfileWithExternalPg(dataDir, resourceDir string, port int, pgUser string, pgURL string, skipOnboardingData bool) componentConfig.Profile {
	return componentConfig.Profile{
		Mode:                       testReleaseMode,
		ExternalURL:                fmt.Sprintf("http://localhost:%d", port),
		GrpcPort:                   port + 1,
		SampleDatabasePort:         0,
		PgUser:                     pgUser,
		DataDir:                    dataDir,
		ResourceDir:                resourceDir,
		AppRunnerInterval:          1 * time.Second,
		BackupRunnerInterval:       10 * time.Second,
		BackupStorageBackend:       api.BackupStorageBackendLocal,
		PgURL:                      pgURL,
		TestOnlySkipOnboardingData: skipOnboardingData,
	}
}

func (ctl *controller) startMockServers(vcsProviderCreator fake.VCSProviderCreator) error {
	// Set up VCS provider.
	vcsPort := getTestPort()
	ctl.vcsProvider = vcsProviderCreator(vcsPort)
	ctl.vcsURL = fmt.Sprintf("http://localhost:%d", vcsPort)

	errChan := make(chan error, 1)

	go func() {
		if err := ctl.vcsProvider.Run(); err != nil {
			errChan <- errors.Wrap(err, "failed to run vcsProvider server")
		}
	}()

	if err := waitForVCSStart(ctl.vcsProvider, errChan); err != nil {
		return errors.Wrap(err, "failed to wait for vcsProvider to start")
	}

	return nil
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
	ctl.client = &http.Client{}

	if err := ctl.waitForHealthz(); err != nil {
		return nil, errors.Wrap(err, "failed to wait for healthz")
	}

	// initialize grpc connection.
	grpcConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", ctl.profile.GrpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial grpc")
	}
	ctl.grpcConn = grpcConn
	ctl.issueServiceClient = v1pb.NewIssueServiceClient(ctl.grpcConn)
	ctl.rolloutServiceClient = v1pb.NewRolloutServiceClient(ctl.grpcConn)
	ctl.orgPolicyServiceClient = v1pb.NewOrgPolicyServiceClient(ctl.grpcConn)
	ctl.projectServiceClient = v1pb.NewProjectServiceClient(ctl.grpcConn)
	ctl.authServiceClient = v1pb.NewAuthServiceClient(ctl.grpcConn)
	ctl.settingServiceClient = v1pb.NewSettingServiceClient(ctl.grpcConn)
	ctl.environmentServiceClient = v1pb.NewEnvironmentServiceClient(ctl.grpcConn)
	ctl.instanceServiceClient = v1pb.NewInstanceServiceClient(ctl.grpcConn)
	ctl.databaseServiceClient = v1pb.NewDatabaseServiceClient(ctl.grpcConn)
	ctl.sheetServiceClient = v1pb.NewSheetServiceClient(ctl.grpcConn)
	ctl.evcsClient = v1pb.NewExternalVersionControlServiceClient(ctl.grpcConn)
	ctl.sqlServiceClient = v1pb.NewSQLServiceClient(ctl.grpcConn)
	ctl.subscriptionServiceClient = v1pb.NewSubscriptionServiceClient(ctl.grpcConn)
	ctl.actuatorServiceClient = v1pb.NewActuatorServiceClient(ctl.grpcConn)

	if err := ctl.signupAndLogin(ctx); err != nil {
		return nil, err
	}

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"Authorization",
		fmt.Sprintf("Bearer %s", ctl.authToken),
	)), nil
}

func waitForVCSStart(p fake.VCSProvider, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			addr := p.ListenerAddr()
			if addr != nil && strings.Contains(addr.String(), ":") {
				return nil // was started
			}
		case err := <-errChan:
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		}
	}
}

func (ctl *controller) waitForHealthz() error {
	begin := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	timer := time.NewTimer(20 * time.Second)
	defer ticker.Stop()
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			gURL := fmt.Sprintf("%s/auth/login", ctl.v1APIURL)
			req, err := http.NewRequest(http.MethodPost, gURL, nil)
			if err != nil {
				slog.Error("Fail to create a new POST request", slog.String("URL", gURL), log.BBError(err))
				continue
			}
			resp, err := ctl.client.Do(req)
			if err != nil {
				slog.Error("Fail to send a POST request", slog.String("URL", gURL), log.BBError(err))
				continue
			}
			if resp.StatusCode == http.StatusServiceUnavailable {
				continue
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
	if ctl.vcsProvider != nil {
		if err := ctl.vcsProvider.Close(); err != nil {
			e = multierr.Append(e, err)
		}
	}
	if ctl.grpcConn != nil {
		if err := ctl.grpcConn.Close(); err != nil {
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

// post sends a POST client request.
func (ctl *controller) post(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	return ctl.request("POST", url, body, nil, map[string]string{
		"Cookie": ctl.cookie,
	})
}

func (ctl *controller) request(method, fullURL string, body io.Reader, params, header map[string]string) (io.ReadCloser, error) {
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create a new %s request(%q)", method, fullURL)
	}

	for k, v := range header {
		req.Header.Set(k, v)
	}

	q := url.Values{}
	for k, v := range params {
		q.Add(k, v)
	}
	if len(q) > 0 {
		req.URL.RawQuery = q.Encode()
	}

	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to send a %s request(%q)", method, fullURL)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read http response body")
		}
		return nil, errors.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

// signupAndLogin will signup and login as user demo@example.com.
func (ctl *controller) signupAndLogin(ctx context.Context) error {
	if _, err := ctl.authServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    "demo@example.com",
			Password: "1024",
			Title:    "demo",
			UserType: v1pb.UserType_USER,
		},
	}); err != nil && !strings.Contains(err.Error(), "exist") {
		return err
	}
	resp, err := ctl.authServiceClient.Login(ctx, &v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024",
	})
	if err != nil {
		return err
	}
	ctl.authToken = resp.Token
	ctl.cookie = fmt.Sprintf("access-token=%s", ctl.authToken)
	return nil
}
