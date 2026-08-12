import { create } from "@bufbuild/protobuf";
import { isUndefined } from "lodash-es";
import { extractSavedQueryConnection } from "@/lib/sqlEditorConnection";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";
import { openSavedQueryByName } from "@/modules/sql-editor/model/Sheet";
import { useAppStore } from "@/stores/app";
import { isValidProjectName } from "@/types";
import { SavedQuerySchema } from "@/types/proto-es/v1/saved_query_service_pb";
import { getSQLEditorEditorState } from "./editor";
import { getSQLEditorTabsState } from "./tab";
import type { SavedQuerySaveSlice, SQLEditorSliceCreator } from "./types";

export const createSavedQuerySaveSlice: SQLEditorSliceCreator<
  SavedQuerySaveSlice
> = (set, get) => ({
  autoSaveController: null,

  setAutoSaveController: (controller) =>
    set({ autoSaveController: controller }),

  abortAutoSave: () => {
    const controller = get().autoSaveController;
    if (controller) {
      controller.abort();
      set({ autoSaveController: null });
    }
  },

  maybeSwitchProject: async (projectName) => {
    const editorStore = getSQLEditorEditorState();

    editorStore.setProjectContextReady(false);
    try {
      if (!isValidProjectName(projectName)) {
        return;
      }
      const project = await useAppStore.getState().fetchProject(projectName);
      if (!project) {
        return;
      }
      // Fetch IAM policy so `hasProjectPermissionV2` sees the bindings. The
      // Pinia permission store falls back to `app.projectPoliciesByName` when
      // its own cache is empty, so populating the app `iam` slice is enough
      // (see `src/store/modules/v1/permission.ts`).
      await useAppStore
        .getState()
        .loadProjectIamPolicy(project.name)
        .catch(() => undefined);
      editorStore.setProject(project.name);
      await sqlEditorEvents.emit("project-context-ready", {
        project: project.name,
      });
      return project.name;
    } catch {
      // ignore
    } finally {
      getSQLEditorEditorState().setProjectContextReady(true);
    }
  },

  maybeUpdateSavedQuery: async ({
    tabId,
    savedQuery,
    title,
    database,
    statement,
    folders,
    signal,
  }) => {
    const tabStore = getSQLEditorTabsState();
    const savedQueryStore = useAppStore.getState();

    const connection = await extractSavedQueryConnection({ database });
    const currentTab = tabStore.tabsById.get(tabId);
    const nextConnection =
      currentTab?.connection.instance === connection.instance &&
      currentTab.connection.database === connection.database
        ? { ...currentTab.connection, ...connection }
        : connection;

    // `title === undefined` means "don't change the title" — preserves
    // the current title on auto-save calls that never pass one.
    // `title === ""` is a real, explicit empty title that should be
    // persisted (renders as the Untitled placeholder elsewhere).
    const currentSheet = savedQuery
      ? savedQueryStore.getSavedQueryByName(savedQuery)
      : undefined;
    if (savedQuery && !currentSheet) {
      return;
    }
    const savedQueryTitle = title ?? currentSheet?.title ?? "";

    if (savedQuery && currentSheet) {
      const updated = await savedQueryStore.patchSavedQuery(
        {
          ...currentSheet,
          title: savedQueryTitle,
          database,
          content: new TextEncoder().encode(statement),
        },
        ["title", "database", "content"],
        signal
      );
      if (!updated) {
        return;
      }
      if (!isUndefined(folders)) {
        await savedQueryStore.upsertSavedQueryOrganizer(
          {
            savedQuery: updated.name,
            folders: folders,
          },
          ["folders"]
        );
      }
    }

    return tabStore.updateTab(tabId, {
      status: "CLEAN",
      connection: nextConnection,
      title: savedQueryTitle,
      savedQuery,
    });
  },

  createSavedQuery: async ({
    tabId,
    title,
    statement = "",
    folders = [],
    database = "",
  }) => {
    const editorStore = getSQLEditorEditorState();
    const tabStore = getSQLEditorTabsState();
    const savedQueryStore = useAppStore.getState();

    const savedQueryTitle = title ?? "";
    const connection = await extractSavedQueryConnection({ database });

    const newSavedQuery = await savedQueryStore.createSavedQuery(
      create(SavedQuerySchema, {
        title: savedQueryTitle,
        database,
        content: new TextEncoder().encode(statement),
        project: editorStore.project,
      })
    );

    if (folders.length > 0) {
      await savedQueryStore.upsertSavedQueryOrganizer(
        {
          savedQuery: newSavedQuery.name,
          folders: folders,
        },
        ["folders"]
      );
    }

    if (tabId) {
      return tabStore.updateTab(tabId, {
        status: "CLEAN",
        title: savedQueryTitle,
        statement,
        connection,
        savedQuery: newSavedQuery.name,
      });
    }

    const tab = await openSavedQueryByName({
      savedQuery: newSavedQuery.name,
      forceNewTab: true,
    });
    queueMicrotask(() => {
      if (tab && !tab.connection?.database) {
        // The zustand store itself owns the UI-state slice, so we can
        // call the action directly through `get()` (avoids the
        // dynamic-import dance we use for cross-store calls).
        get().setShowConnectionPanel(true);
      }
    });
    return tab;
  },
});

// Re-export the SavedQuery proto type so callers don't have to plumb the
// proto path themselves.
export type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
