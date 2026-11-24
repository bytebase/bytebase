// Package mssql is Bytebase's SQL Server driver, adapting
// github.com/bytebase/omni/mssql/driver to Bytebase's db.Driver interface.
//
// Constraints:
//   - TLS settings travel as go-mssqldb parameters; CA material is written to a
//     temp file for go-mssqldb's `certificate` parameter and removed on Close.
//   - Execute parses `-- txn-mode` and ignores `-- txn-isolation`.
//   - QueryConn holds one serialized session for the driver's lifetime.
package mssql

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	odriver "github.com/bytebase/omni/driver"
	mssqldriver "github.com/bytebase/omni/mssql/driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MSSQL, newDriver)
}

// Driver is Bytebase's SQL Server driver.
type Driver struct {
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract without dialing the
// server; Ping or the first statement connects.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if err := d.OpenWith(ctx, util.AdapterOptions{}, func(adapter *util.OmniAdapter) (odriver.Driver, error) {
		cfg, caFile, err := mssqlConfig(config)
		if err != nil {
			return nil, err
		}
		if caFile != "" {
			adapter.AddCloseFunc(func() error {
				removeCAFile(caFile)
				return nil
			})
		}
		return mssqldriver.Open(ctx, cfg)
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// mssqlConfig maps the data source onto the driver contract. caFile is the CA
// written to disk for go-mssqldb, or empty; the caller removes it.
//
// Extra Connection Parameters keep the precedence the old driver gave them:
// they are matched case-insensitively, override the app name and the login, and
// carry the server, port, TLS mode and certificate the client reads. The
// database and an Azure credential come from Bytebase, never from them.
func mssqlConfig(config db.ConnectionConfig) (odriver.Config, string, error) {
	ds := config.DataSource

	// go-mssqldb matches parameter keys case-insensitively and accepts each key
	// once, so the parameters merge by lowercased key, in a stable order.
	params := map[string]string{}
	extra := ds.GetExtraConnectionParameters()
	for _, key := range slices.Sorted(maps.Keys(extra)) {
		params[strings.ToLower(key)] = extra[key]
	}

	host, port := ds.GetHost(), ds.GetPort()
	if server, ok := params["server"]; ok {
		delete(params, "server")
		host = server
	}
	if value, ok := params["port"]; ok {
		delete(params, "port")
		port = value
	}

	database := util.ResolveDatabaseName(config)
	if database == "" {
		database = params["database"]
	}
	delete(params, "database")

	user, password := ds.GetUsername(), config.Password
	if value, ok := params["user id"]; ok {
		user = value
	}
	if value, ok := params["password"]; ok {
		password = value
	}
	delete(params, "user id")
	delete(params, "password")

	appName := "bytebase"
	if value, ok := params["app name"]; ok {
		appName = value
	}
	delete(params, "app name")

	caFile, err := writeCAFile(ds)
	if err != nil {
		return odriver.Config{}, "", err
	}
	if ds.GetUseSsl() {
		// TLS mode TLS sets a floor that Extra Parameters may raise but not
		// lower: encrypt=true unless they ask for strict, and
		// TrustServerCertificate=false when Verify server certificate is on or
		// they ask for it. The section's CA replaces a certificate parameter.
		if !strings.EqualFold(params["encrypt"], "strict") {
			params["encrypt"] = "true"
		}
		trust, err := strconv.ParseBool(params["trustservercertificate"])
		verify := ds.GetVerifyTlsCertificate() || (err == nil && !trust)
		params["trustservercertificate"] = strconv.FormatBool(!verify)
		if caFile != "" {
			params["certificate"] = caFile
		}
	} else if _, ok := params["trustservercertificate"]; !ok {
		params["trustservercertificate"] = "true"
	}

	// tlsmin stays unset: mssql/driver defaults it to 1.0 for a server that
	// offers only TLS 1.0, and an Extra Connection Parameter replaces it.
	cfg := odriver.Config{
		Host:            host,
		Port:            port,
		Database:        database,
		ApplicationName: appName,
		Params:          params,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	if ds.GetAuthenticationType() == storepb.DataSource_AZURE_IAM {
		// The token connector authenticates instead of a login, so the fedauth
		// parameter and the login the old driver put in the DSN have no reader.
		delete(params, "fedauth")
		credential, err := util.GetAzureConnectionConfig(config)
		if err != nil {
			removeCAFile(caFile)
			return odriver.Config{}, "", err
		}
		cfg.TokenProvider = azureTokenProvider(credential)
		return cfg, caFile, nil
	}
	cfg.User, cfg.Password = user, password
	return cfg, caFile, nil
}

// azureAudience is the scope Azure SQL asks an Entra ID token for.
const azureAudience = "https://database.windows.net/.default"

// azureTokenProvider asks credential for an Azure SQL access token. It is
// called for each new connection, as the client's connector does.
func azureTokenProvider(credential azcore.TokenCredential) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		token, err := credential.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{azureAudience}})
		if err != nil {
			return "", err
		}
		return token.Token, nil
	}
}

// writeCAFile writes the data source's CA to a temporary file, because
// go-mssqldb reads a CA only from a .pem or .der path. It returns "" when TLS is
// off or no CA is set.
func writeCAFile(ds *storepb.DataSource) (string, error) {
	if !ds.GetUseSsl() || ds.GetSslCa() == "" {
		return "", nil
	}
	file, err := os.CreateTemp("", "cert-*.pem")
	if err != nil {
		return "", errors.Wrap(err, "failed to create the CA certificate file")
	}
	_, err = file.WriteString(ds.GetSslCa())
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		removeCAFile(file.Name())
		return "", errors.Wrap(err, "failed to write the CA certificate file")
	}
	return file.Name(), nil
}

func removeCAFile(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil {
		slog.Warn("failed to delete temporary file", "path", path, log.BBError(err))
	}
}

// Execute runs the script's batches, in one transaction unless the `-- txn-mode`
// directive turns it off, reporting each step to the task run's log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// -- txn-isolation is parsed and dropped, as it was before: the old driver
	// opened the transaction with the server's default.
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
	})
}

// CountAffectedRows returns the optimizer's estimate for one statement, read
// from the plan SQL Server prints for it.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.Driver.(db.RowEstimator).CountAffectedRows(ctx, statement)
}

// QueryConn runs a script on the held session and converts any results
// (including partial results from earlier batches when an error occurs).
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryHeldSession(ctx, statement, util.OmniQueryOptions(
		queryContext,
		false,
		util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_XML),
	), util.SessionQueryOptions{})
	return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{}), err
}
