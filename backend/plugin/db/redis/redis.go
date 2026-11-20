// Package redis is Bytebase's Redis driver: an adapter over
// github.com/bytebase/omni/redis/driver that maps Bytebase's connection config
// onto the driver contract and the contract's results back into the product's
// protos.
//
// Every db.Driver method delegates except for the product seams:
//
//   - Open maps the data source onto driver.Config and driver.Options: the
//     topology, the database index, the master and sentinel logins, a TLS
//     callback and an SSH Dial. It does not reach the server; Ping does.
//   - Ping, Close, SyncInstance, SyncDBSchema, Dump, Execute and QueryConn
//     delegate. GetDB returns nil because Redis has no database/sql handle.
//   - QueryConn answers QueryOption_ALL_NODES through the contract's
//     AllNodesQuerier probe on the session, and refuses Explain, which Redis
//     cannot answer.
//   - Execute reports each command to the task run's log.
//
// Redis has no schema, DDL or SQL: SyncInstance reads INFO server, CONFIG GET
// databases and INFO keyspace, SyncDBSchema answers the database number, and
// Dump writes nothing, as before.
//
// Behaviour that moves with the driver, all measured in its cutover proof:
// commands run one per request instead of as one pipeline, so a script pays a
// round trip per command; the caller's context deadline interrupts a command;
// a refused command ends the script and carries its error on its own Result
// (earlier results stand, later commands are not sent) instead of failing the
// whole call; a per-result latency is the command's own; the result-size cap is
// byte-accurate rather than proto.Size at power-of-two row counts; an array
// element that holds no value is a NULL cell rather than the text "<nil>"; a
// Redis Cluster refuses a non-zero database and a negative index is refused
// instead of silently running on database 0; a missing port defaults to 6379;
// `#` is an argument's text rather than the comment the old shell lexer made it;
// and MaximumSQLResultSize 0 means no cap rather than a zero-byte cap. A missing
// key is still a NULL row, TLS material is still built once at Open and verified
// against the data source host, and a refused Explain still answers "Redis does
// not support EXPLAIN".
package redis

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	omniredis "github.com/bytebase/omni/redis/driver"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_REDIS, newDriver)
}

// Driver is Bytebase's Redis driver.
type Driver struct {
	inner     *omniredis.Driver
	sshClient *ssh.Client
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	d := &Driver{}
	dataSource := config.DataSource

	// Old Open built the TLS config once, so invalid TLS material still fails
	// here rather than on the first connection.
	tlsConfig, err := util.GetTLSConfig(dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "redis: failed to get tls config")
	}
	var tlsFunc func(host string) (*tls.Config, error)
	if tlsConfig != nil {
		tlsFunc = func(string) (*tls.Config, error) { return tlsConfig, nil }
	}

	// Old Open read the database index before opening the tunnel, so a name that
	// is not a number still fails without an SSH connection. The driver parses it
	// again and refuses a negative index.
	if name := config.ConnectionContext.DatabaseName; name != "" {
		if _, err := strconv.Atoi(name); err != nil {
			return nil, errors.Wrapf(err, "failed to convert database %s to int", name)
		}
	}

	cfg := omnidriver.Config{
		Host:     dataSource.Host,
		Port:     dataSource.Port,
		Database: config.ConnectionContext.DatabaseName,
		User:     dataSource.Username,
		Password: config.Password,
		TLS:      tlsFunc,
	}
	options := omniredis.Options{}
	switch dataSource.GetRedisType() {
	case storepb.DataSource_REDIS_TYPE_UNSPECIFIED, storepb.DataSource_STANDALONE:
	case storepb.DataSource_SENTINEL:
		options = omniredis.Options{
			Type:             omniredis.Sentinel,
			MasterName:       dataSource.GetMasterName(),
			SentinelUser:     dataSource.Username,
			SentinelPassword: config.Password,
		}
		cfg.Host = addressList(dataSource)
		cfg.User = dataSource.GetMasterUsername()
		cfg.Password = dataSource.GetMasterPassword()
	case storepb.DataSource_CLUSTER:
		options = omniredis.Options{Type: omniredis.Cluster}
		cfg.Host = addressList(dataSource)
	default:
		return nil, errors.Errorf("unsupported redis type %s", dataSource.GetRedisType().String())
	}
	if options.Type == "" {
		cfg.Host = joinAddress(dataSource.Host, dataSource.Port)
	}
	if dataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(dataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		cfg.Dial = func(_ context.Context, network, address string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, address)
			if err != nil {
				return nil, err
			}
			return &util.NoDeadlineConn{Conn: conn}, nil
		}
	}

	inner, err := omniredis.Open(ctx, cfg, options)
	if err != nil {
		d.closeSSHClient()
		return nil, err
	}
	d.inner = inner
	return d, nil
}

