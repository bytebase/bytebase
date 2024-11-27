import { cloneDeep, head } from "lodash-es";
import {
  computed,
  inject,
  provide,
  reactive,
  watch,
  type InjectionKey,
  type Ref,
} from "vue";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import type { SQLEditorTab } from "@/types";
import {
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
  instanceV1SupportsTrigger,
} from "@/utils";
import {
  defaultViewState,
  typeToView,
  type EditorPanelView,
  type EditorPanelViewState as ViewState,
} from "../types";

const KEY = Symbol(
  "bb.sql-editor.editor-panel"
) as InjectionKey<EditorPanelContext>;

const viewStateByTab = reactive(new Map</* tab.id */ string, ViewState>());

export const provideEditorPanelContext = (base: {
  tab: Ref<SQLEditorTab | undefined>;
}) => {
  const { tab } = base;
  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const viewState = computed(() => {
    if (!tab.value) return undefined;
    return viewStateByTab.get(tab.value.id) ?? defaultViewState();
  });
  const availableViews = computed(() => {
    const views: EditorPanelView[] = [
      "CODE",
      "INFO",
      "TABLES",
      "VIEWS",
      "FUNCTIONS",
      "PROCEDURES",
    ];
    if (instanceV1SupportsSequence(instance.value)) {
      views.push("SEQUENCES");
    }
    if (instanceV1SupportsTrigger(instance.value)) {
      views.push("TRIGGERS");
    }
    if (instanceV1SupportsExternalTable(instance.value)) {
      views.push("EXTERNAL_TABLES");
    }
    if (instanceV1SupportsPackage(instance.value)) {
      views.push("PACKAGES");
    }
    views.push("DIAGRAM");
    return views;
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

  const cloneViewState = (from: string, to: string) => {
    const vs = viewStateByTab.get(from);
    if (!vs) return;
    const cloned = cloneDeep(vs);
    viewStateByTab.set(to, cloned);
    return cloned;
  };

  watch(
    [() => viewState.value?.view, availableViews],
    ([view, availableViews]) => {
      if (view && !availableViews.includes(view)) {
        updateViewState({
          view: head(availableViews),
        });
      }
    },
    { immediate: true }
  );

  watch(
    () => tab.value?.connection.schema,
    (schema) => {
      if (schema === undefined) return;
      if (!viewState.value) return;
      if (viewState.value.schema === schema) return;
      viewState.value.schema = schema;
    },
    { immediate: true }
  );
  watch(
    () => viewState.value?.schema,
    (schema) => {
      if (!schema) return;
      if (!tab.value) return;
      if (tab.value.connection.schema === undefined) return; // if schema chooser is "ALL", don't sync
      if (tab.value.connection.schema === schema) return;
      tab.value.connection.schema = schema;
    },
    { immediate: true }
  );

  const context = {
    ...base,
    viewState,
    selectedSchemaName,
    availableViews,
    updateViewState,
    cloneViewState,
    typeToView,
  };

  provide(KEY, context);

  return context;
};

export const useEditorPanelContext = () => {
  return inject(KEY)!;
};

export type EditorPanelContext = ReturnType<typeof provideEditorPanelContext>;
