package obo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const systemSchemas = "'DWEXP','OMC','ORAAUDITOR','LBACSYS','SYS'"

func (*Driver) SyncInstance(context.Context) (*db.InstanceMetadata, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) SyncDBSchema(context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) SyncSlowQuery(context.Context, time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) CheckSlowQueryLogEnabled(context.Context) error {
	return errors.New("not implemented")
}

func getDatabases(ctx context.Context, tx *sql.Tx) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT username FROM sys.all_users
		WHERE username NOT IN (%s)
		ORDER BY username
	`, systemSchemas)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return schemas, nil
}

func getTables(ctx context.Context, tx *sql.Tx) (map[string][]*storepb.TableMetadata, error) {
	return nil, nil
}
