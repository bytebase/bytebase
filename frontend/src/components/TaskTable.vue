<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="taskList"
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
import type { TaskMetadata } from "@/types/proto/v1/database_service";
import { TaskMetadata_State } from "@/types/proto/v1/database_service";

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  schemaName: {
    type: String,
    default: "",
  },
  taskList: {
    required: true,
    type: Object as PropType<TaskMetadata[]>,
  },
});

const { t } = useI18n();

const stringifyTaskState = (state: TaskMetadata_State): string => {
  if (state === TaskMetadata_State.STATE_STARTED) {
    return "Started";
  } else if (state === TaskMetadata_State.STATE_SUSPENDED) {
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
