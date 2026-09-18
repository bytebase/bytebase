import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

/**
 * When a task is due to run, or nothing at all: a schedule belongs to a task
 * still waiting for it, and a task that has started or finished is described
 * by its run, not by the time it was once meant to begin.
 */
export function scheduledRunTimeMs(task: Task): number | undefined {
  return task.status === Task_Status.PENDING
    ? getTimeForPbTimestampProtoEs(task.runTime)
    : undefined;
}
