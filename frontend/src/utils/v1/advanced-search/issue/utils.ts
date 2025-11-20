import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
  useDatabaseV1Store,
  userNamePrefix,
} from "@/store";
import type { IssueFilter } from "@/types";
import { unknownDatabase } from "@/types";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { SearchParams } from "../common";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  type SearchScopeId,
} from "../common";

export const buildIssueFilterBySearchParams = (
  params: SearchParams,
  defaultFilter?: Partial<IssueFilter>
) => {
  const { query, scopes } = params;
  const projectScope = scopes.find((s) => s.id === "project");
  const taskTypeScope = scopes.find((s) => s.id === "taskType");
  const databaseScope = scopes.find((s) => s.id === "database");

  let database = "";
  if (databaseScope) {
    const db = useDatabaseV1Store().getDatabaseByName(databaseScope.value);
    if (db.name !== unknownDatabase().name) {
      database = db.name;
    }
  }

  const createdTsRange = getTsRangeFromSearchParams(params, "created");
  const labels = getValuesFromSearchParams(params, "issue-label");
  const approvalStatus = getValueFromSearchParams(params, "approval");

  const filter: IssueFilter = {
    ...defaultFilter,
    query,
    instance: getValueFromSearchParams(params, "instance", instanceNamePrefix),
    database,
    project: `${projectNamePrefix}${projectScope?.value ?? "-"}`,
    createdTsAfter: createdTsRange?.[0],
    createdTsBefore: createdTsRange?.[1],
    taskType: taskTypeScope?.value,
    creator: getValueFromSearchParams(params, "creator", userNamePrefix),
    currentApprover: getValueFromSearchParams(
      params,
      "current-approver",
      userNamePrefix
    ),
    approvalStatus: approvalStatus
      ? Issue_ApprovalStatus[
          approvalStatus as keyof typeof Issue_ApprovalStatus
        ]
      : undefined,
    releaser: getValueFromSearchParams(params, "releaser", userNamePrefix),
    statusList: getValuesFromSearchParams(params, "status").map(status => IssueStatus[status as keyof typeof IssueStatus]),
    labels,
    environment: getValueFromSearchParams(
      params,
      "environment",
      environmentNamePrefix
    ),
  };
  return filter;
};
