import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import type { SchemaEditorContext } from "../context";
import { cleanupUnusedConfigs } from "./utils";

export const useApplyMetadataEdit = (context: SchemaEditorContext) => {
  const {
    getSchemaStatus,
    getTableStatus,
    getColumnStatus,
    getViewStatus,
    getProcedureStatus,
    getFunctionStatus,
  } = context;

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
    // Drop tables, views, procedures and functions
    metadata.schemas.forEach((schema) => {
      schema.tables = schema.tables.filter((table) => {
        const status = getTableStatus(database, {
          database: metadata,
          schema,
          table,
        });
        return status !== "dropped";
      });
      schema.views = schema.views.filter((view) => {
        const status = getViewStatus(database, {
          database: metadata,
          schema,
          view,
        });
        return status !== "dropped";
      });
      schema.procedures = schema.procedures.filter((procedure) => {
        const status = getProcedureStatus(database, {
          database: metadata,
          schema,
          procedure,
        });
        return status !== "dropped";
      });
      schema.functions = schema.functions.filter((func) => {
        const status = getFunctionStatus(database, {
          database: metadata,
          schema,
          function: func,
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

  return { applyMetadataEdit };
};
