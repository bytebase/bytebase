import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaConfig,
  SchemaMetadata,
  TableConfig,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { keyBy } from "@/utils";
import { SchemaEditorContext } from "../context";
import { DiffMerge } from "./diff-merge";

export const useAlgorithm = (context: SchemaEditorContext) => {
  const { getSchemaStatus, getTableStatus, getColumnStatus, clearEditStatus } =
    context;

  const rebuildMetadataEdit = (
    database: ComposedDatabase,
    source: DatabaseMetadata,
    target: DatabaseMetadata
  ) => {
    clearEditStatus();
    const dm = new DiffMerge(context, database, source, target);
    dm.merge();
    dm.timer.printAll();
  };

  const applyMetadataEdit = (
    database: ComposedDatabase,
    metadata: DatabaseMetadata
  ) => {
    // Drop schemas
    metadata.schemas = metadata.schemas.filter((schema) => {
      const status = getSchemaStatus(database, {
        database: metadata,
        schema,
      });
      return status !== "dropped";
    });
    // Drop tables
    metadata.schemas.forEach((schema) => {
      schema.tables = schema.tables.filter((table) => {
        const status = getTableStatus(database, {
          database: metadata,
          schema,
          table,
        });
        return status !== "dropped";
      });
    });
    // Drop columns
    metadata.schemas.forEach((schema) => {
      schema.tables.forEach((table) => {
        table.columns = table.columns.filter((column) => {
          const status = getColumnStatus(database, {
            database: metadata,
            schema,
            table,
            column,
          });
          return status !== "dropped";
        });
      });
    });

    cleanupUnusedConfigs(metadata);
  };

  return { rebuildMetadataEdit, applyMetadataEdit };
};

const cleanupUnusedConfigs = (metadata: DatabaseMetadata) => {
  const cleanupColumnConfigs = (
    table: TableMetadata,
    tableConfig: TableConfig
  ) => {
    const columnMap = keyBy(table.columns, (column) => column.name);
    tableConfig.columnConfigs = tableConfig.columnConfigs.filter((cc) =>
      columnMap.has(cc.name)
    );
  };
  const cleanupTableConfigs = (
    schema: SchemaMetadata,
    schemaConfig: SchemaConfig
  ) => {
    const tableMap = keyBy(schema.tables, (table) => table.name);
    schemaConfig.tableConfigs = schemaConfig.tableConfigs.filter((tc) =>
      tableMap.has(tc.name)
    );
    schemaConfig.tableConfigs.forEach((tc) => {
      cleanupColumnConfigs(tableMap.get(tc.name)!, tc);
    });
  };
  const cleanupSchemaConfigs = (metadata: DatabaseMetadata) => {
    const schemaMap = keyBy(metadata.schemas, (schema) => schema.name);
    metadata.schemaConfigs = metadata.schemaConfigs.filter((sc) =>
      schemaMap.has(sc.name)
    );
    metadata.schemaConfigs.forEach((sc) => {
      cleanupTableConfigs(schemaMap.get(sc.name)!, sc);
    });
  };

  cleanupSchemaConfigs(metadata);
};
