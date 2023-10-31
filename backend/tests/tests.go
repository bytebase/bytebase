// Package tests provides integration tests for backend services.
package tests

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Import pg driver.
	// init() in pgx/v4/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/bytebase/bytebase/backend/common"
	component "github.com/bytebase/bytebase/backend/component/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/resources/postgres"
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
	profile                   component.Profile
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

	cookie  string
	project *v1pb.Project

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

	externalPgPort = time.Now().Second()*200 + 5000

	nextDatabaseNumber = 20210113

	// resourceDir is the shared resource directory.
	resourceDir string
	pgBinDir    string
	mysqlBinDir string
)

func getTestPort() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 2
	return p
}

func getTestPortForEmbeddedPg() int {
	mu.Lock()
	defer mu.Unlock()
	p := nextPort
	nextPort += 3
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
	if err := ctl.startMockServers(config.vcsProviderCreator); err != nil {
		return nil, err
	}

	pgMainURL := fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", postgres.TestPgUser, externalPgPort, "postgres", common.GetPostgresSocketDir())
	db, err := sql.Open("pgx", pgMainURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	databaseName := getTestDatabaseString()
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName)); err != nil {
		return nil, err
	}

	pgURL := fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", postgres.TestPgUser, externalPgPort, databaseName, common.GetPostgresSocketDir())
	serverPort := getTestPort()
	profile := getTestProfileWithExternalPg(config.dataDir, resourceDir, serverPort, postgres.TestPgUser, pgURL, config.skipOnboardingData)
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
				Type: v1pb.PolicyType_ROLLOUT_POLICY,
				Policy: &v1pb.Policy_RolloutPolicy{
					RolloutPolicy: &v1pb.RolloutPolicy{
						Automatic: false,
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
func getTestProfile(dataDir, resourceDir string, port int, readOnly bool) component.Profile {
	return component.Profile{
		Mode:                 common.ReleaseModeDev,
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
func getTestProfileWithExternalPg(dataDir, resourceDir string, port int, pgUser string, pgURL string, skipOnboardingData bool) component.Profile {
	return component.Profile{
		Mode:                       common.ReleaseModeDev,
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

	if err := ctl.waitForHealthz(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to wait for healthz")
	}
	authToken, err := ctl.signupAndLogin(ctx)
	if err != nil {
		return nil, err
	}

	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"Authorization",
		fmt.Sprintf("Bearer %s", authToken),
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

func (ctl *controller) waitForHealthz(ctx context.Context) error {
	begin := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	timer := time.NewTimer(30 * time.Second)
	defer ticker.Stop()
	defer timer.Stop()
	for {
		select {
		case <-ticker.C:
			_, err := ctl.actuatorServiceClient.GetActuatorInfo(ctx, &v1pb.GetActuatorInfoRequest{})
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

// signupAndLogin will signup and login as user demo@example.com.
func (ctl *controller) signupAndLogin(ctx context.Context) (string, error) {
	if _, err := ctl.authServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    "demo@example.com",
			Password: "1024",
			Title:    "demo",
			UserType: v1pb.UserType_USER,
		},
	}); err != nil && !strings.Contains(err.Error(), "exist") {
		return "", err
	}
	resp, err := ctl.authServiceClient.Login(ctx, &v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024",
	})
	if err != nil {
		return "", err
	}
	ctl.cookie = fmt.Sprintf("access-token=%s", resp.Token)
	return resp.Token, nil
}
