import { useRouter } from "vue-router";
import { useStore } from "vuex";

import { SheetState } from "../types";
import { connectionSlug } from "../utils";
import { getDefaultConnectionContext } from "../store/modules/sqlEditor";

const useSQLEditorConnection = () => {
  const router = useRouter();
  const store = useStore();

  /**
   * Set the connection by tab info
   * @param param
   * @param payload
   */
  const setConnectionContextFromCurrentTab = () => {
    const currentTab = store.getters["tab/currentTab"];
    const sheetById = store.state.sheet.sheetById as SheetState["sheetById"];

    if (currentTab.sheetId && sheetById.has(currentTab.sheetId)) {
      const sheet = sheetById.get(currentTab.sheetId);

      const database = store.getters["database/databaseById"](
        sheet?.databaseId,
        sheet?.instanceId
      );

      store.dispatch("sqlEditor/setConnectionContext", {
        hasSlug: true,
        databaseId: sheet?.databaseId,
        instanceId: sheet?.instanceId,
      });

      router.replace({
        name: "sql-editor.detail",
        params: {
          connectionSlug: connectionSlug(database),
        },
      });
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
