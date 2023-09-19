<template>
  <BBTable
    :column-list="columnList"
    :data-source="taskList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :row-clickable="true"
    :custom-footer="true"
  >
    <template #body="{ rowData: task }: { rowData: TaskMetadata }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ schemaName }}
      </BBTableCell>
      <BBTableCell>
        {{ task.name }}
      </BBTableCell>
      <BBTableCell>
        {{ task.warehouse }}
      </BBTableCell>
      <BBTableCell>
        {{ task.schedule || "-" }}
      </BBTableCell>
      <BBTableCell>
        {{ stringifyTaskState(task.state) }}
      </BBTableCell>
      <BBTableCell>
        {{ task.condition || "-" }}
      </BBTableCell>
      <BBTableCell>
        <DefinitionView :definition="task.definition" />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import DefinitionView from "@/components/DefinitionView.vue";
import { ComposedDatabase } from "@/types";
import {
  TaskMetadata,
  TaskMetadata_State,
} from "@/types/proto/v1/database_service";

defineProps({
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

const columnList = computed(() => {
  return [
    {
      title: t("common.schema"),
    },
    {
      title: t("common.name"),
    },
    {
      title: t("common.warehouse"),
    },
    {
      title: t("common.schedule"),
    },
    {
      title: t("common.state"),
    },
    {
      title: t("common.condition"),
    },
    {
      title: t("common.definition"),
    },
  ];
});

const stringifyTaskState = (state: TaskMetadata_State): string => {
  if (state === TaskMetadata_State.STATE_STARTED) {
    return "Started";
  } else if (state === TaskMetadata_State.STATE_SUSPENDED) {
    return "Suspended";
  }
  return "-";
};
</script>
