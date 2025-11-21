import { useLocalStorage } from "@vueuse/core";
import Emittery from "emittery";
import { type IRange } from "monaco-editor";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";
import { useSQLEditorStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";

export type AsidePanelTab = "SCHEMA" | "WORKSHEET" | "HISTORY";

// 30% by default
export const storedAIPanelSize = useLocalStorage("bb.plugin.ai.panel-size", 30);

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
  AIPanelSize: Ref<number>;
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

  maybeSwitchProject: (project: string) => Promise<string>;
  handleAIPanelResize: (panes: { size: number }[], index?: number) => void;
};

export const KEY = Symbol(
  "bb.sql-editor.context"
) as InjectionKey<SQLEditorContext>;

export const useSQLEditorContext = () => {
  return inject(KEY)!;
};

export const provideSQLEditorContext = () => {
  const editorStore = useSQLEditorStore();
  const context: SQLEditorContext = {
    asidePanelTab: ref("WORKSHEET"),
    showConnectionPanel: ref(false),
    showAIPanel: ref(false),
    AIPanelSize: storedAIPanelSize,
    schemaViewer: ref(undefined),
    pendingInsertAtCaret: ref(),
    events: new Emittery(),

    maybeSwitchProject: async (project) => {
      if (editorStore.project !== project) {
        editorStore.project = project;
        await context.events.once("project-context-ready");
        return project;
      }
      return Promise.resolve(editorStore.project);
    },
    handleAIPanelResize: (panes, index = 0) => {
      try {
        if (!panes || !Array.isArray(panes) || panes.length <= index) return;
        const pane = panes[index];
        if (!pane || typeof pane.size !== "number") return;
        storedAIPanelSize.value = pane.size;
      } catch (error) {
        // Silently ignore errors from splitpanes during initialization
        console.debug("Splitpanes resize handler error (ignored):", error);
      }
    },
  };

  provide(KEY, context);

  return context;
};
