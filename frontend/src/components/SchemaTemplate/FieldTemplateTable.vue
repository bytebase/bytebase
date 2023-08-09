<template>
  <BBGrid
    :column-list="columnList"
    :data-source="templateList"
    :is-row-clickable="isRowClickable"
    class="border"
    @click-row="clickRow"
  >
    <template #item="{ item }: { item: SchemaTemplateSetting_FieldTemplate }">
      <div class="bb-grid-cell flex justify-start items-center">
        <EngineIcon :engine="item.engine" custom-class="ml-0 mr-1" />
        {{ item.column?.name }}
      </div>
      <div class="bb-grid-cell">
        {{ item.column?.type }}
      </div>
      <div class="bb-grid-cell">
        {{ getDefaultValue(item.column) }}
      </div>
      <div class="bb-grid-cell">
        {{ item.column?.comment }}
      </div>
      <div class="bb-grid-cell flex items-center justify-start gap-x-5">
        <button
          type="button"
          class="btn-normal flex justify-end !py-1 !px-3"
          @click.stop="$emit('view', item)"
        >
          {{ $t("common.view") }}
        </button>
        <BBButtonConfirm
          v-if="!readonly"
          :style="'DELETE'"
          :ok-text="$t('common.delete')"
          :confirm-title="$t('common.delete') + ` '${item.column?.name}'?`"
          :require-confirm="true"
          @confirm="() => deleteTemplate(item.id)"
        />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn } from "@/bbkit";
import { useSchemaEditorStore } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";
import { getDefaultValue } from "./utils";

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
const store = useSchemaEditorStore();

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: t("schema-template.form.column-name"),
      width: "auto",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.column-type"),
      width: "auto",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.default-value"),
      width: "auto",
      class: "capitalize",
    },
    {
      title: t("schema-template.form.comment"),
      width: "auto",
      class: "capitalize",
    },
    {
      title: t("common.operations"),
      width: "10rem",
      class: "capitalize",
    },
  ];
});

const clickRow = (template: SchemaTemplateSetting_FieldTemplate) => {
  emit("apply", template);
};

const isRowClickable = (template: SchemaTemplateSetting_FieldTemplate) => {
  return template.engine === props.engine;
};

const deleteTemplate = async (id: string) => {
  await store.deleteSchemaTemplate(id);
};
</script>
