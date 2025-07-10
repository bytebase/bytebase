package bigquery

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// SyncInstance syncs the instance.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var databases []*storepb.DatabaseSchemaMetadata

	it := d.client.Datasets(ctx)
	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		databases = append(databases, &storepb.DatabaseSchemaMetadata{Name: dataset.DatasetID})
	}

	return &db.InstanceMetadata{
		Databases: databases,
	}, nil
}

type columnRow struct {
	TableName       string              `bigquery:"table_name"`
	ColumnName      string              `bigquery:"column_name"`
	OrdinalPosition int32               `bigquery:"ordinal_position"`
	IsNullable      string              `bigquery:"is_nullable"`
	DataType        string              `bigquery:"data_type"`
	CollationName   bigquery.NullString `bigquery:"collation_name"`
	ColumnDefault   bigquery.NullString `bigquery:"column_default"`
}

// SyncDBSchema syncs a single database schema.
func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{}

	columnMap := make(map[db.TableKey][]*storepb.ColumnMetadata)
	q := d.client.Query(fmt.Sprintf(`
		SELECT
			table_name,
			column_name,
			ordinal_position,
			is_nullable,
			data_type,
			collation_name,
			column_default
		FROM %s.INFORMATION_SCHEMA.COLUMNS ORDER BY table_name, ordinal_position;`, d.databaseName))
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	for {
		var row columnRow
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		nullableBool, err := util.ConvertYesNo(row.IsNullable)
		if err != nil {
			return nil, err
		}

		column := &storepb.ColumnMetadata{
			Name:     row.ColumnName,
			Position: row.OrdinalPosition,
			Nullable: nullableBool,
			Type:     row.DataType,
		}
		if row.CollationName.Valid {
			column.Collation = row.CollationName.String()
		}
		if row.ColumnDefault.Valid {
			column.Default = row.ColumnDefault.String()
		} else {
			column.DefaultNull = true
		}

		key := db.TableKey{Schema: "", Table: row.TableName}
		columnMap[key] = append(columnMap[key], column)
	}
	ts := d.client.Dataset(d.databaseName).Tables(ctx)
	for {
		t, err := ts.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		key := db.TableKey{Schema: "", Table: t.TableID}
		columns := columnMap[key]
		tmd, err := t.Metadata(ctx)
		if err != nil {
			return nil, err
		}
		schemaMetadata.Tables = append(schemaMetadata.Tables, &storepb.TableMetadata{
			Name:     t.TableID,
			Columns:  columns,
			RowCount: int64(tmd.NumRows),
			Comment:  tmd.Description,
			DataSize: tmd.NumBytes,
		})
	}
	return &storepb.DatabaseSchemaMetadata{
		Name:    d.databaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}
