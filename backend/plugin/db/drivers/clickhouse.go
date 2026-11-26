package drivers

import (
	"context"
	"log/slog"

	omnich "github.com/bytebase/omni/clickhouse/driver"
	odriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var clickhouseSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		dataSource := config.DataSource
		cfg, err := clickhouseConfig(dataSource, config.Password, config.ConnectionContext.DatabaseName)
		if err != nil {
			return nil, err
		}
		if err := util.SetupSSHDial(dataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		slog.Debug("Opening ClickHouse driver",
			slog.String("addr", dataSource.GetHost()+":"+dataSource.GetPort()),
			slog.String("environment", config.ConnectionContext.EnvironmentID),
			slog.String("database", config.ConnectionContext.InstanceID),
		)
		return omnich.Open(ctx, cfg)
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true},
	// ClickHouse cannot roll back, so a script still logs the BEGIN and ROLLBACK
	// its transaction wrapper issues.
	execute: util.ExecuteConfig{ParseTransactionDirective: true},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(ctx, statement, odriver.QueryOptions{
			Limit:          queryContext.Limit,
			Explain:        queryContext.Explain,
			ExplainFormat:  queryContext.Option.GetExplainFormat().String(),
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		}, util.SessionQueryOptions{
			// A statement's error does not say whether the connection survived
			// it: a server exception leaves it in sync, while a decode or
			// protocol error leaves it unusable.
			ShouldDiscardResults: func(ctx context.Context, session odriver.Session, results []*odriver.Result) bool {
				return firstError(results) != nil && !sessionAnswers(ctx, session)
			},
		}, util.QueryResultOptions{
			ClearExplainPlan: queryContext.Explain,
			TypedExplainPlan: true,
			TruncatedWins:    true,
			ConvertValue: func(value odriver.Value) *v1pb.RowValue {
				if value.Kind == odriver.Structured {
					return stringValue(value.String)
				}
				return util.OmniRowValue(value)
			},
		})
	},
}

func clickhouseConfig(dataSource *storepb.DataSource, password, database string) (odriver.Config, error) {
	tlsFn, err := util.SQLStaticTLSFunc(dataSource)
	if err != nil {
		return odriver.Config{}, err
	}
	return odriver.Config{
		Host:     dataSource.GetHost(),
		Port:     dataSource.GetPort(),
		Database: database,
		User:     dataSource.GetUsername(),
		Password: password,
		// A data source's extra parameters are ClickHouse settings.
		Params: dataSource.GetExtraConnectionParameters(),
		TLS:    tlsFn,
	}, nil
}

func firstError(results []*odriver.Result) error {
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

// sessionAnswers reports whether the session can still run a trivial
// statement. It is called only after a statement failed.
func sessionAnswers(ctx context.Context, session odriver.Session) bool {
	results, err := session.Query(ctx, "SELECT 1", odriver.QueryOptions{})
	if err != nil || len(results) != 1 {
		return false
	}
	return results[0].Err == nil
}
