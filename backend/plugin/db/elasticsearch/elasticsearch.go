// Package elasticsearch is Bytebase's Elasticsearch driver: an adapter over
// github.com/bytebase/omni/elasticsearch/driver that maps Bytebase's
// connection config onto the driver contract and the contract's results back
// into the product's protos.
//
// Elasticsearch is not a SQL-shaped engine, so each db.Driver method answers in
// its own way:
//
//   - Open builds the endpoint and the login and reaches nothing. The host and
//     port are the data source's, a scheme in the host wins, and https is
//     selected by use_ssl, by a host that names its own https, or by AWS IAM
//     (which always requires it). TLS material is the data source's;
//     verify_tls_certificate decides whether the endpoint's certificate is
//     checked, and a data source with no TLS settings still skips verification
//     when it is asked for https, as before. A data source with an SSH host
//     dials through the tunnel, which the old driver ignored: it reached the
//     endpoint directly whatever the jump host said.
//   - Ping asks the endpoint for its root document.
//   - QueryConn runs each Kibana Dev Console request of the script and answers
//     one result per request, and fails the call as before when a request
//     fails.
//   - Execute runs the same requests through the driver's script path and
//     reports 0 affected rows: an Elasticsearch answer states its own counts in
//     its columns.
//   - SyncInstance reads the version, the cluster's indices and its users;
//     SyncDBSchema returns the indices as one database named "node", which is
//     the name the old driver gave it.
//   - Dump writes nothing. A synced index states its document count and data
//     size, not the mappings and settings that would recreate it, so a dump of
//     the metadata could not create the index it describes.
//   - GetDB returns nil. Elasticsearch has no database/sql handle, and no
//     caller reached for one.
//
// `db.QueryContext.Schema`, `Limit` and `SkipMasking` are not mapped: a cluster
// is one namespace, a REST answer has no statement to rewrite with a row limit,
// and Elasticsearch answers with text the driver never relates to a column, so
// there is nothing for masking to withhold. `MaximumSQLResultSize` is the
// driver's own cap and marks the answer row truncated with the legacy message.
// `QueryContext.Explain` profiles the search: the older engines' explain runs
// the search rather than describing it, and the result says so.
//
// What stays here is what Bytebase decides and omni must not: the SSH tunnel
// (Config.Dial), the AWS IAM login (a SigV4 round tripper around the caller's
// transport, plus the role-assumption hints the connection test and sync errors
// have always carried), and the task run's execute log.
//
// Known differences from the driver this replaced, measured by running both
// against the pinned Elasticsearch 8.19.25 container over one corpus. The ones
// the cutover plan decided are noted there (docs/migration/cutover-changes.md in
// omni); the rest were found by that run and are fixes unless noted.
//
// Requests:
//
//   - Ping reads `GET /` where the old client sent `HEAD /`. The same answer,
//     the root document, now comes back with the version in it.
//   - The index-settings request is
//     `GET /*/_settings?filter_path=**.hidden&expand_wildcards=open,hidden`
//     where the old client sent the same filters ordered differently and also
//     named index.hidden. The answer, and so the synced metadata, is the same.
//   - The cat-indices request writes `pretty` where the old client wrote
//     `pretty=true`. The answer is the same.
//
// Answers:
//
//   - A search's object answer sorts its keys by name; the old driver took them
//     in map order, which changed from one answer to the next.
//   - A number in an object answer keeps the digits the endpoint wrote, so
//     max_score stays 1.0 where the old driver decoded through float64 and
//     printed 1, and a long past 2^53 is no longer rounded.
//   - Indices and users are sorted by name instead of arriving in the server's
//     and the answer object's order, so one sync and its golden hold one order.
//   - A partially applied bulk request fails the statement: the endpoint
//     answers 200 and states the failures in its body, which the old driver
//     read past.
//   - An answer with no body (a HEAD, an empty 200) is an empty result instead
//     of "failed to parse json body: unexpected end of JSON input", the result
//     size cap is enforced instead of ignored, and an explain of anything but a
//     search is refused instead of run.
//   - Execute reports each request to the task run's log. The old driver
//     reported nothing, because it ran the script through QueryConn.
//   - A transport failure keeps its text but now names the request it was
//     making, because it is net/http's own error: a refused certificate reads
//     `Get "https://...": x509: certificate signed by unknown authority` where
//     the old client had no request to name.
//   - A data source with no username and no password used to send
//     `Authorization: Basic Og==`; the driver now sends no Authorization header.
//     A secured endpoint that refuses the empty login refuses the absence of one
//     too, but the request on the wire differs.
//   - An SSH data source now reaches the endpoint through the jump host. The old
//     driver ignored the SSH settings, so Open dials the jump host where it used
//     to open without touching a network.
//   - A data source with no port used to reach the scheme's default (80 or 443)
//     because the old driver built "host:" and let net/http fill it in; the
//     endpoint now defaults to 9200, which is Elasticsearch's own port.
package elasticsearch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	esdriver "github.com/bytebase/omni/elasticsearch/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/opensearch-project/opensearch-go/v5/signer"
	"github.com/opensearch-project/opensearch-go/v5/signer/awsv2"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_ELASTICSEARCH, newDriver)
}

