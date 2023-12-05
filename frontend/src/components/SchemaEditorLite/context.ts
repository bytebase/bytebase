import { InjectionKey, Ref, computed, inject, provide, ref } from "vue";
import { ComposedProject } from "@/types";
import { TabContext } from "@/types/v1/schemaEditor";
import { EditTarget, ResourceType } from "./types";

export type SchemaEditorContext = {
  resourceType: Ref<ResourceType>;
  readonly: Ref<boolean>;
  project: Ref<ComposedProject>;
  targets: Ref<EditTarget[]>;
  tabMap: Ref<Map<string, TabContext>>;
  currentTabId: Ref<string>;

  // computed
  currentTab: Ref<TabContext | undefined>;
};

export const KEY = Symbol(
  "bb.schema-editor"
) as InjectionKey<SchemaEditorContext>;

export const useSchemaEditorContext = () => {
  return inject(KEY)!;
};

export const provideSchemaEditorContext = (
  params: Pick<
    SchemaEditorContext,
    "targets" | "resourceType" | "readonly" | "project"
  >
) => {
  const tabMap = ref(new Map<string, TabContext>());
  const currentTabId = ref<string>("");
  const currentTab = computed(() => {
    if (!currentTabId.value) return undefined;
    return tabMap.value.get(currentTabId.value);
  });

  const context: SchemaEditorContext = {
    ...params,
    tabMap,
    currentTabId,
    currentTab,
  };

  provide(KEY, context);

  return context;
};
