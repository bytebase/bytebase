import { watch } from "vue";
import { useCurrentSQLEditorTab } from "@/store";
import type { SQLEditorConnection } from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { engineNameV1 } from "@/utils";

export const onConnectionChanged = (
  fn: (
    newConn: SQLEditorConnection,
    oldConn: SQLEditorConnection | undefined
  ) => void,
  immediate = false
) => {
  const tab = useCurrentSQLEditorTab();
  return watch(
    [
      () => tab.value?.connection.instance,
      () => tab.value?.connection.database,
    ],
    (newValues, oldValues) => {
      fn(
        { instance: newValues[0] ?? "", database: newValues[1] ?? "" },
        oldValues[0] && oldValues[1]
          ? { instance: oldValues[0] ?? "", database: oldValues[1] ?? "" }
          : undefined
      );
    },
    { immediate }
  );
};

export const databaseMetadataToText = (
  databaseMetadata: DatabaseMetadata | undefined,
  engine?: Engine
) => {
  const prompts: string[] = [];
  if (engine) {
    if (databaseMetadata) {
      prompts.push(
        `### ${engineNameV1(engine)} tables, with their properties:`
      );
    } else {
      prompts.push(`### ${engineNameV1(engine)} database`);
    }
  } else {
    if (databaseMetadata) {
      prompts.push(`### Giving a database`);
    }
  }
  if (databaseMetadata) {
    databaseMetadata.schemas.forEach((schema) => {
      schema.tables.forEach((table) => {
        const name = schema.name ? `${schema.name}.${table.name}` : table.name;
        const columns = table.columns.map((column) => column.name).join(", ");
        prompts.push(`# ${name}(${columns})`);
      });
    });
  }
  return prompts.join("\n");
};
