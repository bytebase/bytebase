// Package dynamodb is Bytebase's DynamoDB driver: an adapter over
// github.com/bytebase/omni/dynamodb/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
//
// The AWS credential path stays here. util.GetAWSConnectionConfig reads the
// data source's static key pair or the SDK's default chain, then
// util.AssumeRoleIfNeeded assumes the configured role and caches the assumed
// keys. The omni driver takes a plain key pair, so this adapter hands it the
// configured pair, or the assumed role's keys read once at Open. An assumed
// role's keys therefore no longer refresh for the driver's life; a run longer
// than the role session needs a caller-side credential source the contract does
// not have yet.
//
// The endpoint mapping stays here too. An empty host leaves the AWS SDK's own
// endpoint resolution for the region, which is how real AWS is reached; a host
// with no scheme is reached over http, and over https when use_ssl is set. An
// explicit scheme in the host wins, as it did before. The adapter passes a TLS
// function only when it must make the omni driver choose https for a bare host:
// that function verifies with the system roots, so verify_tls_certificate is
// ignored exactly as the old driver ignored it. A data source that sets
// verify_tls_certificate=false keeps verifying until the product can tell an
// explicit opt-out from an unset default.
//
// The account ID is the one fact DynamoDB itself does not answer. The omni
// driver asks STS for it from SyncInstance when Params[accountId] is absent,
// with the error strings this driver reported. An operator can skip that call
// by setting the extra connection parameter accountId; the old driver always
// asked STS.
//
// An SSH data source is still ignored: this driver never opened a tunnel, and
// the adapter does not add one. Config.Dial is therefore not mapped.
//
// Three result shapes move with the contract, each recorded as a known
// difference rather than fixed silently:
//
//   - QueryOptions.Limit now caps the rows of a statement. The old driver
//     passed it to every ExecuteStatement page, so Limit=1 could return 2 rows.
//   - MaximumSQLResultSize is now enforced; the old driver read every page and
//     never truncated.
//   - A binary value crosses as the contract's Bytes kind instead of a proto
//     string holding raw bytes, which could fail protojson; a missing attribute
//     is an explicit NULL instead of a nil row value.
//
// SyncDBSchema also reports global and local secondary indexes and their key
// attributes as columns, and no longer reports an empty RANGE index for a table
// with no sort key. Both are omni's sync, not this adapter's. A sync failure's
// text is reported once where the old driver wrapped it twice: a describe
// failure now reads "failed to describe table: T: ..." instead of the same
// phrase repeated, and a listing failure reads "failed to list tables: ..."
// instead of "failed to list all tables: failed to list tables: ...".
package dynamodb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"strings"

	omnidriver "github.com/bytebase/omni/driver"
	omnidynamodb "github.com/bytebase/omni/dynamodb/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_DYNAMODB, newDriver)
}

// Driver is Bytebase's DynamoDB driver.
type Driver struct {
	inner *omnidynamodb.Driver
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the DynamoDB endpoint or STS; Ping and SyncInstance do.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource

	awsConfig, err := util.GetAWSConnectionConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load AWS config")
	}
	if err := util.AssumeRoleIfNeeded(ctx, &awsConfig, config.ConnectionContext, dataSource.GetAwsCredential()); err != nil {
		return nil, err
	}

	params := map[string]string{
		omnidynamodb.ParamRegion: awsConfig.Region,
	}
	var user, password string
	awsCredential := dataSource.GetAwsCredential()
	switch {
	case awsCredential.GetRoleArn() != "":
		// AssumeRoleIfNeeded put the assumed role's cached keys in awsConfig.
		// The contract takes a key pair, so read them once; a refreshing cache
		// has no place to live on the other side of the seam.
		value, err := awsConfig.Credentials.Retrieve(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve AWS credentials")
		}
		user, password = value.AccessKeyID, value.SecretAccessKey
		params[omnidynamodb.ParamSessionToken] = value.SessionToken
	case awsCredential.GetAccessKeyId() != "":
		// The same static pair util.GetAWSConnectionConfig gave the AWS SDK.
		user, password = awsCredential.GetAccessKeyId(), awsCredential.GetSecretAccessKey()
		params[omnidynamodb.ParamSessionToken] = awsCredential.GetSessionToken()
	default:
		// Leave both empty: the omni driver's LoadDefaultConfig reads the same
		// default chain the old driver read.
	}
	if accountID := dataSource.GetExtraConnectionParameters()["accountId"]; accountID != "" {
		params[omnidynamodb.ParamAccountID] = accountID
	}

	inner, err := omnidynamodb.Open(ctx, omnidriver.Config{
		Host:          dataSource.GetHost(),
		Port:          dataSource.GetPort(),
		Database:      config.ConnectionContext.DatabaseName,
		User:          user,
		Password:      password,
		TLS:           endpointTLS(dataSource),
		Params:        params,
		EngineVersion: config.ConnectionContext.EngineVersion,
	})
	if err != nil {
		return nil, err
	}
	return &Driver{inner: inner}, nil
}

