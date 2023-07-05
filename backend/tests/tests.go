// Package tests provides integration tests for backend services.
package tests

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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
	server                   *server.Server
	profile                  componentConfig.Profile
	client                   *http.Client
	grpcConn                 *grpc.ClientConn
	issueServiceClient       v1pb.IssueServiceClient
	orgPolicyServiceClient   v1pb.OrgPolicyServiceClient
	projectServiceClient     v1pb.ProjectServiceClient
	authServiceClient        v1pb.AuthServiceClient
	settingServiceClient     v1pb.SettingServiceClient
	environmentServiceClient v1pb.EnvironmentServiceClient
	instanceServiceClient    v1pb.InstanceServiceClient
	databaseServiceClient    v1pb.DatabaseServiceClient
	sheetServiceClient       v1pb.SheetServiceClient
	evcsClient               v1pb.ExternalVersionControlServiceClient
	sqlServiceClient         v1pb.SQLServiceClient

	cookie             string
	grpcMDAccessToken  string
	grpcMDRefreshToken string
	grpcMDUser         string

	vcsProvider    fake.VCSProvider
	feishuProvider *fake.Feishu

	rootURL   string
	apiURL    string
	vcsURL    string
	v1APIURL  string
	feishuURL string
}

type config struct {
	dataDir                 string
	vcsProviderCreator      fake.VCSProviderCreator
	feishuProverdierCreator fake.FeishuProviderCreator
	readOnly                bool
	skipOnboardingData      bool
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

// getTestPort reserves two ports, one for server, one for sample pg instance.
func getTestPort() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 3
	return p
}

// getTestPortForEmbeddedPg reserves three ports, one for server, one for sample pg instance, one for postgres server.
func getTestPortForEmbeddedPg() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 4
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
	log.SetLevel(zap.DebugLevel)
	if err := ctl.startMockServers(config.vcsProviderCreator, config.feishuProverdierCreator); err != nil {
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
	profile := getTestProfileWithExternalPg(config.dataDir, resourceDir, serverPort, externalPgUser, pgURL, ctl.feishuProvider.APIURL(ctl.feishuURL), config.skipOnboardingData)
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
	if err := ctl.setLicense(); err != nil {
		return nil, err
	}
	return metaCtx, nil
}

// StartServer starts the main server with embed Postgres.
func (ctl *controller) StartServer(ctx context.Context, config *config) (context.Context, error) {
	log.SetLevel(zap.DebugLevel)
	if err := ctl.startMockServers(config.vcsProviderCreator, config.feishuProverdierCreator); err != nil {
		return nil, err
	}
	serverPort := getTestPortForEmbeddedPg()
	profile := getTestProfile(config.dataDir, resourceDir, serverPort, config.readOnly, ctl.feishuProvider.APIURL(ctl.feishuURL))
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
func getTestProfile(dataDir, resourceDir string, port int, readOnly bool, feishuAPIURL string) componentConfig.Profile {
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
		FeishuAPIURL:         feishuAPIURL,
	}
}

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func getTestProfileWithExternalPg(dataDir, resourceDir string, port int, pgUser string, pgURL string, feishuAPIURL string, skipOnboardingData bool) componentConfig.Profile {
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
		FeishuAPIURL:               feishuAPIURL,
		PgURL:                      pgURL,
		TestOnlySkipOnboardingData: skipOnboardingData,
	}
}

func (ctl *controller) startMockServers(vcsProviderCreator fake.VCSProviderCreator, feishuProviderCreator fake.FeishuProviderCreator) error {
	// Set up VCS provider.
	vcsPort := getTestPort()
	ctl.vcsProvider = vcsProviderCreator(vcsPort)
	ctl.vcsURL = fmt.Sprintf("http://localhost:%d", vcsPort)

	// Set up fake feishu server.
	if feishuProviderCreator != nil {
		feishuPort := getTestPort()
		ctl.feishuProvider = feishuProviderCreator(feishuPort)
		ctl.feishuURL = fmt.Sprintf("http://localhost:%d", feishuPort)
	}

	errChan := make(chan error, 1)

	go func() {
		if err := ctl.vcsProvider.Run(); err != nil {
			errChan <- errors.Wrap(err, "failed to run vcsProvider server")
		}
	}()

	if feishuProviderCreator != nil {
		go func() {
			if err := ctl.feishuProvider.Run(); err != nil {
				errChan <- errors.Wrap(err, "failed to run feishuProvider server")
			}
		}()
	}

	if err := waitForVCSStart(ctl.vcsProvider, errChan); err != nil {
		return errors.Wrap(err, "failed to wait for vcsProvider to start")
	}

	if feishuProviderCreator != nil {
		if err := waitForFeishuStart(ctl.feishuProvider, errChan); err != nil {
			return errors.Wrap(err, "failed to wait for feishuProvider to start")
		}
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

	if err := waitForServerStart(ctl.server, errChan); err != nil {
		return nil, errors.Wrap(err, "failed to wait for server to start")
	}

	// initialize controller clients.
	ctl.client = &http.Client{}

	if err := ctl.waitForHealthz(); err != nil {
		return nil, errors.Wrap(err, "failed to wait for healthz")
	}

	if err := ctl.Signup(); err != nil && !strings.Contains(err.Error(), "exist") {
		return nil, err
	}
	if err := ctl.Login(); err != nil {
		return nil, err
	}

	// initialize grpc connection.
	grpcConn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", ctl.profile.GrpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial grpc")
	}
	ctl.grpcConn = grpcConn
	ctl.issueServiceClient = v1pb.NewIssueServiceClient(ctl.grpcConn)
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

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"Authorization",
		fmt.Sprintf("Bearer %s", ctl.grpcMDAccessToken),
	)), nil
}

