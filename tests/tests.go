// Package tests provides integration tests for backend services.
package tests

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/jsonapi"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/server"
	componentConfig "github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/tests/fake"
)

//go:embed fake
var fakeFS embed.FS

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
				Name: "Staging stage",
				Spec: &api.DeploymentSpec{
					Selector: &api.LabelSelector{
						MatchExpressions: []*api.LabelSelectorRequirement{
							{
								Key:      api.EnvironmentKeyName,
								Operator: api.InOperatorType,
								Values:   []string{"Staging"},
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
								Key:      api.EnvironmentKeyName,
								Operator: api.InOperatorType,
								Values:   []string{"Prod"},
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

	rootURL    string
	apiURL     string
	vcsURL     string
	openAPIURL string
	feishuURL  string
}

type config struct {
	dataDir                 string
	vcsProviderCreator      fake.VCSProviderCreator
	feishuProverdierCreator fake.FeishuProviderCreator
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

	// resourceDirOverride is the shared resource directory.
	resourceDirOverride string
)

// getTestPort reserves and returns a port.
func getTestPort() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort++
	return p
}

// getTestPortForEmbeddedPg reserves two ports, one for server and one for postgres server.
func getTestPortForEmbeddedPg() int {
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
	profile := getTestProfileWithExternalPg(config.dataDir, resourceDirOverride, serverPort, externalPgUser, pgURL, ctl.feishuProvider.APIURL(ctl.feishuURL))
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return err
	}
	ctl.server = server
	ctl.profile = profile

	return ctl.start(ctx, serverPort)
}

// StartServer starts the main server with embed Postgres.
func (ctl *controller) StartServer(ctx context.Context, config *config) error {
	log.SetLevel(zap.DebugLevel)
	if err := ctl.startMockServers(config.vcsProviderCreator, config.feishuProverdierCreator); err != nil {
		return err
	}
	serverPort := getTestPortForEmbeddedPg()
	profile := getTestProfile(config.dataDir, resourceDirOverride, serverPort, ctl.feishuProvider.APIURL(ctl.feishuURL))
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
func getTestProfile(dataDir, resourceDirOverride string, port int, feishuAPIURL string) componentConfig.Profile {
	// Using flags.port + 1 as our datastore port
	datastorePort := port + 1
	return componentConfig.Profile{
		Mode:                 common.ReleaseModeDev,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		DatastorePort:        datastorePort,
		PgUser:               "bbtest",
		DataDir:              dataDir,
		ResourceDirOverride:  resourceDirOverride,
		DemoDataDir:          fmt.Sprintf("demo/%s", testReleaseMode),
		AppRunnerInterval:    1 * time.Second,
		BackupRunnerInterval: 10 * time.Second,
		BackupStorageBackend: api.BackupStorageBackendLocal,
		FeishuAPIURL:         feishuAPIURL,
	}
}

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func getTestProfileWithExternalPg(dataDir, resourceDirOverride string, port int, pgUser string, pgURL string, feishuAPIURL string) componentConfig.Profile {
	return componentConfig.Profile{
		Mode:                 common.ReleaseModeDev,
		ExternalURL:          fmt.Sprintf("http://localhost:%d", port),
		PgUser:               pgUser,
		DataDir:              dataDir,
		ResourceDirOverride:  resourceDirOverride,
		DemoDataDir:          fmt.Sprintf("demo/%s", testReleaseMode),
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
	ctl.openAPIURL = fmt.Sprintf("http://localhost:%d/v1", port)

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
	ticker := time.NewTicker(100 * time.Microsecond)
	timer := time.NewTimer(5 * time.Second)
	defer ticker.Stop()
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			healthzURL := "/healthz"
			gURL := fmt.Sprintf("%s%s", ctl.rootURL, healthzURL)
			req, err := http.NewRequest(http.MethodGet, gURL, nil)
			if err != nil {
				log.Error("Fail to create a new GET request", zap.String("URL", gURL), zap.Error(err))
				continue
			}

			resp, err := ctl.client.Do(req)
			if err != nil {
				log.Error("Fail to send a GET request", zap.String("URL", gURL), zap.Error(err))
				continue
			}

			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Error("Failed to read http response body", zap.Error(err))
				}
				log.Error("http response error", zap.Int("status_code", resp.StatusCode), zap.ByteString("body", body))
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

// Login will login as user demo@example.com and caches its cookie.
func (ctl *controller) Login() error {
	resp, err := ctl.client.Post(
		fmt.Sprintf("%s/auth/login/BYTEBASE", ctl.apiURL),
		"",
		strings.NewReader(`{"data":{"type":"loginInfo","attributes":{"email":"demo@example.com","password":"1024"}}}`))
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
	url := fmt.Sprintf("%s%s", ctl.openAPIURL, shortURL)
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

func (ctl *controller) createPrincipal(principalCreate api.PrincipalCreate) (*api.Principal, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &principalCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal principal create")
	}

	body, err := ctl.post("/principal", buf)
	if err != nil {
		return nil, err
	}

	principal := new(api.Principal)
	if err = jsonapi.UnmarshalPayload(body, principal); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post principal response")
	}
	return principal, nil
}

func (ctl *controller) createMember(memberCreate api.MemberCreate) (*api.Member, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &memberCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal member create")
	}

	body, err := ctl.post("/member", buf)
	if err != nil {
		return nil, err
	}

	member := new(api.Member)
	if err = jsonapi.UnmarshalPayload(body, member); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post member response")
	}
	return member, nil
}

