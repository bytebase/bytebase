import {
  Database,
  empty,
  EMPTY_ID,
  Environment,
  Pipeline,
  Task,
  TaskStatus,
  Stage,
  UNKNOWN_ID,
} from "../types";

export type PipelineType =
  | "NO_PIPELINE"
  | "SINGLE_STAGE"
  | "MULTI_SINGLE_TASK_STAGE"
  | "MULTI_STAGE";

export function pipelineType(pipeline: Pipeline): PipelineType {
  if (pipeline.stageList.length == 0) {
    return "NO_PIPELINE";
  } else if (pipeline.stageList.length == 1) {
    return "SINGLE_STAGE";
  } else {
    for (const stage of pipeline.stageList) {
      if (stage.taskList.length > 1) {
        return "MULTI_STAGE";
      }
    }
    return "MULTI_SINGLE_TASK_STAGE";
  }
}

// Returns all tasks from all stages.
export function allTaskList(pipeline: Pipeline): Task[] {
  const list: Task[] = [];
  pipeline.stageList.forEach((stage) => {
    stage.taskList.forEach((task) => {
      list.push(task);
    });
  });
  return list;
}

export function activeTask(pipeline: Pipeline): Task {
  for (const stage of pipeline.stageList) {
    for (const task of stage.taskList) {
      if (
        task.status == "PENDING" ||
        task.status == "RUNNING" ||
        // "FAILED" is also a transient task status, which requires user
        // to take further action (e.g. Skip, Retry)
        task.status == "FAILED"
      ) {
        return task;
      }
    }
  }
  return empty("TASK") as Task;
}

export function activeEnvironment(pipeline: Pipeline): Environment {
  const stage: Stage = activeStage(pipeline);
  if (stage.id == EMPTY_ID) {
    return empty("ENVIRONMENT") as Environment;
  }
  return stage.environment;
}

export function activeDatabase(pipeline: Pipeline): Database {
  const stage = activeStage(pipeline);
  if (stage.id == EMPTY_ID) {
    return empty("DATABASE") as Database;
  }
  return stage.database;
}

export type TaskStatusTransitionType = "RUN" | "RETRY" | "CANCEL" | "SKIP";

export interface TaskStatusTransition {
  type: TaskStatusTransitionType;
  to: TaskStatus;
  buttonName: string;
  buttonClass: string;
}

const TASK_STATUS_TRANSITION_LIST: Map<
  TaskStatusTransitionType,
  TaskStatusTransition
> = new Map([
  [
    "RUN",
    {
      type: "RUN",
      to: "RUNNING",
      buttonName: "Run",
      buttonClass: "btn-primary",
    },
  ],
  [
    "RETRY",
    {
      type: "RETRY",
      to: "RUNNING",
      buttonName: "Retry",
      buttonClass: "btn-primary",
    },
  ],
  [
    "CANCEL",
    {
      type: "CANCEL",
      to: "PENDING",
      buttonName: "Cancel",
      buttonClass: "btn-primary",
    },
  ],
  [
    "SKIP",
    {
      type: "SKIP",
      actionName: "Skip",
      to: "SKIPPED",
      buttonName: "Skip",
      buttonClass: "btn-normal",
    },
  ],
]);

// The transition button is ordered from right to left on the UI
const APPLICABLE_TASK_TRANSITION_LIST: Map<
  TaskStatus,
  TaskStatusTransitionType[]
> = new Map([
  ["PENDING", ["RUN", "SKIP"]],
  ["RUNNING", ["CANCEL"]],
  ["DONE", []],
  ["FAILED", ["RETRY", "SKIP"]],
  ["SKIPPED", []],
]);

export function applicableTaskTransition(
  pipeline: Pipeline
): TaskStatusTransition[] {
  const task = activeTask(pipeline);

  if (task.id == EMPTY_ID || task.id == UNKNOWN_ID) {
    return [];
  }

  const list: TaskStatusTransitionType[] = APPLICABLE_TASK_TRANSITION_LIST.get(
    task.status
  )!;

  return list.map((type: TaskStatusTransitionType) => {
    return TASK_STATUS_TRANSITION_LIST.get(type)!;
  });
}

function activeStage(pipeline: Pipeline): Stage {
  for (const stage of pipeline.stageList) {
    for (const task of stage.taskList) {
      if (
        task.status == "PENDING" ||
        task.status == "RUNNING" ||
        // "FAILED" is also a transient task status, which requires user
        // to take further action (e.g. Skip, Retry)
        task.status == "FAILED"
      ) {
        return stage;
      }
    }
  }
  return empty("STAGE") as Stage;
}
