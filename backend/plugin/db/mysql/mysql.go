// Package mysql is the plugin for MySQL driver.
package mysql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	baseTableType = "BASE TABLE"
	viewTableType = "VIEW"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MYSQL, newDriver)
	db.Register(storepb.Engine_MARIADB, newDriver)
	db.Register(storepb.Engine_OCEANBASE, newDriver)
}

// Driver is the MySQL driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	connCfg       db.ConnectionConfig
	dbType        storepb.Engine
	dbBinDir      string
	db            *sql.DB
	databaseName  string
	sshClient     *ssh.Client

	// Called upon driver.Open() finishes.
	openCleanUp []func()
}

func newDriver(dc db.DriverConfig) db.Driver {
	return &Driver{
		dbBinDir: dc.DbBinDir,
	}
}

// Open opens a MySQL driver.
func (d *Driver) Open(ctx context.Context, dbType storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	defer func() {
		for _, f := range d.openCleanUp {
			f()
		}
	}()

	var dsn string
	var err error
	switch connCfg.AuthenticationType {
	case storepb.DataSourceOptions_GOOGLE_CLOUD_SQL_IAM:
		dsn, err = getCloudSQLConnection(ctx, connCfg)
	case storepb.DataSourceOptions_AWS_RDS_IAM:
		dsn, err = getRDSConnection(ctx, connCfg)
	default:
		dsn, err = d.getMySQLConnection(connCfg)
	}
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	d.dbType = dbType
	d.db = db
	// TODO(d): remove the work-around once we have clean-up the migration connection hack.
	db.SetConnMaxLifetime(2 * time.Hour)
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(15)
	d.connectionCtx = connCfg.ConnectionContext
	d.connCfg = connCfg
	d.databaseName = connCfg.Database

	return d, nil
}

func (d *Driver) getMySQLConnection(connCfg db.ConnectionConfig) (string, error) {
	protocol := "tcp"
	if strings.HasPrefix(connCfg.Host, "/") {
		protocol = "unix"
	}

	params := []string{"multiStatements=true", "maxAllowedPacket=0"}
	if connCfg.SSHConfig.Host != "" {
		sshClient, err := util.GetSSHClient(connCfg.SSHConfig)
		if err != nil {
			return "", err
		}
		d.sshClient = sshClient
		// Now we register the dialer with the ssh connection as a parameter.
		mysql.RegisterDialContext("mysql+tcp", func(_ context.Context, addr string) (net.Conn, error) {
			return sshClient.Dial("tcp", addr)
		})
		protocol = "mysql+tcp"
	}

	// TODO(zp): mysql and mysqlbinlog doesn't support SSL yet. We need to write certs to temp files and load them as CLI flags.
	tlsConfig, err := connCfg.TLSConfig.GetSslConfig()
	if err != nil {
		return "", errors.Wrap(err, "sql: tls config error")
	}
	tlsKey := uuid.NewString()
	if tlsConfig != nil {
		if err := mysql.RegisterTLSConfig(tlsKey, tlsConfig); err != nil {
			return "", errors.Wrap(err, "sql: failed to register tls config")
		}
		// TLS config is only used during sql.Open, so should be safe to deregister afterwards.
		d.openCleanUp = append(d.openCleanUp, func() { mysql.DeregisterTLSConfig(tlsKey) })
		params = append(params, fmt.Sprintf("tls=%s", tlsKey))
	}

	return fmt.Sprintf("%s:%s@%s(%s:%s)/%s?%s", connCfg.Username, connCfg.Password, protocol, connCfg.Host, connCfg.Port, connCfg.Database, strings.Join(params, "&")), nil
}

// AWS RDS connection with IAM require TLS connection.
//
// refs:
// https://github.com/aws/aws-sdk-go/issues/1248
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/mysql-ssl-connections.html
func registerRDSMysqlCerts(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem", nil)
	if err != nil {
		return errors.Wrapf(err, "failed to build request for rds cert")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	pem, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := resp.Body.Close(); err != nil {
		return errors.Wrapf(err, "failed to close response")
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return err
	}

	return mysql.RegisterTLSConfig("rds", &tls.Config{RootCAs: rootCertPool, InsecureSkipVerify: true})
}

// getRDSConnection returns the connection string with IAM for AWS RDS.
//
// refs:
// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.Connecting.Go.html
// https://repost.aws/knowledge-center/rds-mysql-access-denied
func getRDSConnection(ctx context.Context, connCfg db.ConnectionConfig) (string, error) {
	dbEndpoint := fmt.Sprintf("%s:%s", connCfg.Host, connCfg.Port)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "load aws config failed")
	}

	authenticationToken, err := auth.BuildAuthToken(
		ctx, dbEndpoint, connCfg.Region, connCfg.Username, cfg.Credentials)
	if err != nil {
		return "", errors.Wrap(err, "failed to create authentication token")
	}

	err = registerRDSMysqlCerts(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to register rds certs")
	}

	return fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=rds&allowCleartextPasswords=true",
		connCfg.Username, authenticationToken, dbEndpoint, connCfg.Database,
	), nil
}

func getCloudSQLConnection(ctx context.Context, connCfg db.ConnectionConfig) (string, error) {
	d, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return "", err
	}
	mysql.RegisterDialContext("cloudsqlconn",
		func(ctx context.Context, _ string) (net.Conn, error) {
			return d.Dial(ctx, connCfg.Host)
		})

	return fmt.Sprintf("%s:empty@cloudsqlconn(localhost:3306)/%s?parseTime=true",
		connCfg.Username, connCfg.Database), nil
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	var err error
	err = multierr.Append(err, d.db.Close())
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// getVersion gets the version.
func (d *Driver) getVersion(ctx context.Context) (string, string, error) {
	query := "SELECT VERSION()"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", "", util.FormatErrorWithQuery(err, query)
	}

	return parseVersion(version)
}

