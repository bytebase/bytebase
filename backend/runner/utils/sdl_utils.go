package utils

import (
	"bytes"
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/differ"
	"github.com/bytebase/bytebase/backend/store"
)

// ComputeDatabaseSchemaDiff computes the diff between current database schema
// and the given schema. It returns an empty string if there is no applicable
// diff.
func ComputeDatabaseSchemaDiff(ctx context.Context, instance *store.InstanceMessage, databaseName string, dbFactory *dbfactory.DBFactory, newSchema string) (string, error) {
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, databaseName)
	if err != nil {
		return "", errors.Wrap(err, "get admin driver")
	}
	defer func() {
		_ = driver.Close(ctx)
	}()

	var schema bytes.Buffer
	_, err = driver.Dump(ctx, databaseName, &schema, true /* schemaOnly */)
	if err != nil {
		return "", errors.Wrap(err, "dump old schema")
	}

	var engine parser.EngineType
	switch instance.Engine {
	case db.Postgres:
		engine = parser.Postgres
	case db.MySQL:
		engine = parser.MySQL
	default:
		return "", errors.Errorf("unsupported database engine %q", instance.Engine)
	}

	diff, err := differ.SchemaDiff(engine, schema.String(), newSchema)
	if err != nil {
		return "", errors.Wrapf(err, "compute schema diff")
	}
	return diff, nil
}
