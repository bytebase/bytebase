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
import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable, NTag } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import { humanizeDurationV1 } from "@/utils";
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
      key: "detail",
      title: t("common.detail"),
      render: (taskRun: TaskRun) => {
        return <TaskRunComment taskRun={taskRun} />;
      },
    },
    {
      key: "createTime",
      title: t("task.created"),
      width: 140,
      render: (taskRun: TaskRun) => (
        <HumanizeDate date={getDateForPbTimestampProtoEs(taskRun.createTime)} />
      ),
    },
    {
      key: "startTime",
      title: t("task.started"),
      width: 140,
      render: (taskRun: TaskRun) =>
        taskRun.startTime ? (
          <HumanizeDate
            date={getDateForPbTimestampProtoEs(taskRun.startTime)}
          />
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
        return duration ? humanizeDurationV1(duration) : "-";
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
  return TaskRun_Status[status].replace("_", " ");
};

const executionDurationOfTaskRun = (taskRun: TaskRun): Duration | undefined => {
  const { startTime, updateTime } = taskRun;
  if (!startTime || !updateTime) {
    return undefined;
  }
  if (Number(startTime.seconds) === 0) {
    return undefined;
  }
  if (taskRun.status === TaskRun_Status.RUNNING) {
    const elapsedMS = Date.now() - getTimeForPbTimestampProtoEs(startTime);
    return create(DurationSchema, {
      seconds: BigInt(Math.floor(elapsedMS / 1000)),
      nanos: (elapsedMS % 1000) * 1e6,
    });
  }
  const startMS = getTimeForPbTimestampProtoEs(startTime);
  const updateMS = getTimeForPbTimestampProtoEs(updateTime);
  const elapsedMS = updateMS - startMS;
  return create(DurationSchema, {
    seconds: BigInt(Math.floor(elapsedMS / 1000)),
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
