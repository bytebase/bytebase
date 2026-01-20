<template>
  <NDataTable
    :columns="columns"
    :data="functionList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
    :loading="loading"
    :row-key="(f: FunctionMetadata) => `${database.name}.${schemaName}.${f.signature || f.name}`"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  Database,
  FunctionMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import EllipsisSQLView from "./EllipsisSQLView.vue";

const props = withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    functionList: FunctionMetadata[];
    loading?: boolean;
  }>(),
  {
    schemaName: "",
    loading: false,
  }
);

const { t } = useI18n();

const engine = computed(() => getInstanceResource(props.database).engine);

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
        return row.signature || row.name;
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
