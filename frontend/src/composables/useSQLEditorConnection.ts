import { useRouter } from "vue-router";

import { DEFAULT_PROJECT_ID, Sheet, UNKNOWN_ID } from "../types";
import { connectionSlug } from "../utils";
import {
  useDatabaseStore,
  useTabStore,
  useSQLEditorStore,
  getDefaultConnectionContext,
} from "@/store";

const useSQLEditorConnection = () => {
  const router = useRouter();
  const tabStore = useTabStore();
  const sqlEditorStore = useSQLEditorStore();

  /**
   * Set the connection by tab info
   */
  const setConnectionContextFromCurrentTab = (sheet?: Sheet) => {
    const currentTab = tabStore.currentTab;

    if (sheet) {
      const { project, database, payload } = sheet;
      // If we are opening a sheet.
      // This only happens when we are landing on the page with `sheetId` in the URL.
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
