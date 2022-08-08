package tests

import (
	"bytes"
	"context"
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
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/server/cmd"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server"
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
	bookTableQuery      = "SELECT * FROM sqlite_schema WHERE type = 'table' AND tbl_name = 'book';"
	bookSchemaSQLResult = `[["type","name","tbl_name","rootpage","sql"],["TEXT","TEXT","TEXT","INT","TEXT"],[["table","book","book",2,"CREATE TABLE book (\n\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\t\tname TEXT NULL\n\t)"]]]`
	bookDataQuery       = `SELECT * FROM book;`
	bookDataSQLResult   = `[["id","name"],["INTEGER","TEXT"],[[1,"byte"],[2,null]]]`

	dataUpdateStatement = `
	INSERT INTO book(name) VALUES
		("byte"),
		(NULL);
	`
	dumpedSchema = "" +
		`CREATE TABLE book (
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
	server      *server.Server
	client      *http.Client
	cookie      string
	vcsProvider fake.VCSProvider

	rootURL string
	apiURL  string
	vcsURL  string
}

func getTestPort(testName string) int {
	// We allocates 4 ports for each of the integration test, who probably would start
	// the Bytebase server, Postgres, MySQL and GitLab.
	tests := []string{
		"TestServiceRestart",
		"TestSchemaAndDataUpdate",
		"TestVCS/GitLab",
		"TestVCS/GitHub",
		"TestTenant",
		"TestTenantVCS/GitLab",
		"TestTenantVCS/GitHub",
		"TestTenantDatabaseNameTemplate",
		"TestGhostSchemaUpdate",
		"TestBackupRestoreBasic",
		"TestTenantVCSDatabaseNameTemplate/GitLab",
		"TestTenantVCSDatabaseNameTemplate/GitHub",
		"TestBootWithExternalPg",
		"TestSheetVCS/GitLab",
		"TestSheetVCS/GitHub",
		"TestPrepare",

		// PITR related cases
		"TestPITR",
		"TestPITR/Buggy_Application",
		"TestPITR/Schema_Migration_Failure",
		"TestPITR/Drop_Database",
		"TestPITR/Case_Sensitive",
		"TestPITR/Invalid_Time_Point",
		"TestPITR/PITR_Twice",

		"TestCheckEngineInnoDB",
		"TestCheckServerVersionAndBinlogForPITR",
		"TestFetchBinlogFiles",

		"TestSQLReviewForMySQL",
		"TestSQLReviewForPostgreSQL",
	}
	port := 1234
	for _, name := range tests {
		if testName == name {
			return port
		}
		port += 4
	}
	panic(fmt.Sprintf("test %q doesn't have assigned port, please set it in getTestPort()", testName))
}

// StartServerWithExternalPg starts the main server with external Postgres.
func (ctl *controller) StartServerWithExternalPg(ctx context.Context, dataDir string, vcsProviderCreator fake.VCSProviderCreator, port int, pgUser, pgURL string) error {
	log.SetLevel(zap.DebugLevel)
	profile := cmd.GetTestProfileWithExternalPg(dataDir, port, pgUser, pgURL)
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return err
	}
	ctl.server = server

	return ctl.start(ctx, vcsProviderCreator, port)
}

// StartServer starts the main server with embed Postgres.
func (ctl *controller) StartServer(ctx context.Context, dataDir string, vcsProviderCreator fake.VCSProviderCreator, port int) error {
	// start main server.
	log.SetLevel(zap.DebugLevel)
	profile := cmd.GetTestProfile(dataDir, port)
	server, err := server.NewServer(ctx, profile)
	if err != nil {
		return err
	}
	ctl.server = server

	return ctl.start(ctx, vcsProviderCreator, port)
}

// start only called by StartServer() and StartServerWithExternalPg().
func (ctl *controller) start(ctx context.Context, vcsProviderCreator fake.VCSProviderCreator, port int) error {
	ctl.rootURL = fmt.Sprintf("http://localhost:%d", port)
	ctl.apiURL = fmt.Sprintf("http://localhost:%d/api", port)

	// Set up VCS provider.
	vcsPort := port + 2
	ctl.vcsProvider = vcsProviderCreator(vcsPort)
	ctl.vcsURL = fmt.Sprintf("http://localhost:%d", vcsPort)

	errChan := make(chan error, 1)

	go func() {
		if err := ctl.server.Run(ctx); err != nil {
			errChan <- fmt.Errorf("failed to run main server, error: %w", err)
		}
	}()

	go func() {
		if err := ctl.vcsProvider.Run(); err != nil {
			errChan <- fmt.Errorf("failed to run vcsProvider server, error: %w", err)
		}
	}()

	if err := waitForServerStart(ctl.server, errChan); err != nil {
		return fmt.Errorf("failed to wait for server to start, error: %w", err)
	}
	if err := waitForVCSStart(ctl.vcsProvider, errChan); err != nil {
		return fmt.Errorf("failed to wait for vcsProvider to start, error: %w", err)
	}

	// initialize controller clients.
	ctl.client = &http.Client{}

	if err := ctl.waitForHealthz(); err != nil {
		return fmt.Errorf("failed to wait for healthz, err: %w", err)
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
				fmt.Printf("fail to create a new GET request(%q), error: %s", gURL, err.Error())
				continue

			}

			resp, err := ctl.client.Do(req)
			if err != nil {
				fmt.Printf("fail to send a GET request(%q), error: %s", gURL, err.Error())
				continue
			}

			if resp.StatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("failed to read http response body, error: %s", err.Error())
				}
				fmt.Printf("http response error code %v body %q", resp.StatusCode, string(body))
				continue
			}

			return nil

		case end := <-timer.C:
			return fmt.Errorf("cannot wait for healthz in duration: %v", end.Sub(begin).Seconds())

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
		return fmt.Errorf("fail to post login request, error %w", err)
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
		return fmt.Errorf("unable to find access token in the login response headers")
	}
	ctl.cookie = cookie

	return nil
}

// provisionSQLiteInstance provisions a SQLite instance (a directory).
func (*controller) provisionSQLiteInstance(rootDir, name string) (string, error) {
	p := path.Join(rootDir, name)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to make directory %q, error: %w", p, err)
	}

	return p, nil
}

// get sends a GET client request.
func (ctl *controller) get(shortURL string, params map[string]string) (io.ReadCloser, error) {
	gURL := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("GET", gURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to create a new GET request(%q), error: %w", gURL, err)
	}
	req.Header.Set("Cookie", ctl.cookie)
	q := url.Values{}
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to send a GET request(%q), error: %w", gURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read http response body, error: %w", err)
		}
		return nil, fmt.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

// post sends a POST client request.
func (ctl *controller) post(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("fail to create a new POST request(%q), error: %w", url, err)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to send a POST request(%q), error: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read http response body, error: %w", err)
		}
		return nil, fmt.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

// patch sends a PATCH client request.
func (ctl *controller) patch(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return nil, fmt.Errorf("fail to create a new PATCH request(%q), error: %w", url, err)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to send a PATCH request(%q), error: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read http response body, error: %w", err)
		}
		return nil, fmt.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

func (ctl *controller) delete(shortURL string, body io.Reader) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s%s", ctl.apiURL, shortURL)
	req, err := http.NewRequest("DELETE", url, body)
	if err != nil {
		return nil, fmt.Errorf("fail to create a new DELETE request(%q), error: %w", url, err)
	}
	req.Header.Set("Cookie", ctl.cookie)
	resp, err := ctl.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to send a DELETE request(%q), error: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read http response body, error: %w", err)
		}
		return nil, fmt.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
	}
	return resp.Body, nil
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
		return nil, fmt.Errorf("fail to unmarshal get project response, error: %w", err)
	}
	for _, p := range ps {
		project, ok := p.(*api.Project)
		if !ok {
			return nil, fmt.Errorf("fail to convert project")
		}
		projects = append(projects, project)
	}
	return projects, nil
}

// createProject creates an project.
func (ctl *controller) createProject(projectCreate api.ProjectCreate) (*api.Project, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &projectCreate); err != nil {
		return nil, fmt.Errorf("failed to marshal project create, error: %w", err)
	}

	body, err := ctl.post("/project", buf)
	if err != nil {
		return nil, err
	}

	project := new(api.Project)
	if err = jsonapi.UnmarshalPayload(body, project); err != nil {
		return nil, fmt.Errorf("fail to unmarshal post project response, error: %w", err)
	}
	return project, nil
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
		return nil, fmt.Errorf("fail to unmarshal get environment response, error %w", err)
	}
	for _, p := range ps {
		environment, ok := p.(*api.Environment)
		if !ok {
			return nil, fmt.Errorf("fail to convert environment")
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
	return nil, fmt.Errorf("unable to find environment %q", name)
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
	body, err := ctl.get("/database", params)
	if err != nil {
		return nil, err
	}

	var databases []*api.Database
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Database)))
	if err != nil {
		return nil, fmt.Errorf("fail to unmarshal get database response, error %w", err)
	}
	for _, p := range ps {
		database, ok := p.(*api.Database)
		if !ok {
			return nil, fmt.Errorf("fail to convert database")
		}
		databases = append(databases, database)
	}
	return databases, nil
}

func (ctl *controller) setLicense() error {
	// Switch plan to increase instance limit.
	license, err := fs.ReadFile(fakeFS, "fake/license")
	if err != nil {
		return fmt.Errorf("failed to read fake license, error: %w", err)
	}
	err = ctl.switchPlan(&enterpriseAPI.SubscriptionPatch{
		License: string(license),
	})
	if err != nil {
		return fmt.Errorf("failed to switch plan, error: %w", err)
	}
	return nil
}

func (ctl *controller) switchPlan(patch *enterpriseAPI.SubscriptionPatch) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, patch); err != nil {
		return fmt.Errorf("failed to marshal subscription patch, error: %w", err)
	}

	_, err := ctl.patch("/subscription", buf)
	if err != nil {
		return err
	}

	return nil
}

// addInstance adds an instance.
func (ctl *controller) addInstance(instanceCreate api.InstanceCreate) (*api.Instance, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &instanceCreate); err != nil {
		return nil, fmt.Errorf("failed to marshal instance create, error: %w", err)
	}

	body, err := ctl.post("/instance", buf)
	if err != nil {
		return nil, err
	}

	instance := new(api.Instance)
	if err = jsonapi.UnmarshalPayload(body, instance); err != nil {
		return nil, fmt.Errorf("fail to unmarshal post instance response, error: %w", err)
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
		return nil, fmt.Errorf("fail to unmarshal get migration history response, error %w", err)
	}
	for _, h := range hs {
		history, ok := h.(*api.MigrationHistory)
		if !ok {
			return nil, fmt.Errorf("fail to convert migration history")
		}
		histories = append(histories, history)
	}
	return histories, nil
}

// createIssue creates an issue.
func (ctl *controller) createIssue(issueCreate api.IssueCreate) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issueCreate); err != nil {
		return nil, fmt.Errorf("failed to marshal issue create, error: %w", err)
	}

	body, err := ctl.post("/issue", buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, fmt.Errorf("fail to unmarshal post issue response, error: %w", err)
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
		return nil, fmt.Errorf("fail to unmarshal get issue response, error %w", err)
	}
	return issue, nil
}

// getIssue gets the issue with given ID.
func (ctl *controller) getIssues(issueFind api.IssueFind) ([]*api.Issue, error) {
	params := make(map[string]string)
	if issueFind.ProjectID != nil {
		params["project"] = fmt.Sprintf("%d", *issueFind.ProjectID)
	}
	if issueFind.StatusList != nil && len(*issueFind.StatusList) > 0 {
		var sl []string
		for _, status := range *issueFind.StatusList {
			sl = append(sl, string(status))
		}
		params["status"] = strings.Join(sl, ",")
	}
	body, err := ctl.get("/issue", params)
	if err != nil {
		return nil, err
	}

	var issues []*api.Issue
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Issue)))
	if err != nil {
		return nil, fmt.Errorf("fail to unmarshal get issue response, error %w", err)
	}
	for _, p := range ps {
		issue, ok := p.(*api.Issue)
		if !ok {
			return nil, fmt.Errorf("fail to convert issue")
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

// patchIssue patches the issue with given ID.
func (ctl *controller) patchIssueStatus(issueStatusPatch api.IssueStatusPatch) (*api.Issue, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &issueStatusPatch); err != nil {
		return nil, fmt.Errorf("failed to marshal issue status patch, error: %w", err)
	}

	body, err := ctl.patch(fmt.Sprintf("/issue/%d/status", issueStatusPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	issue := new(api.Issue)
	if err = jsonapi.UnmarshalPayload(body, issue); err != nil {
		return nil, fmt.Errorf("fail to unmarshal patch issue status patch response, error: %w", err)
	}
	return issue, nil
}

// patchTaskStatus patches the status of a task in the pipeline stage.
func (ctl *controller) patchTaskStatus(taskStatusPatch api.TaskStatusPatch, pipelineID int) (*api.Task, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &taskStatusPatch); err != nil {
		return nil, fmt.Errorf("failed to marshal patchTaskStatus, error: %w", err)
	}

	body, err := ctl.patch(fmt.Sprintf("/pipeline/%d/task/%d/status", pipelineID, taskStatusPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	task := new(api.Task)
	if err = jsonapi.UnmarshalPayload(body, task); err != nil {
		return nil, fmt.Errorf("fail to unmarshal patchTaskStatus response, error: %w", err)
	}
	return task, nil
}

// patchStageAllTaskStatus patches the status of all tasks in the pipeline stage.
func (ctl *controller) patchStageAllTaskStatus(stageAllTaskStatusPatch api.StageAllTaskStatusPatch, pipelineID int) ([]*api.Task, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &stageAllTaskStatusPatch); err != nil {
		return nil, fmt.Errorf("failed to marshal StageAllTaskStatusPatch, error: %w", err)
	}

	body, err := ctl.patch(fmt.Sprintf("/pipeline/%d/stage/%d/status", pipelineID, stageAllTaskStatusPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	var tasks []*api.Task
	untypedTasks, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Task)))
	if err != nil {
		return nil, fmt.Errorf("fail to unmarshal get tasks response, error: %w", err)
	}
	for _, t := range untypedTasks {
		task, ok := t.(*api.Task)
		if !ok {
			return nil, fmt.Errorf("fail to convert task")
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// approveIssueNext approves the next pending approval task.
func (ctl *controller) approveIssueNext(issue *api.Issue) error {
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPendingApproval {
				if _, err := ctl.patchTaskStatus(
					api.TaskStatusPatch{
						ID:     task.ID,
						Status: api.TaskPending,
					},
					issue.Pipeline.ID); err != nil {

					return fmt.Errorf("failed to patch task status for task %d, error: %w", task.ID, err)
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
		if _, err := ctl.patchStageAllTaskStatus(
			api.StageAllTaskStatusPatch{
				ID:     stageID,
				Status: api.TaskPending,
			},
			issue.Pipeline.ID,
		); err != nil {
			return fmt.Errorf("failed to patch task status for stage %d, error: %w", stageID, err)
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
				return api.TaskFailed, fmt.Errorf("pipeline task %v failed runs: %v", task.ID, strings.Join(runs, ", "))
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

// waitIssuePipelineImpl waits for the tasks in pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipelineTaskImpl(id int, approveFunc func(issue *api.Issue) error, approveOnce bool) (api.TaskStatus, error) {
	// Sleep for two seconds between issues so that we don't get migration version conflict because we are using second-level timestamp for the version string. We choose sleep because it mimics the user's behavior.
	time.Sleep(2 * time.Second)

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
		return nil, fmt.Errorf("failed to marshal sqlExecute, error: %w", err)
	}

	body, err := ctl.post("/sql/execute", buf)
	if err != nil {
		return nil, err
	}

	sqlResultSet := new(api.SQLResultSet)
	if err = jsonapi.UnmarshalPayload(body, sqlResultSet); err != nil {
		return nil, fmt.Errorf("fail to unmarshal sqlResultSet response, error: %w", err)
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
		return "", fmt.Errorf("failed to execute SQL, error: %v", err)
	}
	if sqlResultSet.Error != "" {
		return "", fmt.Errorf("expect SQL result has no error, got %q", sqlResultSet.Error)
	}
	return sqlResultSet.Data, nil
}

// createVCS creates a VCS.
func (ctl *controller) createVCS(vcsCreate api.VCSCreate) (*api.VCS, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &vcsCreate); err != nil {
		return nil, fmt.Errorf("failed to marshal vcsCreate, error: %w", err)
	}

	body, err := ctl.post("/vcs", buf)
	if err != nil {
		return nil, err
	}

	vcs := new(api.VCS)
	if err = jsonapi.UnmarshalPayload(body, vcs); err != nil {
		return nil, fmt.Errorf("fail to unmarshal vcs response, error: %w", err)
	}
	return vcs, nil
}

// createRepository creates a repository.
func (ctl *controller) createRepository(repositoryCreate api.RepositoryCreate) (*api.Repository, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &repositoryCreate); err != nil {
		return nil, fmt.Errorf("failed to marshal repositoryCreate, error: %w", err)
	}

	body, err := ctl.post(fmt.Sprintf("/project/%d/repository", repositoryCreate.ProjectID), buf)
	if err != nil {
		return nil, err
	}

	repository := new(api.Repository)
	if err = jsonapi.UnmarshalPayload(body, repository); err != nil {
		return nil, fmt.Errorf("fail to unmarshal repository response, error: %w", err)
	}
	return repository, nil
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
		return fmt.Errorf("failed to construct database creation issue CreateContext payload, error: %w", err)
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
		return fmt.Errorf("failed to create database creation issue, error: %v", err)
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return fmt.Errorf("issue %v pipeline %v is supposed to be pending manual approval", issue.ID, issue.Pipeline.ID)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		return fmt.Errorf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		return fmt.Errorf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}
	issue, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	if err != nil {
		return fmt.Errorf("failed to patch issue status %v to done, error: %v", issue.ID, err)
	}
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
		return fmt.Errorf("failed to construct database creation issue CreateContext payload, error: %w", err)
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
		return fmt.Errorf("failed to create database creation issue, error: %v", err)
	}
	if status, _ := getNextTaskStatus(issue); status != api.TaskPendingApproval {
		return fmt.Errorf("issue %v pipeline %v is supposed to be pending manual approval", issue.ID, issue.Pipeline.ID)
	}
	status, err := ctl.waitIssuePipeline(issue.ID)
	if err != nil {
		return fmt.Errorf("failed to wait for issue %v pipeline %v, error: %v", issue.ID, issue.Pipeline.ID, err)
	}
	if status != api.TaskDone {
		return fmt.Errorf("issue %v pipeline %v is expected to finish with status done, got %v", issue.ID, issue.Pipeline.ID, status)
	}
	issue, err = ctl.patchIssueStatus(api.IssueStatusPatch{
		ID:     issue.ID,
		Status: api.IssueDone,
	})
	if err != nil {
		return fmt.Errorf("failed to patch issue status %v to done, error: %v", issue.ID, err)
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
		return "", fmt.Errorf("failed to marshal labels %+v, error %v", labelList, err)
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
		return nil, fmt.Errorf("fail to unmarshal get label response, error: %w", err)
	}
	for _, lk := range lks {
		labelKey, ok := lk.(*api.LabelKey)
		if !ok {
			return nil, fmt.Errorf("fail to convert label key")
		}
		labelKeys = append(labelKeys, labelKey)
	}
	return labelKeys, nil
}

// patchLabelKey patches the label key with given ID.
func (ctl *controller) patchLabelKey(labelKeyPatch api.LabelKeyPatch) (*api.LabelKey, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &labelKeyPatch); err != nil {
		return nil, fmt.Errorf("failed to marshal label key patch, error: %w", err)
	}

	body, err := ctl.patch(fmt.Sprintf("/label/%d", labelKeyPatch.ID), buf)
	if err != nil {
		return nil, err
	}

	labelKey := new(api.LabelKey)
	if err = jsonapi.UnmarshalPayload(body, labelKey); err != nil {
		return nil, fmt.Errorf("fail to unmarshal patch label key response, error: %w", err)
	}
	return labelKey, nil
}

// addLabelValues adds values to an existing label key.
func (ctl *controller) addLabelValues(key string, values []string) error {
	labelKeys, err := ctl.getLabels()
	if err != nil {
		return fmt.Errorf("failed to get labels, error: %w", err)
	}
	var labelKey *api.LabelKey
	for _, lk := range labelKeys {
		if lk.Key == key {
			labelKey = lk
			break
		}
	}
	if labelKey == nil {
		return fmt.Errorf("failed to find label with key %q", key)
	}
	var valueList []string
	valueList = append(valueList, labelKey.ValueList...)
	valueList = append(valueList, values...)
	_, err = ctl.patchLabelKey(api.LabelKeyPatch{
		ID:        labelKey.ID,
		ValueList: valueList,
	})
	if err != nil {
		return fmt.Errorf("failed to patch label key for key %q ID %d values %+v, error: %w", key, labelKey.ID, valueList, err)
	}
	return nil
}

// upsertDeploymentConfig upserts the deployment configuration for a project.
func (ctl *controller) upsertDeploymentConfig(deploymentConfigUpsert api.DeploymentConfigUpsert, deploymentSchedule api.DeploymentSchedule) (*api.DeploymentConfig, error) {
	scheduleBuf, err := json.Marshal(&deploymentSchedule)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment schedule, error: %w", err)
	}
	deploymentConfigUpsert.Payload = string(scheduleBuf)

	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &deploymentConfigUpsert); err != nil {
		return nil, fmt.Errorf("failed to marshal deployment config upsert, error: %w", err)
	}

	body, err := ctl.patch(fmt.Sprintf("/project/%d/deployment", deploymentConfigUpsert.ProjectID), buf)
	if err != nil {
		return nil, err
	}

	deploymentConfig := new(api.DeploymentConfig)
	if err = jsonapi.UnmarshalPayload(body, deploymentConfig); err != nil {
		return nil, fmt.Errorf("fail to unmarshal upsert deployment config response, error: %w", err)
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
		return fmt.Errorf("failed to marshal backupSetting, error: %w", err)
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
		return nil, fmt.Errorf("failed to marshal backupCreate, error: %w", err)
	}

	body, err := ctl.post(fmt.Sprintf("/database/%d/backup", backupCreate.DatabaseID), buf)
	if err != nil {
		return nil, err
	}

	backup := new(api.Backup)
	if err = jsonapi.UnmarshalPayload(body, backup); err != nil {
		return nil, fmt.Errorf("fail to unmarshal backup response, error: %w", err)
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
		return nil, fmt.Errorf("fail to unmarshal get backup response, error: %w", err)
	}
	for _, p := range ps {
		backup, ok := p.(*api.Backup)
		if !ok {
			return nil, fmt.Errorf("fail to convert backup")
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
			return fmt.Errorf("backup %v for database %v not found", backupID, databaseID)
		}
		switch backup.Status {
		case api.BackupStatusDone:
			return nil
		case api.BackupStatusFailed:
			return fmt.Errorf("backup %v for database %v failed", backupID, databaseID)
		}
	}
	// Ideally, this should never happen because the ticker will not stop till the backup is finished.
	return fmt.Errorf("failed to wait for backup as this condition should never be reached")
}

// createSheet creates a sheet.
func (ctl *controller) createSheet(sheetCreate api.SheetCreate) (*api.Sheet, error) {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &sheetCreate); err != nil {
		return nil, fmt.Errorf("failed to marshal sheetCreate, error: %w", err)
	}

	body, err := ctl.post("/sheet", buf)
	if err != nil {
		return nil, err
	}

	sheet := new(api.Sheet)
	if err = jsonapi.UnmarshalPayload(body, sheet); err != nil {
		return nil, fmt.Errorf("fail to unmarshal sheet response, error: %w", err)
	}
	return sheet, nil
}

// listSheets lists sheets for a database.
func (ctl *controller) listSheets(sheetFind api.SheetFind) ([]*api.Sheet, error) {
	params := map[string]string{}
	if sheetFind.ProjectID != nil {
		params["projectId"] = strconv.Itoa(*sheetFind.ProjectID)
	}
	if sheetFind.DatabaseID != nil {
		params["databaseId"] = strconv.Itoa(*sheetFind.DatabaseID)
	}

	body, err := ctl.get("/sheet/my", params)
	if err != nil {
		return nil, err
	}

	var sheets []*api.Sheet
	ps, err := jsonapi.UnmarshalManyPayload(body, reflect.TypeOf(new(api.Sheet)))
	if err != nil {
		return nil, fmt.Errorf("fail to unmarshal get sheet response, error: %w", err)
	}
	for _, p := range ps {
		sheet, ok := p.(*api.Sheet)
		if !ok {
			return nil, fmt.Errorf("fail to convert sheet")
		}
		sheets = append(sheets, sheet)
	}
	return sheets, nil
}

// syncSheet syncs sheets with project.
func (ctl *controller) syncSheet(projectID int) error {
	_, err := ctl.post(fmt.Sprintf("/sheet/project/%d/sync", projectID), nil)
	return err
}

func (ctl *controller) createDataSource(dataSourceCreate api.DataSourceCreate) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &dataSourceCreate); err != nil {
		return fmt.Errorf("failed to marshal dataSourceCreate, error: %w", err)
	}

	body, err := ctl.post(fmt.Sprintf("/database/%d/data-source", dataSourceCreate.DatabaseID), buf)
	if err != nil {
		return err
	}

	dataSource := new(api.DataSource)
	if err = jsonapi.UnmarshalPayload(body, dataSource); err != nil {
		return fmt.Errorf("fail to unmarshal dataSource response, error: %w", err)
	}
	return nil
}

// upsertPolicy upserts the policy.
func (ctl *controller) upsertPolicy(policyUpsert api.PolicyUpsert) error {
	buf := new(bytes.Buffer)
	if err := jsonapi.MarshalPayload(buf, &policyUpsert); err != nil {
		return fmt.Errorf("failed to marshal policyUpsert, error: %w", err)
	}

	body, err := ctl.patch(fmt.Sprintf("/policy/environment/%d?type=%s", policyUpsert.EnvironmentID, policyUpsert.Type), buf)
	if err != nil {
		return err
	}

	policy := new(api.Policy)
	if err = jsonapi.UnmarshalPayload(body, policy); err != nil {
		return fmt.Errorf("fail to unmarshal policy response, error: %w", err)
	}
	return nil
}

// deletePolicy deletes the archived policy.
func (ctl *controller) deletePolicy(policyDelete api.PolicyDelete) error {
	_, err := ctl.delete(fmt.Sprintf("/policy/environment/%d?type=%s", policyDelete.EnvironmentID, policyDelete.Type), new(bytes.Buffer))
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
			return nil, fmt.Errorf("the status of issue %v is not pending approval", id)
		}

		result, yes, err := ctl.sqlReviewTaskCheckRunFinished(issue)
		if err != nil {
			return nil, fmt.Errorf("failed to get SQL review result for issue %v: %w", id, err)
		}
		if yes {
			return result, nil
		}
	}
	return nil, nil
}

// setDefaultSQLReviewRulePayload sets the default payload for this rule.
func setDefaultSQLReviewRulePayload(ruleTp advisor.SQLReviewRuleType) (string, error) {
	var payload []byte
	var err error
	switch ruleTp {
	case advisor.SchemaRuleMySQLEngine:
	case advisor.SchemaRuleStatementNoSelectAll:
	case advisor.SchemaRuleStatementRequireWhere:
	case advisor.SchemaRuleStatementNoLeadingWildcardLike:
	case advisor.SchemaRuleTableRequirePK:
	case advisor.SchemaRuleTableNoFK:
	case advisor.SchemaRuleColumnNotNull:
	case advisor.SchemaRuleSchemaBackwardCompatibility:
	case advisor.SchemaRuleTableDropNamingConvention:
		payload, err = json.Marshal(advisor.NamingRulePayload{
			Format: "_delete$",
		})
	case advisor.SchemaRuleTableNaming:
		fallthrough
	case advisor.SchemaRuleColumnNaming:
		payload, err = json.Marshal(advisor.NamingRulePayload{
			Format: "^[a-z]+(_[a-z]+)*$",
		})
	case advisor.SchemaRuleIDXNaming:
		payload, err = json.Marshal(advisor.NamingRulePayload{
			Format: "^idx_{{table}}_{{column_list}}$",
		})
	case advisor.SchemaRulePKNaming:
		payload, err = json.Marshal(advisor.NamingRulePayload{
			Format: "^pk_{{table}}_{{column_list}}$",
		})
	case advisor.SchemaRuleUKNaming:
		payload, err = json.Marshal(advisor.NamingRulePayload{
			Format: "^uk_{{table}}_{{column_list}}$",
		})
	case advisor.SchemaRuleFKNaming:
		payload, err = json.Marshal(advisor.NamingRulePayload{
			Format: "^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
		})
	case advisor.SchemaRuleRequiredColumn:
		payload, err = json.Marshal(advisor.RequiredColumnRulePayload{
			ColumnList: []string{
				"id",
				"created_ts",
				"updated_ts",
				"creator_id",
				"updater_id",
			},
		})
	default:
		return "", fmt.Errorf("unknown SQL review type for default payload: %s", ruleTp)
	}

	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// prodTemplateSQLReviewPolicy returns the default SQL review policy.
func prodTemplateSQLReviewPolicy() (string, error) {
	policy := advisor.SQLReviewPolicy{
		Name: "Prod",
		RuleList: []*advisor.SQLReviewRule{
			{
				Type:  advisor.SchemaRuleMySQLEngine,
				Level: advisor.SchemaRuleLevelError,
			},
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
				Type:  advisor.SchemaRuleRequiredColumn,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleColumnNotNull,
				Level: advisor.SchemaRuleLevelWarning,
			},
			{
				Type:  advisor.SchemaRuleSchemaBackwardCompatibility,
				Level: advisor.SchemaRuleLevelWarning,
			},
		},
	}

	for _, rule := range policy.RuleList {
		payload, err := setDefaultSQLReviewRulePayload(rule.Type)
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
