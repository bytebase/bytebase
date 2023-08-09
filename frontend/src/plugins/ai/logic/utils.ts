import { watch } from "vue";
import { useCurrentTab } from "@/store";
import type { Connection } from "@/types";
import { DatabaseMetadata } from "@/types/proto/store/database";
import { Engine } from "@/types/proto/v1/common";
import { engineNameV1 } from "@/utils";

export const onConnectionChanged = (
  fn: (newConn: Connection, oldConn: Connection | undefined) => void,
  immediate = false
) => {
  const tab = useCurrentTab();
  return watch(
    [
      () => tab.value.connection.instanceId,
      () => tab.value.connection.databaseId,
    ],
    (newValues, oldValues) => {
      fn(
        { instanceId: newValues[0], databaseId: newValues[1] },
        oldValues[0] && oldValues[1]
          ? { instanceId: oldValues[0], databaseId: oldValues[1] }
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
