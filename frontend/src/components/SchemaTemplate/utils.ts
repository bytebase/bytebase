import { computed } from "vue";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";

export const engineList = [Engine.MYSQL, Engine.POSTGRES];

export const categoryList = computed(() => {
  const settingStore = useSettingV1Store();
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
  const fieldTemplateList =
    setting?.value?.schemaTemplateSettingValue?.fieldTemplates ?? [];
  const tableTemplateList =
    setting?.value?.schemaTemplateSettingValue?.tableTemplates ?? [];
  const resp = [];

  for (const category of new Set([
    ...fieldTemplateList.map((template) => template.category),
    ...tableTemplateList.map((template) => template.category),
  ])) {
    if (!category) {
      continue;
    }
    resp.push(category);
  }
  return resp;
});

export const classificationConfig = computed(() => {
  const settingStore = useSettingV1Store();
  // TODO(ed): it's a temporary solution
  return settingStore.classification[0];
});
