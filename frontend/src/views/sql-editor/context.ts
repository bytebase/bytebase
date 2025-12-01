import { useLocalStorage } from "@vueuse/core";
import Emittery from "emittery";
import { type IRange } from "monaco-editor";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useProjectV1Store, useSQLEditorStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { isValidProjectName } from "@/types";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";

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
};

export const KEY = Symbol(
  "bb.sql-editor.context"
) as InjectionKey<SQLEditorContext>;

export const useSQLEditorContext = () => {
  return inject(KEY)!;
};

export const provideSQLEditorContext = () => {
  const editorStore = useSQLEditorStore();
  const projectStore = useProjectV1Store();

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

  const context: SQLEditorContext = {
    asidePanelTab: ref("WORKSHEET"),
    showConnectionPanel: ref(false),
    showAIPanel,
    editorPanelSize,
    schemaViewer: ref(undefined),
    pendingInsertAtCaret: ref(),
    events: new Emittery(),

    maybeSwitchProject: async (projectName) => {
      if (!isValidProjectName(projectName)) {
        return;
      }
      editorStore.projectContextReady = false;
      try {
        const project = await projectStore.getOrFetchProjectByName(projectName);
        editorStore.setProject(project.name);
        context.events.emit("project-context-ready", { project: project.name });
        return project.name;
      } catch {
        // Nothing
      } finally {
        editorStore.projectContextReady = true;
      }
    },
    handleEditorPanelResize: (size: number) => {
      if (size >= 1) {
        return;
      }
      aiPanelSize.value = 1 - size;
    },
  };

  provide(KEY, context);

  return context;
};
