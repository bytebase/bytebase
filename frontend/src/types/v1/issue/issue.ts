import {
  ComposedProject,
  EMPTY_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
  emptyProject,
  unknownProject,
  emptyUser,
  unknownUser,
} from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { EMPTY_ID, UNKNOWN_ID } from "../../const";
import { Issue, Issue_Type, IssueStatus } from "../../proto/v1/issue_service";
import {
  Plan,
  PlanCheckRun,
  Rollout,
  TaskRun,
} from "../../proto/v1/rollout_service";
import {
  EMPTY_ROLLOUT_NAME,
  UNKNOWN_ROLLOUT_NAME,
  emptyRollout,
  unknownRollout,
} from "./rollout";

export interface ComposedIssue extends Issue {
  planEntity: Plan | undefined;
  planCheckRunList: PlanCheckRun[];
  rolloutEntity: Rollout;
  rolloutTaskRunList: TaskRun[];
  project: string;
  projectEntity: ComposedProject;
  assigneeEntity?: User;
  creatorEntity: User;
}

export const ESTABLISH_BASELINE_SQL =
  "/* Establish baseline using current schema. This SQL won't be applied to the database. */";

export const EMPTY_ISSUE_NAME = `projects/${EMPTY_ID}/issues/${EMPTY_ID}`;
export const UNKNOWN_ISSUE_NAME = `projects/${UNKNOWN_ID}/issues/${UNKNOWN_ID}`;

export const emptyIssue = (): ComposedIssue => {
  return {
    ...Issue.fromJSON({
      name: EMPTY_ISSUE_NAME,
      rollout: EMPTY_ROLLOUT_NAME,
      uid: String(EMPTY_ID),
      type: Issue_Type.DATABASE_CHANGE,
    }),
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: emptyRollout(),
    rolloutTaskRunList: [],
    project: EMPTY_PROJECT_NAME,
    projectEntity: emptyProject(),
    creatorEntity: emptyUser(),
  };
};

export const unknownIssue = (): ComposedIssue => {
  return {
    ...Issue.fromJSON({
      name: UNKNOWN_ISSUE_NAME,
      rollout: UNKNOWN_ROLLOUT_NAME,
      uid: String(UNKNOWN_ID),
      type: Issue_Type.DATABASE_CHANGE,
    }),
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: unknownRollout(),
    rolloutTaskRunList: [],
    project: UNKNOWN_PROJECT_NAME,
    projectEntity: unknownProject(),
    creatorEntity: unknownUser(),
  };
};

export interface IssueFilter {
  project: string;
  query: string;
  principal?: string;
  creator?: string;
  assignee?: string;
  subscriber?: string;
  statusList?: IssueStatus[];
}