// Driver is Bytebase's Elasticsearch driver.
type Driver struct {
	inner     *esdriver.Driver
	sshClient *ssh.Client
	// aws reports that the endpoint is a managed OpenSearch domain reached
	// with SigV4, and awsRoleARN the role that was assumed for it. Both are
	// kept for the hints the connection test carries.
	aws        bool
	awsRoleARN string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the endpoint; Ping does.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource
	d := &Driver{}
	aws := dataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM

	cfg := omnidriver.Config{
		Host: endpointHost(dataSource),
		Port: dataSource.GetPort(),
	}
	var tlsConfig *tls.Config
	if secure(dataSource) {
		var err error
		tlsConfig, err = tlsSettings(dataSource)
		if err != nil {
			return nil, err
		}
	}

	var opts esdriver.Options
	if aws {
		// A managed domain signs each request with SigV4 through a transport
		// of the caller's, which then owns its own connections: the tunnel is
		// not available on this login path, as it was not before.
		d.aws = true
		d.awsRoleARN = dataSource.GetAwsCredential().GetRoleArn()
		transport, err := signedTransport(ctx, config, dataSource, tlsConfig)
		if err != nil {
			return nil, err
		}
		opts = esdriver.Options{OpenSearch: true, Transport: transport}
	} else {
		cfg.User = dataSource.GetUsername()
		cfg.Password = config.Password
		if tlsConfig != nil {
			cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
		}
		if dataSource.GetSshHost() != "" {
			sshClient, err := util.GetSSHClient(dataSource)
			if err != nil {
				return nil, err
			}
			d.sshClient = sshClient
			cfg.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
				return util.DialSSHTunnel(ctx, sshClient, network, address)
			}
		}
	}

	inner, err := esdriver.Open(ctx, cfg, opts)
	if err != nil {
		d.closeSSHClient()
		return nil, err
	}
	d.inner = inner
	return d, nil
}

// endpointHost is the data source's host as the old driver built it: a host that
// carries its own scheme also carries the port glued onto it. The contract's
// endpoint takes a host that names its own port, and ignores Config.Port for it.
func endpointHost(dataSource *storepb.DataSource) string {
	host := dataSource.GetHost()
	if port := dataSource.GetPort(); port != "" && strings.Contains(host, "://") {
		return host + ":" + port
	}
	return host
}

// secure reports whether the endpoint is reached over https: use_ssl asks for
// it, a managed OpenSearch domain requires it, and so does a host that names
// https in its own scheme.
func secure(dataSource *storepb.DataSource) bool {
	return dataSource.GetUseSsl() ||
		dataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(dataSource.GetHost())), "https://")
}