// getProjects gets the projects.
func (ctl *controller) getProjects() ([]*api.Project, error) {
	body, err := ctl.get("/project", nil)
	if err != nil {
		return nil, err
	}

	var projects []*api.Project
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Project)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get project response")
	}
	for _, p := range ps {
		project, ok := p.(*api.Project)
		if !ok {
			return nil, errors.Errorf("fail to convert project")
		}
		projects = append(projects, project)
	}
	return projects, nil
}

// createProject creates an project.
func (ctl *controller) createProject(projectCreate api.ProjectCreate) (*api.Project, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &projectCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal project create")
	}

	body, err := ctl.post("/project", buf)
	if err != nil {
		return nil, err
	}

	project := new(api.Project)
	if err = jsonapi.UnmarshalPayload(body, project); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post project response")
	}
	return project, nil
}

func (ctl *controller) patchProject(projectPatch api.ProjectPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &projectPatch); err != nil {
		return errors.Wrap(err, "failed to marshal project patch")
	}

	_, err := ctl.patch(fmt.Sprintf("/project/%d", projectPatch.ID), buf)
	if err != nil {
		return err
	}

	return nil
}

func (ctl *controller) createEnvironment(environmentCreate api.EnvironmentCreate) (*api.Environment, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &environmentCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal environment create")
	}

	body, err := ctl.post("/environment", buf)
	if err != nil {
		return nil, err
	}

	environment := new(api.Environment)
	if err = jsonapi.UnmarshalPayload(body, environment); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal post project response")
	}
	return environment, nil
}

// getProjects gets the environments.
func (ctl *controller) getEnvironments() ([]*api.Environment, error) {
	body, err := ctl.get("/environment", nil)
	if err != nil {
		return nil, err
	}

	var environments []*api.Environment
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Environment)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get environment response")
	}
	for _, p := range ps {
		environment, ok := p.(*api.Environment)
		if !ok {
			return nil, errors.Errorf("fail to convert environment")
		}
		environments = append(environments, environment)
	}
	return environments, nil
}

func findEnvironment(envs []*api.Environment, name string) (*api.Environment, error) {
	for _, env := range envs {
		if env.Name == name {
			return env, nil
		}
	}
	return nil, errors.Errorf("unable to find environment %q", name)
}

// getDatabases gets the databases.
func (ctl *controller) getDatabases(databaseFind api.DatabaseFind) ([]*api.Database, error) {
	params := make(map[string]string)
	if databaseFind.InstanceID != nil {
		params["instance"] = fmt.Sprintf("%d", *databaseFind.InstanceID)
	}
	if databaseFind.ProjectID != nil {
		params["project"] = fmt.Sprintf("%d", *databaseFind.ProjectID)
	}
	if databaseFind.Name != nil {
		params["name"] = *databaseFind.Name
	}
	body, err := ctl.get("/database", params)
	if err != nil {
		return nil, err
	}

	var databases []*api.Database
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Database)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get database response")
	}
	for _, p := range ps {
		database, ok := p.(*api.Database)
		if !ok {
			return nil, errors.Errorf("fail to convert database")
		}
		databases = append(databases, database)
	}
	return databases, nil
}

// DatabaseEditResult is a subset struct of api.DatabaseEditResult for testing,
// because of jsonapi doesn't support to unmarshal struct pointer slice.
type DatabaseEditResult struct {
	Statement string `jsonapi:"attr,statement"`
}

// postDatabaseEdit posts the database edit.
func (ctl *controller) postDatabaseEdit(databaseEdit api.DatabaseEdit) (*DatabaseEditResult, error) {
	buf, err := json.Marshal(&databaseEdit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal databaseEdit")
	}

	res, err := ctl.post(fmt.Sprintf("/database/%v/edit", databaseEdit.DatabaseID), strings.NewReader(string(buf)))
	if err != nil {
		return nil, err
	}

	databaseEditResult := new(DatabaseEditResult)
	if err = jsonapi.UnmarshalPayload(res, databaseEditResult); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post database edit response")
	}
	return databaseEditResult, nil
}

func (ctl *controller) setLicense() error {
	// Switch plan to increase instance limit.
	license, err := fs.ReadFile(fakeFS, "fake/license")
	if err != nil {
		return errors.Wrap(err, "failed to read fake license")
	}
	err = ctl.switchPlan(&enterpriseAPI.SubscriptionPatch{
		License: string(license),
	})
	if err != nil {
		return errors.Wrap(err, "failed to switch plan")
	}
	return nil
}

func (ctl *controller) removeLicense() error {
	err := ctl.switchPlan(&enterpriseAPI.SubscriptionPatch{
		License: "",
	})
	if err != nil {
		return errors.Wrap(err, "failed to switch plan")
	}
	return nil
}

func (ctl *controller) getSubscription() (*enterpriseAPI.Subscription, error) {
	body, err := ctl.get("/subscription", nil)
	if err != nil {
		return nil, err
	}

	subscription := new(enterpriseAPI.Subscription)
	if err = jsonapi.UnmarshalPayload(body, subscription); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get subscription response")
	}
	return subscription, nil
}

func (ctl *controller) trialPlan(trial *api.TrialPlanCreate) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, trial); err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}

	_, err := ctl.post("/subscription/trial", buf)
	return err
}

