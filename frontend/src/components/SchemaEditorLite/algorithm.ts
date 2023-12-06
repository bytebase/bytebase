import { ComposedDatabase } from "@/types";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { useSchemaEditorContext } from "./context";

export const useAlgorithm = (
  context: ReturnType<typeof useSchemaEditorContext>
) => {
  const { getSchemaStatus, getTableStatus, getColumnStatus } = context;

  const diff = () => {};

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
  };

  return { diff, applyMetadataEdit };
};
