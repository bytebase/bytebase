package drivers

import (
	"context"
	"net"
	"strconv"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	omniredis "github.com/bytebase/omni/redis/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var redisSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, options, err := redisConfig(config)
		if err != nil {
			return nil, err
		}
		if err := util.SetupSSHDialWithWrap(config.DataSource, &cfg, &d.OmniAdapter, false, func(conn net.Conn, _ string) net.Conn {
			return &util.NoDeadlineConn{Conn: conn}
		}); err != nil {
			return nil, err
		}
		return omniredis.Open(ctx, cfg, options)
	},
	execute: util.ExecuteConfig{PassCreateDatabase: true},
	query:   redisQuery,
}

func redisConfig(config db.ConnectionConfig) (odriver.Config, omniredis.Options, error) {
	dataSource := config.DataSource
	tlsFunc, err := util.StaticTLSFunc(dataSource)
	if err != nil {
		return odriver.Config{}, omniredis.Options{}, errors.Wrap(err, "redis: failed to get tls config")
	}
	if name := config.ConnectionContext.DatabaseName; name != "" {
		if _, err := strconv.Atoi(name); err != nil {
			return odriver.Config{}, omniredis.Options{}, errors.Wrapf(err, "failed to convert database %s to int", name)
		}
	}
	cfg := odriver.Config{
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
		cfg.Host = joinHostPort(dataSource.Host, dataSource.Port)
	case storepb.DataSource_SENTINEL:
		options = omniredis.Options{
			Type:             omniredis.Sentinel,
			MasterName:       dataSource.GetMasterName(),
			SentinelUser:     dataSource.Username,
			SentinelPassword: config.Password,
		}
		cfg.Host = redisAddresses(dataSource)
		cfg.User = dataSource.GetMasterUsername()
		cfg.Password = dataSource.GetMasterPassword()
	case storepb.DataSource_CLUSTER:
		options = omniredis.Options{Type: omniredis.Cluster}
		cfg.Host = redisAddresses(dataSource)
	default:
		return odriver.Config{}, omniredis.Options{}, errors.Errorf("unsupported redis type %s", dataSource.GetRedisType().String())
	}
	return cfg, options, nil
}

func redisAddresses(dataSource *storepb.DataSource) string {
	addrs := []string{joinHostPort(dataSource.Host, dataSource.Port)}
	for _, addr := range dataSource.GetAdditionalAddresses() {
		addrs = append(addrs, joinHostPort(addr.Host, addr.Port))
	}
	return strings.Join(addrs, ",")
}

var redisQueryResultOptions = util.QueryResultOptions{
	SkipPlan:      true,
	SkipMessages:  true,
	TruncatedWins: true,
	ConvertValue: func(value odriver.Value) *v1pb.RowValue {
		switch value.Kind {
		case odriver.Null:
			return util.NullRowValue
		case odriver.Int32, odriver.String:
			return util.OmniRowValue(value)
		default:
			return printedValue(value)
		}
	},
}

// redisQuery runs a script on an ephemeral session, on every node of a cluster
// when the request asks for it, and refuses EXPLAIN.
func redisQuery(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, errors.New("Redis does not support EXPLAIN")
	}
	runOn := queryContext.Option.GetRedisRunCommandsOn()
	switch runOn {
	case v1pb.QueryOption_REDIS_RUN_COMMANDS_ON_UNSPECIFIED, v1pb.QueryOption_SINGLE_NODE, v1pb.QueryOption_ALL_NODES:
	default:
		return nil, errors.Errorf("unknown RedisRunCommandOn enum %v", runOn)
	}
	queryOpts := odriver.QueryOptions{MaxResultBytes: queryContext.MaximumSQLResultSize}
	if runOn != v1pb.QueryOption_ALL_NODES {
		return d.QueryEphemeralConn(ctx, statement, queryOpts, redisQueryResultOptions)
	}
	var results []*odriver.Result
	err := d.QueryEphemeral(ctx, func(session odriver.Session) error {
		querier, ok := session.(odriver.AllNodesQuerier)
		if !ok {
			return errors.Errorf("expect *redis.ClusterClient, but get %T", d.Driver)
		}
		var queryErr error
		results, queryErr = querier.QueryAllNodes(ctx, statement, queryOpts)
		return queryErr
	})
	if err != nil {
		return nil, err
	}
	return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, redisQueryResultOptions), nil
}
