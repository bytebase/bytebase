package hive

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

	// schema.
	databaseSchemaMetadata, err := d.SyncDBSchema(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sync database schema")
	}

	// roles.
	rolesMetadata, err := d.getRoles(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role metadata")
	}

	instanceMetadata.InstanceRoles = rolesMetadata
	instanceMetadata.Databases = append(instanceMetadata.Databases, databaseSchemaMetadata)
	instanceMetadata.Version = version
	return &instanceMetadata, nil
}

// It should be noted that Schema and Database have the same meaning in Hive.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	databaseSchemaMetadata := new(storepb.DatabaseSchemaMetadata)

	databaseNames, err := d.getDatabaseNames(ctx)
	if err != nil {
		return nil, err
	}

	for _, databaseName := range databaseNames {
		schemaMetadata, err := d.getDatabaseInfoByName(ctx, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get info from database %s", databaseName)
		}
		databaseSchemaMetadata.Schemas = append(databaseSchemaMetadata.Schemas, schemaMetadata)
	}

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

func (d *Driver) getTablesNames(ctx context.Context) ([]string, error) {
	var tabNames []string
	tabResults, err := d.QueryConn(ctx, nil, "SHOW TABLES", nil)

	for _, row := range tabResults[0].Rows {
		tabNames = append(tabNames, row.Values[0].GetStringValue())
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from instance")
	}
	return tabNames, nil
}

// getTables fetches table info and returns structed table data.
func (d *Driver) getTables(ctx context.Context) (
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
	tabNames, err := d.getTablesNames(ctx)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "failed to list tables")
	}

	// iterations in tables of certain database.
	for _, tabName := range tabNames {
		// filter out index table names.
		if strings.HasSuffix(tabName, "__") {
			continue
		}

		// different processing according to the type of the table.
		tableType, columnMetaData, rowCount, viewDefination, err := d.getTableInfo(ctx, tabName)
		if err != nil {
			return nil, nil, nil, nil, errors.Wrapf(err, "failed to describe table %s's type", tabName)
		}

		if tableType == "MATERIALIZED_VIEW" {
			mtViewMetadatas = append(mtViewMetadatas, &storepb.MaterializedViewMetadata{
				Name:       tabName,
				Definition: viewDefination,
			})
			continue
		} else if tableType == "VIRTUAL_VIEW" {
			viewMetadatas = append(viewMetadatas, &storepb.ViewMetadata{
				Name: tabName,
			})
			continue
		} else if tableType == "EXTERNAL_TABLE" {
			extTableMetadatas = append(extTableMetadatas, &storepb.ExternalTableMetadata{
				Name:    tabName,
				Columns: columnMetaData,
			})
			continue
		} else if tableType != "MANAGED_TABLE" {
			return nil, nil, nil, nil, errors.Errorf("unsupported table type: %s", tableType)
		}

		var tableMetadata storepb.TableMetadata

		// indexes.
		indexResults, err := d.getIndexesByTableName(ctx, tabName)
		if err != nil {
			return nil, nil, nil, nil, errors.Wrapf(err, "failed to get index info from tab %s", tabName)
		}

		// partitions.
		partitionResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW PARTITIONS %s", tabName), nil)
		if err == nil && partitionResults[0].Error == "" {
			for _, row := range partitionResults[0].Rows {
				tableMetadata.Partitions = append(tableMetadata.Partitions, &storepb.TablePartitionMetadata{
					Name: row.Values[0].GetStringValue(),
				})
			}
		}

		tableMetadata.RowCount = int64(rowCount)
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

func (d *Driver) getIndexesByTableName(ctx context.Context, tableName string) ([]*storepb.IndexMetadata, error) {
	var (
		indexMetadata []*storepb.IndexMetadata
		cursor        = d.dbClient.Cursor()
	)
	defer cursor.Close()

	cursor.Execute(ctx, fmt.Sprintf("SHOW INDEX ON %s", tableName), false)
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
	// change database.
	// TODO(tommy): tables in different databases can be accessed by [database].[table].
	_, err := d.Execute(ctx, fmt.Sprintf("use %s", databaseName), db.ExecuteOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to switch to database %s", databaseName)
	}

	// fetch table metadata.
	tableMetadata, extTabMetadata, viewMetadata, mtViewMetadata, err := d.getTables(ctx)
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

func (d *Driver) getTableInfo(ctx context.Context, tabName string) (
	string,
	[]*storepb.ColumnMetadata,
	int,
	string,
	error,
) {
	var (
		columnMetadatas []*storepb.ColumnMetadata
		tabType         string
		viewDefination  string
		numRows         int
		hasReadColumns  = false
		cursor          = d.dbClient.Cursor()
	)
	defer cursor.Close()

	cursor.Exec(ctx, fmt.Sprintf("DESCRIBE FORMATTED %s", tabName))
	if cursor.Err != nil {
		return "", nil, 0, "", errors.Wrapf(cursor.Err, "failed to describe table %s", tabName)
	}

	for idx := 0; cursor.HasMore(ctx); idx++ {
		var (
			typeStr    string
			commentStr string
			rowMap     = cursor.RowMap(ctx)
			colName    = rowMap["col_name"].(string)
		)

		if idx < 2 {
			continue
		}
		if rowMap["data_type"] == nil {
			typeStr = ""
		} else {
			ok := false
			typeStr, ok = rowMap["data_type"].(string)
			if !ok {
				return "", nil, 0, "", errors.New("type assertions fails")
			}
		}
		if rowMap["comment"] == nil {
			commentStr = ""
		} else {
			commentStr = strings.ReplaceAll(rowMap["comment"].(string), " ", "")
		}

		// process table type.
		if strings.Contains(colName, "Table Type") {
			tabType = strings.ReplaceAll(typeStr, " ", "")
		} else if colName == "" {
			hasReadColumns = true
		}
		if !hasReadColumns {
			// process column.
			columnMetadatas = append(columnMetadatas, &storepb.ColumnMetadata{
				Name:    colName,
				Type:    typeStr,
				Comment: commentStr,
			})
		}

		// get row count.
		if strings.Contains(typeStr, "numRows") {
			n, err := strconv.Atoi(commentStr)
			if err != nil {
				return "", nil, 0, "", errors.Wrapf(err, "failed to parse row count")
			}
			numRows = n
		} else if strings.Contains(colName, "View Original Text") {
			// get view definition if it exists.
			viewDefination = typeStr
		}
	}

	return tabType, columnMetadatas, numRows, viewDefination, nil
}
