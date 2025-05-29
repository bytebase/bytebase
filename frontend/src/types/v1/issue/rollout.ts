import { EMPTY_ID, UNKNOWN_ID } from "@/types/const";
import {
  Stage,
  Task,
  Task_Type,
  TaskRun,
  TaskRunLog,
} from "@/types/proto/v1/rollout_service";
import { EMPTY_ROLLOUT_NAME, UNKNOWN_ROLLOUT_NAME } from "@/types/rollout";
import {
  EMPTY_ENVIRONMENT_NAME,
  UNKNOWN_ENVIRONMENT_NAME,
} from "../environment";

export const EMPTY_STAGE_NAME = `${EMPTY_ROLLOUT_NAME}/stages/${EMPTY_ID}`;
export const UNKNOWN_STAGE_NAME = `${UNKNOWN_ROLLOUT_NAME}/stages/${UNKNOWN_ID}`;

export const emptyStage = () => {
  return Stage.fromPartial({
    name: EMPTY_STAGE_NAME,
    environment: EMPTY_ENVIRONMENT_NAME,
  });
};
export const unknownStage = () => {
  return Stage.fromPartial({
    name: UNKNOWN_STAGE_NAME,
    environment: UNKNOWN_ENVIRONMENT_NAME,
  });
};

export const EMPTY_TASK_NAME = `${EMPTY_STAGE_NAME}/tasks/${EMPTY_ID}`;
export const UNKNOWN_TASK_NAME = `${UNKNOWN_STAGE_NAME}/tasks/${UNKNOWN_ID}`;
export const emptyTask = () => {
  return Task.fromPartial({
    name: EMPTY_TASK_NAME,
  });
};
export const unknownTask = () => {
  return Task.fromPartial({
    name: UNKNOWN_TASK_NAME,
  });
};

export const TaskTypeListWithStatement: Task_Type[] = [
  Task_Type.GENERAL,
  Task_Type.DATABASE_CREATE,
  Task_Type.DATABASE_DATA_UPDATE,
  Task_Type.DATABASE_SCHEMA_UPDATE,
  Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
  Task_Type.DATABASE_SCHEMA_UPDATE_GHOST,
  Task_Type.DATABASE_DATA_EXPORT,
];

export interface ComposedTaskRun extends TaskRun {
  taskRunLog: TaskRunLog;
}
