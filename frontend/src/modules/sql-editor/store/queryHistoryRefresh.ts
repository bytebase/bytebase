import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { getSQLEditorEditorState } from "./editor";
import { useSQLEditorStore } from "./index";

// Prepends the newest history of each database to the cache, keeping loaded
// pages, then tells the History pane to re-render.
export const refreshQueryHistory = async (databases: string[]) => {
  const project = getSQLEditorEditorState().project;
  await Promise.allSettled(
    databases.map((database) =>
      useSQLEditorStore.getState().mergeLatest({ project, database })
    )
  );
  await sqlEditorEvents.emit("query-history-updated");
};
