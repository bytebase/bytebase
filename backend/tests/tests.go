// Package tests provides integration tests for backend services.
package tests

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v5/stdlib"

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
	bookTableQuery      = "SELECT * FROM sqlite_schema WHERE type = 'table' AND tbl_name = 'book';"
	bookSchemaSQLResult = `[["type","name","tbl_name","rootpage","sql"],["TEXT","TEXT","TEXT","INT","TEXT"],[["table","book","book",2,"CREATE TABLE book (\n\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\t\tname TEXT NULL\n\t)"]]]`
	bookDataQuery       = `SELECT * FROM book;`
	bookDataSQLResult   = `[["id","name"],["INTEGER","TEXT"],[[1,"byte"],[2,null]]]`

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
	backupDump = "CREATE TABLE book (\n\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\t\tname TEXT NULL\n\t);\nINSERT INTO 'book' VALUES ('1', 'byte');\nINSERT INTO 'book' VALUES ('2', NULL);\n\n"

	deploymentSchedule = api.DeploymentSchedule{
		Deployments: []*api.Deployment{
			{
				Name: "Test stage",
				Spec: &api.DeploymentSpec{
					Selector: &api.LabelSelector{
						MatchExpressions: []*api.LabelSelectorRequirement{
							{
								Key:      api.EnvironmentLabelKey,
								Operator: api.InOperatorType,
								Values:   []string{"test"},
							},
							{
								Key:      api.TenantLabelKey,
								Operator: api.ExistsOperatorType,
							},
						},
					},
				},
			},
			{
				Name: "Prod stage",
				Spec: &api.DeploymentSpec{
					Selector: &api.LabelSelector{
						MatchExpressions: []*api.LabelSelectorRequirement{
							{
								Key:      api.EnvironmentLabelKey,
								Operator: api.InOperatorType,
								Values:   []string{"prod"},
							},
							{
								Key:      api.TenantLabelKey,
								Operator: api.ExistsOperatorType,
							},
						},
					},
				},
			},
		},
	}
)

type controller struct {
	server         *server.Server
	profile        componentConfig.Profile
	client         *http.Client
	cookie         string
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
}

var (
	mu       sync.Mutex
	nextPort = 1234

	// Shared external PG server variables.
	externalPgUser     = "bbexternal"
	externalPgPort     = 21113
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
func (ctl *controller) StartServerWithExternalPg(ctx context.Context, config *config) error {
	log.SetLevel(zap.DebugLevel)
	if err := ctl.startMockServers(config.vcsProviderCreator, config.feishuProverdierCreator); err != nil {
		return err
	}

	pgMainURL := fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", externalPgUser, externalPgPort, "postgres", common.GetPostgresSocketDir())
	db, err := sql.Open("pgx", pgMainURL)
	if err != nil {
		return err
	}
	defer db.Close()
	databaseName := getTestDatabaseString()
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName)); err != nil {
		return err
	}

	pgURL := fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", externalPgUser, externalPgPort, databaseName, common.GetPostgresSocketDir())
	serverPort := getTestPort()
	profile := getTestProfileWithExternalPg(config.dataDir, resourceDir, serverPort, externalPgUser, pgURL, ctl.feishuProvider.APIURL(ctl.feishuURL))
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return err
	}
	ctl.server = server
	ctl.profile = profile

	if err := ctl.start(ctx, serverPort); err != nil {
		return err
	}
	if err := ctl.Signup(); err != nil {
		return err
	}
	return ctl.Login()
}

// StartServer starts the main server with embed Postgres.
func (ctl *controller) StartServer(ctx context.Context, config *config) error {
	log.SetLevel(zap.DebugLevel)
	if err := ctl.startMockServers(config.vcsProviderCreator, config.feishuProverdierCreator); err != nil {
		return err
	}
	serverPort := getTestPortForEmbeddedPg()
	profile := getTestProfile(config.dataDir, resourceDir, serverPort, config.readOnly, ctl.feishuProvider.APIURL(ctl.feishuURL))
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return err
	}
	ctl.server = server
	ctl.profile = profile

	return ctl.start(ctx, serverPort)
}

// GetTestProfile will return a profile for testing.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports.
func getTestProfile(dataDir, resourceDir string, port int, readOnly bool, feishuAPIURL string) componentConfig.Profile {
	return componentConfig.Profile{
		Mode:                 testReleaseMode,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		GrpcPort:             port + 1,
		DatastorePort:        port + 2,
		SampleDatabasePort:   port + 3,
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
func getTestProfileWithExternalPg(dataDir, resourceDir string, port int, pgUser string, pgURL string, feishuAPIURL string) componentConfig.Profile {
	return componentConfig.Profile{
		Mode:                 testReleaseMode,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		GrpcPort:             port + 1,
		SampleDatabasePort:   port + 2,
		PgUser:               pgUser,
		DataDir:              dataDir,
		ResourceDir:          resourceDir,
		AppRunnerInterval:    1 * time.Second,
		BackupRunnerInterval: 10 * time.Second,
		BackupStorageBackend: api.BackupStorageBackendLocal,
		FeishuAPIURL:         feishuAPIURL,
		PgURL:                pgURL,
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
func (ctl *controller) start(ctx context.Context, port int) error {
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
		return errors.Wrap(err, "failed to wait for server to start")
	}

	// initialize controller clients.
	ctl.client = &http.Client{}

	if err := ctl.waitForHealthz(); err != nil {
		return errors.Wrap(err, "failed to wait for healthz")
	}

	return nil
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
	timer := time.NewTimer(5 * time.Second)
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
		e = ctl.server.Shutdown(ctx)
	}
	if ctl.vcsProvider != nil {
		if err := ctl.vcsProvider.Close(); err != nil {
			e = err
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
	req, err := http.NewRequest("GET", gURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create a new GET request(%q)", gURL)
	}
	req.Header.Set("Cookie", ctl.cookie)
	q := url.Values{}
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to send a GET request(%q)", gURL)
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

// postOpenAPI sends a openAPI POST request.
func (ctl *controller) postOpenAPI(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.v1APIURL, shortURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create a new POST request(%q)", url)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to send a POST request(%q)", url)
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

// post sends a POST client request.
func (ctl *controller) post(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create a new POST request(%q)", url)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to send a POST request(%q)", url)
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

// patch sends a PATCH client request.
func (ctl *controller) patch(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create a new PATCH request(%q)", url)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to send a PATCH request(%q)", url)
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

func (ctl *controller) delete(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create a new DELETE request(%q)", url)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to send a DELETE request(%q)", url)
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
		strings.NewReader(`{"email":"demo@example.com","password":"1024","title":"demo"}`),
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

	return nil
}

func (ctl *controller) patchSetting(settingPatch api.SettingPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &settingPatch); err != nil {
		return errors.Wrap(err, "failed to marshal settingPatch")
	}

	body, err := ctl.patch(fmt.Sprintf("/setting/%s", settingPatch.Name), buf)
	if err != nil {
		return err
	}

	setting := new(api.Setting)
	if err = jsonapi.UnmarshalPayload(body, setting); err != nil {
		return errors.Wrap(err, "fail to unmarshal setting response")
	}
	return nil
}
