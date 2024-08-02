import {
  computed,
  inject,
  provide,
  reactive,
  type InjectionKey,
  type Ref,
} from "vue";
import type { SQLEditorTab } from "@/types";
import {
  defaultViewState,
  type EditorPanelViewState as ViewState,
} from "./types";

const KEY = Symbol(
  "bb.sql-editor.editor-panel"
) as InjectionKey<EditorPanelContext>;

const viewStateByTab = reactive(new Map</* tab.id */ string, ViewState>());

export const provideEditorPanelContext = (base: {
  tab: Ref<SQLEditorTab | undefined>;
}) => {
  const { tab } = base;

  const viewState = computed<ViewState | undefined>({
    get() {
      if (!tab.value) return undefined;
      return viewStateByTab.get(tab.value.id) ?? defaultViewState();
    },
    set(vs) {
      if (!tab.value) return;
      if (!vs) return;
      viewStateByTab.set(tab.value.id, vs);
    },
  });

  const context = {
    ...base,
    viewState,
  };

  provide(KEY, context);

  return context;
};

export const useEditorPanelContext = () => {
  return inject(KEY)!;
};

export type EditorPanelContext = ReturnType<typeof provideEditorPanelContext>;
