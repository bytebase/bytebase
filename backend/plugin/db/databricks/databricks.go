package databricks

import (
	"context"
	"database/sql"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/proto/generated-go/store"
	v1 "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(store.Engine_DATABRICKS, NewDatabricksDriver)
}

var _ db.Driver = (*DatabricksDriver)(nil)

type DatabricksDriver struct {
	curCatalog string
	client     *databricks.WorkspaceClient
}

func NewDatabricksDriver(db.DriverConfig) db.Driver {
	return &DatabricksDriver{}
}

// Each Databricks driver is associated with a single Databricks Workspace (Workspace -> catalog -> schema -> table).
func (d *DatabricksDriver) Open(ctx context.Context, dbType store.Engine, config db.ConnectionConfig) (db.Driver, error) {
	databricksConfig := &databricks.Config{
		Host: config.Host,
	}
	// Support Databricks native authentication.
	// ref: https://github.com/databricks/databricks-sdk-go?tab=readme-ov-file#databricks-native-authentication
	if config.AuthenticationPrivateKey != "" {
		// Token.
		databricksConfig.Token = config.AuthenticationPrivateKey
	} else {
		// Basic username and password.
		databricksConfig.Username = config.Username
		databricksConfig.Password = config.Password
		databricksConfig.AccountID = config.AccountID
	}
	client, err := databricks.NewWorkspaceClient(databricksConfig)
	if err != nil {
		return nil, err
	}
	d.client = client
	d.curCatalog = config.Database
	return d, nil
}

func (d *DatabricksDriver) Close(ctx context.Context) error {
	return nil
}

func (d *DatabricksDriver) Ping(ctx context.Context) error {
	_, err := d.client.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{})
	if err != nil {
		return errors.Wrapf(err, "failed to ping instance")
	}
	return nil
}

func (d *DatabricksDriver) GetDB() *sql.DB {
	return nil
}

func (d *DatabricksDriver) GetType() store.Engine {
	return store.Engine_DATABRICKS
}

func (d *DatabricksDriver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1.QueryResult, error) {
	return nil, nil
}

func (d *DatabricksDriver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1.QueryResult, error) {
	return nil, nil
}

func (d *DatabricksDriver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return 0, nil
}

func (d *DatabricksDriver) CheckSlowQueryLogEnabled(ctx context.Context) error {
	return nil
}
