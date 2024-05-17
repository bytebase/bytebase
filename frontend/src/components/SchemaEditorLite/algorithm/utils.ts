import type { ComposedDatabase } from "@/types";
import type {
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
    })
  };
  const cleanupViewConfigs = (
    schema: SchemaMetadata,
    schemaConfig: SchemaConfig
  ) => {
    const viewMap = keyBy(schema.views, (view) => view.name);
    // Remove unused view configs
    schemaConfig.viewConfigs = schemaConfig.viewConfigs.filter((tc) =>
      viewMap.has(tc.name)
    );
  };
  const cleanupFunctionConfigs = (
    schema: SchemaMetadata,
    schemaConfig: SchemaConfig
  ) => {
    const functionMap = keyBy(schema.functions, (func) => func.name);
    // Remove unused function configs
    schemaConfig.functionConfigs = schemaConfig.functionConfigs.filter((tc) =>
      functionMap.has(tc.name)
    );
  };
  const cleanupProcedureConfigs = (
    schema: SchemaMetadata,
    schemaConfig: SchemaConfig
  ) => {
    const procedureMap = keyBy(schema.procedures, (p) => p.name);
    // Remove unused procedure configs
    schemaConfig.procedureConfigs = schemaConfig.procedureConfigs.filter((tc) =>
      procedureMap.has(tc.name)
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
      const schema = schemaMap.get(sc.name)!;
      cleanupTableConfigs(schema, sc);
      cleanupViewConfigs(schema, sc);
      cleanupFunctionConfigs(schema, sc);
      cleanupProcedureConfigs(schema, sc);
    });
    // Cleanup empty schema configs
    metadata.schemaConfigs = metadata.schemaConfigs.filter(
      (sc) =>
        sc.tableConfigs.length > 0 ||
        sc.viewConfigs.length > 0 ||
        sc.functionConfigs.length > 0 ||
        sc.procedureConfigs.length > 0
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
          const key = keyForResourceName({
            database: db.name,
            schema: schemaConfig.name,
            table: tableConfig.name,
            column: columnConfig.name,
          });
          return [key, { schemaConfig, tableConfig, columnConfig }];
        });
      });
    })
  );
};
