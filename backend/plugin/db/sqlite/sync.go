package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	excludedDatabaseList = map[string]bool{
		// Skip our internal "bytebase" database
		bytebaseDatabase: true,
	}
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	databaseNames, err := driver.getDatabases()
	if err != nil {
		return nil, err
	}

	var databases []*storepb.DatabaseSchemaMetadata
	for _, databaseName := range databaseNames {
		if _, ok := excludedDatabaseList[databaseName]; ok {
			continue
		}
		databases = append(databases, &storepb.DatabaseSchemaMetadata{Name: databaseName})
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	var version string
	if err := driver.db.QueryRowContext(ctx, "SELECT sqlite_version();").Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	databases, err := driver.getDatabases()
	if err != nil {
		return nil, err
	}

	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}
	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name:    driver.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}
	found := false
	for _, database := range databases {
		if database == driver.databaseName {
			found = true
			break
		}
	}
	if !found {
		return nil, common.Errorf(common.NotFound, "database %q not found", driver.databaseName)
	}

	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	tables, err := getTables(txn)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Tables = tables

	views, err := getViews(txn)
	if err != nil {
		return nil, err
	}
	schemaMetadata.Views = views

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return databaseMetadata, nil
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx) ([]*storepb.TableMetadata, error) {
	indexMap, err := getIndices(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}

	var tableNames []string
	query := `
		SELECT
			name
		FROM sqlite_schema
		WHERE type ='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	var tables []*storepb.TableMetadata
	for _, name := range tableNames {
		table := &storepb.TableMetadata{
			Name: name,
		}
		if err := func() error {
			// Get columns: cid, name, type, notnull, dflt_value, pk.
			query := fmt.Sprintf("pragma table_info(%s);", name)
			rows, err := txn.Query(query)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				column := &storepb.ColumnMetadata{}
				var notNull, unusedPk bool
				var defaultStr sql.NullString
				if err := rows.Scan(&column.Position, &column.Name, &column.Type, &notNull, &defaultStr, &unusedPk); err != nil {
					return err
				}
				column.Nullable = !notNull
				if defaultStr.Valid {
					column.Default = &wrapperspb.StringValue{Value: defaultStr.String}
				}

				table.Columns = append(table.Columns, column)
				table.Indexes = indexMap[table.Name]
			}
			return rows.Err()
		}(); err != nil {
			return nil, err
		}

		tables = append(tables, table)
	}
	return tables, nil
}

// getIndices gets all indices of a database.
func getIndices(txn *sql.Tx) (map[string][]*storepb.IndexMetadata, error) {
	indexMap := make(map[string][]*storepb.IndexMetadata)
	query := `
		SELECT
			tbl_name, name, sql
		FROM sqlite_schema
		WHERE type ='index' AND name NOT LIKE 'sqlite_%'
		ORDER BY tbl_name, name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, statement string
		index := &storepb.IndexMetadata{}
		if err := rows.Scan(tableName, &index.Name, &statement); err != nil {
			return nil, err
		}
		index.Unique = strings.Contains(statement, " UNIQUE INDEX ")
		indexMap[tableName] = append(indexMap[tableName], index)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	for _, indexes := range indexMap {
		for _, index := range indexes {
			if err := func() error {
				query := fmt.Sprintf("pragma index_info(%s);", index.Name)
				rows, err := txn.Query(query)
				if err != nil {
					return err
				}
				defer rows.Close()
				for rows.Next() {
					var unusedPosition int
					var unusedCid, expression string
					if err := rows.Scan(&unusedPosition, &unusedCid, &expression); err != nil {
						return err
					}
					index.Expressions = append(index.Expressions, expression)
				}
				if err := rows.Err(); err != nil {
					return util.FormatErrorWithQuery(err, query)
				}
				return nil
			}(); err != nil {
				return nil, err
			}
		}
	}

	return indexMap, nil
}

func getViews(txn *sql.Tx) ([]*storepb.ViewMetadata, error) {
	var views []*storepb.ViewMetadata

	query := `
		SELECT
			name, sql
		FROM sqlite_schema
		WHERE type ='view' AND name NOT LIKE 'sqlite_%'
		ORDER BY name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		if err := rows.Scan(&view.Name, &view.Definition); err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}

	return views, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
