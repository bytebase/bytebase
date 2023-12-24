import { ComposedDatabase } from "@/types";
import {
  ColumnConfig,
  ColumnMetadata,
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { keyBy } from "@/utils";
import { keyForResource, keyForResourceName } from "../context/common";

type RichSchemaMetadata = {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
};
type RichTableMetadata = RichSchemaMetadata & {
  table: TableMetadata;
};
type RichColumnMetadata = RichTableMetadata & {
  column: ColumnMetadata;
};
type RichSchemaConfig = {
  schemaConfig: SchemaConfig;
};
type RichTableConfig = RichSchemaConfig & {
  tableConfig: TableConfig;
};
type RichColumnConfig = RichTableConfig & {
  columnConfig: ColumnConfig;
};

export const cleanupUnusedConfigs = (metadata: DatabaseMetadata) => {
  const cleanupColumnConfigs = (
    table: TableMetadata,
    tableConfig: TableConfig
  ) => {
    const columnMap = keyBy(table.columns, (column) => column.name);
    // Remove unused column configs
    tableConfig.columnConfigs = tableConfig.columnConfigs.filter((cc) =>
      columnMap.has(cc.name)
    );
  };
  const cleanupTableConfigs = (
    schema: SchemaMetadata,
    schemaConfig: SchemaConfig
  ) => {
    const tableMap = keyBy(schema.tables, (table) => table.name);
    // Remove unused table configs
    schemaConfig.tableConfigs = schemaConfig.tableConfigs.filter((tc) =>
      tableMap.has(tc.name)
    );
    // Recursively cleanup column configs
    schemaConfig.tableConfigs.forEach((tc) => {
      cleanupColumnConfigs(tableMap.get(tc.name)!, tc);
    });
    // Cleanup empty table configs
    schemaConfig.tableConfigs = schemaConfig.tableConfigs.filter(
      (tc) => tc.columnConfigs.length > 0
    );
  };
  const cleanupSchemaConfigs = (metadata: DatabaseMetadata) => {
    const schemaMap = keyBy(metadata.schemas, (schema) => schema.name);
    // Remove unused schema configs
    metadata.schemaConfigs = metadata.schemaConfigs.filter((sc) =>
      schemaMap.has(sc.name)
    );
    // Recursively cleanup table configs
    metadata.schemaConfigs.forEach((sc) => {
      cleanupTableConfigs(schemaMap.get(sc.name)!, sc);
    });
    // Cleanup empty schema configs
    metadata.schemaConfigs = metadata.schemaConfigs.filter(
      (sc) => sc.tableConfigs.length > 0
    );
  };

  cleanupSchemaConfigs(metadata);
};

export const buildColumnMap = (
  db: ComposedDatabase,
  database: DatabaseMetadata
) => {
  return new Map<string, RichColumnMetadata>(
    database.schemas.flatMap((schema) => {
      return schema.tables.flatMap((table) => {
        return table.columns.map((column) => {
          const key = keyForResource(db, {
            schema,
            table,
            column,
          });
          return [key, { database, schema, table, column }];
        });
      });
    })
  );
};
export const buildColumnConfigMap = (
  db: ComposedDatabase,
  database: DatabaseMetadata
) => {
  return new Map<string, RichColumnConfig>(
    database.schemaConfigs.flatMap((schemaConfig) => {
      return schemaConfig.tableConfigs.flatMap((tableConfig) => {
        return tableConfig.columnConfigs.map((columnConfig) => {
          const key = keyForResourceName(
            db.name,
            schemaConfig.name,
            tableConfig.name,
            columnConfig.name
          );
          return [key, { schemaConfig, tableConfig, columnConfig }];
        });
      });
    })
  );
};
