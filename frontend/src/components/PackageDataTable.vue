<template>
  <NDataTable
    :columns="columns"
    :data="packageList"
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
import type { PackageMetadata } from "@/types/proto/v1/database_service";
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
  packageList: {
    required: true,
    type: Object as PropType<PackageMetadata[]>,
  },
});

const { t } = useI18n();

const engine = computed(() => props.database.instanceResource.engine);

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
