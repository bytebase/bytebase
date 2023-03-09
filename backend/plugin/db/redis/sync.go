package redis

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Sync schema

// SyncInstance syncs the instance metadata.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var instance db.InstanceMetadata

	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get server version")
	}
	instance.Version = version

	clusterEnabled, err := d.getClusterEnabled(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if cluster is enabled")
	}

	databaseCount, err := d.getDatabaseCount(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}

	// Redis cluster can only use database zero.
	if clusterEnabled {
		databaseCount = 1
	}

	var databases []*storepb.DatabaseMetadata
	for i := 0; i < databaseCount; i++ {
		databases = append(databases, &storepb.DatabaseMetadata{
			Name: strconv.Itoa(i),
		})
	}
	instance.Databases = databases

	return &instance, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context, database string) (*storepb.DatabaseMetadata, error) {
	return &storepb.DatabaseMetadata{Name: database}, nil
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	val, err := d.rdb.Info(ctx, "server").Result()
	if err != nil {
		return "", err
	}
	var version string
	for _, line := range strings.Split(val, "\n") {
		if strings.HasPrefix(line, "redis_version:") {
			version = strings.TrimPrefix(line, "redis_version:")
			version = strings.Trim(version, " \n\t")
			break
		}
	}
	if version == "" {
		return "", errors.New("failed to get redis_version")
	}
	return version, nil
}

func (d *Driver) getClusterEnabled(ctx context.Context) (bool, error) {
	val, err := d.rdb.Info(ctx, "cluster").Result()
	if err != nil {
		return false, err
	}
	var enabled string
	for _, line := range strings.Split(val, "\n") {
		if strings.HasPrefix(line, "cluster_enabled:") {
			enabled = strings.TrimPrefix(line, "cluter_enabled:")
			enabled = strings.Trim(enabled, " \n\t")
			break
		}
	}
	if enabled == "" {
		return false, errors.New("failed to get cluster_enabled")
	}
	return enabled == "1", nil
}

func (d *Driver) getDatabaseCount(ctx context.Context) (int, error) {
	val, err := d.rdb.ConfigGet(ctx, "databases").Result()
	if err != nil {
		return 0, err
	}
	if _, ok := val["databases"]; !ok {
		return 0, errors.New("The returned values of 'CONFIG GET databases' dont't have the 'databases' KEY")
	}
	count, err := strconv.Atoi(val["databases"])
	if err != nil {
		return 0, errors.Wrapf(err, "failed to convert to int from %v", val["databases"])
	}
	return count, nil
}
