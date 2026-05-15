import { create } from "@bufbuild/protobuf";
import { isUndefined } from "lodash-es";
import { nextTick } from "vue";
import { isValidProjectName } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetConnection } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { openWorksheetByName } from "@/views/sql-editor/Sheet";
import type { SQLEditorSliceCreator, WorksheetSaveSlice } from "./types";

// Avoid static imports of Pinia stores (`useSQLEditorVueState`,
// `useSQLEditorTabStore`, `useProjectV1Store`, ...) because the Pinia
// store barrel transitively re-imports `@/react/stores/sqlEditor` —
// loading them at module evaluation creates a TDZ cycle. The slice
// pulls them in at call time via the helper below.
const importStores = () => import("@/store");
const importProjectIamPolicyStore = () =>
  import("@/store/modules/v1/projectIamPolicy");

export const createWorksheetSaveSlice: SQLEditorSliceCreator<
  WorksheetSaveSlice
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
    const { useProjectV1Store } = await importStores();
    const { useProjectIamPolicyStore } = await importProjectIamPolicyStore();
    const { useSQLEditorVueState } = await import("./editor-vue-state");
    const editorStore = useSQLEditorVueState();
    const projectStore = useProjectV1Store();
    const projectIamPolicyStore = useProjectIamPolicyStore();

    editorStore.projectContextReady = false;
    try {
      if (!isValidProjectName(projectName)) {
        return;
      }
      const project = await projectStore.getOrFetchProjectByName(projectName);
      // Fetch IAM policy to ensure permission checks work correctly.
      await projectIamPolicyStore.getOrFetchProjectIamPolicy(project.name);
      editorStore.setProject(project.name);
      await sqlEditorEvents.emit("project-context-ready", {
        project: project.name,
      });
      return project.name;
    } catch {
      // ignore
    } finally {
      editorStore.projectContextReady = true;
    }
  },

  maybeUpdateWorksheet: async ({
    tabId,
    worksheet,
    title,
    database,
    statement,
    folders,
    signal,
  }) => {
    const { useWorkSheetStore } = await importStores();
    const { useSQLEditorTabStore } = await import("./tab-vue-state");
    const tabStore = useSQLEditorTabStore();
    const worksheetStore = useWorkSheetStore();

    const connection = await extractWorksheetConnection({ database });

    // `title === undefined` means "don't change the title" — preserves
    // the current title on auto-save calls that never pass one.
    // `title === ""` is a real, explicit empty title that should be
    // persisted (renders as the Untitled placeholder elsewhere).
    const currentSheet = worksheet
      ? worksheetStore.getWorksheetByName(worksheet)
      : undefined;
    if (worksheet && !currentSheet) {
      return;
    }
    const worksheetTitle = title ?? currentSheet?.title ?? "";

    if (worksheet && currentSheet) {
      const updated = await worksheetStore.patchWorksheet(
        {
          ...currentSheet,
          title: worksheetTitle,
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
        await worksheetStore.upsertWorksheetOrganizer(
          {
            worksheet: updated.name,
            folders: folders,
          },
          ["folders"]
        );
      }
    }

    return tabStore.updateTab(tabId, {
      status: "CLEAN",
      connection,
      title: worksheetTitle,
      worksheet,
    });
  },

  createWorksheet: async ({
    tabId,
    title,
    statement = "",
    folders = [],
    database = "",
  }) => {
    const { useWorkSheetStore } = await importStores();
    const { useSQLEditorVueState } = await import("./editor-vue-state");
    const { useSQLEditorTabStore } = await import("./tab-vue-state");
    const editorStore = useSQLEditorVueState();
    const tabStore = useSQLEditorTabStore();
    const worksheetStore = useWorkSheetStore();

    const worksheetTitle = title ?? "";
    const connection = await extractWorksheetConnection({ database });

    const newWorksheet = await worksheetStore.createWorksheet(
      create(WorksheetSchema, {
        title: worksheetTitle,
        database,
        content: new TextEncoder().encode(statement),
        project: editorStore.project,
        visibility: Worksheet_Visibility.PRIVATE,
      })
    );

    if (folders.length > 0) {
      await worksheetStore.upsertWorksheetOrganizer(
        {
          worksheet: newWorksheet.name,
          folders: folders,
        },
        ["folders"]
      );
    }

    if (tabId) {
      return tabStore.updateTab(tabId, {
        status: "CLEAN",
        title: worksheetTitle,
        statement,
        connection,
        worksheet: newWorksheet.name,
      });
    }

    const tab = await openWorksheetByName({
      worksheet: newWorksheet.name,
      forceNewTab: true,
    });
    nextTick(() => {
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

// Re-export the Worksheet proto type so callers don't have to plumb the
// proto path themselves.
export type { Worksheet };
