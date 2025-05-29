<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="streamList"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="ts" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import type { PropType } from "vue";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import DefinitionView from "@/components/DefinitionView.vue";
import type { ComposedDatabase } from "@/types";
import type { StreamMetadata } from "@/types/proto/v1/database_service";
import {
  StreamMetadata_Mode,
  StreamMetadata_Type,
} from "@/types/proto/v1/database_service";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  streamList: {
    required: true,
    type: Object as PropType<StreamMetadata[]>,
  },
});

const { t } = useI18n();

const stringifyStreamType = (t: StreamMetadata_Type): string => {
  if (t === StreamMetadata_Type.TYPE_DELTA) {
    return "Delta";
  }
  return "-";
};

const stringifyStreamMode = (mode: StreamMetadata_Mode): string => {
  if (mode === StreamMetadata_Mode.MODE_APPEND_ONLY) {
    return "Append only";
  } else if (mode === StreamMetadata_Mode.MODE_INSERT_ONLY) {
    return "Insert only";
  } else if (mode === StreamMetadata_Mode.MODE_DEFAULT) {
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
