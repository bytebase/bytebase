// Package dynamodb is Bytebase's DynamoDB driver: an adapter over
// github.com/bytebase/omni/dynamodb/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's protos.
//
// The AWS credential path stays here. util.LoadAWSConfig reads the data
// source's static key pair or the SDK's default chain, then assumes the
// configured role and caches the assumed keys. The omni driver takes a plain
// key pair, so this adapter hands it the configured pair, or the assumed role's
// keys read once at Open. The assumed role's keys do not refresh, so a run
// longer than the role session needs a caller-side credential source the
// contract does not have yet.
//
// The endpoint mapping also stays here. An empty host leaves the AWS SDK's own
// endpoint resolution for the region, which is how real AWS is reached; a host
// with no scheme is reached over http, and over https when use_ssl is set. An
// explicit scheme in the host wins. TLS verifies against the system roots, so
// verify_tls_certificate is ignored: the product cannot yet distinguish an
// explicit opt-out from an unset default. An SSH data source is ignored, and
// Config.Dial is not mapped.
//
// The account ID is the one fact DynamoDB itself does not answer. The omni
// driver asks STS for it from SyncInstance when Params[accountId] is absent,
// with Bytebase's error strings. An operator can skip that call by setting the
// extra connection parameter accountId.
//
// The contract's result shapes are the driver's: QueryOptions.Limit caps a
// statement's rows, QueryOptions.MaxResultBytes truncates a result, a binary is
// the Bytes kind, and a missing attribute is an explicit NULL. SyncDBSchema
// reports every key attribute of the table and of its secondary indexes as a
// column, reports GLOBAL and LOCAL SECONDARY indexes, and reports no RANGE
// index for a table with no sort key. A sync failure names the failed operation
// once.
package dynamodb

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	omnidriver "github.com/bytebase/omni/driver"
	omnidynamodb "github.com/bytebase/omni/dynamodb/driver"
	"github.com/pkg/errors"

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
	util.OmniAdapter
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the DynamoDB endpoint; Ping and SyncInstance do. It reaches STS when the data
// source configures a role to assume, because util.LoadAWSConfig resolves the
// role's keys here.
func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dataSource := config.DataSource

	awsConfig, err := util.LoadAWSConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		omnidynamodb.ParamRegion: awsConfig.Region,
	}
	var user, password string
	awsCredential := dataSource.GetAwsCredential()
	switch {
	case awsCredential.GetRoleArn() != "":
		// util.LoadAWSConfig put the assumed role's cached keys in awsConfig.
		// The contract takes a key pair, so read them once; a refreshing cache
		// has no place to live on the other side of the seam.
		value, err := awsConfig.Credentials.Retrieve(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve AWS credentials")
		}
		user, password = value.AccessKeyID, value.SecretAccessKey
		params[omnidynamodb.ParamSessionToken] = value.SessionToken
	case awsCredential.GetAccessKeyId() != "":
		// The same static pair util.LoadAWSConfig gave the AWS SDK.
		user, password = awsCredential.GetAccessKeyId(), awsCredential.GetSecretAccessKey()
		params[omnidynamodb.ParamSessionToken] = awsCredential.GetSessionToken()
	default:
		// Leave both empty: the omni driver's LoadDefaultConfig reads the
		// default chain.
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
	return &Driver{OmniAdapter: util.OmniAdapter{Driver: inner}}, nil
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

// Execute runs each statement of script as its own ExecuteTransaction and
// reports each to the task run's log. The count is always 0 because PartiQL
// writes report no affected rows.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	return d.ExecuteScript(ctx, statement, opts, util.ExecuteConfig{})
}

// QueryConn runs a read-only PartiQL script, one result per statement.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	return d.QueryEphemeralConn(ctx, statement, omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	}, util.QueryResultOptions{
		SkipPlan:      true,
		SkipMessages:  true,
		TruncatedWins: true,
		ConvertValue:  convertValue,
	})
}

func convertValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Null:
		return util.NullRowValue
	case omnidriver.Bool, omnidriver.Bytes, omnidriver.String:
		return util.OmniRowValue(value)
	case omnidriver.Structured:
		encoded, err := json.Marshal(value.Structured)
		if err != nil {
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value.Structured)}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(encoded)}}
	default:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value)}}
	}
}
