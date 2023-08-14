// Package utils is the package for runner utils.
package utils

import (
	"bytes"
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/differ"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/store"
)

// ComputeDatabaseSchemaDiff computes the diff between current database schema
// and the given schema. It returns an empty string if there is no applicable
// diff.
func ComputeDatabaseSchemaDiff(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, dbFactory *dbfactory.DBFactory, newSchema string) (string, error) {
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return "", errors.Wrap(err, "get admin driver")
	}
	defer func() {
		_ = driver.Close(ctx)
	}()

	var schema bytes.Buffer
	_, err = driver.Dump(ctx, &schema, true /* schemaOnly */)
	if err != nil {
		return "", errors.Wrap(err, "dump old schema")
	}

	var engine parser.EngineType
	switch instance.Engine {
	case db.Postgres, db.RisingWave:
		engine = parser.Postgres
	case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
		engine = parser.MySQL
	default:
		return "", errors.Errorf("unsupported database engine %q", instance.Engine)
	}

	sdlFormat, err := transform.SchemaTransform(engine, schema.String())
	if err != nil {
		return "", errors.Wrapf(err, "failed to transform SDL format")
	}
	diff, err := differ.SchemaDiff(engine, sdlFormat, newSchema, store.IgnoreDatabaseAndTableCaseSensitive(instance))
	if err != nil {
		return "", errors.Wrapf(err, "compute schema diff")
	}
	return diff, nil
}
