package hive

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var instanceMetadata db.InstanceMetadata

	// version.
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, err
	}

	// databases.
	databaseNames, err := d.getDatabaseNames(ctx)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	instanceMetadata.Databases = make([]*storepb.DatabaseSchemaMetadata, len(databaseNames))
	for idx, databaseName := range databaseNames {
		wg.Add(1)
		go func(index int, databaseName string) {
			databaseSchemaMetadata := new(storepb.DatabaseSchemaMetadata)
			databaseSchemaMetadata.Name = databaseName

			schemaMetadata, err := d.getDatabaseInfoByName(ctx, databaseName)
			if err != nil {
				return
			}

			databaseSchemaMetadata.Schemas = append(databaseSchemaMetadata.Schemas, schemaMetadata)
			instanceMetadata.Databases[index] = databaseSchemaMetadata
			wg.Done()
		}(idx, databaseName)
	}
	wg.Wait()

	rolesMetadata, err := d.getRoles(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role metadata")
	}
	instanceMetadata.InstanceRoles = rolesMetadata
	instanceMetadata.Version = version
	return &instanceMetadata, nil
}

func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	dbName := d.config.Database
	if dbName == "" {
		dbName = "default"
	}

	databaseSchemaMetadata := new(storepb.DatabaseSchemaMetadata)
	databaseSchemaMetadata.Name = dbName
	schemaMetadata, err := d.getDatabaseInfoByName(ctx, dbName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get info from database %s", dbName)
	}
	databaseSchemaMetadata.Schemas = append(databaseSchemaMetadata.Schemas, schemaMetadata)

	return databaseSchemaMetadata, nil
}

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("SyncSlowQuery() is not applicable to Hive")
}

func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("CheckSlowQueryLogEnabled() is not applicable to Hive")
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	results, err := d.QueryConn(ctx, nil, "SELECT VERSION()", nil)
	if err != nil || len(results) == 0 {
		return "", errors.Wrap(err, "failed to get version from instance")
	}
	return results[0].Rows[0].Values[0].GetStringValue(), nil
}

func (d *Driver) getDatabaseNames(ctx context.Context) ([]string, error) {
	var databaseNames []string
	results, err := d.QueryConn(ctx, nil, "SHOW DATABASES", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases from instance")
	}
	for _, row := range results[0].Rows {
		databaseNames = append(databaseNames, row.Values[0].GetStringValue())
	}
	return databaseNames, nil
}

func (d *Driver) listTablesNames(ctx context.Context, databaseName string) ([]string, error) {
	var tabNames []string
	tabResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW TABLES FROM %s", databaseName), nil)

	for _, row := range tabResults[0].Rows {
		tabNames = append(tabNames, row.Values[0].GetStringValue())
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from instance")
	}
	return tabNames, nil
}

