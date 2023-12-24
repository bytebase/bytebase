import {
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { keyBy } from "@/utils";

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
