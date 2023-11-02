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
        <HumanizeDate :date="taskRun.startTime" />
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
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { Duration } from "@/types/proto/google/protobuf/duration";
import {
  Task,
  TaskRun,
  TaskRun_Status,
} from "@/types/proto/v1/rollout_service";
import TaskRunComment from "./TaskRunComment.vue";
import TaskRunStatusIcon from "./TaskRunStatusIcon.vue";

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
      title: t("task.created"),
      width: "auto",
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

const executionDurationOfTaskRun = (taskRun: TaskRun): Duration | undefined => {
  const { startTime, updateTime } = taskRun;
  if (!startTime || !updateTime) {
    return undefined;
  }
  if (startTime.getTime() === 0) {
    return undefined;
  }
  if (
    taskRun.status === TaskRun_Status.RUNNING &&
    taskRun.executionStatusUpdateTime
  ) {
    const elapsedMS = Date.now() - taskRun.executionStatusUpdateTime.getTime();
    return Duration.fromPartial({
      seconds: Math.floor(elapsedMS / 1000),
      nanos: (elapsedMS % 1000) * 1e6,
    });
  }
  const startMS = startTime.getTime();
  const updateMS = updateTime.getTime();
  const elapsedMS = updateMS - startMS;
  return Duration.fromPartial({
    seconds: Math.floor(elapsedMS / 1000),
    nanos: (elapsedMS % 1000) * 1e6,
  });
};
</script>
