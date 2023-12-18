<template>
  <NDataTable
    :columns="columns"
    :data="dbExternalTableList"
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
import { ExternalTableMetadata } from "@/types/proto/v1/database_service";

defineProps({
  dbExternalTableList: {
    required: true,
    type: Object as PropType<ExternalTableMetadata[]>,
    schemaName: "",
  },
});

const { t } = useI18n();

const columns = computed(() => {
  return [
    {
      key: "name",
      title: t("common.name"),
      render: (row) => {
        return row.name;
      },
    },
    {
      key: "name",
      title: t("database.external-server-name"),
      render: (row) => {
        return row.externalServerName;
      },
    },
    {
      key: "name",
      title: t("database.external-database-name"),
      render: (row) => {
        return row.externalDatabaseName;
      },
    },
  ] as DataTableColumn<ExternalTableMetadata>[];
});
</script>
