<template>
  <BBGrid
    :column-list="columnList"
    :data-source="templateList"
    :is-row-clickable="isRowClickable"
    class="border"
    @click-row="clickRow"
  >
    <template #item="{ item }: { item: SchemaTemplateSetting_FieldTemplate }">
      <div class="bb-grid-cell">
        {{ item.category || "-" }}
      </div>
      <div class="bb-grid-cell flex justify-start items-center">
        <EngineIcon :engine="item.engine" custom-class="ml-0 mr-1" />
        {{ item.column?.name }}
      </div>
      <div v-if="classificationConfig" class="bb-grid-cell flex gap-x-1">
        <ClassificationLevelBadge
          :classification="item.column?.classification"
          :classification-config="classificationConfig"
        />
      </div>
      <div class="bb-grid-cell">
        {{ item.column?.type }}
      </div>
      <div class="bb-grid-cell">
        {{ getColumnDefaultValuePlaceholder(item.column!) }}
      </div>
      <div class="bb-grid-cell">
        {{ item.column?.userComment }}
      </div>
      <div class="bb-grid-cell">
        <LabelsColumn :labels="item.config?.labels ?? {}" :show-count="2" />
      </div>
      <div class="bb-grid-cell flex items-center justify-start gap-x-2">
        <MiniActionButton @click.stop="$emit('view', item)">
          <PencilIcon class="w-4 h-4" />
        </MiniActionButton>
        <NPopconfirm v-if="!readonly" @positive-click="deleteTemplate(item.id)">
          <template #trigger>
            <MiniActionButton tag="div" @click.stop>
              <TrashIcon class="w-4 h-4" />
            </MiniActionButton>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("common.delete") + ` '${item.column?.name}'?` }}
          </div>
        </NPopconfirm>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { pullAt } from "lodash-es";
import { PencilIcon } from "lucide-vue-next";
import { TrashIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn } from "@/bbkit";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import { MiniActionButton } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_FieldTemplate,
} from "@/types/proto/v1/setting_service";
import { classificationConfig } from "./utils";

const props = defineProps<{
  engine?: Engine;
  readonly: boolean;
  templateList: SchemaTemplateSetting_FieldTemplate[];
}>();

const emit = defineEmits<{
  (event: "view", item: SchemaTemplateSetting_FieldTemplate): void;
  (event: "apply", item: SchemaTemplateSetting_FieldTemplate): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: t("schema-template.form.category"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.column-name"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("schema-template.classification.self"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
      hide: !classificationConfig.value,
    },
    {
      title: t("schema-template.form.column-type"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.default-value"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.comment"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("common.labels"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
    {
      title: t("common.operations"),
      width: "minmax(min-content, auto)",
      class: "capitalize",
    },
  ].filter((col) => !col.hide);
});

const clickRow = (template: SchemaTemplateSetting_FieldTemplate) => {
  emit("apply", template);
};

const isRowClickable = (template: SchemaTemplateSetting_FieldTemplate) => {
  return template.engine === props.engine;
};

const deleteTemplate = async (id: string) => {
  const setting = await settingStore.fetchSettingByName(
    "bb.workspace.schema-template"
  );

  const settingValue = SchemaTemplateSetting.fromJSON({});
  if (setting?.value?.schemaTemplateSettingValue) {
    Object.assign(settingValue, setting.value.schemaTemplateSettingValue);
  }

  const index = settingValue.fieldTemplates.findIndex((t) => t.id === id);
  if (index >= 0) {
    pullAt(settingValue.fieldTemplates, index);

    await settingStore.upsertSetting({
      name: "bb.workspace.schema-template",
      value: {
        schemaTemplateSettingValue: settingValue,
      },
    });
  }
};
</script>
