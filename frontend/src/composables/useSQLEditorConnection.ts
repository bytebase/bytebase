import { useRouter } from "vue-router";

import { DEFAULT_PROJECT_ID, UNKNOWN_ID } from "../types";
import { connectionSlug } from "../utils";
import {
  useDatabaseStore,
  useTabStore,
  useSQLEditorStore,
  getDefaultConnectionContext,
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
    const sheet = currentTab.sheetId
      ? sheetStore.sheetById.get(currentTab.sheetId)
      : undefined;

    if (sheet) {
      // Connect to the sheet connection if sheet exists.
      const { project, database, payload } = sheet;
      const projectId =
        project.id !== UNKNOWN_ID ? project.id : DEFAULT_PROJECT_ID;
      const instanceId =
        database?.id !== UNKNOWN_ID
          ? database?.instance.id
          : payload?.instanceId || UNKNOWN_ID;
      const databaseId = database?.id || UNKNOWN_ID;
      sqlEditorStore.setConnectionContext({
        projectId,
        instanceId,
        databaseId,
      });
    } else {
      // Connect to the tab connection otherwise.
      const { connectionContext } = currentTab;
      if (connectionContext) {
        sqlEditorStore.setConnectionContext({
          ...getDefaultConnectionContext(),
          ...connectionContext,
        });
      }
    }

    const routeArgs: any = {
      name: "sql-editor.home",
      params: {},
      query: {},
    };

    const database = useDatabaseStore().getDatabaseById(
      sqlEditorStore.connectionContext.databaseId
    );
    if (database && database.id !== UNKNOWN_ID) {
      routeArgs.name = "sql-editor.detail";
      routeArgs.params.connectionSlug = connectionSlug(database);
    }

    if (sheet) {
      routeArgs.query.sheetId = sheet.id;
    }

    router.replace(routeArgs);
  };

  return {
    setConnectionContextFromCurrentTab,
  };
};

export { useSQLEditorConnection };
