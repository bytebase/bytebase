<template>
  <BBGrid
    :column-list="columnList"
    :data-source="taskRunList"
    :row-clickable="false"
    row-key="uid"
    class="border"
    v-bind="$attrs"
  >
    <template #item="{ item: taskRun }: TaskRunGridRow">
      <div class="bb-grid-cell block">
        <TaskRunStatusIcon :status="taskRun.status" />
      </div>
      <div class="bb-grid-cell whitespace-pre-wrap break-words !block">
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
      <div class="bb-grid-cell">
        <NButton
          v-if="shouldShowDetailButton(taskRun)"
          size="tiny"
          @click="showDetail(taskRun)"
        >
          {{ $t("common.detail") }}
        </NButton>
      </div>
    </template>
  </BBGrid>

  <Drawer v-model:show="taskRunDetailContext.show">
    <DrawerContent
      :title="$t('common.detail')"
      style="width: calc(100vw - 8rem)"
    >
      <TaskRunLogTable
        v-if="taskRunDetailContext.taskRun"
        :key="taskRunDetailContext.taskRun.name"
        :task-run="taskRunDetailContext.taskRun"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { BBGridColumn, BBGridRow } from "@/bbkit";
import { BBGrid } from "@/bbkit";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { Duration } from "@/types/proto/google/protobuf/duration";
import type { TaskRun } from "@/types/proto/v1/rollout_service";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import TaskRunComment from "./TaskRunComment.vue";
import TaskRunLogTable from "./TaskRunLogTable";
import TaskRunStatusIcon from "./TaskRunStatusIcon.vue";

defineOptions({
  inheritAttrs: false,
});

export type TaskRunGridRow = BBGridRow<TaskRun>;

defineProps<{
  taskRunList: TaskRun[];
}>();

const { t } = useI18n();
const taskRunDetailContext = ref<{
  show: boolean;
  taskRun?: TaskRun;
}>({
  show: false,
});

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
    {
      title: "",
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
    taskRun.status === TaskRun_Status.RUNNING
  ) {
    const elapsedMS = Date.now() - startTime.getTime();
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

const shouldShowDetailButton = (taskRun: TaskRun) => {
  return [
    TaskRun_Status.RUNNING,
    TaskRun_Status.DONE,
    TaskRun_Status.FAILED,
    TaskRun_Status.CANCELED,
  ].includes(taskRun.status);
};

const showDetail = (taskRun: TaskRun) => {
  taskRunDetailContext.value = {
    show: true,
    taskRun,
  };
};
</script>
