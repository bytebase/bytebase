import { Ref, computed, inject, provide, ref } from "vue";
import { ComposedProject } from "@/types";
import { EditStatus, EditTarget, ResourceType, TabContext } from "./types";

export const KEY = Symbol("bb.schema-editor");

export const provideSchemaEditorContext = (params: {
  resourceType: Ref<ResourceType>;
  readonly: Ref<boolean>;
  project: Ref<ComposedProject>;
  targets: Ref<EditTarget[]>;
}) => {
  const tabMap = ref(new Map<string, TabContext>());
  const tabList = computed(() => {
    return Array.from(tabMap.value.values());
  });
  const currentTabId = ref<string>("");
  const currentTab = computed(() => {
    if (!currentTabId.value) return undefined;
    return tabMap.value.get(currentTabId.value);
  });
  const dirtyPaths = ref(new Map<string, EditStatus>());

  const context = {
    ...params,
    tabMap,
    tabList,
    currentTabId,
    currentTab,
    dirtyPaths,
  };

  provide(KEY, context);

  return context;
};

export type SchemaEditorContext = ReturnType<typeof provideSchemaEditorContext>;

export const useSchemaEditorContext = () => {
  return inject(KEY) as SchemaEditorContext;
};
