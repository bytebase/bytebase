package drivers

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	omnicassandra "github.com/bytebase/omni/cassandra/driver"
	odriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	cassandraparser "github.com/bytebase/bytebase/backend/plugin/parser/cassandra"
)

var cassandraSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		dataSource := config.DataSource
		addrs := []string{cassandraAddress(dataSource.Host, dataSource.Port)}
		for _, addr := range dataSource.AdditionalAddresses {
			addrs = append(addrs, cassandraAddress(addr.Host, addr.Port))
		}
		cfg := odriver.Config{
			Host:          strings.Join(addrs, ","),
			Port:          dataSource.Port,
			Database:      config.ConnectionContext.DatabaseName,
			User:          dataSource.Username,
			Password:      config.Password,
			TLS:           util.HostTLSFunc(dataSource),
			EngineVersion: config.ConnectionContext.EngineVersion,
		}
		if err := util.SetupSSHDialWithWrap(dataSource, &cfg, &d.OmniAdapter, true, func(conn net.Conn, address string) net.Conn {
			return &tcpAddrConn{Conn: conn, remote: tcpAddr(address)}
		}); err != nil {
			return nil, err
		}
		return omnicassandra.Open(ctx, cfg)
	},
	query: cassandraQuery,
}

var cassandraQueryResultOptions = util.QueryResultOptions{
	SkipPlan:      true,
	SkipMessages:  true,
	TruncatedWins: true,
	ConvertValue: func(value odriver.Value) *v1pb.RowValue {
		switch value.Kind {
		case odriver.Null:
			return util.NullRowValue
		case odriver.Structured:
			return jsonValue(value)
		case odriver.Bool, odriver.Bytes, odriver.Float32, odriver.Float64,
			odriver.Int32, odriver.Int64, odriver.Uint32, odriver.Uint64,
			odriver.String, odriver.Timestamp, odriver.TimestampTZ:
			return util.OmniRowValue(value)
		default:
			return printedValue(value)
		}
	},
}

// cassandraQuery runs a read-only script, one result per statement, and stops
// at the first statement the editor refuses or the server fails.
func cassandraQuery(ctx context.Context, d *Driver, rawStatement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := cassandraparser.SplitSQL(rawStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}
	var results []*v1pb.QueryResult
	err = d.QueryEphemeral(ctx, func(session odriver.Session) error {
		queryOpts := odriver.QueryOptions{
			Limit:          queryContext.Limit,
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		}
	statements:
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
				break statements
			}
			stmtResults, err := session.Query(ctx, stmt.Text, queryOpts)
			if err != nil {
				results = append(results, &v1pb.QueryResult{
					Statement: stmt.Text,
					Latency:   durationpb.New(time.Since(startTime)),
					Error:     err.Error(),
				})
				break statements
			}
			for _, stmtResult := range stmtResults {
				result := util.OmniResultToQueryResultWith(stmtResult, queryContext.MaximumSQLResultSize, cassandraQueryResultOptions)
				results = append(results, result)
				if result.Error != "" {
					break statements
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// tcpAddrConn satisfies gocql's *net.TCPAddr type assertion on RemoteAddr.
type tcpAddrConn struct {
	net.Conn
	remote *net.TCPAddr
}

func (c *tcpAddrConn) RemoteAddr() net.Addr { return c.remote }

func tcpAddr(address string) *net.TCPAddr {
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

func cassandraAddress(host, port string) string {
	if port == "" {
		return host
	}
	return joinHostPort(host, port)
}
