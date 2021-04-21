import {
  Database,
  empty,
  EMPTY_ID,
  Environment,
  Pipeline,
  Step,
  StepStatus,
  Task,
  UNKNOWN_ID,
} from "../types";

export type PipelineType =
  | "NO_PIPELINE"
  | "SINGLE_TASK"
  | "MULTI_SINGLE_STEP_TASK"
  | "MULTI_TASK";

export function pipelineType(pipeline: Pipeline): PipelineType {
  if (pipeline.taskList.length == 0) {
    return "NO_PIPELINE";
  } else if (pipeline.taskList.length == 1) {
    return "SINGLE_TASK";
  } else {
    for (const task of pipeline.taskList) {
      if (task.stepList.length > 1) {
        return "MULTI_TASK";
      }
    }
    return "MULTI_SINGLE_STEP_TASK";
  }
}

// Returns all steps from all tasks.
export function allStepList(pipeline: Pipeline): Step[] {
  const list: Step[] = [];
  pipeline.taskList.forEach((task) => {
    task.stepList.forEach((step) => {
      list.push(step);
    });
  });
  return list;
}

export function activeStep(pipeline: Pipeline): Step {
  for (const task of pipeline.taskList) {
    for (const step of task.stepList) {
      if (
        step.status == "PENDING" ||
        step.status == "RUNNING" ||
        // "FAILED" is also a transient step status, which requires user
        // to take further action (e.g. Skip, Retry)
        step.status == "FAILED"
      ) {
        return step;
      }
    }
  }
  return empty("STEP") as Step;
}

export function activeEnvironment(pipeline: Pipeline): Environment {
  const task: Task = activeTask(pipeline);
  if (task.id == EMPTY_ID) {
    return empty("ENVIRONMENT") as Environment;
  }
  return task.environment;
}

export function activeDatabase(pipeline: Pipeline): Database {
  const task = activeTask(pipeline);
  if (task.id == EMPTY_ID) {
    return empty("DATABASE") as Database;
  }
  return task.database;
}

export type StepStatusTransitionType = "RUN" | "RETRY" | "CANCEL" | "SKIP";

export interface StepStatusTransition {
  type: StepStatusTransitionType;
  to: StepStatus;
  buttonName: string;
  buttonClass: string;
}

const STEP_STATUS_TRANSITION_LIST: Map<
  StepStatusTransitionType,
  StepStatusTransition
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
const APPLICABLE_STEP_TRANSITION_LIST: Map<
  StepStatus,
  StepStatusTransitionType[]
> = new Map([
  ["PENDING", ["RUN", "SKIP"]],
  ["RUNNING", ["CANCEL"]],
  ["DONE", []],
  ["FAILED", ["RETRY", "SKIP"]],
  ["SKIPPED", []],
]);

export function applicableStepTransition(
  pipeline: Pipeline
): StepStatusTransition[] {
  const step = activeStep(pipeline);

  if (step.id == EMPTY_ID || step.id == UNKNOWN_ID) {
    return [];
  }

  const list: StepStatusTransitionType[] = APPLICABLE_STEP_TRANSITION_LIST.get(
    step.status
  )!;

  return list.map((type: StepStatusTransitionType) => {
    return STEP_STATUS_TRANSITION_LIST.get(type)!;
  });
}

function activeTask(pipeline: Pipeline): Task {
  for (const task of pipeline.taskList) {
    for (const step of task.stepList) {
      if (
        step.status == "PENDING" ||
        step.status == "RUNNING" ||
        // "FAILED" is also a transient step status, which requires user
        // to take further action (e.g. Skip, Retry)
        step.status == "FAILED"
      ) {
        return task;
      }
    }
  }
  return empty("TASK") as Task;
}
