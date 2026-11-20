// Package cassandra is Bytebase's Cassandra driver: an adapter over
// github.com/bytebase/omni/cassandra/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
//
// The pieces that move behaviour, beyond the swap itself:
//
//   - Open no longer reaches the cluster. The omni driver builds its gocql
//     session on first use, so a bad contact point is reported by Ping, not by
//     Open.
//   - SSH tunnels work. The omni driver takes a Dial, so a data source with an
//     SSH host dials through util.DialSSHTunnel, which this driver did not
//     support before. The live test reaches the cluster through one; TLS is
//     mapped the same way but not exercised, because the pinned container
//     speaks plaintext.
//   - A dropped keyspace answers driver.ErrNotFound from SyncDBSchema instead
//     of an empty schema: the omni driver asks the catalog whether the
//     keyspace exists before reading it.
//   - Execute reports each command to ExecuteOptions.CreateTaskRunLog, so a
//     task run's log holds the steps a Cassandra script ran.
//
// The result shapes stay the ones Bytebase showed: values are converted from
// the contract's driver.Value with Cassandra's own types (a CQL timestamp is
// millisecond-accurate, a float is FloatValue, decimal and varint are strings,
// a time-of-day is HH:MM:SS, and a list, map, set or UDT is its JSON spelling).
// One shape moved: a tuple column is now one cell of JSON rather than the
// column's elements expanded into separate cells, because the contract aligns a
// row's values with the columns the server names.
//
// db.ConnectionConfig.Params, ApplicationName and ReadOnly are not mapped:
// gocql takes no connection string this driver fills, Cassandra has no
// application name, and it has no read-only session.
package cassandra

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	omnicassandra "github.com/bytebase/omni/cassandra/driver"
	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	cassandraparser "github.com/bytebase/bytebase/backend/plugin/parser/cassandra"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_CASSANDRA, newDriver)
}

// Driver is Bytebase's Cassandra driver.
type Driver struct {
	inner     *omnicassandra.Driver
	sshClient *ssh.Client
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the cluster: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	d := &Driver{}

	dataSource := config.DataSource
	var dial func(ctx context.Context, network, address string) (net.Conn, error)
	if dataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(dataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := util.DialSSHTunnel(ctx, sshClient, network, address)
			if err != nil {
				return nil, err
			}
			return &tunnelConn{Conn: conn, remote: remoteTCPAddr(address)}, nil
		}
	}

	addrs := []string{formatAddress(dataSource.Host, dataSource.Port)}
	for _, addr := range dataSource.AdditionalAddresses {
		addrs = append(addrs, formatAddress(addr.Host, addr.Port))
	}
	inner, err := omnicassandra.Open(ctx, omnidriver.Config{
		Host:          strings.Join(addrs, ","),
		Port:          dataSource.Port,
		Database:      config.ConnectionContext.DatabaseName,
		User:          dataSource.Username,
		Password:      config.Password,
		TLS:           tlsConfigFunc(dataSource),
		Dial:          dial,
		EngineVersion: config.ConnectionContext.EngineVersion,
	})
	if err != nil {
		d.closeSSHClient()
		return nil, err
	}
	d.inner = inner
	return d, nil
}

func (d *Driver) Close(context.Context) error {
	d.closeSSHClient()
	if d.inner != nil {
		return d.inner.Close()
	}
	return nil
}

func (d *Driver) closeSSHClient() {
	if d.sshClient != nil {
		_ = d.sshClient.Close()
		d.sshClient = nil
	}
}

// Ping asks the cluster for its release version.
func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// Execute runs the script's statements in order, reporting each to the task
// run's log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.inner.Execute(ctx, statement, omnidriver.ExecuteOptions{
		Log: executeLogger{opts: opts},
	})
}

