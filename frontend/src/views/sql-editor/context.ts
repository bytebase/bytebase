import { create } from "@bufbuild/protobuf";
import { useLocalStorage, watchDebounced } from "@vueuse/core";
import Emittery from "emittery";
import { isUndefined } from "lodash-es";
import { type IRange } from "monaco-editor";
import { storeToRefs } from "pinia";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, nextTick, provide, ref } from "vue";
import {
  pushNotification,
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
  isWorksheetWritableV1,
  NEW_WORKSHEET_TITLE,
  STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
  suggestedTabTitleForSQLEditorConnection,
} from "@/utils";
import { openWorksheetByName } from "@/views/sql-editor/Sheet";

export const ASIDE_PANEL_TABS = [
  "SCHEMA",
  "WORKSHEET",
  "HISTORY",
  "ACCESS",
] as const;
export type AsidePanelTab = (typeof ASIDE_PANEL_TABS)[number];

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

  // The resource name of a newly created access grant to highlight in the list.
  highlightAccessGrantName: Ref<string | undefined>;

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
    signal?: AbortSignal;
  }) => Promise<SQLEditorTab | undefined>;
  // Abort any in-progress auto-save (used when manual save takes priority)
  abortAutoSave: () => void;
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
    STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
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
    signal,
  }: {
    tabId: string;
    worksheet?: string;
    title?: string;
    database: string;
    statement: string;
    folders?: string[];
    signal?: AbortSignal;
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

  // Auto-save abort controller - allows manual save to cancel in-progress auto-save
  let autoSaveController: AbortController | null = null;

  const abortAutoSave = () => {
    if (autoSaveController) {
      autoSaveController.abort();
      autoSaveController = null;
    }
  };

  const context: SQLEditorContext = {
    asidePanelTab: ref("WORKSHEET"),
    showConnectionPanel,
    showAIPanel,
    editorPanelSize,
    schemaViewer: ref(undefined),
    pendingInsertAtCaret: ref(),
    highlightAccessGrantName: ref<string | undefined>(),
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
    abortAutoSave,
  };

  // Auto-saving for current tab.
  // This handles several complex scenarios:
  //
  // Scenario 1: Normal auto-save
  //   1. User types "statement1", status → DIRTY
  //   2. After 2s debounce, auto-save starts, status → SAVING
  //   3. Auto-save completes, status → CLEAN
  //
  // Scenario 2: User types during auto-save
  //   1. User types "statement1", status → DIRTY
  //   2. Auto-save #1 starts for "statement1", status → SAVING
  //   3. User types more → "statement1 statement2", status → DIRTY
  //   4. Auto-save #1 completes, status → CLEAN
  //   5. Finally block: statement changed, status → DIRTY
  //   6. After 2s debounce, auto-save #2 starts for "statement1 statement2"
  //
  // Scenario 3: Manual save during auto-save
  //   1. User types "statement1", status → DIRTY
  //   2. Auto-save #1 starts, status → SAVING
  //   3. User clicks save → abortAutoSave() called
  //   4. Auto-save #1 aborted (throws AbortError), wasAborted = true
  //   5. Manual save runs and completes, status → CLEAN
  //   6. Auto-save #1's finally: wasAborted = true, skips logic
  //
  // Scenario 4: Rapid typing (newer auto-save aborts older)
  //   1. User types "statement1", status → DIRTY
  //   2. Auto-save #1 starts (slow network), status → SAVING
  //   3. User types "statement1 statement2", status → DIRTY
  //   4. After 2s debounce, abortAutoSave() aborts #1, auto-save #2 starts
  //   5. Auto-save #1's finally: wasAborted = true, skips logic
  //   6. Auto-save #2 completes with correct content
  //
  // Scenario 5: Auto-save fails (network/server error)
  //   1. User types "statement1", status → DIRTY
  //   2. Auto-save starts, status → SAVING
  //   3. Network error occurs
  //   4. Catch block: status → DIRTY, error notification shown
  //   5. User can manually save or wait for next auto-save attempt
  //
  // Scenario 6: Tab closed during auto-save
  //   1. User types "statement1", status → DIRTY
  //   2. Auto-save starts, status → SAVING
  //   3. User closes tab (warning shown since status !== CLEAN)
  //   4. User confirms, tab removed from store
  //   5. Auto-save completes, finally block runs
  //   6. getTabById returns undefined, updateTab is no-op (safe)
  //
  // Guard conditions (auto-save skipped):
  //   - No worksheet (draft tab): user must manually save to create worksheet
  //   - Read-only worksheet: user doesn't have write permission
  //   - Status is CLEAN: nothing to save
  //
  const { currentTab } = storeToRefs(tabStore);

  watchDebounced(
    () => currentTab.value?.statement,
    async () => {
      const tab = currentTab.value;
      if (!tab || !tab.worksheet || tab.status === "CLEAN") {
        return;
      }

      // Check write permission
      const worksheet = worksheetStore.getWorksheetByName(tab.worksheet);
      if (!worksheet || !isWorksheetWritableV1(worksheet)) {
        return;
      }

      // Abort any in-progress auto-save before starting a new one
      abortAutoSave();

      // Capture the statement and tab id before async operation
      const statementToSave = tab.statement;
      const tabId = tab.id;

      // Create new abort controller for this auto-save
      autoSaveController = new AbortController();
      // Set status to SAVING
      tabStore.updateTab(tabId, { status: "SAVING" });

      // Track if this auto-save was aborted
      let wasAborted = false;

      try {
        // the maybeUpdateWorksheet will set status to CLEAN
        await maybeUpdateWorksheet({
          tabId,
          worksheet: tab.worksheet,
          database: tab.connection.database,
          statement: statementToSave,
          signal: autoSaveController.signal,
        });
      } catch (error) {
        // Don't handle aborted requests - a newer save or manual save took priority
        if (error instanceof Error && error.name === "AbortError") {
          wasAborted = true;
          return;
        }
        // Only revert to DIRTY if still in SAVING state
        // (user might have manually saved successfully during this time)
        if (tabStore.getTabById(tabId)?.status === "SAVING") {
          tabStore.updateTab(tabId, { status: "DIRTY" });
        }
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Auto-save failed",
          description: error instanceof Error ? error.message : "Unknown error",
        });
      } finally {
        autoSaveController = null;

        // Skip for aborted requests - manual save or newer auto-save took priority
        if (wasAborted) {
          return;
        }

        // After save, check if the statement changed while we were saving
        // If it did, the user typed more content, so keep status as DIRTY
        const currentStatement = tabStore.getTabById(tabId)?.statement;
        if (currentStatement !== statementToSave) {
          tabStore.updateTab(tabId, { status: "DIRTY" });
        }
      }
    },
    { debounce: 2000 /* 2 seconds */ }
  );

  provide(KEY, context);

  return context;
};
