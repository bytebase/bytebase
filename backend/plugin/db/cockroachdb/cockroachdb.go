// Package cockroachdb is Bytebase's CockroachDB driver: an adapter over
// github.com/bytebase/omni/cockroachdb/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's. The conversation with the server — connection configuration,
// session setup, metadata sync, statement classification, execution, result
// decoding and the plan diagram — is the driver's.
//
// CockroachDB is a PostgreSQL fork in Bytebase and its own tree in omni, so the
// adapter follows pg's shape where the two overlap: Config carries libpq's host
// and port lists, a TLS func bound to each host the driver dials, and the SSH
// tunnel as Config.Dial; QueryOptions.Schema is the session's search_path. Two
// parts of the pg path are not copied: Config.Params stays unset, because this
// data source has no extra connection parameters, and there is no tenant-mode
// prelude, because tenant mode does not apply to CockroachDB and its Open takes
// no engine options.
//
// QueryConn runs on one session held for the driver's lifetime and serializes
// its calls, as the editor stream's one pooled connection did: what a statement
// sets there (search_path, a temporary table) holds for the stream's next call.
//
// Execute maps the script's transaction directive and passes neither the
// isolation directive nor MaximumRetries: this engine ignored both, and honoring
// them would change what a script does.
package cockroachdb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	omnicockroach "github.com/bytebase/omni/cockroachdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver               = (*Driver)(nil)
	_ db.RowEstimator         = (*Driver)(nil)
	_ db.TaskRunSessionReader = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_COCKROACHDB, newDriver)
}

// Driver is Bytebase's CockroachDB driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         config.ConnectionContext.DatabaseName,
	}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		cfg, err := cockroachConfig(config)
		if err != nil {
			return nil, err
		}
		if err := util.SetupSSHDial(config.DataSource, &cfg, adapter, true); err != nil {
			return nil, err
		}
		return omnicockroach.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// cockroachConfig resolves what Bytebase knows about reaching a server into the
// plain values the contract's CockroachDB driver takes.
func cockroachConfig(config db.ConnectionConfig) (omnidriver.Config, error) {
	dataSource := config.DataSource
	// The old driver refused half a client certificate whether or not TLS was
	// asked for, so the mistake is reported here rather than left to a
	// connection that never presents it.
	if err := util.ValidateTLSCertKeyPair(dataSource); err != nil {
		return omnidriver.Config{}, err
	}
	return omnidriver.Config{
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

// TaskRunSession reads the running session by application name from
// CockroachDB's own catalog. CockroachDB's pg_stat_activity compatibility view
// answers no rows and it has no pg_blocking_pids, so neither PostgreSQL's
// query nor its blocking relationships apply.
func (d *Driver) TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error) {
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

	results, err := session.Query(ctx, query, omnidriver.QueryOptions{})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return &v1pb.TaskRunSession{
			Session: &v1pb.TaskRunSession_Postgres_{
				Postgres: &v1pb.TaskRunSession_Postgres{},
			},
		}, nil
	}
	if results[0].Err != nil {
		return nil, results[0].Err
	}
	return taskRunSessionFromResult(results[0], applicationName), nil
}

func taskRunSessionFromResult(result *omnidriver.Result, applicationName string) *v1pb.TaskRunSession {
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
	return &v1pb.TaskRunSession{
		Session: &v1pb.TaskRunSession_Postgres_{
			Postgres: ss,
		},
	}
}

func valueString(v omnidriver.Value) string {
	if v.Kind == omnidriver.Null {
		return ""
	}
	return v.String
}

func optionalValueString(v omnidriver.Value) *string {
	if v.Kind == omnidriver.Null {
		return nil
	}
	s := v.String
	return &s
}

func optionalValueTimestamp(v omnidriver.Value) *timestamppb.Timestamp {
	if v.Kind != omnidriver.Timestamp && v.Kind != omnidriver.TimestampTZ {
		return nil
	}
	return timestamppb.New(v.Time)
}

// CountAffectedRows returns the planner's estimate of the rows a statement
// modifies. plancheck reads it through db.RowEstimator.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

var cockroachExecuteConfig = util.ExecuteConfig{
	ParseTransactionDirective: true,
	PassCreateDatabase:        true,
}

// Execute runs the script's statements under the transaction directive the
// script starts with, reporting each step to the task run's log. An unspecified
// mode is transactional, and isolation directives and MaximumRetries are ignored.
func (d *Driver) Execute(ctx context.Context, script string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, script, opts, cockroachExecuteConfig)
}

// QueryConn runs the script on the driver's session, one result per statement,
// and serializes its calls.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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
}
