import { useDatabaseV1Store } from "@/store";
import { IssueFilter, UNKNOWN_ID } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  UIIssueFilter,
  UIIssueFilterScopeId,
  isValidIssueApprovalStatus,
} from "./ui-filter";

export type SemanticIssueStatus = "OPEN" | "CLOSED";

export type SearchScopeId =
  | "project"
  | "instance"
  | "database"
  | "type"
  | "creator"
  | "assignee"
  | "subscriber"
  | "status"
  | "created"
  | UIIssueFilterScopeId;

export type SearchScope = {
  id: SearchScopeId;
  value: string;
};

export interface SearchParams {
  query: string;
  scopes: SearchScope[];
}

export const buildIssueFilterBySearchParams = (params: SearchParams) => {
  const { query, scopes } = params;
  const projectScope = scopes.find((s) => s.id === "project");
  const typeScope = scopes.find((s) => s.id === "type");
  const databaseScope = scopes.find((s) => s.id === "database");

  let database = "";
  if (databaseScope) {
    const uid = databaseScope.value.split("-").slice(-1)[0];
    const db = useDatabaseV1Store().getDatabaseByUID(uid);
    if (db.uid !== `${UNKNOWN_ID}`) {
      database = db.name;
    }
  }

  const createdTsRange = getTsRangeFromSearchParams(params, "created");
  const status = getSemanticIssueStatusFromSearchParams(params);

  const filter: IssueFilter = {
    query,
    instance: getValueFromSearchParams(params, "instance", "instances/"),
    database,
    project: `projects/${projectScope?.value ?? "-"}`,
    createdTsAfter: createdTsRange?.[0],
    createdTsBefore: createdTsRange?.[1],
    type: typeScope?.value,
    creator: getValueFromSearchParams(params, "creator", "users/"),
    assignee: getValueFromSearchParams(params, "assignee", "users/"),
    subscriber: getValueFromSearchParams(params, "subscriber", "users/"),
    statusList:
      status === "OPEN"
        ? [IssueStatus.OPEN]
        : status === "CLOSED"
        ? [IssueStatus.DONE, IssueStatus.CANCELED]
        : undefined,
  };
  return filter;
};

export const buildUIIssueFilterBySearchParams = (params: SearchParams) => {
  const { scopes } = params;
  const approverScope = scopes.find((s) => s.id === "approver");
  const approvalScope = scopes.find((s) => s.id === "approval");
  const uiIssueFilter: UIIssueFilter = {};
  if (approverScope && approverScope.value) {
    uiIssueFilter.approver = `users/${approverScope.value}`;
  }
  if (approvalScope && isValidIssueApprovalStatus(approvalScope.value)) {
    uiIssueFilter.approval = approvalScope.value;
  }

  return uiIssueFilter;
};

export const getTsRangeFromSearchParams = (
  params: SearchParams,
  scopeId: SearchScopeId
) => {
  const scope = params.scopes.find((s) => s.id === scopeId);
  if (!scope) return undefined;
  const parts = scope.value.split(",");
  if (parts.length !== 2) return undefined;
  const range = [parseInt(parts[0], 10), parseInt(parts[1], 10)];
  return range;
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

export const getValueFromSearchParams = (
  params: SearchParams,
  scopeId: SearchScopeId,
  prefix: string = "",
  validValues: string[] = []
): string => {
  const scope = params.scopes.find((s) => s.id === scopeId);
  if (!scope) {
    return "";
  }
  const value = scope.value;
  if (validValues.length !== 0) {
    if (!validValues.includes(value)) {
      return "";
    }
  }
  return `${prefix}${scope.value}`;
};
