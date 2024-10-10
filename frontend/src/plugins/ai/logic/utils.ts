import { watch } from "vue";
import { useCurrentSQLEditorTab } from "@/store";
import type { SQLEditorConnection } from "@/types";

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
