package cassandra

import (
	"context"
	"slices"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// https://docs.datastax.com/en/cql/hcd/reference/system-virtual-tables/system-keyspace-tables.html
func isSystemDatabase(database string) bool {
	switch database {
	case
		"HiveMetaStore",
		"dse_analytics",
		"dse_insights",
		"dse_insights_local",
		"dse_leases",
		"dse_perf",
		"dse_security",
		"dse_system",
		"dse_system_local",
		"dsefs",
		"solr_admin",
		"system_traces",
		"system_auth",
		"system_distributed",
		"system_schema",
		"system":
		return true
	default:
		return false
	}
}

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get version")
	}
	databases, err := d.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		if isSystemDatabase(database.Name) {
			continue
		}
		filteredDatabases = append(filteredDatabases, database)
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: filteredDatabases,
	}, nil
}

type primaryKey struct {
	name            string
	clusteringOrder string
	kind            string
	position        int
}

type columnOrderInfo struct {
	column   *storepb.ColumnMetadata
	kind     string // "partition_key", "clustering", or "regular"
	position int    // Position within partition/clustering keys
}

func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	schemaMetadata := &storepb.SchemaMetadata{
		Name: "",
	}

	tablePKMap := map[string][]primaryKey{}
	columnOrderMap := map[string][]columnOrderInfo{}
	columnScanner := d.session.Query(`
		SELECT
			table_name,
			column_name,
			kind,
			position,
			clustering_order,
			type
		FROM system_schema.columns
		WHERE keyspace_name = ?
	`, d.config.ConnectionContext.DatabaseName).WithContext(ctx).Iter().Scanner()
	for columnScanner.Next() {
		var tableName, columnName, kind, clusteringOrder, columnType string
		var position int
		if err := columnScanner.Scan(
			&tableName,
			&columnName,
			&kind,
			&position,
			&clusteringOrder,
			&columnType,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan column")
		}
		columnOrderMap[tableName] = append(columnOrderMap[tableName], columnOrderInfo{
			column: &storepb.ColumnMetadata{
				Name:     columnName,
				Type:     columnType,
				Nullable: true,
			},
			kind:     kind,
			position: position,
		})
		if kind != "regular" {
			tablePKMap[tableName] = append(tablePKMap[tableName], primaryKey{
				name:            columnName,
				kind:            kind,
				clusteringOrder: clusteringOrder,
				position:        position,
			})
		}
	}
	if err := columnScanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "column scanner err")
	}

	// Sort columns to match Cassandra's SELECT * order:
	// 1. Partition keys (by position)
	// 2. Clustering keys (by position)
	// 3. Regular columns (alphabetically)
	// This ensures masking is applied to the correct columns.
	for tableName, columns := range columnOrderMap {
		slices.SortFunc(columns, func(a, b columnOrderInfo) int {
			// Primary key columns come before regular columns
			aPrimary := (a.kind == "partition_key" || a.kind == "clustering")
			bPrimary := (b.kind == "partition_key" || b.kind == "clustering")

			if aPrimary && !bPrimary {
				return -1
			}
			if !aPrimary && bPrimary {
				return 1
			}

			// Within primary key columns, sort by type then position
			if aPrimary && bPrimary {
				// Partition keys before clustering keys
				if a.kind == "partition_key" && b.kind == "clustering" {
					return -1
				}
				if a.kind == "clustering" && b.kind == "partition_key" {
					return 1
				}
				// Within same type, sort by position
				if a.position < b.position {
					return -1
				}
				if a.position > b.position {
					return 1
				}
			}

			// Regular columns are sorted alphabetically
			if !aPrimary && !bPrimary {
				return strings.Compare(a.column.Name, b.column.Name)
			}

			return 0
		})
		columnOrderMap[tableName] = columns
	}

	indexMap := map[string][]*storepb.IndexMetadata{}
	indexScanner := d.session.Query(`
		SELECT
			table_name,
			index_name,
			kind,
			toJson(options)
		FROM system_schema.indexes
		WHERE keyspace_name = ?
		ORDER BY table_name, index_name
	`, d.config.ConnectionContext.DatabaseName).WithContext(ctx).Iter().Scanner()
	for indexScanner.Next() {
		var tableName, indexName, kind, options string
		if err := indexScanner.Scan(
			&tableName,
			&indexName,
			&kind,
			&options,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan index")
		}

		indexMap[tableName] = append(indexMap[tableName], &storepb.IndexMetadata{
			Name:        indexName,
			Type:        kind,
			Expressions: []string{options},
			Definition:  options,
		})
	}
	if err := indexScanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "index scanner err")
	}

	tableScanner := d.session.Query(`
		SELECT
			table_name,
			comment
		FROM system_schema.tables
		WHERE keyspace_name = ?
		ORDER BY table_name
	`, d.config.ConnectionContext.DatabaseName).WithContext(ctx).Iter().Scanner()
	for tableScanner.Next() {
		var tableName, comment string
		if err := tableScanner.Scan(
			&tableName,
			&comment,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan table")
		}

		pks := tablePKMap[tableName]
		slices.SortFunc(pks, func(a, b primaryKey) int {
			if a.kind == "partition_key" && b.kind == "clustering" {
				return -1
			}
			if a.kind == "clustering" && b.kind == "partition_key" {
				return 1
			}
			if a.position < b.position {
				return -1
			}
			return 1
		})
		pk := &storepb.IndexMetadata{
			Name:        "PRIMARY KEY",
			Expressions: getPKExpressions(pks),
			Descending:  getPKDescending(pks),
			Primary:     true,
			Definition:  getPKDefinition(pks),
		}

		// Extract the sorted column metadata
		var sortedColumns []*storepb.ColumnMetadata
		for _, colInfo := range columnOrderMap[tableName] {
			sortedColumns = append(sortedColumns, colInfo.column)
		}

		table := &storepb.TableMetadata{
			Name:    tableName,
			Comment: comment,
			Columns: sortedColumns,
		}
		table.Indexes = append(table.Indexes, pk)
		table.Indexes = append(table.Indexes, indexMap[tableName]...)

		schemaMetadata.Tables = append(schemaMetadata.Tables, table)
	}
	if err := tableScanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "table scanner err")
	}

	return &storepb.DatabaseSchemaMetadata{
		Name:    d.config.ConnectionContext.DatabaseName,
		Schemas: []*storepb.SchemaMetadata{schemaMetadata},
	}, nil
}

