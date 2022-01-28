package tests

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/google/jsonapi"
)

var (
	port    = 1234
	rootURL = fmt.Sprintf("http://localhost:%d/api", port)
)

type controller struct {
	main   *cmd.Main
	client *http.Client
	cookie string
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

	errChan := make(chan error, 1)
	go func() {
		if err := ctl.main.Run(ctx); err != nil {
			errChan <- fmt.Errorf("failed to run main server, error: %w", err)
		}
	}()

	if err := waitForServerStart(ctl.main, errChan); err != nil {
		return fmt.Errorf("failed to wait for server to start, error: %w", err)
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

func (ctl *controller) Close() error {
	if ctl.main != nil {
		return ctl.main.Close()
	}
	return nil
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

// getDatabaseCreateIssueCreateContext gets a create context for create database issue.
func getDatabaseCreateIssueCreateContext(instanceID int, databaseName string) (string, error) {
	m := &api.CreateDatabaseContext{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		Labels:       "",
	}
	createContext, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to construct issue create context payload, error: %w", err)
	}
	return string(createContext), nil
}
