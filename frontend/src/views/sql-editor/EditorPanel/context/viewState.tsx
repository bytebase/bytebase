import { storeToRefs } from "pinia";
import { computed, inject, provide, watch, type InjectionKey } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
  useTabViewStateStore,
} from "@/store";
import {
  type EditorPanelView,
  type EditorPanelViewState as ViewState,
} from "@/types";
import {
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils";
import type { VNodeChild } from "vue";
import { t } from "@/plugins/i18n";
import {
  FunctionIcon,
  TableIcon,
  ViewIcon,
  ProcedureIcon,
  ExternalTableIcon,
  PackageIcon,
  SequenceIcon,
} from "@/components/Icon";
import { InfoIcon } from "lucide-vue-next"
import { SchemaDiagramIcon } from "@/components/SchemaDiagram";

const KEY = Symbol(
  "bb.sql-editor.editor-panel"
) as InjectionKey<EditorPanelContext>;

export const provideCurrentTabViewStateContext = () => {
  const tabStore = useSQLEditorTabStore();
  const tabViewStateStore = useTabViewStateStore();
  const { currentTab: tab } = storeToRefs(tabStore);

  const { instance } = useConnectionOfCurrentSQLEditorTab();

  const viewState = computed(() => {
    if (!tab.value) return undefined;
    return tabViewStateStore.getViewState(tab.value.id);
  });

  const availableActions = computed(() => {
    const actions: { view: EditorPanelView; title: string; icon: () => VNodeChild }[] = [
      {
        view: "INFO",
        title: t("common.info"),
        icon: () => <InfoIcon class="w-4 h-4" />
      },
      {
        view: "TABLES",
        title: t("db.tables"),
        icon: () => <TableIcon class="w-4 h-4" />
      },
      {
        view: "VIEWS",
        title: t("db.views"),
        icon: () => <ViewIcon class="w-4 h-4" />
      },
      {
        view: "FUNCTIONS",
        title: t("db.functions"),
        icon: () => <FunctionIcon class="w-4 h-4" />
      },
      {
        view: "PROCEDURES",
        title: t("db.procedures"),
        icon: () => <ProcedureIcon class="w-4 h-4" />
      },
    ];
    if (instanceV1SupportsSequence(instance.value)) {
      actions.push({
        view: "SEQUENCES",
        title: t("db.sequences"),
        icon: () => <SequenceIcon class="w-4 h-4" />
      });
    }
    if (instanceV1SupportsExternalTable(instance.value)) {
      actions.push({
        view: "EXTERNAL_TABLES",
        title: t("db.external-tables"),
        icon: () => <ExternalTableIcon class="w-4 h-4" />
      });
    }
    if (instanceV1SupportsPackage(instance.value)) {
      actions.push({
        view: "PACKAGES",
        title: t("db.packages"),
        icon: () => <PackageIcon class="w-4 h-4" />
      });
    }
    actions.push({
      view: "DIAGRAM",
      title: t("schema-diagram.self"),
      icon: () => <SchemaDiagramIcon class="w-4 h-4" />
    });
    return actions;
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
    if (!tab.value) return;
    tabViewStateStore.updateViewState(tab.value.id, patch);
  };

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
    viewState,
    selectedSchemaName,
    availableActions,
    updateViewState,
  };

  provide(KEY, context);

  return context;
};

export const useCurrentTabViewStateContext = () => {
  return inject(KEY)!;
};

export type EditorPanelContext = ReturnType<
  typeof provideCurrentTabViewStateContext
>;
