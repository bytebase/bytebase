import { DEFAULT_PROJECT_ID, UNKNOWN_ID } from "../types";
import {
  useTabStore,
  useSQLEditorStore,
  getDefaultConnectionContext,
  useSheetStore,
} from "@/store";

const useSQLEditorConnection = () => {
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
  };

  return {
    setConnectionContextFromCurrentTab,
  };
};

export { useSQLEditorConnection };