func getPKExpressions(pks []primaryKey) []string {
	var partition, clustering []string
	for _, k := range pks {
		switch k.kind {
		case "partition_key":
			partition = append(partition, k.name)
		case "clustering":
			clustering = append(clustering, k.name)
		default:
			// Ignore other key types
		}
	}

	res := []string{strings.Join(partition, ",")}
	if len(partition) > 1 {
		res[0] = "(" + res[0] + ")"
	}
	res = append(res, clustering...)
	return res
}

func getPKDescending(pks []primaryKey) []bool {
	res := []bool{false}
	for _, pk := range pks {
		if pk.kind == "partition_key" {
			continue
		}
		res = append(res, pk.clusteringOrder == "desc")
	}
	return res
}

func getPKDefinition(pks []primaryKey) string {
	var partition, clustering []string
	for _, k := range pks {
		switch k.kind {
		case "partition_key":
			partition = append(partition, k.name)
		case "clustering":
			clustering = append(clustering, k.name)
		default:
			// Ignore other key types
		}
	}
	if len(partition) == 1 {
		return "(" + strings.Join(append(partition, clustering...), ",") + ")"
	}
	return "(" +
		strings.Join(append([]string{"(" + strings.Join(partition, ",") + ")"}, clustering...), ",") +
		")"
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	var version string
	if err := d.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Scan(&version); err != nil {
		return "", errors.Wrapf(err, "failed to query")
	}
	return version, nil
}

func (d *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	scanner := d.session.Query("SELECT keyspace_name FROM system_schema.keyspaces").WithContext(ctx).Iter().Scanner()

	var databases []*storepb.DatabaseSchemaMetadata
	for scanner.Next() {
		var database storepb.DatabaseSchemaMetadata
		if err := scanner.Scan(&database.Name); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		databases = append(databases, &database)
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "scan error")
	}

	return databases, nil
}
