// Package cassandra is Bytebase's Cassandra driver, adapting
// github.com/bytebase/omni/cassandra/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Open configures gocql lazily without dialing the cluster; Ping verifies connectivity.
//   - SSH tunnels wrap dialed connections so RemoteAddr satisfies gocql's *net.TCPAddr assertion.
//   - Params, ApplicationName, and ReadOnly are unused by Cassandra.
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

// Open maps the connection config onto the driver contract without dialing the
// cluster; Ping verifies connectivity.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource
	addrs := make([]string, 0, 1+len(dataSource.AdditionalAddresses))
	addrs = append(addrs, formatAddress(dataSource.Host, dataSource.Port))
	for _, addr := range dataSource.AdditionalAddresses {
		addrs = append(addrs, formatAddress(addr.Host, addr.Port))
	}
	cfg := omnidriver.Config{
		Host:          strings.Join(addrs, ","),
		Port:          dataSource.Port,
		Database:      config.ConnectionContext.DatabaseName,
		User:          dataSource.Username,
		Password:      config.Password,
		TLS:           util.HostTLSFunc(dataSource),
		EngineVersion: config.ConnectionContext.EngineVersion,
	}

	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		if err := util.SetupSSHDialWithWrap(dataSource, &cfg, adapter, true, func(conn net.Conn, address string) net.Conn {
			return &tunnelConn{Conn: conn, remote: remoteTCPAddr(address)}
		}); err != nil {
			return nil, err
		}
		return omnicassandra.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// Execute runs the script's statements in order, reporting each to the task
// run's log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{})
}

var cassandraQueryResultOptions = util.QueryResultOptions{
	SkipPlan:      true,
	SkipMessages:  true,
	TruncatedWins: true,
	ConvertValue:  convertValue,
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
		queryOpts := omnidriver.QueryOptions{
			Limit:          queryContext.Limit,
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		}
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

			stmtResults, err := session.Query(ctx, stmt.Text, queryOpts)
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
				result := util.OmniResultToQueryResultWith(stmtResult, queryContext.MaximumSQLResultSize, cassandraQueryResultOptions)
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

// tunnelConn satisfies gocql's *net.TCPAddr type assertion on RemoteAddr.
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
