// Package hive is Bytebase's Hive driver: an adapter over
// github.com/bytebase/omni/hive/driver that maps Bytebase's connection config
// onto the driver contract, and the contract's results back into the product's.
// The conversation with HiveServer2 — session setup, the sync, the server-DDL
// dump, queries and change scripts — is the driver's.
//
// The adapter keeps what is product: the data source to contract conversion,
// the Kerberos bootstrap (under util.Lock, held through the first connection
// because the environment it writes is process-wide), TLS material,
// QueryContext/ExecuteOptions mapping, the v1pb conversion and the task-run
// logger.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - Config.Dial is unused: gohive v2 keeps its dialer in an unexported
//     connect configuration, so no caller can hand the driver a tunnel. An SSH
//     data source is refused with the local-forward instruction; a local
//     forwarder in db/util would restore it.
//   - Config.ReadOnly, ApplicationName and EngineVersion are unused: Hive has
//     no read-only session setting, no application name, and its version comes
//     from the sync.
//   - Execute ignores the transaction directive: HiveServer2 4.0 refuses
//     BEGIN, COMMIT and ROLLBACK, so every statement commits itself.
//   - QueryContext.Limit is applied to the rows the adapter returns, not to the
//     statement: there is no hive parser to place a LIMIT with, so the server
//     computes every row and MaximumSQLResultSize bounds the transfer. An
//     explain request's plan is never cut.
//   - A plan arrives as rows with no typed QueryPlan, which the API names.
//   - A failed statement's error is gohive's rendering of Hive's TStatus, whose
//     fields gohive keeps as pointers it prints as addresses. That is the
//     engine client's, and the adapter does not hide it.
//   - Dump does not read the metadata it is given: omni has no hive catalog, so
//     nothing renders a synced schema back to HiveQL, and the dump is the
//     server's own SHOW CREATE statements.
//
// Known differences from the driver this replaced, which are the fix this
// cutover is: Open now sends the data source's login, so every operation is
// reachable; SyncDBSchema reads DESCRIBE FORMATTED's NULL cells, so tables,
// external tables, views and materialized views sync; Dump restores; a query's
// schema is selected with USE; extra connection parameters and SSL reach the
// session; a DOUBLE or FLOAT cell is a number rather than 20-decimal text; a
// failing statement ends its script with the error on its result instead of
// failing the call; Execute logs each command; GetDB returns nil; and the dump
// text is requalified and reordered.
package hive

import (
	"context"
	"fmt"

	odriver "github.com/bytebase/omni/driver"
	hivedriver "github.com/bytebase/omni/hive/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_HIVE, newDriver)
}

// Driver is Bytebase's Hive driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract and reaches the
// server, as the driver this replaced did: a HiveServer2 that cannot be reached
// or will not accept the login fails here rather than at the first query.
//
// Kerberos needs the process-wide krb5.conf and keytab BootKerberosEnv writes,
// so the lock that guards them is held through the connect and the ping: a
// second opener that bootstrapped in between would leave this one
// authenticating against its environment.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	kerberos := hasKerberos(config.DataSource)
	if kerberos {
		util.Lock.Lock()
		defer util.Lock.Unlock()
	}
	cfg, err := hiveConfig(config)
	if err != nil {
		return nil, err
	}
	impl, err := hivedriver.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := impl.Ping(ctx); err != nil {
		_ = impl.Close()
		return nil, errors.Wrap(err, "failed to ping hive server")
	}
	d.OmniAdapter = util.OmniAdapter{Driver: impl}
	return d, nil
}

// hasKerberos reports whether the data source authenticates with a Kerberos
// principal.
func hasKerberos(ds *storepb.DataSource) bool {
	_, ok := ds.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig)
	return ok
}

