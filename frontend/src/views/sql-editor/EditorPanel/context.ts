import { inject, provide, ref, type InjectionKey, type Ref } from "vue";
import type { SQLEditorTab } from "@/types";
import type { EditorPanelView } from "./types";

const KEY = Symbol(
  "bb.sql-editor.editor-panel"
) as InjectionKey<EditorPanelContext>;

export const provideEditorPanelContext = (baseContext: {
  tab: Ref<SQLEditorTab | undefined>;
}) => {
  const view = ref<EditorPanelView>("CODE");
  const context = {
    ...baseContext,
    view,
  };

  provide(KEY, context);

  return context;
};

export const useEditorPanelContext = () => {
  return inject(KEY)!;
};

export type EditorPanelContext = ReturnType<typeof provideEditorPanelContext>;
