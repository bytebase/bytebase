<template>
  <BBGrid
    :column-list="columnList"
    :data-source="templateList"
    :is-row-clickable="isRowClickable"
    class="border"
    @click-row="clickRow"
  >
    <template #item="{ item }: { item: SchemaTemplateSetting_TableTemplate }">
      <div class="bb-grid-cell">
        {{ item.category || "-" }}
      </div>
      <div class="bb-grid-cell flex justify-start items-center">
        <EngineIcon :engine="item.engine" custom-class="ml-0 mr-1" />
        {{ item.table?.name }}
      </div>
      <div v-if="classificationConfig" class="bb-grid-cell flex gap-x-1">
        <ClassificationLevelBadge
          :classification="item.table?.classification"
          :classification-config="classificationConfig"
        />
      </div>
      <div class="bb-grid-cell">
        {{ item.table?.userComment }}
      </div>
      <div class="bb-grid-cell flex items-center justify-start gap-x-2">
        <button
          class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
          @click.stop="$emit('view', item)"
        >
          <heroicons-outline:pencil class="w-4 h-4" />
        </button>
        <NPopconfirm v-if="!readonly" @positive-click="deleteTemplate(item.id)">
          <template #trigger>
            <button
              class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
              @click.stop=""
            >
              <heroicons-outline:trash class="w-4 h-4" />
            </button>
          </template>

          <div class="whitespace-nowrap">
            {{ $t("common.delete") + ` '${item.table?.name}'?` }}
          </div>
        </NPopconfirm>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { pullAt } from "lodash-es";
import { NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn } from "@/bbkit";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_TableTemplate,
} from "@/types/proto/v1/setting_service";
import { classificationConfig } from "./utils";

const props = defineProps<{
  engine?: Engine;
  readonly: boolean;
  templateList: SchemaTemplateSetting_TableTemplate[];
}>();

const emit = defineEmits<{
  (event: "view", item: SchemaTemplateSetting_TableTemplate): void;
  (event: "apply", item: SchemaTemplateSetting_TableTemplate): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: t("schema-template.form.category"),
      width: "15rem",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.table-name"),
      width: "15rem",
      class: "capitalize",
    },
    {
      title: t("schema-template.classification.self"),
      width: "auto",
      class: "capitalize",
      hide: !classificationConfig.value,
    },
    {
      title: t("schema-template.form.comment"),
      width: "auto",
      class: "capitalize",
    },
    {
      title: t("common.operations"),
      width: "5rem",
      class: "capitalize",
    },
  ].filter((col) => !col.hide);
});

const clickRow = (template: SchemaTemplateSetting_TableTemplate) => {
  emit("apply", template);
};

const isRowClickable = (template: SchemaTemplateSetting_TableTemplate) => {
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

  const index = settingValue.tableTemplates.findIndex((t) => t.id === id);
  if (index >= 0) {
    pullAt(settingValue.tableTemplates, index);

    await settingStore.upsertSetting({
      name: "bb.workspace.schema-template",
      value: {
        schemaTemplateSettingValue: settingValue,
      },
    });
  }
};
</script>
