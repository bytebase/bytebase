package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/bin/server/cmd"
	"github.com/bytebase/bytebase/tests/fake"
	"github.com/google/jsonapi"
)

var (
	port    = 1234
	rootURL = fmt.Sprintf("http://localhost:%d/api", port)
	gitPort = 1235
)

type controller struct {
	main   *cmd.Main
	client *http.Client
	cookie string
	gitlab *fake.GitLab
}

// StartMain starts the main server.
func (ctl *controller) StartMain(ctx context.Context, dataDir string) error {
	// start main server.
	logger, err := cmd.GetLogger()
	if err != nil {
		return fmt.Errorf("failed to get logger, error: %w", err)
	}
	defer logger.Sync()
	profile := cmd.GetTestProfile(dataDir)
	ctl.main = cmd.NewMain(profile, logger)
	ctl.gitlab = fake.NewGitLab(gitPort)

	errChan := make(chan error, 1)
	go func() {
		if err := ctl.main.Run(ctx); err != nil {
			errChan <- fmt.Errorf("failed to run main server, error: %w", err)
		}
	}()
	go func() {
		if err := ctl.gitlab.Run(); err != nil {
			errChan <- fmt.Errorf("failed to run gitlab server, error: %w", err)
		}
	}()

	if err := waitForServerStart(ctl.main, errChan); err != nil {
		return fmt.Errorf("failed to wait for server to start, error: %w", err)
	}
	if err := waitForGitLabStart(ctl.gitlab, errChan); err != nil {
		return fmt.Errorf("failed to wait for gitlab to start, error: %w", err)
	}

	// initialize controller clients.
	ctl.client = &http.Client{}

	return nil
}

func waitForServerStart(m *cmd.Main, errChan <-chan error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.GetServer() == nil {
				continue
			}
			e := m.GetServer().GetEcho()
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

func (ctl *controller) Close() error {
	var e error
	if ctl.main != nil {
		e = ctl.main.Close()
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
		fmt.Sprintf("%s/auth/login/BYTEBASE", rootURL),
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
	gURL := fmt.Sprintf("%s%s", rootURL, shortURL)
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
	url := fmt.Sprintf("%s%s", rootURL, shortURL)
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
	url := fmt.Sprintf("%s%s", rootURL, shortURL)
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
				return api.TaskFailed, fmt.Errorf("pipeline task %v failed payload %q", task.ID, task.Payload)
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
