package spanner

import (
	"context"
	"sort"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata
	iter := d.dbClient.ListDatabases(ctx, &databasepb.ListDatabasesRequest{
		Parent: d.config.Host,
	})
	for {
		database, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// filter out databases using postgresql dialect which are not supported currently
		// TODO(p0ny): remove this after supporting postgresql dialect
		if database.DatabaseDialect == databasepb.DatabaseDialect_POSTGRESQL {
			continue
		}

		// database.Name is of the form `projects/<project>/instances/<instance>/databases/<database>`
		// We use regular expression to extract <database> from it.
		databaseName, err := getDatabaseFromDSN(database.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database name from %s", database.Name)
		}
		if excludedDatabaseList[databaseName] {
			continue
		}

		databases = append(databases, &storepb.DatabaseSchemaMetadata{Name: databaseName})
	}

	return &db.InstanceMetadata{
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	notFound, err := d.notFoundDatabase(ctx, d.databaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check if database exists")
	}
	if notFound {
		return nil, common.Errorf(common.NotFound, "database %q not found", d.databaseName)
	}

	tx := d.client.ReadOnlyTransaction()
	defer tx.Close()

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name: d.databaseName,
	}
	tableMap, err := getTable(ctx, tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", d.databaseName)
	}
	viewMap, err := getView(ctx, tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", d.databaseName)
	}

	schemaNameMap := make(map[string]bool)
	for schemaName := range tableMap {
		schemaNameMap[schemaName] = true
	}
	for schemaName := range viewMap {
		schemaNameMap[schemaName] = true
	}
	var schemaNames []string
	for schemaName := range schemaNameMap {
		schemaNames = append(schemaNames, schemaName)
	}
	sort.Strings(schemaNames)
	for _, schemaName := range schemaNames {
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:   schemaName,
			Tables: tableMap[schemaName],
			Views:  viewMap[schemaName],
		})
	}

	return databaseMetadata, err
}

