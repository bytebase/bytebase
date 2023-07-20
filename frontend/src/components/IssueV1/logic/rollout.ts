import {
  Task,
  Task_Status,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";

export const isTaskEditable = (task: Task): [boolean, string] => {
  if (
    task.status === Task_Status.PENDING_APPROVAL ||
    task.status === Task_Status.FAILED
  ) {
    return [true, ""];
  }
  if (task.status === Task_Status.PENDING) {
    // If a task's status is "PENDING", its statement is editable if there
    // are at least ONE ERROR task checks.
    // Since once all its task checks are fulfilled, it might be queued by
    // the scheduler.
    // Editing a queued task's SQL statement is dangerous with kinds of race
    // condition risks.
    // TODO
    // const summary = taskCheckRunSummary(task);
    // if (summary.errorCount > 0) {
    //   return true;
    // }
  }

  return [false, `${task_StatusToJSON(task.status)} task is not editable`];
};

export const isTaskFinished = (task: Task): boolean => {
  return [
    Task_Status.DONE,
    Task_Status.FAILED,
    Task_Status.CANCELED,
    Task_Status.SKIPPED,
  ].includes(task.status);
};
