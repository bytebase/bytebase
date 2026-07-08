import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import type { TFunction } from "i18next";
import { extractUserEmail } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { formatAbsoluteDateTime } from "@/utils";

export const sortTaskRunsNewestFirst = (taskRuns: TaskRun[]): TaskRun[] =>
  [...taskRuns].sort((left, right) => {
    const leftTime = left.createTime ? Number(left.createTime.seconds) : 0;
    const rightTime = right.createTime ? Number(right.createTime.seconds) : 0;
    return rightTime - leftTime;
  });

export const executorEmailOfTaskRun = (taskRun: TaskRun): string =>
  extractUserEmail(taskRun.creator);

export const executionDurationOfTaskRun = (
  taskRun: TaskRun
): Duration | undefined => {
  const { startTime, updateTime } = taskRun;
  if (!startTime || !updateTime) {
    return undefined;
  }
  if (Number(startTime.seconds) === 0) {
    return undefined;
  }
  const elapsedMS =
    taskRun.status === TaskRun_Status.RUNNING
      ? Date.now() - getTimeForPbTimestampProtoEs(startTime)
      : getTimeForPbTimestampProtoEs(updateTime) -
        getTimeForPbTimestampProtoEs(startTime);
  return create(DurationSchema, {
    nanos: (elapsedMS % 1000) * 1e6,
    seconds: BigInt(Math.floor(elapsedMS / 1000)),
  });
};

// The one state line that matters for a run that hasn't produced output yet:
// enqueued, scheduled after a time, or held back by the parallel-task limit.
// Returns undefined when the run isn't waiting.
export const getTaskRunWaitingMessage = (
  taskRun: TaskRun,
  t: TFunction
): string | undefined => {
  if (taskRun.status === TaskRun_Status.PENDING) {
    const earliestAllowedTime = taskRun.runTime
      ? getTimeForPbTimestampProtoEs(taskRun.runTime)
      : null;
    if (earliestAllowedTime) {
      return t("task-run.status.enqueued-with-rollout-time", {
        time: formatAbsoluteDateTime(earliestAllowedTime),
      });
    }
    return t("task-run.status.enqueued");
  }
  if (taskRun.status === TaskRun_Status.RUNNING && taskRun.schedulerInfo) {
    const cause = taskRun.schedulerInfo.waitingCause;
    if (cause?.cause?.case === "parallelTasksLimit") {
      return t("task-run.status.waiting-max-tasks-per-rollout", {
        time: formatAbsoluteDateTime(
          getTimeForPbTimestampProtoEs(taskRun.schedulerInfo.reportTime)
        ),
      });
    }
  }
  return undefined;
};

export const getTaskRunComment = (taskRun: TaskRun, t: TFunction): string =>
  getTaskRunWaitingMessage(taskRun, t) ?? (taskRun.detail || "-");
