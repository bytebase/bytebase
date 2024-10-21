package redis

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Sync schema

// SyncInstance syncs the instance metadata.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var instance db.InstanceMetadata
	dbNumber, err := d.getDatabaseNumber(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database number")
	}
	var databases []*storepb.DatabaseSchemaMetadata
	for _, n := range dbNumber {
		databases = append(databases, &storepb.DatabaseSchemaMetadata{
			Name: strconv.Itoa(i),
		})
	}
	instance.Databases = databases
	return &instance, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return &storepb.DatabaseSchemaMetadata{Name: d.databaseName}, nil
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
			version = strings.Trim(version, " \n\t\r")
			break
		}
	}
	if version == "" {
		return "", errors.New("failed to get redis_version")
	}
	return version, nil
}

func (d *Driver) getDatabaseNumber(ctx context.Context) ([]int, error) {
	val, err := d.rdb.Info(ctx, "keyspace").Result()
	if err != nil {
		if strings.Contains(err.Error(), "unknown command") {
			return []int{0}, nil
		}
		return nil, errors.Wrapf(err, "failed to run `INFO KEYSPACE`")
	}
	//get the db number
	re := regexp.MustCompile(`db(\d+)`)
	matches := re.FindAllStringSubmatch(val, -1)
	var dbs []int
	for _, match := range matches {
		dbInt, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, err
		}
		dbs = append(dbs, dbInt)
	}
	return dbs, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
