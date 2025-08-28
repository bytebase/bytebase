import { create as createProto } from "@bufbuild/protobuf";
import {
  EMPTY_PROJECT_NAME,
  EMPTY_ROLLOUT_NAME,
  UNKNOWN_PROJECT_NAME,
  UNKNOWN_ROLLOUT_NAME,
} from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueSchema,
  IssueStatus,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { EMPTY_ID, UNKNOWN_ID } from "../../const";

// For grant request issue, it has no plan and rollout.
// For sql review issue, it has no rollout.
export interface ComposedIssue extends Issue {
  planEntity: Plan | undefined;
  planCheckRunList: PlanCheckRun[];
  rolloutEntity: Rollout | undefined;
  rolloutTaskRunList: TaskRun[];
  project: string;
}

export const EMPTY_ISSUE_NAME = `projects/${EMPTY_ID}/issues/${EMPTY_ID}`;
export const UNKNOWN_ISSUE_NAME = `projects/${UNKNOWN_ID}/issues/${UNKNOWN_ID}`;

export const emptyIssue = (): ComposedIssue => {
  return {
    ...createProto(IssueSchema, {
      name: EMPTY_ISSUE_NAME,
      rollout: EMPTY_ROLLOUT_NAME,
      type: Issue_Type.DATABASE_CHANGE,
    }),
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: undefined,
    rolloutTaskRunList: [],
    project: EMPTY_PROJECT_NAME,
  };
};

export const unknownIssue = (): ComposedIssue => {
  return {
    ...createProto(IssueSchema, {
      name: UNKNOWN_ISSUE_NAME,
      rollout: UNKNOWN_ROLLOUT_NAME,
      type: Issue_Type.DATABASE_CHANGE,
    }),
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: undefined,
    rolloutTaskRunList: [],
    project: UNKNOWN_PROJECT_NAME,
  };
};

export interface IssueFilter {
  project: string;
  instance?: string;
  database?: string;
  query: string;
  creator?: string;
  statusList?: IssueStatus[];
  createdTsAfter?: number;
  createdTsBefore?: number;
  // type is the issue type, for example: GRANT_REQUEST, DATABASE_EXPORT
  type?: Issue_Type;
  // taskType is the task type, for example: DDL, DML
  taskType?: string;
  // filter by labels, for example: labels = "feature & bug"
  labels?: string[];
  hasPipeline?: boolean;
}
