package redis

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

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
	var databases []*metadatapb.DatabaseSchemaMetadata
	for _, n := range databaseNumbers {
		databases = append(databases, &metadatapb.DatabaseSchemaMetadata{
			Name: strconv.Itoa(n),
		})
	}
	instance.Databases = databases

	return &instance, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return &metadatapb.DatabaseSchemaMetadata{Name: d.databaseName}, nil
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

	// Neither source could list the databases. Every Redis has database 0.
	if failopenFromConfig || failopenFromKeyspace {
		return []int{0}, nil
	}

	return nil, nil
}

func (d *Driver) getDatabaseCountFromConfig(ctx context.Context) ([]int, bool, error) {
	val, err := d.rdb.ConfigGet(ctx, "databases").Result()
	return databasesFromConfigReply(val, err)
}

// databasesFromConfigReply reads the reply to CONFIG GET databases. The bool
// reports that the server cannot tell the count, so another source decides.
func databasesFromConfigReply(val map[string]string, err error) ([]int, bool, error) {
	if err != nil {
		if isCommandUnavailable(err) {
			return nil, true, nil
		}
		return nil, false, err
	}
	raw, ok := val["databases"]
	if !ok {
		// Redis Enterprise and Redis Cloud answer CONFIG GET for a subset of
		// settings and send an empty reply, not an error, for the rest.
		return nil, true, nil
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to convert to int from %v", raw)
	}
	var databases []int
	for i := 0; i < count; i++ {
		databases = append(databases, i)
	}
	return databases, false, nil
}

// isCommandUnavailable reports whether the server refused the command itself:
// a cloud vendor removed it, or the user's ACL excludes it.
func isCommandUnavailable(err error) bool {
	return strings.Contains(err.Error(), "unknown command") || redis.HasErrorPrefix(err, "NOPERM")
}

func (d *Driver) getDatabaseNumberFromKeyspace(ctx context.Context) ([]int, bool, error) {
	val, err := d.rdb.Info(ctx, "keyspace").Result()
	if err != nil {
		if isCommandUnavailable(err) {
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
