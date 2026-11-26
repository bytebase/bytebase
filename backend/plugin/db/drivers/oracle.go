package drivers

import (
	"context"
	"crypto/tls"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	oracledriver "github.com/bytebase/omni/oracle/driver"
	oracleparser "github.com/bytebase/omni/oracle/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var oracleSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		dataSource := config.DataSource
		cfg := odriver.Config{
			Host:          dataSource.GetHost(),
			Port:          dataSource.GetPort(),
			Database:      config.ConnectionContext.DatabaseName,
			User:          dataSource.GetUsername(),
			Password:      config.Password,
			Params:        oracleParams(dataSource),
			EngineVersion: config.ConnectionContext.EngineVersion,
		}
		if dataSource.GetUseSsl() {
			cfg.TLS = func(string) (*tls.Config, error) {
				return oracleTLSConfig(dataSource)
			}
		}
		if err := util.SetupSSHDial(dataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		return oracledriver.Open(ctx, cfg)
	},
	// The transaction-mode directive is parsed because the product sends it in
	// the statement text; the isolation directive is not applied.
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassMaxRetries:            true,
		RefuseCreateDatabase:      true,
		Preprocess:                withoutSQLPlusCommands,
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryHeldConn(
			ctx,
			withoutSQLPlusCommands(statement),
			util.OmniQueryOptions(queryContext, true, util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_JSON)),
			util.SessionQueryOptions{WrapOpenError: "failed to open a session"},
			util.QueryResultOptions{},
		)
	},
	// A probe must pass Autocommit: BEGIN opens a PL/SQL block, so EXPLAIN PLAN
	// FOR cannot be wrapped in a transaction, and the session has no other
	// transaction control to roll a probe back with.
	probe: func(*Driver) util.OmniProbeOptions {
		return util.OmniProbeOptions{
			AutocommitOnly: true,
			FormatCell: func(value odriver.Value) any {
				if value.Kind == odriver.Structured {
					return nil
				}
				return util.DefaultProbeCell(value)
			},
		}
	},
}

// oracleParams is the data source's extra parameters in their own spelling,
// then the service name and SID fields unless an extra parameter names the
// same option. go-ora upper-cases option keys, so the fields yield to the
// parameters in any spelling.
func oracleParams(dataSource *storepb.DataSource) map[string]string {
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
	// address in a SERVER list is tried.
	if !named["CONNECTION TIMEOUT"] && !named["CONNECT TIMEOUT"] {
		params["CONNECTION TIMEOUT"] = "10"
	}
	return params
}

// oracleTLSConfig leaves the name check to crypto/tls instead of
// util.GetTLSConfig's verifier, which checks the data source's Host: go-ora
// sets ServerName to each server it dials, so a SERVER address or a listener
// redirect is checked against its own name.
func oracleTLSConfig(dataSource *storepb.DataSource) (*tls.Config, error) {
	config, err := util.GetTLSConfig(dataSource)
	if err != nil || config == nil {
		return config, err
	}
	config.InsecureSkipVerify = !dataSource.GetVerifyTlsCertificate()
	config.VerifyPeerCertificate = nil
	return config, nil
}

// withoutSQLPlusCommands blanks the SQL*Plus commands of a script, which Oracle
// does not execute over a session. They are blanked rather than removed, so
// every other statement keeps its byte offsets and a logged range still names
// the script's own text.
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
		// The command's line break goes with it, so the next statement starts
		// at its own first character.
		for end < len(blanked) && (blanked[end] == '\r' || blanked[end] == '\n') {
			end++
		}
		for i := command.ByteStart; i < end; i++ {
			blanked[i] = ' '
		}
	}
	return string(blanked)
}
