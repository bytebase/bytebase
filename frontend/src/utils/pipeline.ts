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

export function activeTask(pipeline: Pipeline): Task {
  for (const task of pipeline.taskList) {
    if (
      task.status == "PENDING" ||
      task.status == "RUNNING" ||
      // "FAILED" is also a transient task status, which requires user
      // to take further action (e.g. Cancel, Skip, Retry)
      task.status == "FAILED"
    ) {
      return task;
    }
  }
  return empty("TASK") as Task;
}

export function activeTaskIsRunning(pipeline: Pipeline): boolean {
  return activeTask(pipeline).status === "RUNNING";
}

export function activeStep(pipeline: Pipeline): Step {
  const task = activeTask(pipeline);
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
