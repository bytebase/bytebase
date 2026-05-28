import { create } from "@bufbuild/protobuf";
import { isUndefined } from "lodash-es";
import { isValidProjectName } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetConnection } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { openWorksheetByName } from "@/views/sql-editor/Sheet";
import { getSQLEditorEditorState } from "./editor";
import { getSQLEditorTabsState } from "./tab";
import type { SQLEditorSliceCreator, WorksheetSaveSlice } from "./types";

// Pinia store barrel transitively re-imports `@/react/stores/sqlEditor`,
// so we still pull the Pinia bits in at call time via the helpers below
// to avoid a TDZ cycle. The new Zustand editor / tab stores are local
// to this module's directory and safe to import statically.
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
    const editorStore = getSQLEditorEditorState();
    const projectStore = useProjectV1Store();
    const projectIamPolicyStore = useProjectIamPolicyStore();

    editorStore.setProjectContextReady(false);
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
      getSQLEditorEditorState().setProjectContextReady(true);
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
    const tabStore = getSQLEditorTabsState();
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
    const editorStore = getSQLEditorEditorState();
    const tabStore = getSQLEditorTabsState();
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

// Re-export the Worksheet proto type so callers don't have to plumb the
// proto path themselves.
export type { Worksheet };
