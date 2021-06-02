<template>
  <BBTable
    :columnList="columnList"
    :dataSource="taskRunList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="false"
  >
    <template v-slot:body="{ rowData: taskRun }">
      <BBTableCell :leftPadding="4" class="w-8">
        {{ taskRun.status }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(taskRun.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(taskRun.updatedTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { TaskRun } from "../types";

const columnList: BBTableColumn[] = [
  {
    title: "Status",
  },
  {
    title: "Started",
  },
  {
    title: "Ended",
  },
];

export default {
  name: "TaskRunTable",
  components: {},
  props: {
    taskRunList: {
      required: true,
      type: Object as PropType<TaskRun[]>,
    },
  },
  setup(props, ctx) {
    return {
      columnList,
    };
  },
};
</script>
