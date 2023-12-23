<template>
  <NDataTable
    :columns="columns"
    :data="viewList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="ts" setup>
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
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

const columns = computed(() => {
  const columns: (DataTableColumn<ViewMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.name"),
      render: (row) => {
        return getViewName(row.name);
      },
    },
    {
      key: "name",
      title: t("common.definition"),
      render: (row) => {
        return row.definition;
      },
    },
    {
      key: "name",
      title: t("common.comment"),
      render: (row) => {
        return row.comment;
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});

const getViewName = (viewName: string) => {
  if (hasSchemaProperty.value) {
    return `"${props.schemaName}"."${viewName}"`;
  }
  return viewName;
};
</script>