func (ctl *controller) switchPlan(patch *enterpriseAPI.SubscriptionPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, patch); err != nil {
		return errors.Wrap(err, "failed to marshal subscription patch")
	}

	_, err := ctl.patch("/subscription", buf)
	if err != nil {
		return err
	}

	return nil
}

func (ctl *controller) getInstanceByID(instanceID int) (*api.Instance, error) {
	body, err := ctl.get(fmt.Sprintf("/instance/%d", instanceID), nil)
	if err != nil {
		return nil, err
	}

	instance := new(api.Instance)
	if err = jsonapi.UnmarshalPayload(body, instance); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get instance response")
	}
	return instance, nil
}

// addInstance adds an instance.
func (ctl *controller) addInstance(instanceCreate api.InstanceCreate) (*api.Instance, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &instanceCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal instance create")
	}

	body, err := ctl.post("/instance", buf)
	if err != nil {
		return nil, err
	}

	instance := new(api.Instance)
	if err = jsonapi.UnmarshalPayload(body, instance); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post instance response")
	}
	return instance, nil
}

func (ctl *controller) getInstanceMigrationHistory(find db.MigrationHistoryFind) ([]*api.MigrationHistory, error) {
	params := make(map[string]string)
	if find.Database != nil {
		params["database"] = *find.Database
	}
	if find.Version != nil {
		params["version"] = *find.Version
	}
	if find.Limit != nil {
		params["limit"] = fmt.Sprintf("%d", *find.Limit)
	}
	body, err := ctl.get(fmt.Sprintf("/instance/%v/migration/history", *find.ID), params)
	if err != nil {
		return nil, err
	}

	var histories []*api.MigrationHistory
	hs, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.MigrationHistory)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get migration history response")
	}
	for _, h := range hs {
		history, ok := h.(*api.MigrationHistory)
		if !ok {
			return nil, errors.Errorf("fail to convert migration history")
		}
		histories = append(histories, history)
	}
	return histories, nil
}

// createIssue creates an issue.
func (ctl *controller) createIssue(issueCreate api.IssueCreate) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issueCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal issue create")
	}

	body, err := ctl.post("/issue", buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal post issue response")
	}
	return issue, nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssue(id int) (*api.Issue, error) {
	body, err := ctl.get(fmt.Sprintf("/issue/%d", id), nil)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get issue response")
	}
	return issue, nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssues(issueFind api.IssueFind) ([]*api.Issue, error) {
	var ret []*api.Issue
	// call getOnePageIssuesWithToken until no more issues.
	token := ""
	for {
		issues, nextToken, err := ctl.getOnePageIssuesWithToken(issueFind, token)
		if err != nil {
			return nil, err
		}
		if len(issues) == 0 {
			break
		}
		ret = append(ret, issues...)
		token = nextToken
	}
	return ret, nil
}

func (ctl *controller) getOnePageIssuesWithToken(issueFind api.IssueFind, token string) ([]*api.Issue, string, error) {
	params := make(map[string]string)
	if issueFind.ProjectID != nil {
		params["project"] = fmt.Sprintf("%d", *issueFind.ProjectID)
	}
	if len(issueFind.StatusList) > 0 {
		var sl []string
		for _, status := range issueFind.StatusList {
			sl = append(sl, string(status))
		}
		params["status"] = strings.Join(sl, ",")
	}
	if token != "" {
		params["token"] = token
	}
	body, err := ctl.get("/issue", params)
	if err != nil {
		return nil, "", err
	}
	issueResp := new(api.IssueResponse)
	err = jsonapi.UnmarshalPayload(body, issueResp)
	if err != nil {
		return nil, "", errors.Wrap(err, "fail to unmarshal get issue response")
	}
	return issueResp.Issues, issueResp.NextToken, nil
}

func (ctl *controller) patchIssue(issuePatch api.IssuePatch) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issuePatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal issue patch")
	}

	body, err := ctl.patch(fmt.Sprintf("/issue/%d", issuePatch.ID), buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patch issue patch response")
	}
	return issue, nil
}

// patchIssue patches the issue with given ID.
func (ctl *controller) patchIssueStatus(issueStatusPatch api.IssueStatusPatch) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issueStatusPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal issue status patch")
	}

	body, err := ctl.patch(fmt.Sprintf("/issue/%d/status", issueStatusPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patch issue status patch response")
	}
	return issue, nil
}

// patchTaskStatus patches the status of a task in the pipeline stage.
func (ctl *controller) patchTaskStatus(taskStatusPatch api.TaskStatusPatch, pipelineID int, taskID int) (*api.Task, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &taskStatusPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal patchTaskStatus")
	}

	body, err := ctl.patch(fmt.Sprintf("/pipeline/%d/task/%d/status", pipelineID, taskID), buf)
	if err != nil {
		return nil, err
	}

	task := new(api.Task)
	if err = jsonapi.UnmarshalPayload(body, task); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patchTaskStatus response")
	}
	return task, nil
}

// patchStageAllTaskStatus patches the status of all tasks in the pipeline stage.
func (ctl *controller) patchStageAllTaskStatus(stageAllTaskStatusPatch api.StageAllTaskStatusPatch, pipelineID int) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &stageAllTaskStatusPatch); err != nil {
		return errors.Wrap(err, "failed to marshal StageAllTaskStatusPatch")
	}

	_, err := ctl.patch(fmt.Sprintf("/pipeline/%d/stage/%d/status", pipelineID, stageAllTaskStatusPatch.ID), buf)
	return err
}

