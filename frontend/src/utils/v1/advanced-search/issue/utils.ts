import { useDatabaseV1Store } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  userNamePrefix,
  projectNamePrefix,
} from "@/store";
import type { IssueFilter } from "@/types";
import { unknownDatabase } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { SearchParams, SemanticIssueStatus } from "../common";
import {
  type SearchScopeId,
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
} from "../common";

const getValuesFromSearchParams = (
  params: SearchParams,
  scopeId: SearchScopeId
) => {
  return params.scopes.filter((s) => s.id === scopeId).map((s) => s.value);
};

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
  const status = getSemanticIssueStatusFromSearchParams(params);
  const labels = getValuesFromSearchParams(params, "issue-label");

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
    statusList:
      status === "OPEN"
        ? [IssueStatus.OPEN]
        : status === "CLOSED"
          ? [IssueStatus.DONE, IssueStatus.CANCELED]
          : undefined,
    labels,
    environment: getValueFromSearchParams(
      params,
      "environment",
      environmentNamePrefix
    ),
  };
  return filter;
};

export const getSemanticIssueStatusFromSearchParams = (
  params: SearchParams
) => {
  const status = getValueFromSearchParams(
    params,
    "status",
    "" /* prefix='' */,
    ["OPEN", "CLOSED"]
  ) as SemanticIssueStatus | "";
  if (!status) return undefined;
  return status;
};
