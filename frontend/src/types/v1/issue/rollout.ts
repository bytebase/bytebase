import { EMPTY_ID, UNKNOWN_ID } from "@/types/const";
import {
  Rollout,
  Stage,
  Task,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import {
  EMPTY_ENVIRONMENT_NAME,
  UNKNOWN_ENVIRONMENT_NAME,
} from "../environment";

export const EMPTY_ROLLOUT_NAME = `projects/${EMPTY_ID}/rollouts/${EMPTY_ID}`;
export const UNKNOWN_ROLLOUT_NAME = `projects/${UNKNOWN_ID}/rollouts/${UNKNOWN_ID}`;
export const emptyRollout = () => {
  return Rollout.fromJSON({
    name: EMPTY_ROLLOUT_NAME,
    uid: String(EMPTY_ID),
  });
};
export const unknownRollout = () => {
  return Rollout.fromJSON({
    name: UNKNOWN_ROLLOUT_NAME,
    uid: String(UNKNOWN_ID),
  });
};

export const EMPTY_STAGE_NAME = `${EMPTY_ROLLOUT_NAME}/stages/${EMPTY_ID}`;
export const UNKNOWN_STAGE_NAME = `${UNKNOWN_ROLLOUT_NAME}/stages/${UNKNOWN_ID}`;
export const emptyStage = () => {
  return Stage.fromJSON({
    name: EMPTY_STAGE_NAME,
    uid: String(EMPTY_ID),
    environment: EMPTY_ENVIRONMENT_NAME,
    title: "",
  });
};
export const unknownStage = () => {
  return Stage.fromJSON({
    name: UNKNOWN_STAGE_NAME,
    uid: String(UNKNOWN_ID),
    environment: UNKNOWN_ENVIRONMENT_NAME,
    title: "<<Unknown stage>>",
  });
};

export const EMPTY_TASK_NAME = `${EMPTY_STAGE_NAME}/tasks/${EMPTY_ID}`;
export const UNKNOWN_TASK_NAME = `${UNKNOWN_STAGE_NAME}/tasks/${UNKNOWN_ID}`;
export const emptyTask = () => {
  return Task.fromJSON({
    name: EMPTY_TASK_NAME,
    uid: String(EMPTY_ID),
    title: "",
  });
};
export const unknownTask = () => {
  return Task.fromJSON({
    name: UNKNOWN_TASK_NAME,
    uid: String(UNKNOWN_ID),
    title: "<<Unknown task>>",
  });
};

export const TaskTypeV1WithStatement: Task_Type[] = [
  Task_Type.GENERAL,
  Task_Type.DATABASE_CREATE,
  Task_Type.DATABASE_DATA_UPDATE,
  Task_Type.DATABASE_SCHEMA_BASELINE,
  Task_Type.DATABASE_SCHEMA_UPDATE,
  Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
  Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
];
