import { Rollout, Task, Task_Status } from "@/types/proto/v1/rollout_service";

export const activeTaskOfRollout = (rollout: Rollout): Task => {
  for (const stage of rollout.stages) {
    for (const task of stage.tasks) {
      if (
        task.status === Task_Status.PENDING ||
        task.status === Task_Status.PENDING_APPROVAL ||
        task.status === Task_Status.RUNNING ||
        // "FAILED" is also a transient task status, which requires user
        // to take further action (e.g. Skip, Retry)
        task.status === Task_Status.FAILED ||
        task.status === Task_Status.CANCELED
      ) {
        return task;
      }
    }
  }

  return Task.fromJSON({});

  // const theLastTask = lastTask(pipeline);
  // if (theLastTask.id != EMPTY_ID) {
  //   return theLastTask;
  // }

  // return empty("TASK") as Task;
};
