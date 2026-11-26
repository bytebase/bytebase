package drivers

import (
	"context"
	"fmt"
	"strings"

	omnicockroach "github.com/bytebase/omni/cockroachdb/driver"
	odriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var cockroachdbSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := cockroachdbConfig(config)
		if err != nil {
			return nil, err
		}
		if err := util.SetupSSHDial(config.DataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		return omnicockroach.Open(ctx, cfg)
	},
	adapter: util.AdapterOptions{WithInstanceMetadata: true, NotFoundResource: "database"},
	// An unspecified mode is transactional; isolation directives and
	// MaximumRetries are not passed.
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(
			ctx,
			statement,
			util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON)),
			util.SessionQueryOptions{},
			util.QueryResultOptions{
				ClearExplainPlan:     queryContext.Explain,
				FillZeroAffectedRows: true,
			},
		)
	},
	taskRunSession: cockroachdbTaskRunSession,
}

func cockroachdbConfig(config db.ConnectionConfig) (odriver.Config, error) {
	dataSource := config.DataSource
	// Half a client certificate is refused whether or not TLS is asked for,
	// rather than left to a connection that never presents it.
	if err := util.ValidateTLSCertKeyPair(dataSource); err != nil {
		return odriver.Config{}, err
	}
	return odriver.Config{
		Host:            dataSource.GetHost(),
		Port:            dataSource.GetPort(),
		Database:        util.ResolveDatabaseName(config),
		User:            dataSource.GetUsername(),
		Password:        config.Password,
		ApplicationName: util.ApplicationName(config.ConnectionContext),
		ReadOnly:        config.ConnectionContext.ReadOnly,
		EngineVersion:   config.ConnectionContext.EngineVersion,
		TLS:             util.PGTLSFunc(dataSource),
	}, nil
}

// cockroachdbTaskRunSession reads the running session by application name from
// CockroachDB's own catalog. Its pg_stat_activity view answers no rows and it
// has no pg_blocking_pids, so PostgreSQL's query does not apply.
func cockroachdbTaskRunSession(ctx context.Context, d *Driver, applicationName string) (*v1pb.TaskRunSession, error) {
	if d.Driver == nil {
		return nil, errors.New("cockroachdb: driver is closed")
	}
	session, err := d.OpenSession(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	query := fmt.Sprintf(`
		SELECT
			q.session_id,
			q.query,
			q.phase,
			q.database,
			q.user_name,
			q.application_name,
			q.client_address,
			s.session_start,
			q.start
		FROM crdb_internal.cluster_queries AS q
		LEFT JOIN crdb_internal.cluster_sessions AS s
			ON s.session_id = q.session_id
		WHERE q.application_name = '%s'
		ORDER BY q.session_id
	`, strings.ReplaceAll(applicationName, "'", "''"))
	results, err := session.Query(ctx, query, odriver.QueryOptions{})
	if err != nil {
		return nil, err
	}
	postgres := &v1pb.TaskRunSession_Postgres{}
	if len(results) > 0 {
		if results[0].Err != nil {
			return nil, results[0].Err
		}
		postgres = cockroachdbSessions(results[0], applicationName)
	}
	return &v1pb.TaskRunSession{
		Session: &v1pb.TaskRunSession_Postgres_{Postgres: postgres},
	}, nil
}

func cockroachdbSessions(result *odriver.Result, applicationName string) *v1pb.TaskRunSession_Postgres {
	ss := &v1pb.TaskRunSession_Postgres{}
	for _, row := range result.Rows {
		if len(row) < 9 {
			continue
		}
		s := v1pb.TaskRunSession_Postgres_Session{
			Pid:             valueString(row[0]),
			Query:           valueString(row[1]),
			State:           optionalValueString(row[2]),
			Datname:         optionalValueString(row[3]),
			Usename:         optionalValueString(row[4]),
			ApplicationName: valueString(row[5]),
			ClientAddr:      optionalValueString(row[6]),
			BackendStart:    optionalValueTimestamp(row[7]),
			QueryStart:      optionalValueTimestamp(row[8]),
		}
		if s.ApplicationName == applicationName {
			ss.Session = &s
		}
	}
	return ss
}

func valueString(v odriver.Value) string {
	if v.Kind == odriver.Null {
		return ""
	}
	return v.String
}

func optionalValueString(v odriver.Value) *string {
	if v.Kind == odriver.Null {
		return nil
	}
	s := v.String
	return &s
}

func optionalValueTimestamp(v odriver.Value) *timestamppb.Timestamp {
	if v.Kind != odriver.Timestamp && v.Kind != odriver.TimestampTZ {
		return nil
	}
	return timestamppb.New(v.Time)
}