func (d *Driver) notFoundDatabase(ctx context.Context, databaseName string) (bool, error) {
	dsn := getDSN(d.config.Host, databaseName)
	_, err := d.dbClient.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: dsn})
	if status.Code(err) == codes.NotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func getTable(ctx context.Context, tx *spanner.ReadOnlyTransaction) (map[string][]*storepb.TableMetadata, error) {
	columnMap, err := getColumn(ctx, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get table columns")
	}
	indexMap, err := getIndex(ctx, tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}
	foreignKeyMap, err := getForeignKey(ctx, tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get foreign keys")
	}
	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
    SELECT
      TABLE_SCHEMA,
      TABLE_NAME
    FROM INFORMATION_SCHEMA.TABLES
    WHERE TABLE_SCHEMA NOT IN ('INFORMATION_SCHEMA', 'SPANNER_SYS') AND TABLE_TYPE = 'BASE TABLE'
    ORDER BY TABLE_SCHEMA, TABLE_NAME
  `
	iter := tx.Query(ctx, spanner.NewStatement(query))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var table storepb.TableMetadata
		var schema string
		if err := row.Columns(&schema, &table.Name); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schema, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = foreignKeyMap[key]

		tableMap[schema] = append(tableMap[schema], &table)
	}
	return tableMap, nil
}

func getColumn(ctx context.Context, tx *spanner.ReadOnlyTransaction) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	query := `
    SELECT 
      TABLE_SCHEMA,
      TABLE_NAME,
      COLUMN_NAME,
      ORDINAL_POSITION,
      COLUMN_DEFAULT,
      IS_NULLABLE = 'YES',
      SPANNER_TYPE
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA NOT IN ('INFORMATION_SCHEMA', 'SPANNER_SYS')
    ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION
  `
	iter := tx.Query(ctx, spanner.NewStatement(query))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		column := &storepb.ColumnMetadata{}
		var (
			schemaName string
			tableName  string
			position   int64
			defaultStr spanner.NullString
		)
		if err := row.Columns(&schemaName, &tableName, &column.Name, &position, &defaultStr, &column.Nullable, &column.Type); err != nil {
			return nil, err
		}
		column.Position = int32(position)
		if defaultStr.Valid {
			column.Default = &wrapperspb.StringValue{Value: defaultStr.StringVal}
		}
		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	return columnsMap, nil
}

func getIndex(ctx context.Context, tx *spanner.ReadOnlyTransaction) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	query := `
    SELECT
      TABLE_SCHEMA,
      TABLE_NAME,
      INDEX_NAME,
      IS_UNIQUE,
      INDEX_TYPE = 'PRIMARY_KEY' AS IS_PRIMARY,
      ARRAY (
        SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.INDEX_COLUMNS AS ic
        WHERE ic.TABLE_SCHEMA = i.TABLE_SCHEMA AND ic.TABLE_NAME = i.TABLE_NAME AND ic.INDEX_NAME = i.INDEX_NAME
        ORDER BY ic.ORDINAL_POSITION
      )
    FROM INFORMATION_SCHEMA.INDEXES as i
    WHERE TABLE_SCHEMA NOT IN ('INFORMATION_SCHEMA', 'SPANNER_SYS') AND SPANNER_IS_MANAGED = FALSE
    ORDER BY TABLE_SCHEMA, TABLE_NAME, INDEX_NAME
  `

	keyIndexMap := make(map[db.TableKey][]*storepb.IndexMetadata)

	iter := tx.Query(ctx, spanner.NewStatement(query))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var idx storepb.IndexMetadata
		var schema, table string
		if err := row.Columns(
			&schema,
			&table,
			&idx.Name,
			&idx.Unique,
			&idx.Primary,
			&idx.Expressions,
		); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schema, Table: table}
		keyIndexMap[key] = append(keyIndexMap[key], &idx)
	}

	return keyIndexMap, nil
}

func getForeignKey(ctx context.Context, tx *spanner.ReadOnlyTransaction) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	foreignKeyMap := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	query := `
    WITH t AS (
      SELECT DISTINCT
        tc.CONSTRAINT_SCHEMA,
        tc.CONSTRAINT_NAME,
        tc.TABLE_NAME,
        rc.UNIQUE_CONSTRAINT_SCHEMA,
        rc.UNIQUE_CONSTRAINT_NAME,
        ccu.TABLE_SCHEMA AS REFERENCED_TABLE_SCHEMA,
        ccu.TABLE_NAME AS REFERENCED_TABLE_NAME,
        rc.DELETE_RULE,
        rc.UPDATE_RULE,
        rc.MATCH_OPTION
      FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS AS tc
      JOIN INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS AS rc
      ON tc.CONSTRAINT_SCHEMA = rc.CONSTRAINT_SCHEMA AND tc.CONSTRAINT_NAME = rc.CONSTRAINT_NAME 
      JOIN INFORMATION_SCHEMA.CONSTRAINT_COLUMN_USAGE as ccu
      ON tc.CONSTRAINT_SCHEMA = ccu.CONSTRAINT_SCHEMA AND tc.CONSTRAINT_NAME = ccu.CONSTRAINT_NAME 
      WHERE tc.CONSTRAINT_TYPE = 'FOREIGN KEY'
    )
    SELECT
      t.CONSTRAINT_SCHEMA,
      t.CONSTRAINT_NAME,
      t.TABLE_NAME,
      t.REFERENCED_TABLE_SCHEMA,
      t.REFERENCED_TABLE_NAME,
      ARRAY (
        SELECT
          COLUMN_NAME
        FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE AS kcu
        WHERE t.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA AND t.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
        ORDER BY kcu.ORDINAL_POSITION
      ) AS TABLE_COLUMNS,
      ARRAY (
        SELECT
          COLUMN_NAME
        FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE AS kcu
        WHERE t.UNIQUE_CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA AND t.UNIQUE_CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
        ORDER BY kcu.ORDINAL_POSITION
      ) AS REFERENCED_TABLE_COLUMNS,
      t.DELETE_RULE,
      t.UPDATE_RULE,
      t.MATCH_OPTION
    FROM t
  `
	iter := tx.Query(ctx, spanner.NewStatement(query))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var fk storepb.ForeignKeyMetadata
		var schema, table string
		var columns, referencedColumns []string
		if err := row.Columns(
			&schema,
			&fk.Name,
			&table,
			&fk.ReferencedSchema,
			&fk.ReferencedTable,
			&columns,
			&referencedColumns,
			&fk.OnDelete,
			&fk.OnUpdate,
			&fk.MatchType,
		); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schema, Table: table}
		foreignKeyMap[key] = append(foreignKeyMap[key], &fk)
	}
	return foreignKeyMap, nil
}

func getView(ctx context.Context, tx *spanner.ReadOnlyTransaction) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)
	query := `
  SELECT
    TABLE_SCHEMA,
    TABLE_NAME,
    VIEW_DEFINITION
  FROM INFORMATION_SCHEMA.VIEWS
  WHERE TABLE_SCHEMA NOT IN ('INFORMATION_SCHEMA', 'SPANNER_SYS')
  `
	iter := tx.Query(ctx, spanner.NewStatement(query))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var (
			schema     string
			name       string
			definition string
		)
		if err := row.Columns(&schema, &name, &definition); err != nil {
			return nil, err
		}
		viewMap[schema] = append(viewMap[schema], &storepb.ViewMetadata{
			Name:       name,
			Definition: definition,
		})
	}
	return viewMap, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
