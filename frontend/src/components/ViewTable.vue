<template>
  <BBGrid
    :column-list="columnList"
    :data-source="viewList"
    :show-header="true"
    :row-clickable="false"
    class="border"
  >
    <template #item="{ item: view }: BBGridRow<ViewMetadata>">
      <div class="bb-grid-cell">
        {{ getViewName(view.name) }}
      </div>
      <div class="bb-grid-cell break-all">
        {{ view.definition }}
      </div>
      <div class="bb-grid-cell">
        {{ view.comment }}
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { ViewMetadata } from "@/types/proto/v1/database_service";

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

const columnList = computed((): BBGridColumn[] => [
  {
    title: t("common.name"),
    width: "minmax(auto, 1fr)",
  },
  {
    title: t("common.definition"),
    width: "3fr",
  },
  {
    title: t("database.comment"),
    width: "minmax(auto, 12rem)",
  },
]);

const getViewName = (viewName: string) => {
  if (hasSchemaProperty.value) {
    return `"${props.schemaName}"."${viewName}"`;
  }
  return viewName;
};
</script>
