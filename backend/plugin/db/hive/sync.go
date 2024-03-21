package hive

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TODO(tommy): another approch for this is fetching data from metastore database.
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
func (d *Driver) getTables(ctx context.Context, viewMap map[string]bool) ([]*storepb.TableMetadata, error) {
	var tableMetadatas []*storepb.TableMetadata

	// tables' names.
	tabNames, err := d.getTablesNames(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list tables")
	}

	// iterations in tables of certain database.
	for _, tabName := range tabNames {
		// filter out view and index names from table names.
		_, ok := viewMap[tabName]
		if ok || strings.HasSuffix(tabName, "__") {
			continue
		}

		var (
			tableMetadata storepb.TableMetadata
			isPartitioned = false
		)

		// columns info.
		columnResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("DESC %s", tabName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get columns from table %s", tabName)
		}
		for _, columnRow := range columnResults[0].Rows {
			column := &storepb.ColumnMetadata{
				Name:    columnRow.Values[0].GetStringValue(),
				Type:    columnRow.Values[1].GetStringValue(),
				Comment: columnRow.Values[2].GetStringValue(),
			}
			// there will be an empty row if the table is partitioned.
			if column.Name == "" {
				isPartitioned = true
				break
			}
			tableMetadata.Columns = append(tableMetadata.Columns, column)
		}

		// row counts.
		countResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SELECT COUNT(*) FROM %s", tabName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get number of rows from table %s", tabName)
		}

		// indexes.
		indexResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW INDEX on %s", tabName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index info from table %s", tabName)
		}
		for _, row := range indexResults[0].Rows {
			tableMetadata.Indexes = append(tableMetadata.Indexes, &storepb.IndexMetadata{
				Name:    row.Values[0].GetStringValue(),
				Type:    row.Values[4].GetStringValue(),
				Comment: row.Values[5].GetStringValue(),
			})
		}

		// partitions.
		if isPartitioned {
			partitionResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW PARTITIONS %s", tabName), nil)
			// check whether the table is partitioned.
			if err == nil && partitionResults[0].Error == "" {
				for _, row := range partitionResults[0].Rows {
					tableMetadata.Partitions = append(tableMetadata.Partitions, &storepb.TablePartitionMetadata{
						Name: row.Values[0].GetStringValue(),
					})
				}
			}
		}

		tableMetadata.RowCount = countResults[0].Rows[0].Values[0].GetInt64Value()
		tableMetadata.Name = tabName
		tableMetadatas = append(tableMetadatas, &tableMetadata)
	}

	return tableMetadatas, nil
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

func (d *Driver) getViews(ctx context.Context) ([]*storepb.ViewMetadata, map[string]bool, error) {
	var (
		viewMetadata []*storepb.ViewMetadata
		viewMap      = map[string]bool{}
	)

	viewResults, err := d.QueryConn(ctx, nil, "SHOW VIEWS", nil)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get views")
	}
	for _, row := range viewResults[0].Rows {
		viewName := row.Values[0].GetStringValue()
		viewMap[viewName] = true

		viewMetadata = append(viewMetadata, &storepb.ViewMetadata{
			Name: viewName,
		})
	}
	return viewMetadata, viewMap, nil
}

// This function gets certain database info by name.
func (d *Driver) getDatabaseInfoByName(ctx context.Context, databaseName string) (*storepb.SchemaMetadata, error) {
	// change database.
	// TODO(tommy): tables in different databases can be accessed by [database].[table].
	_, err := d.Execute(ctx, fmt.Sprintf("use %s", databaseName), db.ExecuteOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to switch to database %s", databaseName)
	}

	// fetch view metadata.
	viewMetadata, viewMap, err := d.getViews(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get view metadata from database %s", databaseName)
	}

	// fetch table metadata.
	tableMetadata, err := d.getTables(ctx, viewMap)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table metadata from database %s", databaseName)
	}

	return &storepb.SchemaMetadata{
		Name:   databaseName,
		Tables: tableMetadata,
		Views:  viewMetadata,
	}, nil
}
