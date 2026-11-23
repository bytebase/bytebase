// Package redis is Bytebase's Redis driver, adapting
// github.com/bytebase/omni/redis/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Open validates TLS material, database index, and topology lazily without dialing; Ping checks connectivity.
//   - SSH tunnels wrap connections in util.NoDeadlineConn because ssh channels reject SetDeadline.
//   - QueryConn refuses Explain and dispatches ALL_NODES through omni/driver.AllNodesQuerier.
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

// Open maps the connection config onto the driver contract without dialing the
// server; Ping verifies connectivity.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource

	tlsFunc, err := util.StaticTLSFunc(dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "redis: failed to get tls config")
	}
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
		cfg.Host = joinAddress(dataSource.Host, dataSource.Port)
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

	d := &Driver{}
	if err := d.OpenWith(ctx, util.AdapterOptions{}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		if err := util.SetupSSHDialWithWrap(dataSource, &cfg, adapter, false, func(conn net.Conn, _ string) net.Conn {
			return &util.NoDeadlineConn{Conn: conn}
		}); err != nil {
			return nil, err
		}
		return omniredis.Open(ctx, cfg, options)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

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

var redisQueryResultOptions = util.QueryResultOptions{
	SkipPlan:      true,
	SkipMessages:  true,
	TruncatedWins: true,
	ConvertValue:  toRowValue,
}

// QueryConn runs a script on an ephemeral session, dispatching ALL_NODES
// through omni/driver.AllNodesQuerier and refusing EXPLAIN.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, errors.New("Redis does not support EXPLAIN")
	}
	runOn := queryContext.Option.GetRedisRunCommandsOn()
	switch runOn {
	case v1pb.QueryOption_REDIS_RUN_COMMANDS_ON_UNSPECIFIED, v1pb.QueryOption_SINGLE_NODE, v1pb.QueryOption_ALL_NODES:
	default:
		return nil, errors.Errorf("unknown RedisRunCommandOn enum %v", runOn)
	}

	queryOpts := omnidriver.QueryOptions{MaxResultBytes: queryContext.MaximumSQLResultSize}
	if runOn != v1pb.QueryOption_ALL_NODES {
		return d.QueryEphemeralConn(ctx, statement, queryOpts, redisQueryResultOptions)
	}
	var innerResults []*omnidriver.Result
	err := d.QueryEphemeral(ctx, func(session omnidriver.Session) error {
		probe, ok := session.(omnidriver.AllNodesQuerier)
		if !ok {
			return errors.Errorf("expect *redis.ClusterClient, but get %T", d.Driver)
		}
		var queryErr error
		innerResults, queryErr = probe.QueryAllNodes(ctx, statement, queryOpts)
		return queryErr
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResultsWith(innerResults, queryContext.MaximumSQLResultSize, redisQueryResultOptions), nil
}

func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Int32, omnidriver.String:
		return util.OmniRowValue(value)
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}
