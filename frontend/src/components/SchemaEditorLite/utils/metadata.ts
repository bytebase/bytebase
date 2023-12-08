import { uniq } from "lodash-es";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";

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
          messages.push(`Column name is required.`);
          continue;
        }
        if (!column.type) {
          messages.push(`Column ${column.name} type is required.`);
        }
      }
    }
  }

  return uniq(messages);
};
