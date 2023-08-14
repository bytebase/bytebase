<template>
  <BBTable
    :column-list="columnList"
    :data-source="viewList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: view }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ getViewName(view.name) }}
      </BBTableCell>
      <BBTableCell class="w-64">
        {{ view.definition }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ view.comment }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedDatabase } from "@/types";
import { ViewMetadata } from "@/types/proto/store/database";
import { Engine } from "@/types/proto/v1/common";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  viewList: {
    required: true,
    type: Object as PropType<ViewMetadata[]>,
  },
});

const { t } = useI18n();

const engine = computed(() => props.database.instanceEntity.engine);

const hasSchemaProperty = computed(() => {
  return (
    engine.value === Engine.POSTGRES ||
    engine.value === Engine.SNOWFLAKE ||
    engine.value === Engine.RISINGWAVE
  );
});

const columnList = computed(() => [
  {
    title: t("common.name"),
  },
  {
    title: t("common.definition"),
  },
  {
    title: t("database.comment"),
  },
]);

const getViewName = (viewName: string) => {
  if (hasSchemaProperty.value) {
    return `"${props.schemaName}"."${viewName}"`;
  }
  return viewName;
};
</script>
