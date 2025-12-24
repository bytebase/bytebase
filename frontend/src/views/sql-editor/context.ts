import { create } from "@bufbuild/protobuf";
import { useLocalStorage } from "@vueuse/core";
import Emittery from "emittery";
import { isUndefined } from "lodash-es";
import { type IRange } from "monaco-editor";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, nextTick, provide, ref } from "vue";
import {
  useProjectIamPolicyStore,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import type { SQLEditorTab } from "@/types";
import { isValidDatabaseName, isValidProjectName } from "@/types";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";
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
import { openWorksheetByName } from "@/views/sql-editor/Sheet";

export type AsidePanelTab = "SCHEMA" | "WORKSHEET" | "HISTORY";

const minimumEditorPanelSize = 0.5;

export type SQLEditorEvents = Emittery<{
  "save-sheet": {
    tab: SQLEditorTab;
    editTitle?: boolean;
  };
  "alter-schema": {
    // Format: instances/{instance}/databases/{database}
    databaseName: string;
    schema: string;
    table: string;
  };
  "format-content": undefined;
  "tree-ready": undefined;
  "project-context-ready": {
    project: string;
  };
  "set-editor-selection": IRange;
  "append-editor-content": { content: string; select: boolean };
  "insert-at-caret": {
    content: string;
  };
}>;

export type SQLEditorContext = {
  asidePanelTab: Ref<AsidePanelTab>;
  showConnectionPanel: Ref<boolean>;
  showAIPanel: Ref<boolean>;
  editorPanelSize: ComputedRef<{
    size: number;
    min: number;
    max: number;
  }>;
  schemaViewer: Ref<
    | {
        schema?: string;
        object?: string;
        type?: GetSchemaStringRequest_ObjectType;
      }
    | undefined
  >;

  pendingInsertAtCaret: Ref<string | undefined>;

  events: SQLEditorEvents;

  maybeSwitchProject: (project: string) => Promise<string | undefined>;
  handleEditorPanelResize: (size: number) => void;
  createWorksheet: (options: {
    tabId?: string;
    title?: string;
    folders?: string[];
    statement?: string;
    database?: string;
  }) => Promise<SQLEditorTab | undefined>;
  maybeUpdateWorksheet: (options: {
    tabId: string;
    worksheet: string;
    title?: string;
    database: string;
    statement: string;
    folders?: string[];
  }) => Promise<SQLEditorTab | undefined>;
};

export const KEY = Symbol(
  "bb.sql-editor.context"
) as InjectionKey<SQLEditorContext>;

export const useSQLEditorContext = () => {
  return inject(KEY)!;
};

export const provideSQLEditorContext = () => {
  const editorStore = useSQLEditorStore();
  const tabStore = useSQLEditorTabStore();
  const projectStore = useProjectV1Store();
  const projectIamPolicyStore = useProjectIamPolicyStore();
  const worksheetStore = useWorkSheetStore();
  const showConnectionPanel = ref(false);

  const aiPanelSize = useLocalStorage(
    "bb.plugin.editor.ai-panel-size",
    0.3 /* panel size should in [0.1, 1-minimumEditorPanelSize]*/
  );
  const showAIPanel = ref(false);
  const editorPanelSize = computed(() => {
    if (!showAIPanel.value) {
      return {
        size: 1,
        max: 1,
        min: 1,
      };
    }
    return {
      size: Math.max(1 - aiPanelSize.value, minimumEditorPanelSize),
      max: 0.9,
      min: minimumEditorPanelSize,
    };
  });

  const maybeUpdateWorksheet = async ({
    tabId,
    worksheet,
    title,
    database,
    statement,
    folders,
  }: {
    tabId: string;
    worksheet?: string;
    title?: string;
    database: string;
    statement: string;
    folders?: string[];
  }) => {
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
        ["title", "database", "content"]
      );
      if (updated && !isUndefined(folders)) {
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
  }) => {
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
          showConnectionPanel.value = true;
        }
      });
      return tab;
    }
  };

  const maybeSwitchProject = async (projectName: string) => {
    editorStore.projectContextReady = false;
    try {
      if (!isValidProjectName(projectName)) {
        return;
      }
      const project = await projectStore.getOrFetchProjectByName(projectName);
      // Fetch IAM policy to ensure permission checks work correctly
      await projectIamPolicyStore.getOrFetchProjectIamPolicy(project.name);
      editorStore.setProject(project.name);
      context.events.emit("project-context-ready", { project: project.name });
      return project.name;
    } catch {
      // Nothing
    } finally {
      editorStore.projectContextReady = true;
    }
  };

  const context: SQLEditorContext = {
    asidePanelTab: ref("WORKSHEET"),
    showConnectionPanel,
    showAIPanel,
    editorPanelSize,
    schemaViewer: ref(undefined),
    pendingInsertAtCaret: ref(),
    events: new Emittery(),

    maybeSwitchProject,
    handleEditorPanelResize: (size: number) => {
      if (size >= 1) {
        return;
      }
      aiPanelSize.value = 1 - size;
    },
    createWorksheet,
    maybeUpdateWorksheet,
  };

  provide(KEY, context);

  return context;
};
