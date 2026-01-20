<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="taskList"
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
  TaskMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { TaskMetadata_State } from "@/types/proto-es/v1/database_service_pb";

const props = withDefaults(
  defineProps<{
    database: Database;
    schemaName?: string;
    taskList: TaskMetadata[];
    loading?: boolean;
  }>(),
  {
    schemaName: "",
    loading: false,
  }
);

const { t } = useI18n();

const stringifyTaskState = (state: TaskMetadata_State): string => {
  if (state === TaskMetadata_State.STARTED) {
    return "Started";
  } else if (state === TaskMetadata_State.SUSPENDED) {
    return "Suspended";
  }
  return "-";
};

const columns = computed((): DataTableColumn<TaskMetadata>[] => {
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
      title: t("common.warehouse"),
      key: "warehouse",
      render: (row) => row.warehouse || "-",
    },
    {
      title: t("common.schedule"),
      key: "schedule",
      render: (row) => row.schedule || "-",
    },
    {
      title: t("common.state"),
      key: "state",
      render: (row) => stringifyTaskState(row.state),
    },
    {
      title: t("common.condition"),
      key: "condition",
      render: (row) => row.condition || "-",
    },
    {
      title: t("common.definition"),
      key: "definition",
      render: (row) => h(DefinitionView, { definition: row.definition }),
    },
  ];
});
</script>
