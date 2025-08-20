import { create as createProto } from "@bufbuild/protobuf";
import { getProjectNameRolloutId } from "@/store/modules/v1/common";
import { EMPTY_ID, UNKNOWN_ID } from "./const";
import type { Rollout } from "./proto-es/v1/rollout_service_pb";
import { RolloutSchema } from "./proto-es/v1/rollout_service_pb";
import { EMPTY_PROJECT_NAME, UNKNOWN_PROJECT_NAME } from "./v1/project";

export const EMPTY_ROLLOUT_NAME = `${EMPTY_PROJECT_NAME}/rollouts/${EMPTY_ID}`;
export const UNKNOWN_ROLLOUT_NAME = `${UNKNOWN_PROJECT_NAME}/rollouts/${UNKNOWN_ID}`;

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

export const isValidRolloutName = (name: any): name is string => {
  if (typeof name !== "string") return false;
  const [projectName, rolloutName] = getProjectNameRolloutId(name);
  return Boolean(
    projectName &&
      projectName !== String(UNKNOWN_ID) &&
      rolloutName &&
      rolloutName !== String(UNKNOWN_ID)
  );
};