// tlsSettings returns the TLS settings the endpoint is verified with. A data
// source with no TLS settings keeps the old driver's default of encrypting
// without verifying, and a CA certificate is added to the endpoint's roots
// whether or not verification is on, which is where the old client put it.
func tlsSettings(dataSource *storepb.DataSource) (*tls.Config, error) {
	tlsConfig, err := util.GetTLSConfig(dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get TLS config")
	}
	if tlsConfig == nil {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	tlsConfig.MinVersion = tls.VersionTLS12
	if ca := dataSource.GetSslCa(); ca != "" {
		if tlsConfig.RootCAs == nil {
			tlsConfig.RootCAs = x509.NewCertPool()
		}
		if ok := tlsConfig.RootCAs.AppendCertsFromPEM([]byte(ca)); !ok {
			return nil, errors.New("failed to add CA certificate from PEM-encoded certificate")
		}
	}
	return tlsConfig, nil
}

// signedTransport builds the AWS IAM login path: the credentials are resolved
// here, and the transport signs every request the driver sends.
func signedTransport(ctx context.Context, config db.ConnectionConfig, dataSource *storepb.DataSource, tlsConfig *tls.Config) (http.RoundTripper, error) {
	if dataSource.GetRegion() == "" {
		return nil, errors.New("region is required for AWS IAM authentication")
	}
	awsCfg, err := util.GetAWSConnectionConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS config")
	}
	if err := util.AssumeRoleIfNeeded(ctx, &awsCfg, config.ConnectionContext, dataSource.GetAwsCredential()); err != nil {
		return nil, err
	}
	if _, err := awsCfg.Credentials.Retrieve(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve AWS credentials")
	}
	sig, err := awsv2.NewSigner(awsCfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS signer")
	}
	return &signedHTTP{
		base: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 5 * time.Second,
			TLSClientConfig:       tlsConfig,
		},
		signer: sig,
	}, nil
}

// signedHTTP signs each request before the transport under it sends it, which
// is what a managed OpenSearch domain requires.
type signedHTTP struct {
	base   http.RoundTripper
	signer signer.Signer
}

func (t *signedHTTP) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.signer.SignRequest(req); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}

func (d *Driver) Close(context.Context) error {
	d.closeSSHClient()
	if d.inner != nil {
		return d.inner.Close()
	}
	return nil
}

func (d *Driver) closeSSHClient() {
	if d.sshClient != nil {
		_ = d.sshClient.Close()
		d.sshClient = nil
	}
}

// Ping asks the endpoint for its root document. On a managed OpenSearch domain
// a refusal keeps the hints the old client added, since a support case reads
// them: the role that could not be assumed, and where to look otherwise.
func (d *Driver) Ping(ctx context.Context) error {
	if err := d.inner.Ping(ctx); err != nil {
		return d.connectionError(err)
	}
	return nil
}

// connectionError names what a managed OpenSearch domain's refusal means. The
// status is in the driver's error text, which is all the old client matched on
// either.
func (d *Driver) connectionError(err error) error {
	if !d.aws {
		return err
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "401") || strings.Contains(text, "403"):
		if d.awsRoleARN != "" {
			return errors.Errorf("authentication failed: unable to assume role %s: %v", d.awsRoleARN, err)
		}
		return errors.Errorf("authentication failed (consider using cross-account role if accessing different AWS account): %v", err)
	case strings.Contains(text, "404"):
		return errors.Errorf("endpoint not found (check if path is correct): %v", err)
	default:
		return errors.Wrapf(err, "failed to connect to OpenSearch")
	}
}

// GetDB returns nil: Elasticsearch has no database/sql handle.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute runs the script's requests in order, reporting each to the task run's
// log, and returns 0 affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.inner.Execute(ctx, statement, omnidriver.ExecuteOptions{
		Log: executeLogger{opts: opts},
	})
}

