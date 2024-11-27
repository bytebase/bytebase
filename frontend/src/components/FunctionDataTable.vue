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

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabase } from "@/types";
import type { FunctionMetadata } from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";
import EllipsisSQLView from "./EllipsisSQLView.vue";

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

const engine = computed(() => props.database.instanceResource.engine);

const columns = computed(() => {
  const columns: (DataTableColumn<FunctionMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.schema"),
      hide: !hasSchemaProperty(engine.value),
      ellipsis: {
        tooltip: true,
      },
      render: () => {
        return props.schemaName || t("db.schema.default");
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
      render: (row) => {
        return <EllipsisSQLView sql={row.definition} />;
      },
    },
  ];

  return columns;
});
</script>
