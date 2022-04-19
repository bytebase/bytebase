import { useRouter } from "vue-router";

import { DEFAULT_PROJECT_ID, SheetState, UNKNOWN_ID } from "../types";
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
   * @param param
   * @param payload
   */
  const setConnectionContextFromCurrentTab = () => {
    const currentTab = tabStore.currentTab;
    const sheetById = sheetStore.sheetById as SheetState["sheetById"];

    if (currentTab.sheetId && sheetById.has(currentTab.sheetId)) {
      const sheet = sheetById.get(currentTab.sheetId);

      sqlEditorStore.setConnectionContext({
        hasSlug: true,
        projectId: sheet?.database?.projectId || DEFAULT_PROJECT_ID,
        instanceId: sheet?.database?.instanceId || UNKNOWN_ID,
        databaseId: sheet?.databaseId || UNKNOWN_ID,
      });

      // deal with the sheet is without databaseId
      if (sheet?.databaseId) {
        const database = useDatabaseStore().getDatabaseById(sheet?.databaseId);

        router.replace({
          name: "sql-editor.detail",
          params: {
            connectionSlug: connectionSlug(database),
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
