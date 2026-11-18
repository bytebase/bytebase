// Package mssql is Bytebase's SQL Server driver: an adapter over
// github.com/bytebase/omni/mssql/driver that maps Bytebase's connection config
// onto the driver contract, and the contract's results back into the product's.
// The conversation with SQL Server — session setup, metadata sync, the catalog's
// dump, interactive queries and change scripts — is the driver's.
//
// The adapter keeps what is product: the data source to contract conversion
// (the go-mssqldb parameters the old driver composed, its CA file and its
// parameter precedence), the Azure login as a Config.TokenProvider,
// QueryContext/ExecuteOptions mapping, the v1pb conversion and the task-run
// logger.
//
// Constraints of the engine and the contract, here rather than at each call:
//   - GetDB returns nil. The driver owns its pool; db.Driver still requires the
//     method, and every production caller already guards a nil handle. The one
//     behaviour that changes with it is the editor's session: QueryConn opens
//     its own session per call, so a script's statements share a connection
//     while the separate requests of one editor stream no longer do.
//   - Config.TLS is not set. The data source's TLS settings are go-mssqldb DSN
//     parameters, and the client's own parse of them is what the old driver and
//     its tests described; handing the client a *tls.Config instead would move
//     certificate verification to util's custom verifier and report different
//     errors for the same failure. The CA therefore travels as the
//     `certificate` parameter, which go-mssqldb reads only from a path, so the
//     adapter writes the data source's CA to a temporary file and removes it on
//     Close. Mutual TLS is not carried: the old driver did not read ssl_cert or
//     ssl_key either.
//   - Config.Dial is not set. The driver this replaced never read the SSH
//     fields, so an SSH data source connected to the direct host; preserving
//     that is parity, and adding a tunnel is a behavior change of its own.
//   - Config.ReadOnly is ignored: SQL Server has no read-only session setting
//     the driver can open.
//   - QueryContext.Schema is ignored: unqualified names resolve in the
//     session's default schema, as they did before.
//   - QueryContext.Limit is enforced by rewriting a statement that returns
//     rows, and, for the rows a rewrite cannot reach (a procedure call or an
//     OUTPUT clause), by the driver reading past the limit.
//   - Execute parses the `-- txn-mode` directive and ignores `-- txn-isolation`,
//     which the driver this replaced parsed and then dropped: BeginTx ran with
//     the server's default isolation. The defect is preserved rather than fixed
//     here.
//   - A masked procedure call or OUTPUT clause keeps its rows and sets the
//     result's error, as before; the driver withholds them and states why.
//
// Known differences from the driver this replaced. The first group was measured
// by running the old path and this adapter over one corpus on the pinned server
// (the recording is testdata/cutover_parity.json, replayed by
// cutover_parity_test.go); the second group is read from the two
// implementations.
//
// Measured:
//   - The synced metadata is the richer one omni's sync reads: an index keeps
//     its INCLUDE list and its filtered-index WHERE (the old sync made the
//     INCLUDE column a key column and dropped the predicate), a DATETIMEOFFSET
//     column keeps its precision, a sequence keeps its type, start, increment,
//     bounds, cache and cycle, an INSTEAD OF trigger is read from
//     is_instead_of_trigger where the old sync called it AFTER, and a
//     check-constraint and foreign-key comment arrive.
//   - Dump renders through omni's catalog instead of Bytebase's schema renderer:
//     sequences and triggers are written at all (the old renderer dropped both,
//     so its dump could not restore them), a filtered index keeps its INCLUDE
//     list and predicate, a view definition ends with exactly one `;` where the
//     old renderer wrote the one the server stored plus its own, and a default
//     constraint keeps the name the server gave it.
//   - Execute names the statement the server reported a failure at.
//   - A lone failing QueryConn statement answers with its error on the result
//     rather than as the call's error; a failure the driver meets while reading
//     a batch's rows comes back as the client's own text (mssql: ... (N)) rather
//     than the server messages the old driver unpacked; and a procedure's later
//     result sets are still returned after every statement's result when a later
//     batch fails, where the old driver dropped them.
//   - An empty, non-NULL binary cell is an empty byte value; the old driver
//     reported NULL for it because it tested the byte slice's length.
//   - An explain result carries the plan's format in QueryPlan as well as the
//     plan's rows. The API already labelled it, so nothing the user sees moves.
//   - The query-result size cap is byte-accurate and does not stop reading the
//     result set, where the old driver measured proto size at power-of-two row
//     counts. The same cap can therefore keep a different number of rows; the
//     "exceeds max allowed output size" message is unchanged.
//
// Read from the two implementations:
//   - SyncInstance states ORDER BY name and wraps a failed query with the
//     statement text; the corpus's server returned the name order anyway.
//   - The sync's row-count query is restricted to the synced schemas, which
//     changes the statement it sends but not a synced table's count, and a
//     procedure comment is read where the old sync did not.
//   - A batch of more than one statement reports no affected-rows rows, because
//     SQL Server delivers a batch's per-statement counts only through the
//     client's message loop and the driver reads them for a lone statement. The
//     statements still run, in order, in one request.
//   - A statement that reports only its affected rows carries RowsCount 1, the
//     one row the API shows; the old driver left it at 0. The API and the other
//     engines read the count as len(rows).
//   - A RECEIVE or READTEXT result is withheld under masking, where the old
//     driver could not match its rows to its statement and failed the call.
//   - CountAffectedRows turns SHOWPLAN_ALL off and reports the server's message;
//     a database the login may not open is driver.ErrNotFound; and an Azure
//     credential that cannot be built fails at Open rather than at the first
//     connection.
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
	_ db.Driver       = (*Driver)(nil)
	_ db.RowEstimator = (*Driver)(nil)
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

// Close closes the driver's connections and removes the CA file it wrote.
func (d *Driver) Close(context.Context) error {
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
// *sql.DB, and every production caller guards a nil handle. The editor keeps a
// session per call, which is the one behaviour this changes.
func (*Driver) GetDB() *sql.DB {
	return nil
}

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
