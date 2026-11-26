package drivers

import (
	"context"

	odriver "github.com/bytebase/omni/driver"
	omnitrino "github.com/bytebase/omni/trino/driver"
	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var trinoSpec = spec{
	// The catalog is Config.Database: the connection's database, else the data
	// source's, else Trino's default.
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		dataSource := config.DataSource
		if dataSource == nil {
			return nil, errors.New("trino: a data source is required")
		}
		return omnitrino.Open(ctx, odriver.Config{
			Host:            dataSource.Host,
			Port:            dataSource.Port,
			Database:        util.ResolveDatabaseName(config),
			User:            dataSource.Username,
			Password:        config.Password,
			ApplicationName: "bytebase",
			TLS:             util.LazyTLSFunc(dataSource),
		})
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true},
	// Trino commits DDL on its own and supports transactions for only some
	// connectors' DML, so each statement runs on its own.
	execute: util.ExecuteConfig{
		PassMaxRetries:          true,
		RangeIncludesTerminator: true,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(
			ctx,
			statement,
			util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON)),
			util.SessionQueryOptions{WrapOpenError: "failed to open a session"},
			// A statement that reports no count (DDL, SET SESSION) shows
			// "Affected Rows 0".
			util.QueryResultOptions{FillZeroAffectedRows: true},
		)
	},
}
