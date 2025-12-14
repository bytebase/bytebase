import { cloneDeep } from "lodash-es";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { SchemaEditorContext } from "../context";

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
    const response = cloneDeep(metadata);
    // Drop schemas
    response.schemas = response.schemas.filter((schema) => {
      const status = getSchemaStatus(database, {
        schema,
      });
      return status !== "dropped";
    });
    // Drop tables, views, procedures and functions
    response.schemas.forEach((schema) => {
      schema.tables = schema.tables.filter((table) => {
        const status = getTableStatus(database, {
          schema,
          table,
        });
        return status !== "dropped";
      });
      // Drop columns
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

    return { metadata: response };
  };

  return { applyMetadataEdit };
};
