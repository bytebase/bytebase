<template>
  <NDataTable
    :columns="columns"
    :data="functionList"
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
import { FunctionMetadata } from "@/types/proto/v1/database_service";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  functionList: {
    required: true,
    type: Object as PropType<FunctionMetadata[]>,
  },
});

const { t } = useI18n();

const engine = computed(() => props.database.instanceEntity.engine);

const isPostgres = computed(() => engine.value === Engine.POSTGRES);

const hasSchemaProperty = computed(() => {
  return (
    isPostgres.value ||
    engine.value === Engine.SNOWFLAKE ||
    engine.value === Engine.ORACLE ||
    engine.value === Engine.DM ||
    engine.value === Engine.MSSQL ||
    engine.value === Engine.RISINGWAVE
  );
});

const columns = computed(() => {
  const columns: (DataTableColumn<FunctionMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.schema"),
      hide: !hasSchemaProperty.value,
      ellipsis: {
        tooltip: true,
      },
      render: () => {
        return props.schemaName;
      },
    },
    {
      key: "name",
      title: t("common.name"),
      ellipsis: {
        tooltip: true,
      },
      render: (row) => {
        return row.name;
      },
    },
    {
      key: "name",
      title: t("common.definition"),
      ellipsis: {
        tooltip: true,
      },
      render: (row) => {
        return row.definition;
      },
    },
  ];

  return columns;
});
</script>