// approveIssueNext approves the next pending approval task.
func (ctl *controller) approveIssueNext(issue *api.Issue) error {
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				if _, err := ctl.patchTaskStatus(
					api.TaskStatusPatch{
						Status: api.TaskPending,
					},
					issue.Pipeline.ID, task.ID); err != nil {
					return errors.Wrapf(err, "failed to patch task status for task %d", task.ID)
				}
				return nil
			}
		}
	}
	return nil
}

// approveIssueTasksWithStageApproval approves all pending approval tasks in the next stage.
func (ctl *controller) approveIssueTasksWithStageApproval(issue *api.Issue) error {
	stageID := 0
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				stageID = stage.ID
				break
			}
		}
		if stageID != 0 {
			break
		}
	}
	if stageID != 0 {
		if err := ctl.patchStageAllTaskStatus(
			api.StageAllTaskStatusPatch{
				ID:     stageID,
				Status: api.TaskPending,
			},
			issue.Pipeline.ID,
		); err != nil {
			return errors.Wrapf(err, "failed to patch task status for stage %d", stageID)
		}
	}
	return nil
}

// getNextTaskStatus gets the next task status that needs to be handle.
func getNextTaskStatus(issue *api.Issue) (api.TaskStatus, error) {
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskDone {
				continue
			}
			if task.Status == api.TaskFailed {
				var runs []string
				for _, run := range task.TaskRunList {
					runs = append(runs, fmt.Sprintf("%+v", run))
				}
				return api.TaskFailed, errors.Errorf("pipeline task %v failed runs: %v", task.ID, strings.Join(runs, ", "))
			}
			return task.Status, nil
		}
	}
	return api.TaskDone, nil
}

// waitIssueNextTaskWithTaskApproval waits for next task in pipeline to finish and approves it when necessary.
func (ctl *controller) waitIssueNextTaskWithTaskApproval(id int) (api.TaskStatus, error) {
	return ctl.waitIssuePipelineTaskImpl(id, ctl.approveIssueNext, true)
}

// waitIssuePipeline waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipeline(id int) (api.TaskStatus, error) {
	return ctl.waitIssuePipelineTaskImpl(id, ctl.approveIssueNext, false)
}

// waitIssuePipelineWithStageApproval waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipelineWithStageApproval(id int) (api.TaskStatus, error) {
	return ctl.waitIssuePipelineTaskImpl(id, ctl.approveIssueTasksWithStageApproval, false)
}

// waitIssuePipelineWithNoApproval waits for pipeline to finish and do nothing when approvals are needed.
func (ctl *controller) waitIssuePipelineWithNoApproval(id int) (api.TaskStatus, error) {
	noop := func(*api.Issue) error {
		return nil
	}
	return ctl.waitIssuePipelineTaskImpl(id, noop, false)
}

// waitIssuePipelineImpl waits for the tasks in pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipelineTaskImpl(id int, approveFunc func(issue *api.Issue) error, approveOnce bool) (api.TaskStatus, error) {
	// Sleep for 1 second between issues so that we don't get migration version conflict because we are using second-level timestamp for the version string. We choose sleep because it mimics the user's behavior.
	time.Sleep(1 * time.Second)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	approved := false

	log.Debug("Waiting for issue pipeline tasks.")
	prevStatus := "UNKNOWN"
	for range ticker.C {
		issue, err := ctl.getIssue(id)
		if err != nil {
			return api.TaskFailed, err
		}

		status, err := getNextTaskStatus(issue)
		if err != nil {
			return status, err
		}
		if string(status) != prevStatus {
			log.Debug(fmt.Sprintf("Status changed: %s -> %s.", prevStatus, status))
			prevStatus = string(status)
		}
		switch status {
		case api.TaskPendingApproval:
			if approveOnce && approved {
				return api.TaskDone, nil
			}
			if err := approveFunc(issue); err != nil {
				return api.TaskFailed, err
			}
			approved = true
		case api.TaskFailed, api.TaskDone, api.TaskCanceled:
			return status, err
		case api.TaskPending, api.TaskRunning:
			approved = true
			// keep waiting
		}
	}
	return api.TaskDone, nil
}

// executeSQL executes a SQL query on the database.
func (ctl *controller) executeSQL(sqlExecute api.SQLExecute) (*api.SQLResultSet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sqlExecute); err != nil {
		return nil, errors.Wrap(err, "failed to marshal sqlExecute")
	}

	body, err := ctl.post("/sql/execute", buf)
	if err != nil {
		return nil, err
	}

	sqlResultSet := new(api.SQLResultSet)
	if err = jsonapi.UnmarshalPayload(body, sqlResultSet); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal sqlResultSet response")
	}
	return sqlResultSet, nil
}

