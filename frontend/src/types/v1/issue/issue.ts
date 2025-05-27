import type { ComposedProject, ComposedTaskRun } from "@/types";
import {
  EMPTY_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
  emptyProject,
  unknownProject,
  EMPTY_ROLLOUT_NAME,
  UNKNOWN_ROLLOUT_NAME,
} from "@/types";
import type { Rollout } from "@/types//proto/v1/rollout_service";
import { Issue, IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";
import { EMPTY_ID, UNKNOWN_ID } from "../../const";

// For grant request issue, it has no plan and rollout.
// For sql review issue, it has no rollout.
export interface ComposedIssue extends Issue {
  planEntity: Plan | undefined;
  planCheckRunList: PlanCheckRun[];
  rolloutEntity: Rollout | undefined;
  rolloutTaskRunList: ComposedTaskRun[];
  project: string;
  projectEntity: ComposedProject;
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
    rolloutEntity: undefined,
    rolloutTaskRunList: [],
    project: EMPTY_PROJECT_NAME,
    projectEntity: emptyProject(),
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
    rolloutEntity: undefined,
    rolloutTaskRunList: [],
    project: UNKNOWN_PROJECT_NAME,
    projectEntity: unknownProject(),
  };
};

export interface IssueFilter {
  project: string;
  instance?: string;
  database?: string;
  query: string;
  creator?: string;
  subscriber?: string;
  statusList?: IssueStatus[];
  createdTsAfter?: number;
  createdTsBefore?: number;
  // type is the issue type, for example: GRANT_REQUEST, DATABASE_DATA_EXPORT
  type?: Issue_Type;
  // taskType is the task type, for example: DDL, DML
  taskType?: string;
  // filter by labels, for example: labels = "feature & bug"
  labels?: string[];
  hasPipeline?: boolean;
}
