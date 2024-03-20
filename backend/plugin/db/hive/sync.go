package hive

import (
	"context"
	"fmt"
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

// It should be noted that Schema and database have the same meaning in Hive.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	databaseSchemaMetadata := new(storepb.DatabaseSchemaMetadata)

	databaseNames, err := d.getDatabaseNames(ctx)
	if err != nil {
		return nil, err
	}

	var schemaMetadata []*storepb.SchemaMetadata
	execOpts := db.ExecuteOptions{}
	for _, database := range databaseNames {
		// change database.
		_, err := d.Execute(ctx, fmt.Sprintf("use %s", database), execOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to switch to database %s", database)
		}

		// fetch table metadata.
		tableMetadata, err := d.getTables(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get table metadata from database %s", database)
		}

		// fetch view metadata.
		viewMetadata, err := d.getViews(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get view metadata from database %s", database)
		}

		schemaMetadata = append(schemaMetadata, &storepb.SchemaMetadata{
			Tables: tableMetadata,
			Views:  viewMetadata,
		})
	}

	databaseSchemaMetadata.Schemas = schemaMetadata
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

// getTables fetches table info and returns structed table data.
func (d *Driver) getTables(ctx context.Context) ([]*storepb.TableMetadata, error) {
	var tableMetadatas []*storepb.TableMetadata

	// tables' names.
	tabResults, err := d.QueryConn(ctx, nil, "SHOW TABLES", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tables from instance")
	}

	// iterations in tables of certain database.
	for _, row := range tabResults[0].Rows {
		var tableMetadata storepb.TableMetadata
		tableName := row.Values[0].GetStringValue()

		// columns info.
		columnResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("DESC %s", tableName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get columns from table %s", tableName)
		}
		for _, row := range columnResults[0].Rows {
			tableMetadata.Columns = append(tableMetadata.Columns, &storepb.ColumnMetadata{
				Name:    row.Values[0].GetStringValue(),
				Type:    row.Values[1].GetStringValue(),
				Comment: row.Values[2].GetStringValue(),
			})
		}

		// row counts.
		countResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get number of rows from table %s", tableName)
		}

		// index.
		indexResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW INDEX on %s", tableName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index info from table %s", tableName)
		}
		for _, row := range indexResults[0].Rows {
			tableMetadata.Indexes = append(tableMetadata.Indexes, &storepb.IndexMetadata{
				Name:    row.Values[0].GetStringValue(),
				Type:    row.Values[4].GetStringValue(),
				Comment: row.Values[5].GetStringValue(),
			})
		}

		// partitions.
		partitionResults, err := d.QueryConn(ctx, nil, fmt.Sprintf("SHOW PARTITIONS on %s", tableName), nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get index info from table %s", tableName)
		}
		for _, row := range partitionResults[0].Rows {
			tableMetadata.Partitions = append(tableMetadata.Partitions, &storepb.TablePartitionMetadata{
				Name: row.Values[0].GetStringValue(),
			})
		}

		tableMetadata.RowCount = countResults[0].Rows[0].Values[0].GetInt64Value()
		tableMetadata.Name = tableName
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

func (d *Driver) getViews(ctx context.Context) ([]*storepb.ViewMetadata, error) {
	var viewMetadata []*storepb.ViewMetadata
	viewResults, err := d.QueryConn(ctx, nil, "SHOW VIEWS", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get views")
	}
	for _, row := range viewResults[0].Rows {
		viewMetadata = append(viewMetadata, &storepb.ViewMetadata{
			Name: row.Values[0].GetStringValue(),
		})
	}
	return viewMetadata, nil
}
