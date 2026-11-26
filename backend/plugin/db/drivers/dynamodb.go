package drivers

import (
	"context"
	"crypto/tls"
	"strings"

	odriver "github.com/bytebase/omni/driver"
	omnidynamodb "github.com/bytebase/omni/dynamodb/driver"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// dynamodbSpec opens without reaching the endpoint, though it reaches STS when
// the data source assumes a role, because util.LoadAWSConfig resolves the
// role's keys.
var dynamodbSpec = spec{
	open: func(ctx context.Context, _ *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
		cfg, err := dynamodbConfig(ctx, config)
		if err != nil {
			return nil, err
		}
		return omnidynamodb.Open(ctx, cfg)
	},
	query: func(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
		return d.QueryEphemeralConn(ctx, statement, odriver.QueryOptions{
			Limit:          queryContext.Limit,
			Explain:        queryContext.Explain,
			MaxResultBytes: queryContext.MaximumSQLResultSize,
		}, util.QueryResultOptions{
			SkipPlan:      true,
			SkipMessages:  true,
			TruncatedWins: true,
			ConvertValue: func(value odriver.Value) *v1pb.RowValue {
				switch value.Kind {
				case odriver.Null:
					return util.NullRowValue
				case odriver.Bool, odriver.Bytes, odriver.String:
					return util.OmniRowValue(value)
				case odriver.Structured:
					return jsonValue(value)
				default:
					return printedValue(value)
				}
			},
		})
	},
}

func dynamodbConfig(ctx context.Context, config db.ConnectionConfig) (odriver.Config, error) {
	dataSource := config.DataSource
	awsConfig, err := util.LoadAWSConfig(ctx, config)
	if err != nil {
		return odriver.Config{}, err
	}
	params := map[string]string{
		omnidynamodb.ParamRegion: awsConfig.Region,
	}
	var user, password string
	awsCredential := dataSource.GetAwsCredential()
	switch {
	case awsCredential.GetRoleArn() != "":
		// The contract takes a key pair, so the assumed role's cached keys are
		// read once; a refreshing cache has no place on the other side.
		value, err := awsConfig.Credentials.Retrieve(ctx)
		if err != nil {
			return odriver.Config{}, errors.Wrap(err, "failed to retrieve AWS credentials")
		}
		user, password = value.AccessKeyID, value.SecretAccessKey
		params[omnidynamodb.ParamSessionToken] = value.SessionToken
	case awsCredential.GetAccessKeyId() != "":
		user, password = awsCredential.GetAccessKeyId(), awsCredential.GetSecretAccessKey()
		params[omnidynamodb.ParamSessionToken] = awsCredential.GetSessionToken()
	default:
		// Both stay empty, so the omni driver reads the default chain.
	}
	if accountID := dataSource.GetExtraConnectionParameters()["accountId"]; accountID != "" {
		params[omnidynamodb.ParamAccountID] = accountID
	}
	return odriver.Config{
		Host:          dataSource.GetHost(),
		Port:          dataSource.GetPort(),
		Database:      config.ConnectionContext.DatabaseName,
		User:          user,
		Password:      password,
		TLS:           dynamodbEndpointTLS(dataSource),
		Params:        params,
		EngineVersion: config.ConnectionContext.EngineVersion,
	}, nil
}

// dynamodbEndpointTLS makes the driver reach a bare host over https, or is nil
// to leave the endpoint's scheme and transport alone. The empty config verifies
// against the system roots and the host name, as the AWS SDK's transport does.
func dynamodbEndpointTLS(dataSource *storepb.DataSource) func(string) (*tls.Config, error) {
	host := dataSource.GetHost()
	if !dataSource.GetUseSsl() || host == "" || strings.Contains(host, "://") {
		return nil
	}
	return func(string) (*tls.Config, error) {
		return &tls.Config{}, nil
	}
}
