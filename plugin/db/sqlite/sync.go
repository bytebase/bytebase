package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	excludedDatabaseList = map[string]bool{
		// Skip our internal "bytebase" database
		bytebaseDatabase: true,
	}
)

// indexSchema describes the schema of an index.
type indexSchema struct {
	name      string
	tableName string
	statement string
	unique    bool
}

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMeta, error) {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	databases, err := driver.getDatabases()
	if err != nil {
		return nil, err
	}

	var databaseList []db.DatabaseMeta
	for _, dbName := range databases {
		if _, ok := excludedDatabaseList[dbName]; ok {
			continue
		}

		databaseList = append(
			databaseList,
			db.DatabaseMeta{
				Name: dbName,
			},
		)
	}

	return &db.InstanceMeta{
		Version:      version,
		UserList:     nil,
		DatabaseList: databaseList,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context, databaseName string) (*storepb.DatabaseMetadata, error) {
	databases, err := driver.getDatabases()
	if err != nil {
		return nil, err
	}

	schema := db.Schema{
		Name: databaseName,
	}
	found := false
	for _, database := range databases {
		if database == databaseName {
			found = true
			break
		}
	}
	if !found {
		return nil, common.Errorf(common.NotFound, "database %q not found", databaseName)
	}

	sqldb, err := driver.GetDBConnection(ctx, databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database connection for %q", databaseName)
	}
	txn, err := sqldb.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()
	// Index statements.
	indicesMap := make(map[string][]indexSchema)
	indices, err := getIndices(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices from database %q", databaseName)
	}
	for _, idx := range indices {
		indicesMap[idx.tableName] = append(indicesMap[idx.tableName], idx)
	}

	tbls, err := getTables(txn, indicesMap)
	if err != nil {
		return nil, err
	}
	schema.TableList = tbls

	views, err := getViews(txn)
	if err != nil {
		return nil, err
	}
	schema.ViewList = views

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return util.ConvertDBSchema(&schema, nil), nil
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, indicesMap map[string][]indexSchema) ([]db.Table, error) {
	var tables []db.Table
	query := "SELECT name FROM sqlite_schema WHERE type ='table' AND name NOT LIKE 'sqlite_%';"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
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
	for _, name := range tableNames {
		var tbl db.Table
		tbl.Name = name
		tbl.ShortName = name
		tbl.Type = "BASE TABLE"

		if err := func() error {
			// Get columns: cid, name, type, notnull, dflt_value, pk.
			query := fmt.Sprintf("pragma table_info(%s);", name)
			rows, err := txn.Query(query)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var col db.Column

				var cid int
				var notnull, pk bool
				var name, ctype string
				var dfltValue sql.NullString
				if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
					return err
				}
				col.Position = cid
				col.Name = name
				col.Nullable = !notnull
				col.Type = ctype
				if dfltValue.Valid {
					col.Default = &dfltValue.String
				}

				tbl.ColumnList = append(tbl.ColumnList, col)
			}
			return rows.Err()
		}(); err != nil {
			return nil, err
		}

		for _, idx := range indicesMap[tbl.Name] {
			if err := func() error {
				query := fmt.Sprintf("pragma index_info(%s);", idx.name)
				rows, err := txn.Query(query)
				if err != nil {
					return err
				}
				defer rows.Close()
				for rows.Next() {
					var dbIdx db.Index
					dbIdx.Name = idx.name
					dbIdx.Unique = idx.unique
					var cid string
					if err := rows.Scan(&dbIdx.Position, &cid, &dbIdx.Expression); err != nil {
						return err
					}
					tbl.IndexList = append(tbl.IndexList, dbIdx)
				}
				if err := rows.Err(); err != nil {
					return util.FormatErrorWithQuery(err, query)
				}
				return nil
			}(); err != nil {
				return nil, err
			}
		}

		tables = append(tables, tbl)
	}
	return tables, nil
}

// getIndices gets all indices of a database.
func getIndices(txn *sql.Tx) ([]indexSchema, error) {
	var indices []indexSchema
	query := "SELECT name, tbl_name, sql FROM sqlite_schema WHERE type ='index' AND name NOT LIKE 'sqlite_%';"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idx indexSchema
		if err := rows.Scan(&idx.name, &idx.tableName, &idx.statement); err != nil {
			return nil, err
		}
		idx.unique = strings.Contains(idx.statement, " UNIQUE INDEX ")
		indices = append(indices, idx)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return indices, nil
}

func getViews(txn *sql.Tx) ([]db.View, error) {
	var views []db.View
	query := "SELECT name, sql FROM sqlite_schema WHERE type ='view' AND name NOT LIKE 'sqlite_%';"
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var view db.View
		if err := rows.Scan(&view.Name, &view.Definition); err != nil {
			return nil, err
		}
		view.ShortName = view.Name
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return views, nil
}
