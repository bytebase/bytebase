<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="streamList"
    :striped="true"
    :bordered="true"
    :loading="loading"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import DefinitionView from "@/components/DefinitionView.vue";
import type {
  Database,
  StreamMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  StreamMetadata_Mode,
  StreamMetadata_Type,
} from "@/types/proto-es/v1/database_service_pb";

const props = withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    streamList: StreamMetadata[];
    loading?: boolean;
  }>(),
  {
    schemaName: "",
    loading: false,
  }
);

const { t } = useI18n();

const stringifyStreamType = (t: StreamMetadata_Type): string => {
  if (t === StreamMetadata_Type.DELTA) {
    return "Delta";
  }
  return "-";
};

const stringifyStreamMode = (mode: StreamMetadata_Mode): string => {
  if (mode === StreamMetadata_Mode.APPEND_ONLY) {
    return "Append only";
  } else if (mode === StreamMetadata_Mode.INSERT_ONLY) {
    return "Insert only";
  } else if (mode === StreamMetadata_Mode.MODE_UNSPECIFIED) {
    return "default";
  }
  return "-";
};

const columns = computed((): DataTableColumn<StreamMetadata>[] => {
  return [
    {
      title: t("common.schema"),
      key: "schema",
      render: () => props.schemaName,
    },
    {
      title: t("common.name"),
      key: "name",
      render: (row) => row.name || "-",
    },
    {
      title: t("common.table"),
      key: "tableName",
      render: (row) => row.tableName || "-",
    },
    {
      title: t("common.type"),
      key: "type",
      render: (row) => stringifyStreamType(row.type),
    },
    {
      title: t("common.mode"),
      key: "mode",
      render: (row) => stringifyStreamMode(row.mode),
    },
    {
      title: t("common.definition"),
      key: "definition",
      render: (row) => h(DefinitionView, { definition: row.definition }),
    },
  ];
});
</script>
