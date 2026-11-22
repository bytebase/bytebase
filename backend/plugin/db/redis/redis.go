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
// Dump writes nothing.
package redis

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	omnidriver "github.com/bytebase/omni/driver"
	omniredis "github.com/bytebase/omni/redis/driver"
	"github.com/pkg/errors"

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
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	d := &Driver{}
	opened := false
	defer func() {
		if !opened {
			_ = d.Close(ctx)
		}
	}()

	dataSource := config.DataSource

	// Old Open built the TLS config once, so invalid TLS material still fails
	// here rather than on the first connection.
	tlsFunc, err := util.StaticTLSFunc(dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "redis: failed to get tls config")
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
		d.AddCloser(sshClient)
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
		return nil, err
	}
	d.Driver = inner
	opened = true
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

// Execute runs the script's commands in order, reporting each to the task run's
// log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		PassCreateDatabase: true,
	})
}

// QueryConn runs a script, one result per command. ALL_NODES asks the session
// for the contract's all-nodes probe; every other option runs on one session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, errors.New("Redis does not support EXPLAIN")
	}
	switch queryContext.Option.GetRedisRunCommandsOn() {
	case v1pb.QueryOption_REDIS_RUN_COMMANDS_ON_UNSPECIFIED, v1pb.QueryOption_SINGLE_NODE, v1pb.QueryOption_ALL_NODES:
	default:
		return nil, errors.Errorf("unknown RedisRunCommandOn enum %v", queryContext.Option.GetRedisRunCommandsOn())
	}

	var innerResults []*omnidriver.Result
	err := d.QueryEphemeral(ctx, func(session omnidriver.Session) error {
		options := omnidriver.QueryOptions{MaxResultBytes: queryContext.MaximumSQLResultSize}
		if queryContext.Option.GetRedisRunCommandsOn() == v1pb.QueryOption_ALL_NODES {
			probe, ok := session.(omnidriver.AllNodesQuerier)
			if !ok {
				return errors.Errorf("expect *redis.ClusterClient, but get %T", d.Driver)
			}
			var queryErr error
			innerResults, queryErr = probe.QueryAllNodes(ctx, statement, options)
			return queryErr
		}
		var queryErr error
		innerResults, queryErr = session.Query(ctx, statement, options)
		return queryErr
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResultsWith(innerResults, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		SkipPlan:      true,
		SkipMessages:  true,
		TruncatedWins: true,
		ConvertValue:  toRowValue,
	}), nil
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
