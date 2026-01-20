<template>
  <NDataTable
    :columns="columns"
    :data="packageList"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
    :loading="loading"
    :row-key="(p: PackageMetadata) => `${database.name}.${schemaName}.${p.name}`"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type {
  Database,
  PackageMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, hasSchemaProperty } from "@/utils";
import EllipsisSQLView from "./EllipsisSQLView.vue";

const props = withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    packageList: PackageMetadata[];
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
  const columns: (DataTableColumn<PackageMetadata> & { hide?: boolean })[] = [
    {
      key: "name",
      title: t("common.name"),
      width: 240,
      resizable: true,
      minWidth: 120,
      render: (row) => {
        return getPackageName(row.name);
      },
    },
    {
      key: "definition",
      title: t("common.definition"),
      render: (row) => {
        return <EllipsisSQLView sql={row.definition} />;
      },
    },
  ];

  return columns.filter((column) => !column.hide);
});

const getPackageName = (packageName: string) => {
  if (hasSchemaProperty(engine.value) && props.schemaName) {
    return `"${props.schemaName}"."${packageName}"`;
  }
  return packageName;
};
</script>
