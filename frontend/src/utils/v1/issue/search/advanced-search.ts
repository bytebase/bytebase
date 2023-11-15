import { useDatabaseV1Store } from "@/store";
import { IssueFilter, UNKNOWN_ID } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  SearchParams,
  SemanticIssueStatus,
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
} from "./common";

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