// QueryConn runs a read-only script, one result per statement, with the
// editor's SELECT-only rule.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, rawStatement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := cassandraparser.SplitSQL(rawStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}

	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}
		startTime := time.Now()
		if _, _, err := base.ValidateSQLForEditor(storepb.Engine_CASSANDRA, stmt.Text); err != nil {
			results = append(results, &v1pb.QueryResult{
				Statement: stmt.Text,
				Latency:   durationpb.New(time.Since(startTime)),
				Error:     connect.NewError(connect.CodeInvalidArgument, errors.Errorf("support Cassandra SELECT statement only, err: %s", err.Error())).Error(),
			})
			break
		}

		stmtResults, err := session.Query(ctx, stmt.Text, omnidriver.QueryOptions{
			Limit:          queryContext.Limit,
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		})
		if err != nil {
			results = append(results, &v1pb.QueryResult{
				Statement: stmt.Text,
				Latency:   durationpb.New(time.Since(startTime)),
				Error:     err.Error(),
			})
			break
		}

		stop := false
		for _, stmtResult := range stmtResults {
			result := toQueryResult(stmtResult, queryContext)
			results = append(results, result)
			if result.Error != "" {
				stop = true
				break
			}
		}
		if stop {
			break
		}
	}
	return results, nil
}

// SyncInstance reads the cluster version and its user keyspaces.
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

// SyncDBSchema reads the attached keyspace's tables. A keyspace the server no
// longer has is driver.ErrNotFound.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.inner.SyncDatabase(ctx)
}

// Dump writes nothing: a synced keyspace does not carry its replication
// strategy, so the metadata could not create the keyspace it describes.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

// toQueryResult maps one contract result onto the product's result.
func toQueryResult(result *omnidriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
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
			convertedRow.Values = append(convertedRow.Values, convertValue(value))
		}
		converted.Rows = append(converted.Rows, convertedRow)
	}
	converted.RowsCount = int64(len(converted.Rows))
	if result.Truncated {
		converted.Error = util.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
	} else if result.Err != nil {
		converted.Error = result.Err.Error()
	}
	return converted
}

// convertValue maps one contract value onto the product's row value, keeping
// the shapes Bytebase showed: a 32-bit float stays a FloatValue, a decimal, a
// varint, a time of day and a collection are strings, and a timestamp is
// millisecond-accurate.
func convertValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case omnidriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: float32(value.Float64)}}
	case omnidriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.Int64)}}
	case omnidriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.Uint32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(value.Uint64)}}
	case omnidriver.Uint64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: value.Uint64}}
	case omnidriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case omnidriver.Structured:
		encoded, err := json.Marshal(value.Structured)
		if err != nil {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value.Structured)}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(encoded)}}
	case omnidriver.Timestamp:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
			GoogleTimestamp: timestamppb.New(value.Time),
			Accuracy:        value.Precision,
		}}}
	case omnidriver.TimestampTZ:
		zone, offset := value.Time.Zone()
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
			GoogleTimestamp: timestamppb.New(value.Time),
			Zone:            zone,
			Offset:          int32(offset),
			Accuracy:        value.Precision,
		}}}
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

// tunnelConn reports the address the tunnel dialled. gocql reads the remote
// port from the connection with a *net.TCPAddr assertion, and a tunnel's own
// address type is not one.
type tunnelConn struct {
	net.Conn
	remote *net.TCPAddr
}

func (c *tunnelConn) RemoteAddr() net.Addr { return c.remote }

func remoteTCPAddr(address string) *net.TCPAddr {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return &net.TCPAddr{}
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		return &net.TCPAddr{}
	}
	return &net.TCPAddr{IP: net.ParseIP(host), Port: number}
}

func tlsConfigFunc(dataSource *storepb.DataSource) func(string) (*tls.Config, error) {
	if !dataSource.GetUseSsl() {
		return nil
	}
	return func(host string) (*tls.Config, error) {
		return util.GetTLSConfigForHost(dataSource, host)
	}
}

func formatAddress(host, port string) string {
	if port == "" {
		return host
	}
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return net.JoinHostPort(bare, port)
}