// getTables fetches table info and returns structed table data.
func (d *Driver) getTables(ctx context.Context, databaseName string) (
	[]*storepb.TableMetadata,
	[]*storepb.ExternalTableMetadata,
	[]*storepb.ViewMetadata,
	[]*storepb.MaterializedViewMetadata,
	error,
) {
	var (
		tableMetadatas    []*storepb.TableMetadata
		extTableMetadatas []*storepb.ExternalTableMetadata
		viewMetadatas     []*storepb.ViewMetadata
		mtViewMetadatas   []*storepb.MaterializedViewMetadata
	)

	// list tables' names.
	tabNames, err := d.listTablesNames(ctx, databaseName)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "failed to list tables")
	}

	// iterations in tables of certain database.
	for _, tabName := range tabNames {
		// filter out index table names.
		if strings.HasSuffix(tabName, "__") {
			continue
		}

		tabInfo, err := d.getTableInfo(ctx, tabName, databaseName)
		if err != nil {
			return nil, nil, nil, nil, errors.Wrapf(err, "failed to describe table %s's type", tabName)
		}

		// different processing way according to the type of the table.
		if tabInfo.tabType == "MATERIALIZED_VIEW" {
			mtViewMetadatas = append(mtViewMetadatas, &storepb.MaterializedViewMetadata{
				Name:       tabName,
				Definition: tabInfo.viewDef,
				Comment:    tabInfo.comment,
			})
			continue
		} else if tabInfo.tabType == "VIRTUAL_VIEW" {
			viewMetadatas = append(viewMetadatas, &storepb.ViewMetadata{
				Name:       tabName,
				Definition: tabInfo.viewDef,
				Comment:    tabInfo.comment,
			})
			continue
		} else if tabInfo.tabType == "EXTERNAL_TABLE" {
			extTableMetadatas = append(extTableMetadatas, &storepb.ExternalTableMetadata{
				Name:    tabName,
				Columns: tabInfo.colMetadatas,
			})
			continue
		} else if tabInfo.tabType != "MANAGED_TABLE" {
			return nil, nil, nil, nil, errors.Errorf("unsupported table type: %s", tabInfo.tabType)
		}

		var tableMetadata storepb.TableMetadata

		// indexes.
		indexResults, err := d.getIndexesByTableName(ctx, tabName, databaseName)
		if err != nil {
			return nil, nil, nil, nil, errors.Wrapf(err, "failed to get index info from tab %s", tabName)
		}

		// partitions.
		partitionResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW PARTITIONS `%s`.`%s`", databaseName, tabName), nil)
		if err == nil {
			for _, row := range partitionResults[0].Rows {
				tableMetadata.Partitions = append(tableMetadata.Partitions, &storepb.TablePartitionMetadata{
					Name: row.Values[0].GetStringValue(),
				})
			}
		}

		tableMetadata.Comment = tabInfo.comment
		tableMetadata.DataSize = int64(tabInfo.totalSize)
		tableMetadata.RowCount = int64(tabInfo.numRows)
		tableMetadata.Name = tabName
		tableMetadata.Indexes = indexResults
		tableMetadatas = append(tableMetadatas, &tableMetadata)
	}

	return tableMetadatas, extTableMetadatas, viewMetadatas, mtViewMetadatas, nil
}

// getRoles fetches role names and grant info from instance and returns structed role data.
func (d *Driver) getRoles(ctx context.Context) ([]*storepb.InstanceRoleMetadata, error) {
	var roleMetadata []*storepb.InstanceRoleMetadata
	roleMessages, err := d.ListRole(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role names")
	}

	for _, roleMessage := range roleMessages {
		roleName := roleMessage.Name
		grantString, err := d.GetRoleGrant(ctx, roleName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get grant info from role %s", roleName)
		}
		roleMetadata = append(roleMetadata, &storepb.InstanceRoleMetadata{
			Name:  roleName,
			Grant: grantString,
		})
	}

	return roleMetadata, nil
}

func (d *Driver) getIndexesByTableName(ctx context.Context, tableName string, databaseName string) ([]*storepb.IndexMetadata, error) {
	var (
		indexMetadata []*storepb.IndexMetadata
	)

	conn, err := d.ConnPool.Get()
	if err != nil {
		return nil, err
	}
	cursor := conn.Cursor()
	defer cursor.Close()
	defer d.ConnPool.Put(conn)

	cursor.Execute(ctx, fmt.Sprintf("use %s", databaseName), false)
	if cursor.Err != nil {
		return nil, errors.Wrapf(cursor.Err, "failed to switch to database %s", databaseName)
	}

	cursor.Execute(ctx, fmt.Sprintf("SHOW INDEX ON `%s`", tableName), false)
	if cursor.Err != nil {
		return nil, errors.Wrapf(cursor.Err, "failed to get index info from table %s", tableName)
	}

	for cursor.HasMore(ctx) {
		rowMap := cursor.RowMap(ctx)
		indexMetadata = append(indexMetadata, &storepb.IndexMetadata{
			Name:        strings.ReplaceAll(rowMap["idx_name"].(string), " ", ""),
			Comment:     strings.ReplaceAll(rowMap["comment"].(string), " ", ""),
			Type:        strings.ReplaceAll(rowMap["idx_type"].(string), " ", ""),
			Expressions: strings.Split(strings.ReplaceAll(rowMap["col_names"].(string), " ", ""), ","),
		})
	}

	return indexMetadata, nil
}

