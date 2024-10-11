import { useLocalStorage } from "@vueuse/core";
import Emittery from "emittery";
import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";
import { useSQLEditorStore } from "@/store";
import type { ComposedDatabase, SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";

export type AsidePanelTab = "SCHEMA" | "WORKSHEET" | "HISTORY";

// 30% by default
export const storedAIPanelSize = useLocalStorage("bb.plugin.ai.panel-size", 30);

type SQLEditorEvents = Emittery<{
  "save-sheet": {
    tab: SQLEditorTab;
    editTitle?: boolean;
    mask?: Array<keyof Worksheet>;
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
  "set-editor-selection": {
    start: { line: number; column: number };
    end?: { line: number; column: number };
  };
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
        database: ComposedDatabase;
        schema?: string;
        table?: string;
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

    maybeSwitchProject: (project) => {
      if (editorStore.project !== project) {
        editorStore.project = project;
        return context.events.once("project-context-ready").then(() => project);
      }
      return Promise.resolve(editorStore.project);
    },
    handleAIPanelResize: (panes, index = 0) => {
      if (panes.length <= index) return;
      const { size } = panes[index];
      storedAIPanelSize.value = size;
    },
  };

  provide(KEY, context);

  return context;
};
