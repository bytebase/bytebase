import { uniq } from "lodash-es";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";

export const validateDatabaseMetadata = (
  databaseMetadata: DatabaseMetadata
): string[] => {
  const messages: string[] = [];

  for (const schema of databaseMetadata.schemas) {
    for (const table of schema.tables) {
      if (!table.name) {
        messages.push(`Table name is required.`);
        continue;
      }

      for (const column of table.columns) {
        if (!column.name) {
          messages.push(`Column name is required in table ${table.name}`);
          continue;
        }
        if (!column.type) {
          messages.push(`Missing column type in ${table.name}.${column.name}`);
        }
      }
    }
  }

  return uniq(messages);
};
