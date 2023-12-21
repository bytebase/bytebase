<template>
  <NDataTable
    :columns="columns"
    :data="filteredData"
    :row-props="rowProps"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
  />

  <ExternalTableDetailDrawer
    :show="!!state.selectedTableName"
    :database-name="database.name"
    :schema-name="schemaName"
    :external-table-name="state.selectedTableName ?? ''"
    @dismiss="state.selectedTableName = undefined"
  />
</template>

<script lang="ts" setup>
import { DataTableColumn, NDataTable } from "naive-ui";
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { ComposedDatabase } from "@/types";
import { ExternalTableMetadata } from "@/types/proto/v1/database_service";

type LocalState = {
  selectedTableName?: string;
};

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  externalTableList: {
    required: true,
    type: Object as PropType<ExternalTableMetadata[]>,
    schemaName: "",
  },
  search: {
    type: String,
    default: "",
  },
});

const { t } = useI18n();
const state = reactive<LocalState>({});

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

const filteredData = computed(() => {
  return props.externalTableList.filter((row) => {
    return (
      row.name.toLowerCase().includes(props.search.toLowerCase()) ||
      row.externalServerName
        .toLowerCase()
        .includes(props.search.toLowerCase()) ||
      row.externalDatabaseName
        .toLowerCase()
        .includes(props.search.toLowerCase())
    );
  });
});

const rowProps = (row: ExternalTableMetadata) => {
  return {
    style: "cursor: pointer;",
    onClick: () => {
      state.selectedTableName = row.name;
    },
  };
};
</script>
