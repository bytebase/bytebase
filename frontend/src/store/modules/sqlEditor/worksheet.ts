import { create } from "@bufbuild/protobuf";
import { isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { nextTick, ref } from "vue";
import {
  useProjectIamPolicyStore,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLEditorUIStore,
  useWorkSheetStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import { isValidDatabaseName, isValidProjectName } from "@/types";
import {
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  extractWorksheetConnection,
  isSimilarDefaultSQLEditorTabTitle,
  NEW_WORKSHEET_TITLE,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { openWorksheetByName } from "@/views/sql-editor/Sheet";

export const useSQLEditorWorksheetStore = defineStore(
  "sqlEditorWorksheet",
  () => {
    const editorStore = useSQLEditorStore();
    const tabStore = useSQLEditorTabStore();
    const projectStore = useProjectV1Store();
    const projectIamPolicyStore = useProjectIamPolicyStore();
    const worksheetStore = useWorkSheetStore();
    const uiStore = useSQLEditorUIStore();

    const autoSaveController = ref<AbortController | null>(null);

    const abortAutoSave = () => {
      if (autoSaveController.value) {
        autoSaveController.value.abort();
        autoSaveController.value = null;
      }
    };

    const maybeSwitchProject = async (
      projectName: string
    ): Promise<string | undefined> => {
      editorStore.projectContextReady = false;
      try {
        if (!isValidProjectName(projectName)) {
          return;
        }
        const project = await projectStore.getOrFetchProjectByName(projectName);
        // Fetch IAM policy to ensure permission checks work correctly
        await projectIamPolicyStore.getOrFetchProjectIamPolicy(project.name);
        editorStore.setProject(project.name);
        await sqlEditorEvents.emit("project-context-ready", {
          project: project.name,
        });
        return project.name;
      } catch {
        // Nothing
      } finally {
        editorStore.projectContextReady = true;
      }
    };

    const maybeUpdateWorksheet = async ({
      tabId,
      worksheet,
      title,
      database,
      statement,
      folders,
      signal,
    }: {
      tabId: string;
      worksheet?: string;
      title?: string;
      database: string;
      statement: string;
      folders?: string[];
      signal?: AbortSignal;
    }): Promise<SQLEditorTab | undefined> => {
      const connection = await extractWorksheetConnection({ database });
      let worksheetTitle = title ?? "";
      if (isSimilarDefaultSQLEditorTabTitle(worksheetTitle)) {
        worksheetTitle = suggestedTabTitleForSQLEditorConnection(connection);
      }

      if (worksheet) {
        const currentSheet = worksheetStore.getWorksheetByName(worksheet);
        if (!currentSheet) {
          return;
        }
        if (!isSimilarDefaultSQLEditorTabTitle(currentSheet.title)) {
          worksheetTitle = currentSheet.title;
        }

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
    };

    const createWorksheet = async ({
      tabId,
      title,
      statement = "",
      folders = [],
      database = "",
    }: {
      tabId?: string;
      title?: string;
      statement?: string;
      folders?: string[];
      database?: string;
    }): Promise<SQLEditorTab | undefined> => {
      let worksheetTitle = title || NEW_WORKSHEET_TITLE;
      const connection = await extractWorksheetConnection({ database });
      if (!title && isValidDatabaseName(database)) {
        worksheetTitle = suggestedTabTitleForSQLEditorConnection(connection);
      }

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
      } else {
        const tab = await openWorksheetByName({
          worksheet: newWorksheet.name,
          forceNewTab: true,
        });
        nextTick(() => {
          if (tab && !tab.connection?.database) {
            uiStore.showConnectionPanel = true;
          }
        });
        return tab;
      }
    };

    return {
      autoSaveController,
      abortAutoSave,
      maybeSwitchProject,
      maybeUpdateWorksheet,
      createWorksheet,
    };
  }
);
