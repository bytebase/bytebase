import { watch } from "vue";
import type { Connection } from "@/types";
import { useCurrentTab } from "@/store";

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