// endpointTLS returns the TLS settings that make the omni driver reach a bare
// host over https, or nil to leave the endpoint scheme and transport as the old
// driver left them. The empty config verifies the endpoint against the system
// roots and its host name, which is what the AWS SDK's own transport did.
func endpointTLS(dataSource *storepb.DataSource) func(string) (*tls.Config, error) {
	host := dataSource.GetHost()
	if !dataSource.GetUseSsl() || host == "" || strings.Contains(host, "://") {
		return nil
	}
	return func(string) (*tls.Config, error) {
		return &tls.Config{}, nil
	}
}

// Close closes the driver.
func (d *Driver) Close(context.Context) error {
	if d.inner == nil {
		return nil
	}
	return d.inner.Close()
}

// Ping lists one table as a connectivity check, as DynamoDB has no ping.
func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// GetDB returns nil: DynamoDB has no database/sql handle.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute runs each statement of script as its own ExecuteTransaction and
// reports each to the task run's log. The count is always 0 because PartiQL
// writes report no affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.inner.Execute(ctx, statement, omnidriver.ExecuteOptions{
		Log: executeLogger{opts: opts},
	})
}

// QueryConn runs a read-only PartiQL script, one result per statement. The
// *sql.Conn is ignored: DynamoDB has no session state.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	results, err := session.Query(ctx, statement, omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}

	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, toQueryResult(result, queryContext))
	}
	return converted, nil
}

// SyncInstance reads the account and region that name DynamoDB's pseudo
// database. The account comes from STS unless accountId was configured.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema reads every table of the region.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.inner.SyncDatabase(ctx)
}

// Dump writes nothing: a synced table does not carry the attribute types,
// billing mode or throughput a create needs, so the metadata could not recreate
// the table it describes.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

func toQueryResult(result *omnidriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	converted := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	for _, column := range result.Columns {
		converted.ColumnNames = append(converted.ColumnNames, column.Name)
		converted.ColumnTypeNames = append(converted.ColumnTypeNames, column.TypeName)
	}
	for _, row := range result.Rows {
		convertedRow := &v1pb.QueryRow{}
		for _, value := range row {
			convertedRow.Values = append(convertedRow.Values, convertValue(value))
		}
		converted.Rows = append(converted.Rows, convertedRow)
	}
	converted.RowsCount = int64(len(converted.Rows))
	if result.Truncated {
		converted.Error = util.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
	} else if result.Err != nil {
		converted.Error = result.Err.Error()
	}
	return converted
}

func convertValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case omnidriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}

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
	var transactionType storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case omnidriver.Begin:
		transactionType = storepb.TaskRunLog_TransactionControl_BEGIN
	case omnidriver.Commit:
		transactionType = storepb.TaskRunLog_TransactionControl_COMMIT
	case omnidriver.Rollback:
		transactionType = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		// An operation the contract does not name is left unspecified.
	}
	l.opts.LogTransactionControl(transactionType, errorString(err))
}

func (l executeLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
