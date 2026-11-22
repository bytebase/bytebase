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
//     SSH host dials through util.DialSSHTunnel.
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
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	omnicassandra "github.com/bytebase/omni/cassandra/driver"
	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

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
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the cluster: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	d := &Driver{}
	opened := false
	defer func() {
		if !opened {
			_ = d.Close(ctx)
		}
	}()

	dataSource := config.DataSource
	var dial func(ctx context.Context, network, address string) (net.Conn, error)
	if dataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(dataSource)
		if err != nil {
			return nil, err
		}
		d.AddCloser(sshClient)
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
		TLS:           util.HostTLSFunc(dataSource),
		Dial:          dial,
		EngineVersion: config.ConnectionContext.EngineVersion,
	})
	if err != nil {
		return nil, err
	}
	d.Driver = inner
	opened = true
	return d, nil
}

// Execute runs the script's statements in order, reporting each to the task
// run's log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{})
}

// QueryConn runs a read-only script, one result per statement, with the
// editor's SELECT-only rule.
func (d *Driver) QueryConn(ctx context.Context, rawStatement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := cassandraparser.SplitSQL(rawStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}

	var results []*v1pb.QueryResult
	err = d.QueryEphemeral(ctx, func(session omnidriver.Session) error {
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
				result := util.OmniResultToQueryResultWith(stmtResult, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
					SkipPlan:      true,
					SkipMessages:  true,
					TruncatedWins: true,
					ConvertValue:  convertValue,
				})
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
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// convertValue maps one contract value onto the product's row value, keeping
// the shapes Bytebase showed: a 32-bit float stays a FloatValue, a decimal, a
// varint, a time of day and a collection are strings, and a timestamp is
// millisecond-accurate.
func convertValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Structured:
		encoded, err := json.Marshal(value.Structured)
		if err != nil {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value.Structured)}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(encoded)}}
	case omnidriver.Bool, omnidriver.Bytes, omnidriver.Float32, omnidriver.Float64,
		omnidriver.Int32, omnidriver.Int64, omnidriver.Uint32, omnidriver.Uint64,
		omnidriver.String, omnidriver.Timestamp, omnidriver.TimestampTZ:
		return util.OmniRowValue(value)
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
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

func formatAddress(host, port string) string {
	if port == "" {
		return host
	}
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return net.JoinHostPort(bare, port)
}
