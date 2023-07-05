import slug from "slug";
import { last } from "lodash-es";

import { EMPTY_TASK_NAME, emptyStage, emptyTask } from "@/types";
import {
  Rollout,
  Stage,
  Task,
  Task_Status,
} from "@/types/proto/v1/rollout_service";

export const stageV1Slug = (stage: Stage): string => {
  return [slug(stage.title), stage.uid].join("-");
};

export const taskV1Slug = (task: Task): string => {
  return [slug(task.title), task.uid].join("-");
};

export const activeTaskInStageV1 = (stage: Stage): Task => {
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

  return last(stage.tasks) ?? emptyTask();
};

export const activeTaskInRollout = (rollout: Rollout): Task => {
  for (const stage of rollout.stages) {
    const activeTask = activeTaskInStageV1(stage);
    if (activeTask && activeTask.name !== EMPTY_TASK_NAME) {
      return activeTask;
    }
  }

  return last(last(rollout.stages)?.tasks ?? []) ?? emptyTask();
};

export const activeStageInRollout = (rollout: Rollout): Stage => {
  const activeTask = activeTaskInRollout(rollout);
  if (activeTask.name !== EMPTY_TASK_NAME) {
    const stage = rollout.stages.find((stage) =>
      stage.tasks.findIndex((task) => task.name === activeTask.name)
    );
    if (stage) {
      return stage;
    }
  }
  return emptyStage();
};