func parseVersion(version string) (string, string, error) {
	if loc := regexp.MustCompile(`^\d+.\d+.\d+`).FindStringIndex(version); loc != nil {
		return version[loc[0]:loc[1]], version[loc[1]:], nil
	}
	return "", "", errors.Errorf("failed to parse version %q", version)
}

// Execute executes a SQL statement.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	statement, err := mysqlparser.DealWithDelimiter(statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to deal with delimiter")
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	connectionID, err := getConnectionID(ctx, conn)
	if err != nil {
		return 0, err
	}
	slog.Debug("connectionID", slog.String("connectionID", connectionID))

	var totalCommands int
	var commands []base.SingleSQL
	var originalIndex []int32
	oneshot := true
	if len(statement) <= common.MaxSheetCheckSize {
		singleSQLs, err := mysqlparser.SplitSQL(statement)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to split sql")
		}
		singleSQLs, originalIndex = base.FilterEmptySQLWithIndexes(singleSQLs)
		if len(singleSQLs) == 0 {
			return 0, nil
		}
		commands = singleSQLs
		totalCommands = len(commands)
		if totalCommands <= common.MaximumCommands {
			oneshot = false
		}
	}
	if oneshot {
		commands = []base.SingleSQL{
			{
				Text: statement,
			},
		}
		originalIndex = []int32{0}
	}

	var totalRowsAffected int64

	if err := conn.Raw(func(driverConn any) error {
		//nolint
		exer := driverConn.(driver.ExecerContext)
		//nolint
		txer := driverConn.(driver.ConnBeginTx)
		tx, err := txer.BeginTx(ctx, driver.TxOptions{})
		if err != nil {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
			return err
		} else {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")
		}

		committed := false
		defer func() {
			err := tx.Rollback()
			if committed {
				return
			}
			var rerr string
			if err != nil {
				rerr = err.Error()
			}
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
		}()

		for i, command := range commands {
			indexes := []int32{originalIndex[i]}
			opts.LogCommandExecute(indexes)

			sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(command.Text)
			sqlResult, err := exer.ExecContext(ctx, sqlWithBytebaseAppComment, nil)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					slog.Info("cancel connection", slog.String("connectionID", connectionID))
					if err := d.StopConnectionByID(connectionID); err != nil {
						slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
					}
				}

				opts.LogCommandResponse(indexes, 0, nil, err.Error())

				return &db.ErrorWithPosition{
					Err: errors.Wrapf(err, "failed to execute context in a transaction"),
					Start: &storepb.TaskRunResult_Position{
						Line:   int32(command.FirstStatementLine),
						Column: int32(command.FirstStatementColumn),
					},
					End: &storepb.TaskRunResult_Position{
						Line:   int32(command.LastLine),
						Column: int32(command.LastColumn),
					},
				}
			}

			allRowsAffected := sqlResult.(mysql.Result).AllRowsAffected()
			var rowsAffected int64
			var allRowsAffectedInt32 []int32
			for _, a := range allRowsAffected {
				rowsAffected += a
				allRowsAffectedInt32 = append(allRowsAffectedInt32, int32(a))
			}
			totalRowsAffected += rowsAffected

			opts.LogCommandResponse(indexes, int32(rowsAffected), allRowsAffectedInt32, "")
		}

		if err := tx.Commit(); err != nil {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
			return errors.Wrapf(err, "failed to commit execute transaction")
		} else {
			opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
			committed = true
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return totalRowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_MYSQL, statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	connectionID, err := getConnectionID(ctx, conn)
	if err != nil {
		return nil, err
	}
	slog.Debug("connectionID", slog.String("connectionID", connectionID))

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}
		sqlWithBytebaseAppComment := util.MySQLPrependBytebaseAppComment(statement)

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_MYSQL, statement)
		if err != nil {
			slog.Error("failed to validate sql", slog.String("statement", statement), log.BBError(err))
			allQuery = true
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, sqlWithBytebaseAppComment)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, d.connCfg.MaximumSQLResultSize)
				if err != nil {
					return nil, err
				}
				if err := rows.Err(); err != nil {
					return nil, err
				}
				return r, nil
			}

			sqlResult, err := conn.ExecContext(ctx, statement)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				slog.Info("rowsAffected returns error", log.BBError(err))
			}
			return util.BuildAffectedRowsResult(affectedRows), nil
		}()
		stop := false
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				slog.Info("cancel connection", slog.String("connectionID", connectionID))
				if err := d.StopConnectionByID(connectionID); err != nil {
					slog.Error("failed to cancel connection", slog.String("connectionID", connectionID), log.BBError(err))
				}
			}
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		queryResult.Statement = statement
		queryResult.Latency = durationpb.New(time.Since(startTime))
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

func (d *Driver) StopConnectionByID(id string) error {
	// We cannot use placeholder parameter because TiDB doesn't accept it.
	_, err := d.db.Exec(fmt.Sprintf("KILL QUERY %s", id))
	return err
}

func getConnectionID(ctx context.Context, conn *sql.Conn) (string, error) {
	var id string
	if err := conn.QueryRowContext(ctx, `SELECT CONNECTION_ID();`).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}
