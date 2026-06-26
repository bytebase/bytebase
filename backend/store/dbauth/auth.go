package dbauth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
)

// Configure applies metadata database authentication settings to pgxConfig.
func Configure(ctx context.Context, pgxConfig *pgx.ConnConfig) ([]stdlib.OptionOpenDB, func() error, error) {
	var configures []func() ([]stdlib.OptionOpenDB, func() error, error)

	awsConfig, err := awsConfigFromPGXConfig(pgxConfig)
	if err != nil {
		return nil, nil, err
	}
	if awsConfig != nil && awsConfig.enabled {
		configures = append(configures, func() ([]stdlib.OptionOpenDB, func() error, error) {
			return configureAWS(ctx, awsConfig)
		})
	}

	gcpConfig, err := gcpConfigFromPGXConfig(pgxConfig)
	if err != nil {
		return nil, nil, err
	}
	if gcpConfig != nil && gcpConfig.enabled {
		configures = append(configures, func() ([]stdlib.OptionOpenDB, func() error, error) {
			return configureGCP(ctx, pgxConfig, gcpConfig)
		})
	}

	return configureSingleProvider(configures)
}

func configureSingleProvider(configures []func() ([]stdlib.OptionOpenDB, func() error, error)) ([]stdlib.OptionOpenDB, func() error, error) {
	switch len(configures) {
	case 0:
		return nil, nil, nil
	case 1:
		return configures[0]()
	default:
		return nil, nil, errors.New("multiple metadata database IAM auth providers are enabled")
	}
}

func configureAWS(ctx context.Context, authConfig *awsConfig) ([]stdlib.OptionOpenDB, func() error, error) {
	tokenProvider, err := newAWSMetadataDBTokenProvider(ctx, authConfig.region)
	if err != nil {
		return nil, nil, err
	}
	return awsOpenOptions(authConfig, tokenProvider), nil, nil
}

func configureGCP(ctx context.Context, pgxConfig *pgx.ConnConfig, authConfig *gcpConfig) ([]stdlib.OptionOpenDB, func() error, error) {
	dialer, err := newGCPMetadataDBDialer(ctx)
	if err != nil {
		return nil, nil, err
	}
	return configureGCPWithDialer(pgxConfig, authConfig, dialer)
}

// IsKeywordValueRuntimeParam reports whether key is a Bytebase metadata DB auth runtime parameter.
func IsKeywordValueRuntimeParam(key string) bool {
	switch key {
	case awsRDSIAMParam, awsRegionParam, gcpCloudSQLIAMParam, gcpCloudSQLInstanceConnectionNameParam:
		return true
	default:
		return false
	}
}
