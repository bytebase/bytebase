import { create as createProto } from "@bufbuild/protobuf";
import { getProjectNamePlanIdFromRolloutName } from "@/store/modules/v1/common";
import { EMPTY_ID, UNKNOWN_ID } from "./const";
import type { Rollout } from "./proto-es/v1/rollout_service_pb";
import { RolloutSchema } from "./proto-es/v1/rollout_service_pb";
import { EMPTY_PROJECT_NAME, UNKNOWN_PROJECT_NAME } from "./v1/project";

export const EMPTY_ROLLOUT_NAME = `${EMPTY_PROJECT_NAME}/plans/${EMPTY_ID}/rollout`;
export const UNKNOWN_ROLLOUT_NAME = `${UNKNOWN_PROJECT_NAME}/plans/${UNKNOWN_ID}/rollout`;

export const emptyRollout = (): Rollout => {
  return createProto(RolloutSchema, {
    name: EMPTY_ROLLOUT_NAME,
  });
};

export const unknownRollout = (): Rollout => {
  return createProto(RolloutSchema, {
    name: UNKNOWN_ROLLOUT_NAME,
  });
};

export const isValidRolloutName = (name: unknown): name is string => {
  if (typeof name !== "string") return false;
  const [projectName, rolloutName] = getProjectNamePlanIdFromRolloutName(name);
  return Boolean(
    projectName &&
      projectName !== String(UNKNOWN_ID) &&
      rolloutName &&
      rolloutName !== String(UNKNOWN_ID)
  );
};
