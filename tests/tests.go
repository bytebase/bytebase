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
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/server/cmd"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/tests/fake"
	"github.com/google/jsonapi"
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
	server    *server.Server
	client    *http.Client
	cookie    string
	gitlab    *fake.GitLab
	apiURL    string
	gitURL    string
	gitAPIURL string
}

func getTestPort(testName string) int {
	// The port should be incremented by 3 for bytebase, Postgres, and GitLab.
	switch testName {
	case "TestServiceRestart":
		return 1234
	case "TestSchemaAndDataUpdate":
		return 1237
	case "TestVCS":
		return 1240
	case "TestTenant":
		return 1243
	case "TestTenantVCS":
		return 1246
	case "TestTenantDatabaseNameTemplate":
		return 1249
	case "TestGhostSimpleNoop":
		return 1252
	case "TestBackupRestoreBasic":
		return 1255
	case "TestPITR":
		return 1258
	case "TestTenantVCSDatabaseNameTemplate":
		return 1261
	}
	panic(fmt.Sprintf("test %q doesn't have assigned port, please set it in getTestPort()", testName))
}

// StartServer starts the main server.
func (ctl *controller) StartServer(ctx context.Context, dataDir string, port int) error {
	// start main server.
	logger, lvl, err := cmd.GetLogger()
	if err != nil {
		return fmt.Errorf("failed to get logger, error: %w", err)
	}
	defer logger.Sync()

	profile := cmd.GetTestProfile(dataDir, port)
	ctl.server, err = server.NewServer(ctx, profile, logger, lvl)
	if err != nil {
		return err
	}

	ctl.apiURL = fmt.Sprintf("http://localhost:%d/api", port)

	// set up gitlab.
	gitlabPort := port + 2
	ctl.gitlab = fake.NewGitLab(gitlabPort)
	ctl.gitURL = fmt.Sprintf("http://localhost:%d", gitlabPort)
	ctl.gitAPIURL = fmt.Sprintf("%s/api/v4", ctl.gitURL)

	errChan := make(chan error, 1)

	go func() {
		if err := ctl.server.Run(ctx); err != nil {
			errChan <- fmt.Errorf("failed to run main server, error: %w", err)
		}
	}()

	go func() {
		if err := ctl.gitlab.Run(); err != nil {
			errChan <- fmt.Errorf("failed to run gitlab server, error: %w", err)
		}
	}()

	if err := waitForServerStart(ctl.server, errChan); err != nil {
		return fmt.Errorf("failed to wait for server to start, error: %w", err)
	}
	if err := waitForGitLabStart(ctl.gitlab, errChan); err != nil {
		return fmt.Errorf("failed to wait for gitlab to start, error: %w", err)
	}

	// initialize controller clients.
	ctl.client = &http.Client{}

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

func waitForGitLabStart(g *fake.GitLab, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if g.Echo == nil {
				continue
			}
			addr := g.Echo.ListenerAddr()
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

func (ctl *controller) Close(ctx context.Context) error {
	var e error
	if ctl.server != nil {
		e = ctl.server.Shutdown(ctx)
	}
	if ctl.gitlab != nil {
		if err := ctl.gitlab.Close(); err != nil {
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
func (ctl *controller) provisionSQLiteInstance(rootDir, name string) (string, error) {
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

// getAggregatedTaskStatus gets pipeline status.
func getAggregatedTaskStatus(issue *api.Issue) (api.TaskStatus, error) {
	running := false
	for _, stage := range issue.Pipeline.StageList {
		for _, task := range stage.TaskList {
			switch task.Status {
			case api.TaskPendingApproval:
				return api.TaskPendingApproval, nil
			case api.TaskFailed:
				var runs []string
				for _, run := range task.TaskRunList {
					runs = append(runs, fmt.Sprintf("%+v", run))
				}
				return api.TaskFailed, fmt.Errorf("pipeline task %v failed runs: %v", task.ID, strings.Join(runs, ", "))
			case api.TaskCanceled:
				return api.TaskCanceled, nil
			case api.TaskRunning:
				running = true
			case api.TaskPending:
				running = true
			}
		}
	}
	if running {
		return api.TaskRunning, nil
	}
	return api.TaskDone, nil
}

// waitIssuePipeline waits for pipeline to finish and approves tasks when necessary.
func (ctl *controller) waitIssuePipeline(id int) (api.TaskStatus, error) {
	// Sleep for two seconds between issues so that we don't get migration version conflict because we are using second-level timestamp for the version string. We choose sleep because it mimics the user's behavior.
	time.Sleep(2 * time.Second)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		issue, err := ctl.getIssue(id)
		if err != nil {
			return api.TaskFailed, err
		}

		status, err := getAggregatedTaskStatus(issue)
		if err != nil {
			return status, err
		}
		switch status {
		case api.TaskPendingApproval:
			if err := ctl.approveIssueNext(issue); err != nil {
				return api.TaskFailed, err
			}
		case api.TaskFailed:
			return status, err
		case api.TaskDone:
			return status, err
		case api.TaskCanceled:
			return status, err
		case api.TaskPending:
		case api.TaskRunning:
			// no-op, keep waiting
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

func (ctl *controller) createDatabase(project *api.Project, instance *api.Instance, databaseName string, labelMap map[string]string) error {
	labels, err := marshalLabels(labelMap, instance.Environment.Name)
	if err != nil {
		return err
	}

	createContext, err := json.Marshal(&api.CreateDatabaseContext{
		InstanceID:   instance.ID,
		DatabaseName: databaseName,
		Labels:       labels,
	})
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
	if status, _ := getAggregatedTaskStatus(issue); status != api.TaskPendingApproval {
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
	if status, _ := getAggregatedTaskStatus(issue); status != api.TaskPendingApproval {
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
	valueList := append(labelKey.ValueList, values...)
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
func (ctl *controller) listSheets(databaseID int) ([]*api.Sheet, error) {
	body, err := ctl.get(fmt.Sprintf("/sheet/my?databaseId=%d", databaseID), nil)
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