func (ctl *controller) query(instance *api.Instance, databaseName, query string) (string, error) {
	sqlResultSet, err := ctl.executeSQL(api.SQLExecute{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Statement:    query,
		Readonly:     true,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to execute SQL")
	}
	if sqlResultSet.Error != "" {
		return "", errors.Errorf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	return sqlResultSet.Data, nil
}

// adminExecuteSQL executes a SQL query on the database.
func (ctl *controller) adminExecuteSQL(sqlExecute api.SQLExecute) (*api.SQLResultSet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sqlExecute); err != nil {
		return nil, errors.Wrap(err, "failed to marshal sqlExecute")
	}

	body, err := ctl.post("/sql/execute/admin", buf)
	if err != nil {
		return nil, err
	}

	sqlResultSet := new(api.SQLResultSet)
	if err = jsonapi.UnmarshalPayload(body, sqlResultSet); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal sqlResultSet response")
	}
	return sqlResultSet, nil
}

func (ctl *controller) adminQuery(instance *api.Instance, databaseName, query string) (string, error) {
	sqlResultSet, err := ctl.adminExecuteSQL(api.SQLExecute{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Statement:    query,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to execute SQL")
	}
	if sqlResultSet.Error != "" {
		return "", errors.Errorf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	return sqlResultSet.Data, nil
}

// createVCS creates a VCS.
func (ctl *controller) createVCS(vcsCreate api.VCSCreate) (*api.VCS, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &vcsCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal vcsCreate")
	}

	body, err := ctl.post("/vcs", buf)
	if err != nil {
		return nil, err
	}

	vcs := new(api.VCS)
	if err = jsonapi.UnmarshalPayload(body, vcs); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal vcs response")
	}
	return vcs, nil
}

// createRepository links the repository with the project.
func (ctl *controller) createRepository(repositoryCreate api.RepositoryCreate) (*api.Repository, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &repositoryCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal repositoryCreate")
	}

	body, err := ctl.post(fmt.Sprintf("/project/%d/repository", repositoryCreate.ProjectID), buf)
	if err != nil {
		return nil, err
	}

	repository := new(api.Repository)
	if err = jsonapi.UnmarshalPayload(body, repository); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal repository response")
	}
	return repository, nil
}

// unlinkRepository unlinks the repository from the project by projectID.
func (ctl *controller) unlinkRepository(projectID int) error {
	_, err := ctl.delete(fmt.Sprintf("/project/%d/repository", projectID), nil)
	if err != nil {
		return err
	}
	return nil
}

// listRepository gets the repository list.
func (ctl *controller) listRepository(projectID int) ([]*api.Repository, error) {
	body, err := ctl.get(fmt.Sprintf("/project/%d/repository", projectID), nil)
	if err != nil {
		return nil, err
	}

	var repositories []*api.Repository
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Repository)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get repository response")
	}
	for _, p := range ps {
		repository, ok := p.(*api.Repository)
		if !ok {
			return nil, errors.Errorf("fail to convert repository")
		}
		repositories = append(repositories, repository)
	}
	return repositories, nil
}

// createSQLReviewCI set up the SQL review CI.
func (ctl *controller) createSQLReviewCI(projectID, repositoryID int) (*api.SQLReviewCISetup, error) {
	body, err := ctl.post(fmt.Sprintf("/project/%d/repository/%d/sql-review-ci", projectID, repositoryID), new(bytes.Buffer))
	if err != nil {
		return nil, err
	}

	sqlReviewCISetup := new(api.SQLReviewCISetup)
	if err = jsonapi.UnmarshalPayload(body, sqlReviewCISetup); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal SQL reivew CI response")
	}
	return sqlReviewCISetup, nil
}

func (ctl *controller) getLatestSchemaDump(databaseID int) (string, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/schema", databaseID), nil)
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (ctl *controller) getLatestSchemaMetadata(databaseID int) (string, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/schema", databaseID), map[string]string{"metadata": "true"})
	if err != nil {
		return "", err
	}
	bs, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (ctl *controller) createDatabase(project *api.Project, instance *api.Instance, databaseName string, owner string, labelMap map[string]string) error {
	labels, err := marshalLabels(labelMap, instance.Environment.Name)
	if err != nil {
		return err
	}
	ctx := &api.CreateDatabaseContext{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Labels:       labels,
		CharacterSet: "utf8mb4",
		Collation:    "utf8mb4_general_ci",
	}
	if instance.Engine == db.Postgres {
		ctx.Owner = owner
		ctx.CharacterSet = "UTF8"
		ctx.Collation = "en_US.UTF-8"
	}
	createContext, err := json.Marshal(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to construct database creation issue CreateContext payload")
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("create database %q", databaseName),
		Type:        api.IssueDatabaseCreate,
		Description: fmt.Sprintf("This creates a database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create database creation issue")
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return errors.Errorf("issue %v pipeline %v is supposed to be pending manual approval", issue.ID, issue.Pipeline.ID)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to wait for issue %v pipeline %v", issue.ID, issue.Pipeline.ID)
	}
	if status != api.TaskDone {
		return errors.Errorf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}
	issue, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to patch issue status %v to done", issue.ID)
	}
	// Add a second sleep to avoid schema version conflict.
	time.Sleep(time.Second)
	return nil
}

// cloneDatabaseFromBackup clones the database from an existing backup.
func (ctl *controller) cloneDatabaseFromBackup(project *api.Project, instance *api.Instance, databaseName string, backup *api.Backup, labelMap map[string]string) error {
	labels, err := marshalLabels(labelMap, instance.Environment.Name)
	if err != nil {
		return err
	}

	createContext, err := json.Marshal(&api.CreateDatabaseContext{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		BackupID:     backup.ID,
		Labels:       labels,
	})
	if err != nil {
		return errors.Wrap(err, "failed to construct database creation issue CreateContext payload")
	}
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("create database %q from backup %q", databaseName, backup.Name),
		Type:        api.IssueDatabaseCreate,
		Description: fmt.Sprintf("This creates a database %q from backup %q.", databaseName, backup.Name),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create database creation issue")
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return errors.Errorf("issue %v pipeline %v is supposed to be pending manual approval", issue.ID, issue.Pipeline.ID)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to wait for issue %v pipeline %v", issue.ID, issue.Pipeline.ID)
	}
	if status != api.TaskDone {
		return errors.Errorf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}
	issue, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to patch issue status %v to done", issue.ID)
	}
	return nil
}

