import { create as createProto } from "@bufbuild/protobuf";
import { EMPTY_PROJECT_NAME, UNKNOWN_PROJECT_NAME } from "@/types";
import type {
  Issue,
  Issue_ApprovalStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_Type,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
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
  query: string;
  creator?: string;
  currentApprover?: string;
  approvalStatus?: Issue_ApprovalStatus;
  statusList?: IssueStatus[];
  createdTsAfter?: number;
  createdTsBefore?: number;
  // type is the issue type, for example: GRANT_REQUEST, DATABASE_EXPORT
  type?: Issue_Type;
  // filter by labels, for example: labels = "feature & bug"
  labels?: string[];
}
