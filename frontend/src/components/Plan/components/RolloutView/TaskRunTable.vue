<template>
  <NDataTable
    size="small"
    :row-key="rowKey"
    :columns="columnList"
    :data="taskRuns"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { getDateForPbTimestamp, getTimeForPbTimestamp } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import type { TaskRun } from "@/types/proto/v1/rollout_service";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import { humanizeDurationV1 } from "@/utils";
import { convertDurationToNew } from "@/utils/v1/common-conversions";

defineProps<{
  taskRuns: TaskRun[];
}>();

const { t } = useI18n();

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
      key: "updateTime",
      title: "Update Time",
      width: 140,
      render: (taskRun: TaskRun) =>
        taskRun.updateTime ? (
          <HumanizeDate date={getDateForPbTimestamp(taskRun.updateTime)} />
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
        return duration ? humanizeDurationV1(convertDurationToNew(duration)) : "-";
      },
    },
    {
      key: "executionSummary",
      title: "Summary",
      render: (taskRun: TaskRun) => {
        return getExecutionSummary(taskRun);
      },
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

const getExecutionSummary = (taskRun: TaskRun) => {
  if (taskRun.status === TaskRun_Status.DONE) {
    return "Completed successfully";
  }
  if (taskRun.status === TaskRun_Status.FAILED) {
    return "Failed";
  }
  if (taskRun.status === TaskRun_Status.CANCELED) {
    return "Canceled";
  }
  if (taskRun.status === TaskRun_Status.RUNNING) {
    return "Running...";
  }
  if (taskRun.status === TaskRun_Status.PENDING) {
    return "Pending";
  }
  return "Unknown";
};
</script>
