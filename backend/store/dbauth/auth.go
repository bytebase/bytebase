package dbauth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Configure applies metadata database authentication settings to pgxConfig.
func Configure(ctx context.Context, pgxConfig *pgx.ConnConfig) ([]stdlib.OptionOpenDB, error) {
	authConfig, err := awsConfigFromPGXConfig(pgxConfig)
	if err != nil {
		return nil, err
	}
	if authConfig == nil || !authConfig.enabled {
		return nil, nil
	}

	tokenProvider, err := newAWSMetadataDBTokenProvider(ctx, authConfig.region)
	if err != nil {
		return nil, err
	}
	return awsOpenOptions(authConfig, tokenProvider), nil
}

// IsKeywordValueRuntimeParam reports whether key is a Bytebase metadata DB auth runtime parameter.
func IsKeywordValueRuntimeParam(key string) bool {
	switch key {
	case awsRDSIAMParam, awsRegionParam:
		return true
	default:
		return false
	}
}
