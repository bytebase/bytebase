<template>
  <BBGrid
    :column-list="columnList"
    :data-source="taskRunList"
    :row-clickable="false"
    row-key="uid"
    class="border"
  >
    <template #item="{ item: taskRun }: TaskRunGridRow">
      <div class="bb-grid-cell">
        <TaskRunStatusIcon :status="taskRun.status" />
      </div>
      <div class="bb-grid-cell whitespace-pre-wrap break-words">
        <TaskRunComment :task-run="taskRun" />
      </div>
      <div class="bb-grid-cell">
        <HumanizeDate :date="taskRun.createTime" />
      </div>
      <div class="bb-grid-cell">
        {{ humanizeDurationV1(executionDurationOfTaskRun(taskRun)) }}
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { Task, TaskRun } from "@/types/proto/v1/rollout_service";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import TaskRunStatusIcon from "./TaskRunStatusIcon.vue";
import TaskRunComment from "./TaskRunComment.vue";

export type MergedTaskRunItem = {
  task: Task;
  taskRun: TaskRun;
};
export type TaskRunGridRow = BBGridRow<TaskRun>;

defineProps<{
  taskRunList: TaskRun[];
}>();

const { t } = useI18n();

const columnList = computed((): BBGridColumn[] => {
  return [
    {
      title: "",
      width: "auto",
    },
    {
      title: t("task.comment"),
      width: "1fr",
    },
    {
      title: t("task.started"),
      width: "auto",
    },
    {
      title: t("task.execution-time"),
      width: "auto",
    },
  ];
});

const executionDurationOfTaskRun = (taskRun: TaskRun): Duration => {
  const { createTime, updateTime } = taskRun;
  if (!createTime || !updateTime) return { seconds: 0, nanos: 0 };
  const createMS = createTime.getTime();
  const updateMS = updateTime.getTime();
  const elapsedMS = updateMS - createMS;
  return {
    seconds: Math.floor(elapsedMS / 1000),
    nanos: (elapsedMS % 1000) * 1e6,
  };
};
</script>
