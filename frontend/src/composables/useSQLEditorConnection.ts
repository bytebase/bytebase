import { useRouter } from "vue-router";
import { useStore } from "vuex";

import { DEFAULT_PROJECT_ID, SheetState, UNKNOWN_ID } from "../types";
import { connectionSlug } from "../utils";
import { getDefaultConnectionContext } from "../store/modules/sqlEditor";
import { useTabStore } from "@/store/pinia/tab";

const useSQLEditorConnection = () => {
  const router = useRouter();
  const store = useStore();
  const tabStore = useTabStore();

  /**
   * Set the connection by tab info
   * @param param
   * @param payload
   */
  const setConnectionContextFromCurrentTab = () => {
    const currentTab = tabStore.currentTab;
    const sheetById = store.state.sheet.sheetById as SheetState["sheetById"];

    if (currentTab.sheetId && sheetById.has(currentTab.sheetId)) {
      const sheet = sheetById.get(currentTab.sheetId);

      store.dispatch("sqlEditor/setConnectionContext", {
        hasSlug: true,
        projectId: sheet?.database?.projectId || DEFAULT_PROJECT_ID,
        instanceId: sheet?.database?.instanceId || UNKNOWN_ID,
        databaseId: sheet?.databaseId || UNKNOWN_ID,
      });

      // deal with the sheet is without databaseId
      if (sheet?.databaseId) {
        const database = store.getters["database/databaseById"](
          sheet?.databaseId
        );

        router.replace({
          name: "sql-editor.detail",
          params: {
            connectionSlug: connectionSlug(database),
          },
        });
      }
    } else {
      store.dispatch(
        "sqlEditor/setConnectionContext",
        getDefaultConnectionContext()
      );

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
