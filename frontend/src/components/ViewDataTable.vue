<template>
  <NDataTable
    :columns="columns"
    :data="viewList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :loading="loading"
    :row-key="(view: ViewMetadata) => `${database.name}.${schemaName}.${view.name}`"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  Database,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import EllipsisSQLView from "./EllipsisSQLView.vue";

const props = withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    viewList: ViewMetadata[];
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
        return <EllipsisSQLView sql={row.definition} />;
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
  if (hasSchemaProperty(engine.value) && props.schemaName) {
    return `"${props.schemaName}"."${viewName}"`;
  }
  return viewName;
};
</script>
