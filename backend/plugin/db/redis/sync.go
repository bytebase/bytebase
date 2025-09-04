package redis

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
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

	databaseNumbers, err := d.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases")
	}
	var databases []*storepb.DatabaseSchemaMetadata
	for _, n := range databaseNumbers {
		databases = append(databases, &storepb.DatabaseSchemaMetadata{
			Name: strconv.Itoa(n),
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
	for line := range strings.SplitSeq(val, "\n") {
		if after, ok := strings.CutPrefix(line, "redis_version:"); ok {
			version = after
			version = strings.Trim(version, " \n\t\r")
			break
		}
	}
	if version == "" {
		return "", errors.New("failed to get redis_version")
	}
	return version, nil
}

func (d *Driver) getDatabases(ctx context.Context) ([]int, error) {
	dbsFromConfig, failopenFromConfig, err := d.getDatabaseCountFromConfig(ctx)
	if err != nil {
		return nil, err
	}
	if len(dbsFromConfig) > 0 {
		return dbsFromConfig, nil
	}

	dbsFromKeyspace, failopenFromKeyspace, err := d.getDatabaseNumberFromKeyspace(ctx)
	if err != nil {
		return nil, err
	}
	if len(dbsFromKeyspace) > 0 {
		return dbsFromKeyspace, nil
	}

	// Cloud vendors may have disabled this command. The default database is 0.
	if failopenFromConfig || failopenFromKeyspace {
		return []int{0}, nil
	}

	return nil, nil
}

func (d *Driver) getDatabaseCountFromConfig(ctx context.Context) ([]int, bool, error) {
	val, err := d.rdb.ConfigGet(ctx, "databases").Result()
	if err != nil {
		// Cloud vendors may have disabled this command.
		if strings.Contains(err.Error(), "unknown command") {
			return nil, true, nil
		}
		return nil, false, err
	}
	if _, ok := val["databases"]; !ok {
		return nil, false, errors.New("The returned values of 'CONFIG GET databases' dont't have the 'databases' KEY")
	}
	count, err := strconv.Atoi(val["databases"])
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to convert to int from %v", val["databases"])
	}
	var databases []int
	for i := 0; i < count; i++ {
		databases = append(databases, i)
	}
	return databases, false, nil
}

func (d *Driver) getDatabaseNumberFromKeyspace(ctx context.Context) ([]int, bool, error) {
	val, err := d.rdb.Info(ctx, "keyspace").Result()
	if err != nil {
		// Cloud vendors may have disabled this command.
		if strings.Contains(err.Error(), "unknown command") {
			return nil, true, nil
		}
		return nil, false, err
	}
	// Get the db number.
	re := regexp.MustCompile(`db(\d+)`)
	matches := re.FindAllStringSubmatch(val, -1)
	var dbs []int
	for _, match := range matches {
		dbInt, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, false, err
		}
		dbs = append(dbs, dbInt)
	}
	return dbs, false, nil
}
