import { getProjectNameRolloutId } from "@/store/modules/v1/common";
import { EMPTY_ID, UNKNOWN_ID } from "./const";
import { Rollout } from "./proto/v1/rollout_service";
import type { User } from "./proto-es/v1/user_service_pb";
import { emptyUser, unknownUser } from "./v1";
import {
  EMPTY_PROJECT_NAME,
  emptyProject,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
  type ComposedProject,
} from "./v1/project";

export interface ComposedRollout extends Rollout {
  // Format: projects/{project}
  project: string;
  projectEntity: ComposedProject;
  creatorEntity: User;
}

export const EMPTY_ROLLOUT_NAME = `${EMPTY_PROJECT_NAME}/rollouts/${EMPTY_ID}`;
export const UNKNOWN_ROLLOUT_NAME = `${UNKNOWN_PROJECT_NAME}/rollouts/${UNKNOWN_ID}`;

export const emptyRollout = (): ComposedRollout => {
  const projectEntity = emptyProject();
  const rollout = Rollout.fromPartial({
    name: `${projectEntity.name}/rollouts/${EMPTY_ID}`,
  });
  return {
    ...rollout,
    project: projectEntity.name,
    projectEntity,
    creatorEntity: emptyUser(),
  };
};

export const unknownRollout = (): ComposedRollout => {
  const projectEntity = unknownProject();
  const rollout = Rollout.fromPartial({
    name: `${projectEntity.name}/rollouts/${UNKNOWN_ID}`,
  });
  return {
    ...rollout,
    project: projectEntity.name,
    projectEntity,
    creatorEntity: unknownUser(),
  };
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
