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
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	// Import pg driver.
	// init() in pgx/v5/stdlib will register it's pgx driver.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	component "github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/server"
)

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
	grpcConn                     *grpc.ClientConn
	issueServiceClient           v1pb.IssueServiceClient
	rolloutServiceClient         v1pb.RolloutServiceClient
	planServiceClient            v1pb.PlanServiceClient
	orgPolicyServiceClient       v1pb.OrgPolicyServiceClient
	reviewConfigServiceClient    v1pb.ReviewConfigServiceClient
	projectServiceClient         v1pb.ProjectServiceClient
	databaseGroupServiceClient   v1pb.DatabaseGroupServiceClient
	authServiceClient            v1pb.AuthServiceClient
	userServiceClient            v1pb.UserServiceClient
	settingServiceClient         v1pb.SettingServiceClient
	instanceServiceClient        v1pb.InstanceServiceClient
	databaseServiceClient        v1pb.DatabaseServiceClient
	databaseCatalogServiceClient v1pb.DatabaseCatalogServiceClient
	sheetServiceClient           v1pb.SheetServiceClient
	sqlServiceClient             v1pb.SQLServiceClient
	subscriptionServiceClient    v1pb.SubscriptionServiceClient
	actuatorServiceClient        v1pb.ActuatorServiceClient
	workspaceServiceClient       v1pb.WorkspaceServiceClient

	cookie  string
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
		},
		ProjectId: projectID,
	})
	if err != nil {
		return nil, err
	}
	ctl.project = project

	return metaCtx, nil
}

func (ctl *controller) initWorkspaceProfile(ctx context.Context) error {
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, &v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: fmt.Sprintf("settings/%s", base.SettingWorkspaceProfile),
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceProfileSettingValue{
					WorkspaceProfileSettingValue: &v1pb.WorkspaceProfileSetting{
						ExternalUrl:    ctl.profile.ExternalURL,
						DisallowSignup: false,
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{
				"value.workspace_profile_setting_value.disallow_signup",
				"value.workspace_profile_setting_value.external_url",
			},
		},
	})
	return err
}

// GetTestProfileWithExternalPg will return a profile for testing with external Postgres.
// We require port as an argument of GetTestProfile so that test can run in parallel in different ports,
// pgURL for connect to Postgres.
func getTestProfileWithExternalPg(port int, pgURL string) *component.Profile {
	return &component.Profile{
		Mode:               common.ReleaseModeDev,
		ExternalURL:        fmt.Sprintf("http://localhost:%d", port),
		Port:               port,
		SampleDatabasePort: 0,
		PgURL:              pgURL,
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
	ctl.client = &http.Client{}

	// initialize grpc connection.
	grpcConn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial grpc")
	}
	ctl.grpcConn = grpcConn
	ctl.issueServiceClient = v1pb.NewIssueServiceClient(ctl.grpcConn)
	ctl.rolloutServiceClient = v1pb.NewRolloutServiceClient(ctl.grpcConn)
	ctl.planServiceClient = v1pb.NewPlanServiceClient(ctl.grpcConn)
	ctl.orgPolicyServiceClient = v1pb.NewOrgPolicyServiceClient(ctl.grpcConn)
	ctl.reviewConfigServiceClient = v1pb.NewReviewConfigServiceClient(ctl.grpcConn)
	ctl.projectServiceClient = v1pb.NewProjectServiceClient(ctl.grpcConn)
	ctl.databaseGroupServiceClient = v1pb.NewDatabaseGroupServiceClient(ctl.grpcConn)
	ctl.authServiceClient = v1pb.NewAuthServiceClient(ctl.grpcConn)
	ctl.userServiceClient = v1pb.NewUserServiceClient(ctl.grpcConn)
	ctl.settingServiceClient = v1pb.NewSettingServiceClient(ctl.grpcConn)
	ctl.instanceServiceClient = v1pb.NewInstanceServiceClient(ctl.grpcConn)
	ctl.databaseServiceClient = v1pb.NewDatabaseServiceClient(ctl.grpcConn)
	ctl.databaseCatalogServiceClient = v1pb.NewDatabaseCatalogServiceClient(ctl.grpcConn)
	ctl.sheetServiceClient = v1pb.NewSheetServiceClient(ctl.grpcConn)
	ctl.sqlServiceClient = v1pb.NewSQLServiceClient(ctl.grpcConn)
	ctl.subscriptionServiceClient = v1pb.NewSubscriptionServiceClient(ctl.grpcConn)
	ctl.actuatorServiceClient = v1pb.NewActuatorServiceClient(ctl.grpcConn)
	ctl.workspaceServiceClient = v1pb.NewWorkspaceServiceClient(ctl.grpcConn)

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

func (ctl *controller) waitForHealthz(ctx context.Context) error {
	begin := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	timer := time.NewTimer(10 * time.Second)
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
	principal, err := ctl.userServiceClient.CreateUser(ctx, &v1pb.CreateUserRequest{
		User: &v1pb.User{
			Email:    "demo@example.com",
			Password: "1024bytebase",
			Title:    "demo",
			UserType: v1pb.UserType_USER,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "exist") {
		return "", err
	}
	resp, err := ctl.authServiceClient.Login(ctx, &v1pb.LoginRequest{
		Email:    "demo@example.com",
		Password: "1024bytebase",
	})
	if err != nil {
		return "", err
	}
	ctl.principalName = principal.Name
	ctl.cookie = fmt.Sprintf("access-token=%s", resp.Token)
	return resp.Token, nil
}