// addressList is the primary address followed by the additional ones, each in
// the host:port form the old driver handed its clients.
func addressList(dataSource *storepb.DataSource) string {
	addrs := make([]string, 0, 1+len(dataSource.GetAdditionalAddresses()))
	addrs = append(addrs, joinAddress(dataSource.Host, dataSource.Port))
	for _, addr := range dataSource.GetAdditionalAddresses() {
		addrs = append(addrs, joinAddress(addr.Host, addr.Port))
	}
	return strings.Join(addrs, ",")
}

func joinAddress(host, port string) string {
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return net.JoinHostPort(bare, port)
}

func (d *Driver) closeSSHClient() {
	if d.sshClient != nil {
		_ = d.sshClient.Close()
		d.sshClient = nil
	}
}

// Close closes the client and the SSH tunnel, if any.
func (d *Driver) Close(context.Context) error {
	var err error
	if d.inner != nil {
		err = multierr.Append(err, d.inner.Close())
	}
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
	}
	return err
}

// Ping pings the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// GetDB returns nil: Redis has no database/sql handle.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute runs the script's commands in order, reporting each to the task run's
// log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.inner.Execute(ctx, statement, omnidriver.ExecuteOptions{
		CreateDatabase: opts.CreateDatabase,
		Log:            executeLogger{opts: opts},
	})
}

// QueryConn runs a script, one result per command. ALL_NODES asks the session
// for the contract's all-nodes probe; every other option runs on one session.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, errors.New("Redis does not support EXPLAIN")
	}
	switch queryContext.Option.GetRedisRunCommandsOn() {
	case v1pb.QueryOption_REDIS_RUN_COMMANDS_ON_UNSPECIFIED, v1pb.QueryOption_SINGLE_NODE, v1pb.QueryOption_ALL_NODES:
	default:
		return nil, errors.Errorf("unknown RedisRunCommandOn enum %v", queryContext.Option.GetRedisRunCommandsOn())
	}

	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	options := omnidriver.QueryOptions{MaxResultBytes: queryContext.MaximumSQLResultSize}
	var innerResults []*omnidriver.Result
	if queryContext.Option.GetRedisRunCommandsOn() == v1pb.QueryOption_ALL_NODES {
		probe, ok := session.(omnidriver.AllNodesQuerier)
		if !ok {
			return nil, errors.Errorf("expect *redis.ClusterClient, but get %T", d.inner)
		}
		innerResults, err = probe.QueryAllNodes(ctx, statement, options)
	} else {
		innerResults, err = session.Query(ctx, statement, options)
	}
	if err != nil {
		return nil, err
	}

	results := make([]*v1pb.QueryResult, 0, len(innerResults))
	for _, result := range innerResults {
		results = append(results, toQueryResult(result, queryContext.MaximumSQLResultSize))
	}
	return results, nil
}

// SyncInstance reads the server version and its database numbers.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema answers the database number: Redis keys are not a schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.inner.SyncDatabase(ctx)
}

// Dump writes nothing, as before: Redis has no schema to render.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

// toQueryResult maps one contract result onto the product's result.
func toQueryResult(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	converted := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	for _, column := range result.Columns {
		converted.ColumnNames = append(converted.ColumnNames, column.Name)
		converted.ColumnTypeNames = append(converted.ColumnTypeNames, column.TypeName)
	}
	for _, row := range result.Rows {
		convertedRow := &v1pb.QueryRow{}
		for _, value := range row {
			convertedRow.Values = append(convertedRow.Values, toRowValue(value))
		}
		converted.Rows = append(converted.Rows, convertedRow)
	}
	converted.RowsCount = int64(len(converted.Rows))
	if result.Truncated {
		converted.Error = util.FormatMaximumSQLResultSizeMessage(maxResultBytes)
	} else if result.Err != nil {
		converted.Error = result.Err.Error()
	}
	return converted
}

// toRowValue maps one contract value onto the product's row value. Redis
// reports a command's answer as text (a number included) and a missing key or
// array element as NULL.
func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.Int64)}}
	case omnidriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}

// executeLogger reports the driver contract's execute steps to the task run's
// log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(command omnidriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart),
		End:   int32(command.ByteEnd),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l executeLogger) TransactionControl(op omnidriver.TransactionOp, err error) {
	l.opts.LogTransactionControl(transactionControlType(op), errorString(err))
}

func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func transactionControlType(op omnidriver.TransactionOp) storepb.TaskRunLog_TransactionControl_Type {
	switch op {
	case omnidriver.Begin:
		return storepb.TaskRunLog_TransactionControl_BEGIN
	case omnidriver.Commit:
		return storepb.TaskRunLog_TransactionControl_COMMIT
	case omnidriver.Rollback:
		return storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return storepb.TaskRunLog_TransactionControl_TYPE_UNSPECIFIED
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
