// Package oracle is Bytebase's Oracle driver, adapting
// github.com/bytebase/omni/oracle/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - Execute parses `-- txn-mode` without applying isolation (BEGIN opens a PL/SQL block).
//   - SQL*Plus commands are blanked in place before execution/querying to preserve
//     byte offsets for task run log ranges.
//   - TLS leaves hostname verification to crypto/tls so go-ora binds ServerName per dial.
package oracle

import (
	"context"
	"crypto/tls"
	"strings"

	omnidriver "github.com/bytebase/omni/driver"
	oracledriver "github.com/bytebase/omni/oracle/driver"
	oracleparser "github.com/bytebase/omni/oracle/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver       = (*Driver)(nil)
	_ db.Prober       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_ORACLE, newDriver)
}

// Driver is Bytebase's Oracle driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract and validates the
// target schema when DatabaseName is set.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{}, func(adapter *util.OmniAdapter) (omnidriver.Driver, error) {
		dataSource := config.DataSource
		cfg := omnidriver.Config{
			Host:          dataSource.GetHost(),
			Port:          dataSource.GetPort(),
			Database:      config.ConnectionContext.DatabaseName,
			User:          dataSource.GetUsername(),
			Password:      config.Password,
			Params:        connectionParams(dataSource),
			EngineVersion: config.ConnectionContext.EngineVersion,
		}
		if dataSource.GetUseSsl() {
			cfg.TLS = func(string) (*tls.Config, error) {
				return getTLSConfig(dataSource)
			}
		}
		if err := util.SetupSSHDial(dataSource, &cfg, adapter, true); err != nil {
			return nil, err
		}
		return oracledriver.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// connectionParams is the driver contract's Params for a data source: the
// data source's extra parameters in their own spelling, then the endpoint's
// service name field and SID field unless an extra parameter names the same
// option.
//
// go-ora upper-cases option keys, so two spellings of one option would leave the
// winner to map iteration order; the fields yield to the caller's parameters in
// any spelling, which is what the old driver's option map did.
func connectionParams(dataSource *storepb.DataSource) map[string]string {
	params := make(map[string]string, len(dataSource.GetExtraConnectionParameters())+2)
	named := make(map[string]bool, len(params))
	for key, value := range dataSource.GetExtraConnectionParameters() {
		params[key] = value
		named[strings.ToUpper(key)] = true
	}
	for option, value := range map[string]string{
		"SERVICE_NAME": dataSource.GetServiceName(),
		"SID":          dataSource.GetSid(),
	} {
		if value == "" || named[option] {
			continue
		}
		params[option] = value
	}
	// An unset dial timeout leaves go-ora to the transport's own, which behind an
	// unreachable primary can outlast the whole connect check before the next
	// address in a SERVER list is tried. go-ora reads either spelling, and
	// upper-cases the keys, so the check is on the upper-cased names.
	if !named["CONNECTION TIMEOUT"] && !named["CONNECT TIMEOUT"] {
		params["CONNECTION TIMEOUT"] = "10"
	}
	return params
}

// getTLSConfig returns the TLS configuration for the data source's TLS
// section. It leaves the name check to crypto/tls instead of
// util.GetTLSConfig's verifier, which checks the data source's Host: go-ora
// sets ServerName to each server it dials, so a SERVER address or a listener
// redirect is checked against its own name.
func getTLSConfig(dataSource *storepb.DataSource) (*tls.Config, error) {
	config, err := util.GetTLSConfig(dataSource)
	if err != nil || config == nil {
		return config, err
	}
	config.InsecureSkipVerify = !dataSource.GetVerifyTlsCertificate()
	config.VerifyPeerCertificate = nil
	return config, nil
}

// Execute runs the script's statements, reporting each to the task run's log.
// The script's transaction-mode directive is parsed here, because the product
// sends it inside the statement text; the isolation directive is not applied,
// as before.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("create database is not supported for Oracle")
	}
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassMaxRetries:            true,
		Preprocess:                withoutSQLPlusCommands,
	})
}

// withoutSQLPlusCommands blanks the SQL*Plus commands of a script, which Oracle
// does not execute over a session: a script that starts with SET DEFINE OFF, as
// the SQL*Plus scripts the console imports do, used to run because Bytebase's
// splitter skipped them, and the driver sends every segment it splits.
//
// The commands are blanked rather than removed, so every other statement keeps
// its byte offsets and the range a task run logs still names the script's own
// text.
func withoutSQLPlusCommands(script string) string {
	var commands []oracleparser.Segment
	for _, segment := range oracleparser.Split(script) {
		if segment.Kind == oracleparser.SegmentSQLPlusCommand {
			commands = append(commands, segment)
		}
	}
	if len(commands) == 0 {
		return script
	}
	blanked := []byte(script)
	for _, command := range commands {
		end := command.ByteEnd
		// The command's line break goes with it, so the statement after it starts
		// at its own first character rather than at a blank line.
		for end < len(blanked) && (blanked[end] == '\r' || blanked[end] == '\n') {
			end++
		}
		for i := command.ByteStart; i < end; i++ {
			blanked[i] = ' '
		}
	}
	return string(blanked)
}

// CountAffectedRows returns the optimizer's estimate of the rows a DML
// statement changes.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// OwnsPrivateDatabaseLink reports whether the driver's account owns a private
// database link, querying on a session managed by the underlying omni driver.
func (d *Driver) OwnsPrivateDatabaseLink(ctx context.Context) (bool, error) {
	return d.Driver.(omnidriver.PrivateDatabaseLinkOwner).OwnsPrivateDatabaseLink(ctx)
}

// QueryConn runs a script for the editor on the held session.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryHeldConn(ctx, withoutSQLPlusCommands(statement), queryOptions(queryContext), util.SessionQueryOptions{
		WrapOpenError: "failed to open a session",
	}, util.QueryResultOptions{})
}

func queryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON))
}

// Probe runs a live review rule's probe on a session of its own. The Oracle
// rule's dry run passes Autocommit, because BEGIN opens a PL/SQL block, so
// EXPLAIN PLAN FOR cannot be wrapped in a transaction.
//
// A probe that is not autocommit is refused rather than run: the contract's
// Session has no transaction control, so a probe's statements could not be
// rolled back, and running them would leave what a dry run exists to avoid.
func (d *Driver) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return d.ProbeOmni(ctx, pCtx, statement, util.OmniProbeOptions{
		AutocommitOnly: true,
		FormatCell:     probeValue,
	})
}

// probeValue is one plan cell in the plain types the advisor reads.
func probeValue(value omnidriver.Value) any {
	if value.Kind == omnidriver.Structured {
		return nil
	}
	return util.DefaultProbeCell(value)
}