func marshalLabels(labelMap map[string]string, environmentName string) (string, error) {
	var labelList []*api.DatabaseLabel
	for k, v := range labelMap {
		labelList = append(labelList, &api.DatabaseLabel{
			Key:   k,
			Value: v,
		})
	}
	labelList = append(labelList, &api.DatabaseLabel{
		Key:   api.EnvironmentKeyName,
		Value: environmentName,
	})

	labels, err := json.Marshal(labelList)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal labels %+v", labelList)
	}
	return string(labels), nil
}

// getLabels gets all the labels.
func (ctl *controller) getLabels() ([]*api.LabelKey, error) {
	body, err := ctl.get("/label", nil)
	if err != nil {
		return nil, err
	}

	var labelKeys []*api.LabelKey
	lks, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.LabelKey)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get label response")
	}
	for _, lk := range lks {
		labelKey, ok := lk.(*api.LabelKey)
		if !ok {
			return nil, errors.Errorf("fail to convert label key")
		}
		labelKeys = append(labelKeys, labelKey)
	}
	return labelKeys, nil
}

// patchLabelKey patches the label key with given ID.
func (ctl *controller) patchLabelKey(labelKeyPatch api.LabelKeyPatch) (*api.LabelKey, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &labelKeyPatch); err != nil {
		return nil, errors.Wrap(err, "failed to marshal label key patch")
	}

	body, err := ctl.patch(fmt.Sprintf("/label/%d", labelKeyPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	labelKey := new(api.LabelKey)
	if err = jsonapi.UnmarshalPayload(body, labelKey); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal patch label key response")
	}
	return labelKey, nil
}

// addLabelValues adds values to an existing label key.
func (ctl *controller) addLabelValues(key string, values []string) error {
	labelKeys, err := ctl.getLabels()
	if err != nil {
		return errors.Wrap(err, "failed to get labels")
	}
	var labelKey *api.LabelKey
	for _, lk := range labelKeys {
		if lk.Key == key {
			labelKey = lk
			break
		}
	}
	if labelKey == nil {
		return errors.Errorf("failed to find label with key %q", key)
	}
	var valueList []string
	valueList = append(valueList, labelKey.ValueList...)
	valueList = append(valueList, values...)
	_, err = ctl.patchLabelKey(api.LabelKeyPatch{
		ID:        labelKey.ID,
		ValueList: valueList,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to patch label key for key %q ID %d values %+v", key, labelKey.ID, valueList)
	}
	return nil
}

// upsertDeploymentConfig upserts the deployment configuration for a project.
func (ctl *controller) upsertDeploymentConfig(deploymentConfigUpsert api.DeploymentConfigUpsert, deploymentSchedule api.DeploymentSchedule) (*api.DeploymentConfig, error) {
	scheduleBuf, err := json.Marshal(&deploymentSchedule)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal deployment schedule")
	}
	deploymentConfigUpsert.Payload = string(scheduleBuf)

	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &deploymentConfigUpsert); err != nil {
		return nil, errors.Wrap(err, "failed to marshal deployment config upsert")
	}

	body, err := ctl.patch(fmt.Sprintf("/project/%d/deployment", deploymentConfigUpsert.ProjectID), buf)
	if err != nil {
		return nil, err
	}

	deploymentConfig := new(api.DeploymentConfig)
	if err = jsonapi.UnmarshalPayload(body, deploymentConfig); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal upsert deployment config response")
	}
	return deploymentConfig, nil
}

// disableAutomaticBackup disables the automatic backup of a database.
func (ctl *controller) disableAutomaticBackup(databaseID int) error {
	backupSetting := api.BackupSettingUpsert{
		DatabaseID: databaseID,
		Enabled:    false,
	}
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &backupSetting); err != nil {
		return errors.Wrap(err, "failed to marshal backupSetting")
	}

	if _, err := ctl.patch(fmt.Sprintf("/database/%d/backup-setting", databaseID), buf); err != nil {
		return err
	}
	return nil
}

// createBackup creates a backup.
func (ctl *controller) createBackup(backupCreate api.BackupCreate) (*api.Backup, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &backupCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal backupCreate")
	}

	body, err := ctl.post(fmt.Sprintf("/database/%d/backup", backupCreate.DatabaseID), buf)
	if err != nil {
		return nil, err
	}

	backup := new(api.Backup)
	if err = jsonapi.UnmarshalPayload(body, backup); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal backup response")
	}
	return backup, nil
}

// listBackups lists backups for a database.
func (ctl *controller) listBackups(databaseID int) ([]*api.Backup, error) {
	body, err := ctl.get(fmt.Sprintf("/database/%d/backup", databaseID), nil)
	if err != nil {
		return nil, err
	}

	var backups []*api.Backup
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Backup)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get backup response")
	}
	for _, p := range ps {
		backup, ok := p.(*api.Backup)
		if !ok {
			return nil, errors.Errorf("fail to convert backup")
		}
		backups = append(backups, backup)
	}
	return backups, nil
}

