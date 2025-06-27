import type { ComposedDatabase } from "@/types";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { SchemaEditorContext } from "../context";
import { cleanupUnusedCatalog } from "./utils";

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
    metadata: DatabaseMetadata,
    catalog: DatabaseCatalog
  ) => {
    // Drop schemas
    metadata.schemas = metadata.schemas.filter((schema) => {
      const status = getSchemaStatus(database, {
        schema,
      });
      return status !== "dropped";
    });
    // Drop tables, views, procedures and functions
    metadata.schemas.forEach((schema) => {
      schema.tables = schema.tables.filter((table) => {
        const status = getTableStatus(database, {
          schema,
          table,
        });
        return status !== "dropped";
      });
      schema.views = schema.views.filter((view) => {
        const status = getViewStatus(database, {
          schema,
          view,
        });
        return status !== "dropped";
      });
      schema.procedures = schema.procedures.filter((procedure) => {
        const status = getProcedureStatus(database, {
          schema,
          procedure,
        });
        return status !== "dropped";
      });
      schema.functions = schema.functions.filter((func) => {
        const status = getFunctionStatus(database, {
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
            schema,
            table,
            column,
          });
          return status !== "dropped";
        });
      });
    });

    cleanupUnusedCatalog(metadata, catalog);
  };

  return { applyMetadataEdit };
};
