import { useRouter } from "vue-router";

import { DEFAULT_PROJECT_ID, Sheet, UNKNOWN_ID } from "../types";
import { connectionSlug, getDefaultConnection } from "../utils";
import { useDatabaseStore, useTabStore, useSheetStore } from "@/store";

const useSQLEditorConnection = () => {
  const router = useRouter();
  const tabStore = useTabStore();
  const sheetStore = useSheetStore();

  /**
   * Set the connection by tab info
   */
  const setConnectionContextFromCurrentTab = () => {
    const currentTab = tabStore.currentTab;
    const sheetById = sheetStore.sheetById;

    if (currentTab.sheetId && sheetById.has(currentTab.sheetId)) {
      const sheet = sheetById.get(currentTab.sheetId) as Sheet;

      tabStore.updateCurrentTab({
        connection: {
          ...getDefaultConnection(),
          projectId: sheet.database?.projectId || DEFAULT_PROJECT_ID,
          instanceId: sheet.database?.instanceId || UNKNOWN_ID,
          databaseId: sheet.databaseId || UNKNOWN_ID,
        },
      });

      if (sheet.databaseId) {
        const database = useDatabaseStore().getDatabaseById(sheet.databaseId);

        router.replace({
          name: "sql-editor.detail",
          params: {
            connectionSlug: connectionSlug(database),
          },
          query: {
            sheetId: sheet.id,
          },
        });
      }
    } else {
      tabStore.updateCurrentTab({
        connection: getDefaultConnection(),
      });

      router.push({
        path: "/sql-editor",
      });
    }
  };

  return {
    setConnectionContextFromCurrentTab,
  };
};

export { useSQLEditorConnection };
