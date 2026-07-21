import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import type { TFunction } from "i18next";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { formatAbsoluteDateTime, humanizeDurationV1 } from "@/utils";

// Millisecond resolution so two runs created in the same wall-clock second
// (e.g. a quick double rerun) still sort by their real order.
const createTimeMs = (taskRun: TaskRun): number =>
  taskRun.createTime ? getTimeForPbTimestampProtoEs(taskRun.createTime) : 0;

export const sortTaskRunsNewestFirst = (taskRuns: TaskRun[]): TaskRun[] =>
  [...taskRuns].sort((left, right) => createTimeMs(right) - createTimeMs(left));

export const executorEmailOfTaskRun = (taskRun: TaskRun): string =>
  extractUserEmail(taskRun.creator);

export const executionDurationOfTaskRun = (
  taskRun: TaskRun
): Duration | undefined => {
  const { startTime, updateTime, status } = taskRun;
  if (!startTime || Number(startTime.seconds) === 0) {
    return undefined;
  }
  const startMS = getTimeForPbTimestampProtoEs(startTime);
  // A running run counts up to now (no updateTime needed); a finished run uses
  // its last update. Clamp negatives to 0 so client/server clock skew can't
  // render a nonsensical negative duration.
  let elapsedMS: number;
  if (status === TaskRun_Status.RUNNING) {
    elapsedMS = Date.now() - startMS;
  } else {
    if (!updateTime) {
      return undefined;
    }
    elapsedMS = getTimeForPbTimestampProtoEs(updateTime) - startMS;
  }
  elapsedMS = Math.max(0, elapsedMS);
  return create(DurationSchema, {
    nanos: (elapsedMS % 1000) * 1e6,
    seconds: BigInt(Math.floor(elapsedMS / 1000)),
  });
};

// The canonical rendered duration for a task run: empty string when the run
// has no duration yet. One place so the card, expanded body, and history sheet
// stay identical.
export const formatTaskRunDuration = (taskRun: TaskRun): string => {
  const duration = executionDurationOfTaskRun(taskRun);
  return duration ? humanizeDurationV1(duration) : "";
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
