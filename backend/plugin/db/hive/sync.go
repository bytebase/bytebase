package hive

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
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

	for _, v := range databaseNames {
		instanceMetadata.Databases = append(instanceMetadata.Databases, &storepb.DatabaseSchemaMetadata{
			Name: v,
		})
	}

	instanceMetadata.Version = version
	return &instanceMetadata, nil
}

func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	dbName := d.config.Database
	if dbName == "" {
		dbName = "default"
	}

	schemaMetadata, err := d.getDatabaseInfoByName(ctx, dbName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get info from database %s", dbName)
	}
	return &storepb.DatabaseSchemaMetadata{
		Name:    dbName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("SyncSlowQuery() is not applicable to Hive")
}

func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("CheckSlowQueryLogEnabled() is not applicable to Hive")
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	result, err := runSingleStatement(ctx, d.conn, "SELECT VERSION()", d.config.MaximumSQLResultSize)
	if err != nil {
		return "", errors.Wrap(err, "failed to get version from instance")
	}

	if len(result.Rows) == 0 || len(result.Rows[0].Values) == 0 {
		return "", errors.New("invalid version result")
	}
	// rawVersion has the format of "1.2.3 commitID".
	rawVersion := result.Rows[0].Values[0].GetStringValue()
	tokens := strings.Split(rawVersion, " ")
	if len(tokens) == 0 {
		return "", errors.Errorf("invalid version %q", rawVersion)
	}
	return tokens[0], nil
}

func (d *Driver) getDatabaseNames(ctx context.Context) ([]string, error) {
	var databaseNames []string
	result, err := runSingleStatement(ctx, d.conn, "SHOW DATABASES", d.config.MaximumSQLResultSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get version from instance")
	}
	for _, row := range result.Rows {
		if row == nil || len(row.Values) == 0 {
			return nil, errors.New("row values have zero length")
		}
		databaseNames = append(databaseNames, row.Values[0].GetStringValue())
	}
	return databaseNames, nil
}

func (d *Driver) listTablesNames(ctx context.Context, databaseName string) ([]string, error) {
	result, err := runSingleStatement(ctx, d.conn, fmt.Sprintf("SHOW TABLES FROM %s", databaseName), d.config.MaximumSQLResultSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get version from instance")
	}

	var tableNames []string
	for _, row := range result.Rows {
		if row == nil || len(row.Values) == 0 {
			return nil, errors.New("row values have zero length")
		}
		tableNames = append(tableNames, row.Values[0].GetStringValue())
	}
	return tableNames, nil
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
	tableNames, err := d.listTablesNames(ctx, databaseName)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "failed to list tables")
	}

	// iterations in tables of certain database.
	for _, tableName := range tableNames {
		// filter out index table names.
		if strings.HasSuffix(tableName, "__") {
			continue
		}

		tableInfo, err := d.getTableInfo(ctx, tableName, databaseName)
		if err != nil {
			return nil, nil, nil, nil, errors.Wrapf(err, "failed to describe table %s's type", tableName)
		}

		// different processing way according to the type of the table.
		switch tableInfo.tableType {
		case "MATERIALIZED_VIEW":
			mtViewMetadatas = append(mtViewMetadatas, &storepb.MaterializedViewMetadata{
				Name:       tableName,
				Definition: tableInfo.viewDef,
				Comment:    tableInfo.comment,
			})
		case "VIRTUAL_VIEW":
			viewMetadatas = append(viewMetadatas, &storepb.ViewMetadata{
				Name:       tableName,
				Definition: tableInfo.viewDef,
				Comment:    tableInfo.comment,
			})
		case "EXTERNAL_TABLE":
			extTableMetadatas = append(extTableMetadatas, &storepb.ExternalTableMetadata{
				Name:    tableName,
				Columns: tableInfo.colMetadatas,
			})
		case "MANAGED_TABLE":
			partitions, err := d.getPartitions(ctx, databaseName, tableName)
			if err != nil {
				// Ignore partitions error as some tables aren't partitioned.
				slog.Debug("failed to get partitions", log.BBError(err))
				continue
			}
			tableMetadatas = append(tableMetadatas, &storepb.TableMetadata{
				Engine:     "HDFS",
				Comment:    tableInfo.comment,
				Columns:    tableInfo.colMetadatas,
				DataSize:   int64(tableInfo.totalSize),
				RowCount:   int64(tableInfo.numRows),
				Partitions: partitions,
				Name:       tableName,
			})
		default:
			// ignore other types of tables.
		}
	}

	return tableMetadatas, extTableMetadatas, viewMetadatas, mtViewMetadatas, nil
}