// This function gets certain database info by name.
func (d *Driver) getDatabaseInfoByName(ctx context.Context, databaseName string) (*storepb.SchemaMetadata, error) {
	// fetch table metadata.
	tableMetadata, extTabMetadata, viewMetadata, mtViewMetadata, err := d.getTables(ctx, databaseName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table metadata from database %s", databaseName)
	}

	return &storepb.SchemaMetadata{
		Name:              databaseName,
		Tables:            tableMetadata,
		ExternalTables:    extTabMetadata,
		Views:             viewMetadata,
		MaterializedViews: mtViewMetadata,
	}, nil
}

type TableInfo struct {
	tabType      string
	colMetadatas []*storepb.ColumnMetadata
	numRows      int
	viewDef      string
	totalSize    int
	comment      string
}

func (d *Driver) getTableInfo(ctx context.Context, tabName string, databaseName string) (
	*TableInfo,
	error,
) {
	var (
		columnMetadatas []*storepb.ColumnMetadata
		comment         string
		tabType         string
		viewDefination  string
		totalSize       int
		numRows         int
		hasReadColumns  = false
	)
	conn, err := d.ConnPool.Get()
	if err != nil {
		return nil, err
	}
	cursor := conn.Cursor()
	defer func() {
		cursor.Close()
		d.ConnPool.Put(conn)
	}()

	cursor.Exec(ctx, fmt.Sprintf("DESCRIBE FORMATTED `%s`.`%s`", databaseName, tabName))
	if cursor.Err != nil {
		return nil, errors.Wrapf(cursor.Err, "failed to describe table %s", tabName)
	}

	for idx := 0; cursor.HasMore(ctx); idx++ {
		var (
			typeColStr    string
			commentColStr string
			colNameColStr string
			rowMap        = cursor.RowMap(ctx)
		)

		if idx < 2 {
			continue
		}
		if rowMap["data_type"] == nil {
			typeColStr = ""
		} else {
			ok := false
			typeColStr, ok = rowMap["data_type"].(string)
			if !ok {
				return nil, errors.New("type assertions fails: data_type")
			}
		}
		if rowMap["comment"] == nil {
			commentColStr = ""
		} else {
			commentColStr = strings.TrimRight(rowMap["comment"].(string), " ")
		}
		if rowMap["col_name"] == nil {
			colNameColStr = ""
		} else {
			ok := false
			colNameColStr, ok = rowMap["col_name"].(string)
			if !ok {
				return nil, errors.New("type assertions fails: col_name")
			}
		}

		// process table type.
		if strings.Contains(colNameColStr, "Table Type") {
			tabType = strings.ReplaceAll(typeColStr, " ", "")
		} else if colNameColStr == "" {
			hasReadColumns = true
		}
		if !hasReadColumns {
			// process column.
			columnMetadatas = append(columnMetadatas, &storepb.ColumnMetadata{
				Name:    colNameColStr,
				Type:    typeColStr,
				Comment: commentColStr,
			})
		}

		// get row count.
		if strings.Contains(typeColStr, "numRows") {
			n, err := strconv.Atoi(commentColStr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse row count")
			}
			numRows = n
		}
		if strings.Contains(typeColStr, "totalSize") {
			size, err := strconv.Atoi(commentColStr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse table size")
			}
			totalSize = size
		}
		if strings.Contains(colNameColStr, "View Original Text") {
			// get view definition if it exists.
			viewDefination = typeColStr
		}
		if strings.Contains(typeColStr, "comment") {
			comment = commentColStr
		}
	}

	return &TableInfo{
		tabType,
		columnMetadatas,
		numRows,
		viewDefination,
		totalSize,
		comment,
	}, nil
}
