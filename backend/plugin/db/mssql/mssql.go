// Package mssql is Bytebase's SQL Server driver: an adapter over
// github.com/bytebase/omni/mssql/driver that maps Bytebase's connection config
// onto the driver contract, and the contract's results back into the product's.
// The conversation with SQL Server — session setup, metadata sync, the catalog's
// dump, interactive queries and change scripts — is the driver's.
//
// The adapter keeps what is product: the data source to contract conversion (the
// go-mssqldb parameters, CA file and parameter precedence), the Azure login as a
// Config.TokenProvider, QueryContext/ExecuteOptions mapping, the v1pb conversion
// and the task-run logger.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - GetDB returns nil: omni's driver owns its pool, and every production
//     caller guards a nil handle. QueryConn holds one session for the driver's
//     life, serialized, so a stream's requests see the session state the pooled
//     connection carried before.
//   - Config.TLS is not set: the data source's TLS settings stay go-mssqldb
//     parameters, so verification, the CA and the TLS minimum are the client's
//     parse of them; tlsmin is 1.0 unless a parameter sets it. The CA travels as
//     the `certificate` parameter, which the client reads only from a path, so
//     the adapter writes it to a temporary file and removes it on Close.
//   - Config.Dial is not set: an SSH data source is dialled directly, as it was
//     before this adapter. Adding the tunnel is a separate change.
//   - Config.ReadOnly is ignored: SQL Server has no read-only session setting.
//   - QueryContext.Schema is ignored: unqualified names resolve in the session's
//     default schema.
//   - QueryContext.Limit rewrites a statement that returns rows; for rows a
//     rewrite cannot reach (a procedure call, an OUTPUT clause) it is applied as
//     they are read.
//   - Execute parses `-- txn-mode` and ignores `-- txn-isolation`, so a
//     transaction keeps the server's default isolation, as before.
//   - A masked procedure call or OUTPUT clause keeps its rows and reports the
//     result's error.
//   - A count-only statement's result carries one "Affected Rows" row, and
//     RowsCount counts it.
//
// cutover_parity_test.go replays a recording of the driver this replaced over
// the same corpus and names the cases whose behavior moved.
package mssql

import (
	"context"
	"database/sql"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	mssqldriver "github.com/bytebase/omni/mssql/driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ db.Driver         = (*Driver)(nil)
	_ db.RowEstimator   = (*Driver)(nil)
	_ db.SessionQuerier = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_MSSQL, newDriver)
}

// Driver is Bytebase's SQL Server driver.
type Driver struct {
	impl *mssqldriver.Driver
	// certFilePath is the data source's CA written to disk, because go-mssqldb
	// reads a certificate only from a path. Close removes it.
	certFilePath string

	// sessionMu serializes QueryConn on the session it shares.
	sessionMu sync.Mutex
	// session is the connection the driver's interactive queries share, opened
	// on the first QueryConn. The editor keeps one driver for a stream, so its
	// requests see each other's session state as they did on the pooled
	// connection this replaced.
	session odriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server, as the driver this replaced did not: the pool connects on first
// use, and Ping or the first statement reports an unreachable server.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, caFile, err := mssqlConfig(config)
	if err != nil {
		return nil, err
	}
	impl, err := mssqldriver.Open(ctx, cfg)
	if err != nil {
		removeCAFile(caFile)
		return nil, err
	}
	d.impl = impl
	d.certFilePath = caFile
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

	database := config.ConnectionContext.DatabaseName
	if database == "" {
		database = ds.GetDatabase()
	}
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

// Close closes the driver's connections, the session its queries shared, and
// the CA file it wrote.
func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	if d.session != nil {
		_ = d.session.Close()
		d.session = nil
	}
	d.sessionMu.Unlock()
	removeCAFile(d.certFilePath)
	d.certFilePath = ""
	if d.impl == nil {
		return nil
	}
	return d.impl.Close()
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.impl.Ping(ctx)
}

// GetDB returns nil: SQL Server is driven through omni's sessions, not a
// *sql.DB, and every production caller guards a nil handle.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// OwnsQuerySession reports that QueryConn runs on the session the adapter holds
// rather than on the *sql.Conn a caller passes, which is unused.
func (*Driver) OwnsQuerySession() {}

// Execute runs the script's batches, in one transaction unless the `-- txn-mode`
// directive turns it off, reporting each step to the task run's log.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	// -- txn-isolation is parsed and dropped, as it was before: the old driver
	// opened the transaction with the server's default.
	transactionMode := config.Mode
	if transactionMode == transaction.ModeUnspecified {
		transactionMode = transaction.DefaultMode()
	}

	executeOptions := odriver.ExecuteOptions{
		Transaction:    odriver.Transactional,
		CreateDatabase: opts.CreateDatabase,
		MaxRetries:     opts.MaximumRetries,
		Log:            executeLogger{opts: opts},
	}
	if transactionMode == transaction.ModeOff {
		executeOptions.Transaction = odriver.AutoCommit
	}
	return d.impl.Execute(ctx, statement, executeOptions)
}

// CountAffectedRows returns the optimizer's estimate for one statement, read
// from the plan SQL Server prints for it.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.impl.CountAffectedRows(ctx, statement)
}

// SyncInstance reads the server version and its user databases.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema reads the database the driver was opened on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.impl.SyncDatabase(ctx)
}
