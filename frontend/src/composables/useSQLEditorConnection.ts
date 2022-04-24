import { useRouter } from "vue-router";

import { DEFAULT_PROJECT_ID, Sheet, UNKNOWN_ID } from "../types";
import { connectionSlug } from "../utils";
import { getDefaultConnectionContext } from "@/store";
import {
  useDatabaseStore,
  useTabStore,
  useSQLEditorStore,
  useSheetStore,
} from "@/store";

const useSQLEditorConnection = () => {
  const router = useRouter();
  const tabStore = useTabStore();
  const sqlEditorStore = useSQLEditorStore();
  const sheetStore = useSheetStore();

  /**
   * Set the connection by tab info
   */
  const setConnectionContextFromCurrentTab = () => {
    const currentTab = tabStore.currentTab;
    const sheetById = sheetStore.sheetById;

    if (currentTab.sheetId && sheetById.has(currentTab.sheetId)) {
      const sheet = sheetById.get(currentTab.sheetId) as Sheet;

      sqlEditorStore.setConnectionContext({
        hasSlug: true,
        projectId: sheet.database?.projectId || DEFAULT_PROJECT_ID,
        instanceId: sheet.database?.instanceId || UNKNOWN_ID,
        databaseId: sheet.databaseId || UNKNOWN_ID,
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
      sqlEditorStore.setConnectionContext(getDefaultConnectionContext());

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
