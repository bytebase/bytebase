import { computed } from "vue";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { ColumnMetadata } from "@/types/proto/v1/database_service";

export const engineList = [Engine.MYSQL, Engine.POSTGRES];

export const getDefaultValue = (column: ColumnMetadata | undefined) => {
  if (column?.default) {
    return column.default;
  }
  if (column?.nullable) {
    return "NULL";
  }
  return "EMPTY";
};

export const caregoryList = computed(() => {
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
