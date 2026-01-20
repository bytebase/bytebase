<template>
  <NDataTable
    :columns="columns"
    :data="filteredData"
    :row-props="rowProps"
    :max-height="640"
    :virtual-scroll="true"
    :striped="true"
    :bordered="true"
    :loading="loading"
    :row-key="
      (ex: ExternalTableMetadata) => `${database.name}.${schemaName}.${ex.name}`
    "
  />

  <ExternalTableDetailDrawer
    :show="!!state.selectedTableName"
    :database-name="database.name"
    :schema-name="schemaName"
    :external-table-name="state.selectedTableName ?? ''"
    @dismiss="state.selectedTableName = undefined"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { HighlightLabelText } from "@/components/v2";
import type {
  Database,
  ExternalTableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import ExternalTableDetailDrawer from "./ExternalTableDetailDrawer.vue";

type LocalState = {
  selectedTableName?: string;
};

const props = withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    externalTableList: ExternalTableMetadata[];
    search?: string;
    loading?: boolean;
  }>(),
  {
    schemaName: "",
    search: "",
    loading: false,
  }
);

const { t } = useI18n();
const state = reactive<LocalState>({});

const columns = computed(() => {
  return [
    {
      key: "name",
      title: t("common.name"),
      render: (row) => {
        return <HighlightLabelText keyword={props.search} text={row.name} />;
      },
    },
    {
      key: "name",
      title: t("database.external-server-name"),
      render: (row) => {
        return (
          <HighlightLabelText
            keyword={props.search}
            text={row.externalServerName}
          />
        );
      },
    },
    {
      key: "name",
      title: t("database.external-database-name"),
      render: (row) => {
        return (
          <HighlightLabelText
            keyword={props.search}
            text={row.externalDatabaseName}
          />
        );
      },
    },
  ] as DataTableColumn<ExternalTableMetadata>[];
});

const filteredData = computed(() => {
  return props.externalTableList.filter((row) => {
    return (
      row.name.toLowerCase().includes(props.search) ||
      row.externalServerName.toLowerCase().includes(props.search) ||
      row.externalDatabaseName.toLowerCase().includes(props.search)
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