// waitBackup waits for a backup to be done.
func (ctl *controller) waitBackup(databaseID, backupID int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Debug("Waiting for backup.", zap.Int("id", backupID))
	for range ticker.C {
		backups, err := ctl.listBackups(databaseID)
		if err != nil {
			return err
		}
		var backup *api.Backup
		for _, b := range backups {
			if b.ID == backupID {
				backup = b
				break
			}
		}
		if backup == nil {
			return errors.Errorf("backup %v for database %v not found", backupID, databaseID)
		}
		switch backup.Status {
		case api.BackupStatusDone:
			return nil
		case api.BackupStatusFailed:
			return errors.Errorf("backup %v for database %v failed", backupID, databaseID)
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return errors.Errorf("failed to wait for backup as this condition should never be reached")
}

// waitBackupArchived waits for a backup to be archived.
func (ctl *controller) waitBackupArchived(databaseID, backupID int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Debug("Waiting for backup.", zap.Int("id", backupID))
	for range ticker.C {
		backups, err := ctl.listBackups(databaseID)
		if err != nil {
			return err
		}
		var backup *api.Backup
		for _, b := range backups {
			if b.ID == backupID {
				backup = b
				break
			}
		}
		if backup == nil {
			return errors.Errorf("backup %d for database %d not found", backupID, databaseID)
		}
		if backup.RowStatus == api.Archived {
			return nil
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return errors.Errorf("failed to wait for backup as this condition should never be reached")
}

// createSheet creates a sheet.
func (ctl *controller) createSheet(sheetCreate api.SheetCreate) (*api.Sheet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sheetCreate); err != nil {
		return nil, errors.Wrap(err, "failed to marshal sheetCreate")
	}

	body, err := ctl.post("/sheet", buf)
	if err != nil {
		return nil, err
	}

	sheet := new(api.Sheet)
	if err = jsonapi.UnmarshalPayload(body, sheet); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal sheet response")
	}
	return sheet, nil
}

// listMySheets lists caller's sheets.
func (ctl *controller) listMySheets() ([]*api.Sheet, error) {
	params := map[string]string{}
	body, err := ctl.get("/sheet/my", params)
	if err != nil {
		return nil, err
	}

	var sheets []*api.Sheet
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Sheet)))
	if err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal get sheet response")
	}
	for _, p := range ps {
		sheet, ok := p.(*api.Sheet)
		if !ok {
			return nil, errors.Errorf("fail to convert sheet")
		}
		sheets = append(sheets, sheet)
	}
	return sheets, nil
}

// syncSheet syncs sheets with project.
func (ctl *controller) syncSheet(projectID int) error {
	_, err := ctl.post(fmt.Sprintf("/project/%d/sync-sheet", projectID), nil)
	return err
}

func (ctl *controller) createDataSource(dataSourceCreate api.DataSourceCreate) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &dataSourceCreate); err != nil {
		return errors.Wrap(err, "failed to marshal dataSourceCreate")
	}

	body, err := ctl.post(fmt.Sprintf("/database/%d/data-source", dataSourceCreate.DatabaseID), buf)
	if err != nil {
		return err
	}

	dataSource := new(api.DataSource)
	if err = jsonapi.UnmarshalPayload(body, dataSource); err != nil {
		return errors.Wrap(err, "fail to unmarshal dataSource response")
	}
	return nil
}

func (ctl *controller) patchDataSource(databaseID int, dataSourcePatch api.DataSourcePatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &dataSourcePatch); err != nil {
		return errors.Wrap(err, "failed to marshal dataSourcePatch")
	}

	body, err := ctl.patch(fmt.Sprintf("/database/%d/data-source/%d", databaseID, dataSourcePatch.ID), buf)
	if err != nil {
		return err
	}

	dataSource := new(api.DataSource)
	if err = jsonapi.UnmarshalPayload(body, dataSource); err != nil {
		return errors.Wrap(err, "fail to unmarshal dataSource response")
	}
	return nil
}

func (ctl *controller) deleteDataSource(databaseID, dataSourceID int) error {
	_, err := ctl.delete(fmt.Sprintf("/database/%d/data-source/%d", databaseID, dataSourceID), nil)
	if err != nil {
		return err
	}
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

// upsertPolicy upserts the policy.
func (ctl *controller) upsertPolicy(policyUpsert api.PolicyUpsert) (*api.Policy, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &policyUpsert); err != nil {
		return nil, errors.Wrap(err, "failed to marshal policyUpsert")
	}

	body, err := ctl.patch(fmt.Sprintf("/policy/%s/%d?type=%s", strings.ToLower(string(policyUpsert.ResourceType)), policyUpsert.ResourceID, policyUpsert.Type), buf)
	if err != nil {
		return nil, err
	}

	policy := new(api.Policy)
	if err = jsonapi.UnmarshalPayload(body, policy); err != nil {
		return nil, errors.Wrap(err, "fail to unmarshal policy response")
	}
	return policy, nil
}

