import { useRouter } from "vue-router";

import { Sheet } from "../types";
import { connectionSlug } from "../utils";
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