func (d *Driver) getPartitions(ctx context.Context, databaseName, tableName string) ([]*storepb.TablePartitionMetadata, error) {
	// partitions.
	partitionResult, err := runSingleStatement(ctx, d.conn, fmt.Sprintf("SHOW PARTITIONS `%s`.`%s`", databaseName, tableName), d.config.MaximumSQLResultSize)
	if err != nil {
		slog.Debug("failed to get partitions", log.BBError(err))
		return nil, nil
	}
	if partitionResult == nil {
		return nil, nil
	}
	var partitions []*storepb.TablePartitionMetadata
	for _, row := range partitionResult.Rows {
		if row == nil || len(row.Values) == 0 {
			return nil, errors.New("partitions result row has zero length")
		}
		partitions = append(partitions, &storepb.TablePartitionMetadata{
			Name: row.Values[0].GetStringValue(),
		})
	}
	return partitions, nil
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
	tableType    string
	colMetadatas []*storepb.ColumnMetadata
	numRows      int
	viewDef      string
	totalSize    int
	comment      string
}

const (
	columnSection = iota
	tableInfoSection
	storageSection
	viewSection
)

func (d *Driver) getTableInfo(ctx context.Context, tableName string, databaseName string) (
	*TableInfo,
	error,
) {
	tableInfo := &TableInfo{}

	cursor := d.conn.Cursor()
	query := fmt.Sprintf("DESCRIBE FORMATTED `%s`.`%s`", databaseName, tableName)
	if err := executeCursor(ctx, cursor, query); err != nil {
		return nil, errors.Wrapf(err, "failed to describe table %s", tableName)
	}

	section := columnSection
	for cursor.HasMore(ctx) {
		rowMap := cursor.RowMap(ctx)
		var colName, dataType, comment string
		colNameVal, dataTypeVal, commentVal := rowMap["col_name"], rowMap["data_type"], rowMap["comment"]
		if v, ok := colNameVal.(string); ok {
			colName = v
		}
		if v, ok := dataTypeVal.(string); ok {
			dataType = v
		}
		if v, ok := commentVal.(string); ok {
			comment = v
		}

		// The first rows are column metadata, followed by "# Detailed Table Information" and "# Storage Information"
		switch {
		case strings.HasPrefix(colName, "# col_name"):
			continue
		case strings.HasPrefix(colName, "# Detailed Table Information"):
			section = tableInfoSection
		case strings.HasPrefix(colName, "# Storage Information"):
			section = storageSection
		case strings.HasPrefix(colName, "# View Information"):
			section = viewSection
		}
		switch section {
		case columnSection:
			if colNameVal != nil && dataTypeVal != nil && colName != "" && dataType != "" {
				// Column metadata.
				position := len(tableInfo.colMetadatas) + 1
				tableInfo.colMetadatas = append(tableInfo.colMetadatas, &storepb.ColumnMetadata{
					Name:     colName,
					Type:     dataType,
					Comment:  comment,
					Position: int32(position),
				})
			}
		case tableInfoSection:
			switch {
			case trimField(colName) == "Table Type:":
				tableInfo.tableType = trimField(dataType)
			case trimField(dataType) == "numRows":
				n, err := strconv.Atoi(trimField(comment))
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse row count")
				}
				tableInfo.numRows = n
			case trimField(dataType) == "totalSize":
				n, err := strconv.Atoi(trimField(comment))
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse row count")
				}
				tableInfo.totalSize = n
			case trimField(dataType) == "comment":
				tableInfo.comment = trimField(comment)
			}
		case viewSection:
			switch {
			case strings.HasPrefix(colName, "View Original Text:"):
				tableInfo.viewDef = trimField(dataType)
			case strings.HasPrefix(colName, "Original Query:"):
				tableInfo.viewDef = trimField(dataType)
			}
		}
	}

	return tableInfo, nil
}

func trimField(field string) string {
	return strings.Trim(field, " ")
}