// QueryConn runs each request of a read-only script and answers one result per
// request. A request that fails fails the call, which is how the driver it
// replaced answered: earlier results in the same script did not come back.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	stmtResults, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}
	var results []*v1pb.QueryResult
	for _, stmtResult := range stmtResults {
		if stmtResult.Err != nil {
			return nil, stmtResult.Err
		}
		results = append(results, toQueryResult(stmtResult, queryContext))
	}
	return results, nil
}

// SyncInstance reads the server version, the cluster's indices and its users.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	var roles []*storepb.InstanceRole
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{Name: role.Name, Attribute: role.Attribute})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata:  &storepb.Instance{Roles: roles},
	}, nil
}

// SyncDBSchema reads the cluster's indices as one database.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.inner.SyncDatabase(ctx)
}

// Dump writes nothing: a synced index does not carry what recreates it.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

// toQueryResult maps one contract result onto the product's result.
func toQueryResult(result *omnidriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	converted := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	for _, column := range result.Columns {
		// Elasticsearch states no column type, and Bytebase left
		// ColumnTypeNames empty, so it is not filled with empty names either.
		converted.ColumnNames = append(converted.ColumnNames, column.Name)
	}
	for _, row := range result.Rows {
		convertedRow := &v1pb.QueryRow{}
		for _, value := range row {
			convertedRow.Values = append(convertedRow.Values, toRowValue(value))
		}
		converted.Rows = append(converted.Rows, convertedRow)
	}
	converted.RowsCount = int64(len(converted.Rows))
	if result.Truncated {
		converted.Error = util.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
	}
	converted.QueryPlan = toQueryPlan(result.Plan)
	return converted
}

// toRowValue maps one cell. Every answer this driver reads is text: a JSON
// value re-encoded, or the body as the endpoint wrote it.
func toRowValue(value omnidriver.Value) *v1pb.RowValue {
	if value.Kind == omnidriver.Null {
		return util.NullRowValue
	}
	if value.Kind != omnidriver.String {
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
	return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
}

// toQueryPlan maps the plan an explain answers with. The format crosses the
// contract as its enum name, and Elasticsearch states one plan format: JSON.
func toQueryPlan(plan *omnidriver.Plan) *v1pb.QueryResult_QueryPlan {
	if plan == nil {
		return nil
	}
	format := v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED
	switch strings.ToUpper(plan.Format) {
	case "JSON":
		format = v1pb.QueryOption_JSON
	case "TEXT":
		format = v1pb.QueryOption_TEXT
	case "XML":
		format = v1pb.QueryOption_XML
	case "YAML":
		format = v1pb.QueryOption_YAML
	default:
		// A format this product cannot name keeps the unspecified value.
	}
	return &v1pb.QueryResult_QueryPlan{
		Format:   format,
		Executed: plan.Executed,
		Content:  plan.Content,
	}
}

// executeLogger reports the driver contract's execute steps to the task run's
// log.
type executeLogger struct {
	opts db.ExecuteOptions
}

func (l executeLogger) CommandExecute(command omnidriver.Command) {
	l.opts.LogCommandExecute(&storepb.Range{
		Start: int32(command.ByteStart),
		End:   int32(command.ByteEnd),
	}, command.Text)
}

func (l executeLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l executeLogger) TransactionControl(op omnidriver.TransactionOp, err error) {
	l.opts.LogTransactionControl(transactionControlType(op), errorString(err))
}

func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func transactionControlType(op omnidriver.TransactionOp) storepb.TaskRunLog_TransactionControl_Type {
	switch op {
	case omnidriver.Begin:
		return storepb.TaskRunLog_TransactionControl_BEGIN
	case omnidriver.Commit:
		return storepb.TaskRunLog_TransactionControl_COMMIT
	case omnidriver.Rollback:
		return storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return storepb.TaskRunLog_TransactionControl_TYPE_UNSPECIFIED
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
