<template>
  <NDataTable
    size="small"
    :row-key="rowKey"
    :columns="columnList"
    :data="taskRuns"
    :striped="true"
    :bordered="true"
  />

  <Drawer v-model:show="taskRunDetailContext.show">
    <DrawerContent
      :title="$t('common.detail')"
      style="width: calc(100vw - 14rem)"
    >
      <TaskRunDetail
        v-if="taskRunDetailContext.taskRun"
        :key="taskRunDetailContext.taskRun.name"
        :task-run="taskRunDetailContext.taskRun"
        :database="databaseForTask(project, task)"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable, NTag } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import { getDateForPbTimestamp, getTimeForPbTimestamp } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import type { Task, TaskRun } from "@/types/proto/v1/rollout_service";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import { databaseForTask } from "@/utils";
import { humanizeDurationV1 } from "@/utils";
import { convertDurationToNew } from "@/utils/v1/common-conversions";
import TaskRunComment from "./TaskRunComment.vue";

defineProps<{
  task: Task;
  taskRuns: TaskRun[];
}>();

const { t } = useI18n();
const { project } = useCurrentProjectV1();

const taskRunDetailContext = ref<{
  show: boolean;
  taskRun?: TaskRun;
}>({
  show: false,
});

const rowKey = (taskRun: TaskRun) => {
  return taskRun.name;
};

const columnList = computed((): DataTableColumn<TaskRun>[] => {
  return [
    {
      key: "status",
      title: t("common.status"),
      width: 100,
      render: (taskRun: TaskRun) => {
        const statusType = getStatusType(taskRun.status);
        const statusText = getStatusText(taskRun.status);
        return (
          <NTag type={statusType} size="small">
            {statusText}
          </NTag>
        );
      },
    },
    {
      key: "comment",
      title: t("common.comment"),
      render: (taskRun: TaskRun) => {
        return <TaskRunComment taskRun={taskRun} />;
      },
    },
    {
      key: "createTime",
      title: t("task.created"),
      width: 140,
      render: (taskRun: TaskRun) => (
        <HumanizeDate date={getDateForPbTimestamp(taskRun.createTime)} />
      ),
    },
    {
      key: "startTime",
      title: t("task.started"),
      width: 140,
      render: (taskRun: TaskRun) =>
        taskRun.startTime ? (
          <HumanizeDate date={getDateForPbTimestamp(taskRun.startTime)} />
        ) : (
          "-"
        ),
    },
    {
      key: "executionDuration",
      title: t("task.execution-time"),
      width: 120,
      render: (taskRun: TaskRun) => {
        const duration = executionDurationOfTaskRun(taskRun);
        return duration
          ? humanizeDurationV1(convertDurationToNew(duration))
          : "-";
      },
    },
    {
      key: "actions",
      title: "",
      width: 80,
      render: (taskRun: TaskRun) =>
        shouldShowDetailButton(taskRun) ? (
          <NButton size="tiny" onClick={() => showDetail(taskRun)}>
            {t("common.detail")}
          </NButton>
        ) : null,
    },
  ];
});

const getStatusType = (status: TaskRun_Status) => {
  switch (status) {
    case TaskRun_Status.DONE:
      return "success";
    case TaskRun_Status.FAILED:
    case TaskRun_Status.CANCELED:
      return "error";
    case TaskRun_Status.RUNNING:
      return "info";
    case TaskRun_Status.PENDING:
      return "warning";
    default:
      return "default";
  }
};

const getStatusText = (status: TaskRun_Status) => {
  return status.replace("_", " ");
};

const executionDurationOfTaskRun = (taskRun: TaskRun): Duration | undefined => {
  const { startTime, updateTime } = taskRun;
  if (!startTime || !updateTime) {
    return undefined;
  }
  if (startTime.seconds.toString() === "0") {
    return undefined;
  }
  if (taskRun.status === TaskRun_Status.RUNNING) {
    const elapsedMS = Date.now() - getTimeForPbTimestamp(startTime);
    return Duration.fromPartial({
      seconds: Math.floor(elapsedMS / 1000),
      nanos: (elapsedMS % 1000) * 1e6,
    });
  }
  const startMS = getTimeForPbTimestamp(startTime);
  const updateMS = getTimeForPbTimestamp(updateTime);
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
