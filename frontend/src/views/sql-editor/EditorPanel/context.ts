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

  const viewState = computed(() => {
    if (!tab.value) return undefined;
    return viewStateByTab.get(tab.value.id) ?? defaultViewState();
  });

  const selectedSchemaName = computed({
    get() {
      return viewState.value?.schema;
    },
    set(schema) {
      if (schema === undefined) return;
      updateViewState({
        schema,
      });
    },
  });

  const updateViewState = (patch: Partial<ViewState>) => {
    const curr = viewState.value;
    if (!curr) return;
    if (!tab.value) return;

    Object.assign(curr, patch);
    viewStateByTab.set(tab.value.id, curr);
  };

  const context = {
    ...base,
    viewState,
    selectedSchemaName,
    updateViewState,
  };

  provide(KEY, context);

  return context;
};

export const useEditorPanelContext = () => {
  return inject(KEY)!;
};

export type EditorPanelContext = ReturnType<typeof provideEditorPanelContext>;
