package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var version, fullVersion string
	if err := driver.db.QueryRowContext(ctx, "SELECT SERVERPROPERTY('productversion'), @@VERSION").Scan(&version, &fullVersion); err != nil {
		return nil, err
	}
	tokens := strings.Fields(fullVersion)
	for _, token := range tokens {
		if len(token) == 4 && strings.HasPrefix(token, "20") {
			version = fmt.Sprintf("%s (%s)", version, token)
			break
		}
	}

	var databases []*storepb.DatabaseSchemaMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT name, collation_name FROM sys.databases WHERE name NOT IN ('master', 'model', 'msdb', 'tempdb', 'rdscore')")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseSchemaMetadata{}
		var collation sql.NullString
		if err := rows.Scan(&database.Name, &collation); err != nil {
			return nil, err
		}
		if collation.Valid {
			database.Collation = collation.String
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (driver *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	schemaNames, err := getSchemas(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get schemas from database %q", driver.databaseName)
	}
	columnMap, err := getTableColumns(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table columns from database %q", driver.databaseName)
	}
	tableMap, err := getTables(txn, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from database %q", driver.databaseName)
	}
	viewMap, err := getViews(txn, columnMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views from database %q", driver.databaseName)
	}
	sequenceMap, err := getSequences(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sequences from database %q", driver.databaseName)
	}
	functionMap, err := getFunctions(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get functions from database %q", driver.databaseName)
	}
	procedureMap, err := getProcedures(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get procedures from database %q", driver.databaseName)
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	databaseMetadata := &storepb.DatabaseSchemaMetadata{
		Name: driver.databaseName,
	}
	for _, schemaName := range schemaNames {
		databaseMetadata.Schemas = append(databaseMetadata.Schemas, &storepb.SchemaMetadata{
			Name:       schemaName,
			Tables:     tableMap[schemaName],
			Views:      viewMap[schemaName],
			Sequences:  sequenceMap[schemaName],
			Functions:  functionMap[schemaName],
			Procedures: procedureMap[schemaName],
		})
	}
	return databaseMetadata, nil
}

func getSchemas(txn *sql.Tx) ([]string, error) {
	query := `
		SELECT schema_name FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE schema_name NOT IN ('db_owner', 'db_accessadmin', 'db_backupoperator', 'db_datareader', 'db_datawriter', 'db_ddladmin', 'db_denydatareader', 'db_denydatawriter', 'db_securityadmin', 'guest', 'INFORMATION_SCHEMA', 'sys');`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return nil, err
		}
		result = append(result, schemaName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// getTables gets all tables of a database.
func getTables(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.TableMetadata, error) {
	indexMap, err := getIndexes(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get indices")
	}

	fkMap, err := getForeignKeys(txn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get foreign keys")
	}

	// TODO(d): foreign keys.
	tableMap := make(map[string][]*storepb.TableMetadata)
	query := `
		SELECT
			SCHEMA_NAME(t.schema_id),
			t.name,
			SUM(ps.row_count),
			lj.PropertyValue AS comment
		FROM sys.tables t
		INNER JOIN sys.dm_db_partition_stats ps ON ps.object_id = t.object_id
		LEFT JOIN (
			SELECT
				EP.value AS PropertyValue,
				S.name AS SchemaName,
				O.name AS TableName
			FROM
				(SELECT major_id, name, value FROM sys.extended_properties WHERE name = 'MS_Description' AND minor_id = 0) AS EP
				INNER JOIN sys.all_objects AS O ON EP.major_id = O.object_id
				INNER JOIN sys.schemas AS S ON O.schema_id = S.schema_id
		) lj ON lj.SchemaName = SCHEMA_NAME(t.schema_id) AND lj.TableName = t.name
        WHERE index_id < 2
		GROUP BY t.name, t.schema_id, lj.PropertyValue
		ORDER BY 1, 2 ASC
		OPTION (RECOMPILE);`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		table := &storepb.TableMetadata{}
		var schemaName string
		var comment sql.NullString
		if err := rows.Scan(&schemaName, &table.Name, &table.RowCount, &comment); err != nil {
			return nil, err
		}
		key := db.TableKey{Schema: schemaName, Table: table.Name}
		table.Columns = columnMap[key]
		table.Indexes = indexMap[key]
		table.ForeignKeys = fkMap[key]
		if comment.Valid {
			table.Comment = comment.String
		}

		tableMap[schemaName] = append(tableMap[schemaName], table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tableMap, nil
}

// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-foreign-key-columns-transact-sql?view=sql-server-ver16#example-query
var listForeignKeyQuery = `
SELECT fk.name AS ForeignKeyName,
       s_parent.name AS ParentSchemaName,
       t_parent.name AS ParentTableName,
       c_parent.name AS ParentColumnName,
       s_child.name AS ReferencedSchemaName,
       t_child.name AS ReferencedTableName,
       c_child.name AS ReferencedColumnName,
       CASE fk.delete_referential_action
        WHEN 0 THEN ''
        WHEN 1 THEN 'CASCADE'
        WHEN 2 THEN 'SET NULL'
        WHEN 3 THEN 'SET DEFAULT'
       END AS OnDeleteAction,
       CASE fk.update_referential_action
        WHEN 0 THEN ''
        WHEN 1 THEN 'CASCADE'
        WHEN 2 THEN 'SET NULL'
        WHEN 3 THEN 'SET DEFAULT'
       END AS OnUpdateAction
FROM sys.foreign_keys fk
INNER JOIN sys.foreign_key_columns fkc
    ON fkc.constraint_object_id = fk.object_id
INNER JOIN sys.tables t_parent
    ON t_parent.object_id = fk.parent_object_id
INNER JOIN sys.schemas s_parent
    ON s_parent.schema_id = t_parent.schema_id
INNER JOIN sys.columns c_parent
    ON fkc.parent_column_id = c_parent.column_id
    AND c_parent.object_id = t_parent.object_id
INNER JOIN sys.tables t_child
    ON t_child.object_id = fk.referenced_object_id
INNER JOIN sys.schemas s_child
    ON s_child.schema_id = t_child.schema_id
INNER JOIN sys.columns c_child
    ON c_child.object_id = t_child.object_id
    AND fkc.referenced_column_id = c_child.column_id
ORDER BY fk.name, t_parent.name, t_child.name, c_parent.name, c_child.name;
`

func getForeignKeys(txn *sql.Tx) (map[db.TableKey][]*storepb.ForeignKeyMetadata, error) {
	fkMap := make(map[db.TableKey]map[string]*storepb.ForeignKeyMetadata)

	rows, err := txn.Query(listForeignKeyQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fkName, parentSchemaName, parentTableName, parentColumnName, referencedSchemaName, referencedTableName, referencedColumnName, onDelete, onUpdate string
		if err := rows.Scan(&fkName, &parentSchemaName, &parentTableName, &parentColumnName, &referencedSchemaName, &referencedTableName, &referencedColumnName, &onDelete, &onUpdate); err != nil {
			return nil, err
		}
		outerKey := db.TableKey{Schema: parentSchemaName, Table: parentTableName}
		if _, ok := fkMap[outerKey]; !ok {
			fkMap[outerKey] = make(map[string]*storepb.ForeignKeyMetadata)
		}

		if _, ok := fkMap[outerKey][fkName]; !ok {
			fkMap[outerKey][fkName] = &storepb.ForeignKeyMetadata{
				Name:              fkName,
				Columns:           []string{parentColumnName},
				ReferencedSchema:  referencedSchemaName,
				ReferencedTable:   referencedTableName,
				ReferencedColumns: []string{referencedColumnName},
				OnDelete:          onDelete,
				OnUpdate:          onUpdate,
			}
		} else {
			fkMap[outerKey][fkName].Columns = append(fkMap[outerKey][fkName].Columns, parentColumnName)
			fkMap[outerKey][fkName].ReferencedColumns = append(fkMap[outerKey][fkName].ReferencedColumns, referencedColumnName)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Aggregate the map to a slice.
	result := make(map[db.TableKey][]*storepb.ForeignKeyMetadata)
	for k, m := range fkMap {
		var foreignkeyNames []string
		for _, v := range m {
			foreignkeyNames = append(foreignkeyNames, v.Name)
		}
		sort.Strings(foreignkeyNames)
		for _, fkName := range foreignkeyNames {
			result[k] = append(result[k], m[fkName])
		}
	}

	return result, nil
}

// getTableColumns gets the columns of a table.
func getTableColumns(txn *sql.Tx) (map[db.TableKey][]*storepb.ColumnMetadata, error) {
	columnsMap := make(map[db.TableKey][]*storepb.ColumnMetadata)

	query := `
		SELECT
			IC.table_schema,
			IC.table_name,
			IC.column_name,
			IC.data_type,
			IC.character_maximum_length,
			IC.ordinal_position,
			IC.column_default,
			IC.is_nullable,
			IC.collation_name,
			IJ.PropertyValue AS ColumnComment
		FROM INFORMATION_SCHEMA.COLUMNS IC
		LEFT JOIN (
			SELECT
				EP.value AS PropertyValue,
				S.name AS SchemaName,
				O.name AS TableName,
				C.name AS ColumnName
			FROM
				(SELECT major_id, minor_id, name, value FROM sys.extended_properties WHERE name = 'MS_Description') AS EP
				INNER JOIN sys.all_objects AS O ON EP.major_id = O.object_id
				INNER JOIN sys.schemas AS S ON O.schema_id = S.schema_id
				INNER JOIN sys.columns AS C ON EP.major_id = C.object_id AND EP.minor_id = C.column_id
			WHERE S.name IS NOT NULL AND O.name IS NOT NULL AND C.name IS NOT NULL
		) IJ ON IJ.SchemaName = IC.table_schema AND IJ.TableName = IC.table_name AND IJ.ColumnName = IC.column_name
		ORDER BY IC.table_schema, IC.table_name, IC.ordinal_position;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		column := &storepb.ColumnMetadata{}
		var schemaName, tableName, columnType, nullable string
		var defaultStr, collation, comment sql.NullString
		var characterLength sql.NullInt32
		if err := rows.Scan(&schemaName, &tableName, &column.Name, &columnType, &characterLength, &column.Position, &defaultStr, &nullable, &collation, &comment); err != nil {
			return nil, err
		}
		if characterLength.Valid {
			column.Type = fmt.Sprintf("%s(%d)", columnType, characterLength.Int32)
		} else {
			column.Type = columnType
		}
		if defaultStr.Valid {
			// TODO: use correct default type
			column.DefaultValue = &storepb.ColumnMetadata_DefaultExpression{DefaultExpression: defaultStr.String}
		}
		isNullBool, err := util.ConvertYesNo(nullable)
		if err != nil {
			return nil, err
		}
		column.Nullable = isNullBool
		column.Collation = collation.String
		if comment.Valid {
			column.Comment = comment.String
		}
		key := db.TableKey{Schema: schemaName, Table: tableName}
		columnsMap[key] = append(columnsMap[key], column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnsMap, nil
}

// getIndexes gets all indices of a database.
func getIndexes(txn *sql.Tx) (map[db.TableKey][]*storepb.IndexMetadata, error) {
	// MSSQL doesn't support function-based indexes.
	indexMap := make(map[db.TableKey]map[string]*storepb.IndexMetadata)

	query := `
		SELECT
			s.name,
			t.name,
			ind.name,
			col.name,
			ind.type_desc,
			ind.is_primary_key,
			ind.is_unique
		FROM
			sys.indexes ind
		INNER JOIN
			sys.index_columns ic ON  ind.object_id = ic.object_id and ind.index_id = ic.index_id
		INNER JOIN
			sys.columns col ON ic.object_id = col.object_id and ic.column_id = col.column_id
		INNER JOIN
			sys.tables t ON ind.object_id = t.object_id
		INNER JOIN
			sys.schemas s ON s.schema_id = t.schema_id
		WHERE
			t.is_ms_shipped = 0 AND ind.name IS NOT NULL
		ORDER BY 
			s.name, t.name, ind.name, ic.key_ordinal;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var schemaName, tableName, indexName, indexType, colName string
		var primary, unique bool
		if err := rows.Scan(&schemaName, &tableName, &indexName, &colName, &indexType, &primary, &unique); err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: schemaName, Table: tableName}
		if _, ok := indexMap[key]; !ok {
			indexMap[key] = make(map[string]*storepb.IndexMetadata)
		}
		if _, ok := indexMap[key][indexName]; !ok {
			indexMap[key][indexName] = &storepb.IndexMetadata{
				Name:    indexName,
				Type:    indexType,
				Unique:  unique,
				Primary: primary,
			}
		}
		indexMap[key][indexName].Expressions = append(indexMap[key][indexName].Expressions, colName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tableIndexes := make(map[db.TableKey][]*storepb.IndexMetadata)
	for k, m := range indexMap {
		for _, v := range m {
			tableIndexes[k] = append(tableIndexes[k], v)
		}
		sort.Slice(tableIndexes[k], func(i, j int) bool {
			return tableIndexes[k][i].Name < tableIndexes[k][j].Name
		})
	}
	return tableIndexes, nil
}

// getViews gets all views of a database.
func getViews(txn *sql.Tx, columnMap map[db.TableKey][]*storepb.ColumnMetadata) (map[string][]*storepb.ViewMetadata, error) {
	viewMap := make(map[string][]*storepb.ViewMetadata)

	query := `
		SELECT
			SCHEMA_NAME(v.schema_id) AS schema_name,
			v.name AS view_name,
			m.definition
		FROM sys.views v
		INNER JOIN sys.sql_modules m ON v.object_id = m.object_id
		ORDER BY schema_name, view_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		view := &storepb.ViewMetadata{}
		var schemaName string
		var definition sql.NullString
		if err := rows.Scan(&schemaName, &view.Name, &definition); err != nil {
			return nil, err
		}

		var viewDefinition string
		if !definition.Valid {
			// Definition is null if the view is encrypted.
			// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-all-sql-modules-transact-sql?view=sql-server-ver16
			// https://www.mssqltips.com/sqlservertip/7465/encrypt-stored-procedure-sql-server/
			// We will write a pseudo definition in pure comment.
			viewDefinition = fmt.Sprintf("/* Definition of view %s.%s is encrypted. */", schemaName, view.Name)
		} else {
			viewDefinition = definition.String
		}
		view.Definition = viewDefinition

		key := db.TableKey{Schema: schemaName, Table: view.Name}
		view.Columns = columnMap[key]

		viewMap[schemaName] = append(viewMap[schemaName], view)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return viewMap, nil
}

// getSequences gets all sequences of a database.
func getSequences(txn *sql.Tx) (map[string][]*storepb.SequenceMetadata, error) {
	query := `
	SELECT
		s.name,
		seq.name,
		tp.name
	FROM
		sys.sequences seq
	INNER JOIN
		sys.schemas s ON s.schema_id = seq.schema_id
	INNER JOIN
		sys.types tp ON tp.system_type_id = seq.system_type_id
	ORDER BY s.name, seq.name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sequenceMap := make(map[string][]*storepb.SequenceMetadata)
	for rows.Next() {
		sequence := &storepb.SequenceMetadata{}
		var schemaName string
		if err := rows.Scan(&schemaName, &sequence.Name, &sequence.DataType); err != nil {
			return nil, err
		}
		sequenceMap[schemaName] = append(sequenceMap[schemaName], sequence)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sequenceMap, nil
}

func getProcedures(txn *sql.Tx) (map[string][]*storepb.ProcedureMetadata, error) {
	procedureMap := make(map[string][]*storepb.ProcedureMetadata)

	// The CAST(...) = 0 means the procedure is not a system function.
	query := `
	SELECT
		SCHEMA_NAME(ao.schema_id) AS schema_name,
		ao.name AS procedure_name,
		asm.definition
	FROM sys.all_objects ao
        INNER JOIN sys.all_sql_modules asm ON asm.object_id = ao.object_id
	WHERE ao.type IN ('P', 'RF')
		AND ao.is_ms_shipped = 0 AND
		(
			SELECT major_id
			FROM sys.extended_properties
			WHERE major_id = ao.object_id
				AND minor_id = 0
				AND class = 1
				AND name = 'microsoft_database_tools_support'
		) IS NULL
	ORDER BY schema_name, procedure_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		procedure := &storepb.ProcedureMetadata{}
		var schemaName string
		var definition sql.NullString
		if err := rows.Scan(&schemaName, &procedure.Name, &definition); err != nil {
			return nil, err
		}
		var procedureDefinition string
		if !definition.Valid {
			// Definition is null if the procedure is encrypted.
			// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-all-sql-modules-transact-sql?view=sql-server-ver16
			// https://www.mssqltips.com/sqlservertip/7465/encrypt-stored-procedure-sql-server/
			// We will write a pseudo definition in pure comment.
			procedureDefinition = fmt.Sprintf("/* Definition of procedure %s.%s is encrypted. */", schemaName, procedure.Name)
		} else {
			procedureDefinition = definition.String
		}
		procedure.Definition = procedureDefinition
		procedureMap[schemaName] = append(procedureMap[schemaName], procedure)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return procedureMap, nil
}

func getFunctions(txn *sql.Tx) (map[string][]*storepb.FunctionMetadata, error) {
	funcMap := make(map[string][]*storepb.FunctionMetadata)

	// The CAST(...) = 0 means the function is not a system function.
	query := `
	SELECT
		SCHEMA_NAME(ao.schema_id) AS schema_name,
		ao.name AS func_name,
		asm.definition
	FROM sys.all_objects ao
        INNER JOIN sys.all_sql_modules asm ON asm.object_id = ao.object_id
	WHERE ao.type IN ('FN', 'IF', 'TF')
		AND ao.is_ms_shipped = 0 AND
		(
			SELECT major_id
			FROM sys.extended_properties
			WHERE major_id = ao.object_id
				AND minor_id = 0
				AND class = 1
				AND name = 'microsoft_database_tools_support'
		) IS NULL
	ORDER BY schema_name, func_name;`
	rows, err := txn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		function := &storepb.FunctionMetadata{}
		var schemaName string
		var definition sql.NullString
		if err := rows.Scan(&schemaName, &function.Name, &definition); err != nil {
			return nil, err
		}
		var functionDefinition string
		if !definition.Valid {
			// Definition is null if the function is encrypted.
			// https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-all-sql-modules-transact-sql?view=sql-server-ver16
			// https://www.mssqltips.com/sqlservertip/7465/encrypt-stored-procedure-sql-server/
			// We will write a pseudo definition in pure comment.
			functionDefinition = fmt.Sprintf("/* Definition of function %s.%s is encrypted. */", schemaName, function.Name)
		} else {
			functionDefinition = definition.String
		}
		function.Definition = functionDefinition
		funcMap[schemaName] = append(funcMap[schemaName], function)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return funcMap, nil
}
