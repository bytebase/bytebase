package dbauth

import (
	"context"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
)

const (
	gcpCloudSQLIAMParam                    = "bytebase_gcp_cloud_sql_iam"
	gcpCloudSQLInstanceConnectionNameParam = "bytebase_gcp_cloud_sql_instance_connection_name"
	gcpCloudSQLIPTypeParam                 = "bytebase_gcp_cloud_sql_ip_type"

	gcpCloudSQLIPTypePublic  = "public"
	gcpCloudSQLIPTypePrivate = "private"
	gcpCloudSQLIPTypePSC     = "psc"
)

type gcpConfig struct {
	enabled                bool
	instanceConnectionName string
	ipType                 string
}

type gcpDialer interface {
	Dial(ctx context.Context, instanceConnectionName string) (net.Conn, error)
	Close() error
}

type gcpMetadataDBDialer struct {
	dialer *cloudsqlconn.Dialer
}

func newGCPMetadataDBDialer(ctx context.Context, ipType string) (*gcpMetadataDBDialer, error) {
	opts := append([]cloudsqlconn.Option{cloudsqlconn.WithIAMAuthN()}, gcpDialIPOptions(ipType)...)
	dialer, err := cloudsqlconn.NewDialer(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GCP Cloud SQL dialer")
	}
	return &gcpMetadataDBDialer{dialer: dialer}, nil
}

// gcpDialIPOptions selects the Cloud SQL IP type (private or PSC) for the metadata
// database connection. Public IP is the cloudsqlconn default, so no option is
// needed for public or unspecified.
func gcpDialIPOptions(ipType string) []cloudsqlconn.Option {
	switch ipType {
	case gcpCloudSQLIPTypePrivate:
		return []cloudsqlconn.Option{cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP())}
	case gcpCloudSQLIPTypePSC:
		return []cloudsqlconn.Option{cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPSC())}
	default:
		return nil
	}
}

func (d *gcpMetadataDBDialer) Dial(ctx context.Context, instanceConnectionName string) (net.Conn, error) {
	return d.dialer.Dial(ctx, instanceConnectionName)
}

func (d *gcpMetadataDBDialer) Close() error {
	return d.dialer.Close()
}

func gcpConfigFromPGXConfig(pgxConfig *pgx.ConnConfig) (*gcpConfig, error) {
	iamEnabled := pgxConfig.RuntimeParams[gcpCloudSQLIAMParam] == "true"
	instanceConnectionName := pgxConfig.RuntimeParams[gcpCloudSQLInstanceConnectionNameParam]
	ipType := pgxConfig.RuntimeParams[gcpCloudSQLIPTypeParam]
	delete(pgxConfig.RuntimeParams, gcpCloudSQLIAMParam)
	delete(pgxConfig.RuntimeParams, gcpCloudSQLInstanceConnectionNameParam)
	delete(pgxConfig.RuntimeParams, gcpCloudSQLIPTypeParam)

	if !iamEnabled {
		return nil, nil
	}

	if instanceConnectionName == "" {
		return nil, errors.Errorf("%s is required when metadata database GCP Cloud SQL IAM auth is enabled", gcpCloudSQLInstanceConnectionNameParam)
	}

	if pgxConfig.User == "" {
		return nil, errors.New("database user is required when metadata database GCP Cloud SQL IAM auth is enabled")
	}

	switch ipType {
	case "", gcpCloudSQLIPTypePublic, gcpCloudSQLIPTypePrivate, gcpCloudSQLIPTypePSC:
	default:
		return nil, errors.Errorf("%s must be one of %q, %q, or %q", gcpCloudSQLIPTypeParam, gcpCloudSQLIPTypePublic, gcpCloudSQLIPTypePrivate, gcpCloudSQLIPTypePSC)
	}

	return &gcpConfig{
		enabled:                true,
		instanceConnectionName: instanceConnectionName,
		ipType:                 ipType,
	}, nil
}

func applyGCPConfig(pgxConfig *pgx.ConnConfig, authConfig *gcpConfig, dialer gcpDialer) {
	if authConfig == nil || !authConfig.enabled {
		return
	}
	pgxConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		conn, err := dialer.Dial(ctx, authConfig.instanceConnectionName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to dial metadata database GCP Cloud SQL instance %q", authConfig.instanceConnectionName)
		}
		return conn, nil
	}
}

func configureGCPWithDialer(pgxConfig *pgx.ConnConfig, authConfig *gcpConfig, dialer gcpDialer) ([]stdlib.OptionOpenDB, func() error, error) {
	applyGCPConfig(pgxConfig, authConfig, dialer)
	return nil, dialer.Close, nil
}
