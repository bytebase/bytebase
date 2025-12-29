import { projectNamePrefix, userNamePrefix } from "@/store";
import type { IssueFilter } from "@/types";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { SearchParams } from "../common";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  getValuesFromSearchParams,
} from "../common";

export const buildIssueFilterBySearchParams = (
  params: SearchParams,
  defaultFilter?: Partial<IssueFilter>
) => {
  const { query } = params;
  const projectScope = getValueFromSearchParams(params, "project");

  const createdTsRange = getTsRangeFromSearchParams(params, "created");
  const labels = getValuesFromSearchParams(params, "issue-label");
  const approvalStatus = getValueFromSearchParams(params, "approval");

  const filter: IssueFilter = {
    ...defaultFilter,
    query,
    project: `${projectNamePrefix}${projectScope || "-"}`,
    createdTsAfter: createdTsRange?.[0],
    createdTsBefore: createdTsRange?.[1],
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
    statusList: getValuesFromSearchParams(params, "status").map(
      (status) => IssueStatus[status as keyof typeof IssueStatus]
    ),
    labels,
  };
  return filter;
};
