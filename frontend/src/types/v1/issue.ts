import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { Issue } from "../proto/v1/issue_service";
import { Plan, Rollout } from "../proto/v1/rollout_service";
import {
  ComposedProject,
  EMPTY_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
  emptyProject,
  unknownProject,
} from "./project";

export interface ComposedIssue extends Issue {
  planEntity: Plan | undefined;
  rolloutEntity: Rollout;
  project: string;
  projectEntity: ComposedProject;
}

export const EMPTY_ISSUE_NAME = `projects/${EMPTY_ID}/issues/${EMPTY_ID}`;
export const EMPTY_ROLLOUT_NAME = `projects/${EMPTY_ID}/rollouts/${EMPTY_ID}`;
export const UNKNOWN_ISSUE_NAME = `projects/${UNKNOWN_ID}/issues/${UNKNOWN_ID}`;
export const UNKNOWN_ROLLOUT_NAME = `projects/${UNKNOWN_ID}/rollouts/${UNKNOWN_ID}`;

export const emptyIssue = (): ComposedIssue => {
  return {
    ...Issue.fromJSON({
      name: EMPTY_ISSUE_NAME,
      rollout: EMPTY_ROLLOUT_NAME,
    }),
    planEntity: undefined,
    rolloutEntity: Rollout.fromJSON({
      name: EMPTY_ROLLOUT_NAME,
    }),
    project: EMPTY_PROJECT_NAME,
    projectEntity: emptyProject(),
  };
};

export const unknownIssue = (): ComposedIssue => {
  return {
    ...Issue.fromJSON({
      name: UNKNOWN_ISSUE_NAME,
      rollout: UNKNOWN_ROLLOUT_NAME,
    }),
    planEntity: undefined,
    rolloutEntity: Rollout.fromJSON({
      name: UNKNOWN_ROLLOUT_NAME,
    }),
    project: UNKNOWN_PROJECT_NAME,
    projectEntity: unknownProject(),
  };
};