func waitForServerStart(s *server.Server, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s == nil {
				continue
			}
			e := s.GetEcho()
			if e == nil {
				continue
			}
			addr := e.ListenerAddr()
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

func waitForFeishuStart(f *fake.Feishu, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			addr := f.ListenerAddr()
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
				log.Error("Fail to create a new POST request", zap.String("URL", gURL), zap.Error(err))
				continue
			}
			resp, err := ctl.client.Do(req)
			if err != nil {
				log.Error("Fail to send a POST request", zap.String("URL", gURL), zap.Error(err))
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

// get sends a GET client request.
func (ctl *controller) get(shortURL string, params map[string]string) (io.ReadCloser, error) {
	gURL := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	return ctl.request("GET", gURL, nil, params, map[string]string{
		"Cookie": ctl.cookie,
	})
}

// OpenAPI sends a GET OpenAPI client request.
func (ctl *controller) getOpenAPI(shortURL string, params map[string]string) (io.ReadCloser, error) {
	gURL := fmt.Sprintf("%s%s", ctl.v1APIURL, shortURL)
	return ctl.request("GET", gURL, nil, params, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", strings.ReplaceAll(ctl.cookie, "access-token=", "")),
	})
}

// postOpenAPI sends a openAPI POST request.
func (ctl *controller) postOpenAPI(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.v1APIURL, shortURL)
	return ctl.request("POST", url, body, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", strings.ReplaceAll(ctl.cookie, "access-token=", "")),
	})
}

// post sends a POST client request.
func (ctl *controller) post(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	return ctl.request("POST", url, body, nil, map[string]string{
		"Cookie": ctl.cookie,
	})
}

// patch sends a PATCH client request.
func (ctl *controller) patch(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	return ctl.request("PATCH", url, body, nil, map[string]string{
		"Cookie": ctl.cookie,
	})
}

// patchOpenAPI sends a openAPI PATCH client request.
func (ctl *controller) patchOpenAPI(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.v1APIURL, shortURL)
	return ctl.request("PATCH", url, body, nil, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", strings.ReplaceAll(ctl.cookie, "access-token=", "")),
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

// Signup will signup user demo@example.com and caches its cookie.
func (ctl *controller) Signup() error {
	resp, err := ctl.client.Post(
		fmt.Sprintf("%s/users", ctl.v1APIURL),
		"",
		strings.NewReader(`{"email":"demo@example.com","password":"1024","title":"demo","user_type":"USER"}`),
	)
	if err != nil {
		return errors.Wrap(err, "fail to post login request")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read body")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to create user with status %v body %s", resp.Status, body)
	}
	return nil
}

// Login will login as user demo@example.com and caches its cookie.
func (ctl *controller) Login() error {
	resp, err := ctl.client.Post(
		fmt.Sprintf("%s/auth/login", ctl.v1APIURL),
		"",
		strings.NewReader(`{"email":"demo@example.com","password":"1024","web": true}`))
	if err != nil {
		return errors.Wrap(err, "fail to post login request")
	}

	cookie := ""
	h := resp.Header.Get("Set-Cookie")
	parts := strings.Split(h, "; ")
	for _, p := range parts {
		if strings.HasPrefix(p, "access-token=") {
			cookie = p
			break
		}
	}
	if cookie == "" {
		return errors.Errorf("unable to find access token in the login response headers")
	}
	ctl.cookie = cookie

	ctl.grpcMDAccessToken = resp.Header.Get("grpc-metadata-bytebase-access-token")
	ctl.grpcMDRefreshToken = resp.Header.Get("grpc-metadata-bytebase-refresh-token")
	ctl.grpcMDUser = resp.Header.Get("grpc-metadata-bytebase-user")
	return nil
}