// hiveConfig resolves what Bytebase knows about reaching a HiveServer2 into the
// plain values hive/driver's gohive connector takes.
func hiveConfig(config db.ConnectionConfig) (odriver.Config, error) {
	ds := config.DataSource
	if ds.GetSshHost() != "" {
		// gohive v2's connector keeps its dialer in an unexported field, so the
		// driver cannot dial through a tunnel. Say what to do instead of dialling
		// the direct host and silently ignoring the jump host, as the old driver
		// did.
		return odriver.Config{}, errors.Errorf("an SSH data source is not supported for Hive: forward a local port to %s and use the forwarded host and port instead", serverAddress(ds))
	}

	params := map[string]string{}
	if t, ok := ds.GetSaslConfig().GetMechanism().(*storepb.SASLConfig_KrbConfig); ok {
		// Open holds util.Lock here: the krb5.conf and keytab this writes are
		// process-wide, and the connection that reads them is established under
		// the same lock.
		if err := util.BootKerberosEnv(t); err != nil {
			return odriver.Config{}, errors.Wrapf(err, "failed to init SASL environment")
		}
		params["auth"] = "KERBEROS"
		service := "hive"
		if t.KrbConfig.Primary != "" {
			service = t.KrbConfig.Primary
		}
		params["service"] = service
	}
	// The caller's own parameters reach the session, and win over the SASL
	// settings above.
	for key, value := range ds.GetExtraConnectionParameters() {
		params[key] = value
	}

	cfg := odriver.Config{
		Host:     ds.GetHost(),
		Port:     ds.GetPort(),
		Database: config.ConnectionContext.DatabaseName,
		User:     ds.GetUsername(),
		Password: config.Password,
		Params:   params,
		// GetTLSConfig binds certificate verification to the data source's host,
		// which is the host the driver dials: an SSH tunnel is refused above, so
		// the two cannot differ.
		TLS: util.LazyTLSFunc(ds),
	}
	return cfg, nil
}

// serverAddress is the host and port a refusal names.
func serverAddress(ds *storepb.DataSource) string {
	port := ds.GetPort()
	if port == "" {
		port = "10000"
	}
	return ds.GetHost() + ":" + port
}

// Execute runs the script's statements in order, reporting each to the task
// run's log. Hive has no transaction the client controls — HiveServer2 4.0
// refuses BEGIN, COMMIT and ROLLBACK — so every statement commits itself and
// the script's transaction directive stays a comment, as it was before.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		DefaultTransaction: odriver.AutoCommit,
		PassMaxRetries:     true,
	})
}

// QueryConn runs a script for the editor, one result per statement, with the
// caller's row limit applied to each result. Hive answers a statement that
// returns no rows with an operation that has none, so a DDL statement sent here
// reports Hive's error on its result instead of failing the call, which is how
// the driver this replaced behaved too.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	var results []*odriver.Result
	err := d.QueryEphemeral(ctx, func(session odriver.Session) error {
		var queryErr error
		results, queryErr = session.Query(ctx, statement, odriver.QueryOptions{
			Schema:         queryContext.Schema,
			Explain:        queryContext.Explain,
			ExplainFormat:  util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON),
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		})
		return queryErr
	})
	if err != nil {
		return nil, err
	}
	limit := 0
	if !queryContext.Explain && queryContext.Limit > 0 {
		limit = queryContext.Limit
	}
	return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		SkipPlan:             true,
		SkipMessages:         true,
		UppercaseColumnTypes: true,
		TruncatedWins:        true,
		Limit:                limit,
		ConvertValue:         toRowValue,
	}), nil
}

func toQueryResult(result *odriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	limit := 0
	if !queryContext.Explain && queryContext.Limit > 0 {
		limit = queryContext.Limit
	}
	return util.OmniResultToQueryResultWith(result, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		SkipPlan:             true,
		SkipMessages:         true,
		UppercaseColumnTypes: true,
		TruncatedWins:        true,
		Limit:                limit,
		ConvertValue:         toRowValue,
	})
}

// toRowValue maps one contract value onto the product's row value. A float is
// the number Hive stored, where the old driver wrote it as text with twenty
// fractional digits.
func toRowValue(value odriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case odriver.Null:
		return util.NullRowValue
	case odriver.Bool, odriver.Bytes, odriver.Float32, odriver.Float64,
		odriver.Int32, odriver.Int64, odriver.Uint32, odriver.Uint64, odriver.String:
		return util.OmniRowValue(value)
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}
