package drivers

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

var mssqlSpec = spec{
	open: func(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, caFile, err := mssqlConfig(config)
		if err != nil {
			return nil, err
		}
		if caFile != "" {
			d.AddCloseFunc(func() error {
				removeCAFile(caFile)
				return nil
			})
		}
		return mssqldriver.Open(ctx, cfg)
	},
	// -- txn-isolation is parsed and dropped: the transaction opens with the
	// server's default.
	execute: util.ExecuteConfig{
		ParseTransactionDirective: true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
	},
	// The results of the batches before a failing one are returned with the
	// error.
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		results, err := d.QueryHeldSession(ctx, statement, util.OmniQueryOptions(
			queryContext,
			false,
			util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "TEXT", v1pb.QueryOption_XML),
		), util.SessionQueryOptions{})
		return util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{}), err
	},
}

// mssqlConfig maps the data source onto the driver contract. caFile is the CA
// written to disk for go-mssqldb, or empty; the caller removes it.
//
// Extra Connection Parameters are matched case-insensitively, override the app
// name and the login, and carry the server, port, TLS mode and certificate the
// client reads. The database and an Azure credential come from Bytebase, never
// from them.
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
		// TLS mode sets a floor that Extra Parameters may raise but not lower:
		// encrypt=true unless they ask for strict, and
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
		// parameter has no reader.
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
// called for each new connection.
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