// deletePolicy deletes the archived policy.
func (ctl *controller) deletePolicy(policyDelete api.PolicyDelete) error {
	_, err := ctl.delete(fmt.Sprintf("/policy/environment/%d?type=%s", policyDelete.ResourceID, policyDelete.Type), new(bytes.Buffer))
	if err != nil {
		return err
	}
	return nil
}

// sqlReviewTaskCheckRunFinished will return SQL review task check result for next task.
// If the SQL review task check is not done, return nil, false, nil.
func (*controller) sqlReviewTaskCheckRunFinished(issue *api.Issue) ([]api.TaskCheckResult, bool, error) {
	var result []api.TaskCheckResult
	var latestTs int64
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				for _, taskCheck := range task.TaskCheckRunList {
					if taskCheck.Type == api.TaskCheckDatabaseStatementAdvise {
						switch taskCheck.Status {
						case api.TaskCheckRunRunning:
							return nil, false, nil
						case api.TaskCheckRunDone:
							// return the latest result
							if latestTs != 0 && latestTs > taskCheck.UpdatedTs {
								continue
							}
							checkResult := &api.TaskCheckRunResultPayload{}
							if err := json.Unmarshal([]byte(taskCheck.Result), checkResult); err != nil {
								return nil, false, err
							}
							result = checkResult.ResultList
						}
					}
				}
				return result, true, nil
			}
		}
	}
	return nil, true, nil
}

// GetSQLReviewResult will wait for next task SQL review task check to finish and return the task check result.
func (ctl *controller) GetSQLReviewResult(id int) ([]api.TaskCheckResult, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		issue, err := ctl.getIssue(id)
		if err != nil {
			return nil, err
		}

		status, err := getNextTaskStatus(issue)
		if err != nil {
			return nil, err
		}

		if status != api.TaskPendingApproval {
			return nil, errors.Errorf("the status of issue %v is not pending approval", id)
		}

		result, yes, err := ctl.sqlReviewTaskCheckRunFinished(issue)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get SQL review result for issue %v", id)
		}
		if yes {
			return result, nil
		}
	}
	return nil, nil
}

// prodTemplateSQLReviewPolicy returns the default SQL review policy.
func prodTemplateSQLReviewPolicy() (string, error) {
	policy := advisor.SQLReviewPolicy{
		Name: "Prod",
		RuleList: []*advisor.SQLReviewRule{
			// Engine
			{
				Type:  advisor.SchemaRuleMySQLEngine,
				Level: advisor.SchemaRuleLevelError,
			},
			// Naming
			{
				Type:  advisor.SchemaRuleTableNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIDXNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRulePKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleUKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleFKNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleAutoIncrementColumnNaming,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// Statement
			{
				Type:  advisor.SchemaRuleStatementNoSelectAll,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementRequireWhere,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementNoLeadingWildcardLike,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowCommit,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementDisallowOrderBy,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementMergeAlterTable,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertRowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertMustSpecifyColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementInsertDisallowOrderByRand,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementAffectedRowLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleStatementDMLDryRun,
				Level: advisor.SchemaRuleLevelError,
			},
			// TABLE
			{
				Type:  advisor.SchemaRuleTableRequirePK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableNoFK,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableDropNamingConvention,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleTableCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleTableDisallowPartition,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// COLUMN
			{
				Type:  advisor.SchemaRuleRequiredColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChangeType,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnSetDefaultForNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChange,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowChangingOrder,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnCommentConvention,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementMustInteger,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleColumnTypeDisallowList,
				Level: advisor.SchemaRuleLevelError,
			},
			{
				Type:  advisor.SchemaRuleColumnDisallowSetCharset,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnMaximumCharacterLength,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementInitialValue,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnAutoIncrementMustUnsigned,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleCurrentTimeColumnCountLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnRequireDefault,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SCHEMA
			{
				Type:  advisor.SchemaRuleSchemaBackwardCompatibility,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// DATABASE
			{
				Type:  advisor.SchemaRuleDropEmptyDatabase,
				Level: advisor.SchemaRuleLevelError,
			},
			// INDEX
			{
				Type:  advisor.SchemaRuleIndexNoDuplicateColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexKeyNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexPKTypeLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexTypeNoBlob,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleIndexTotalNumberLimit,
				Level: advisor.SchemaRuleLevelWarning,
			},
			// SYSTEM
			{
				Type:  advisor.SchemaRuleCharsetAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleCollationAllowlist,
				Level: advisor.SchemaRuleLevelWarning,
			},
		},
	}

	for _, rule := range policy.RuleList {
		payload, err := advisor.SetDefaultSQLReviewRulePayload(rule.Type)
		if err != nil {
			return "", err
		}
		rule.Payload = payload
	}

	s, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

type schemaDiffRequest struct {
	EngineType   parser.EngineType `json:"engineType"`
	SourceSchema string            `json:"sourceSchema"`
	TargetSchema string            `json:"targetSchema"`
}

// getSchemaDiff gets the schema diff.
func (ctl *controller) getSchemaDiff(schemaDiff schemaDiffRequest) (string, error) {
	buf, err := json.Marshal(&schemaDiff)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal schemaDiffRequest")
	}

	body, err := ctl.postOpenAPI("/sql/schema/diff", strings.NewReader(string(buf)))
	if err != nil {
		return "", err
	}

	diff, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	diffString := ""
	if err := json.Unmarshal(diff, &diffString); err != nil {
		return "", err
	}
	return diffString, nil
}
