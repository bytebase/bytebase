import { projectNamePrefix, userNamePrefix } from "@/store";
import type { IssueFilter } from "@/types";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import {
  Issue_ApprovalStatus,
  Issue_Type,
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
    riskLevelList: getValuesFromSearchParams(params, "risk-level").map(
      (risk) => RiskLevel[risk as keyof typeof RiskLevel]
    ),
    typeList: getValuesFromSearchParams(params, "issue-type").map(
      (type) => Issue_Type[type as keyof typeof Issue_Type]
    ),
    labels,
  };
  return filter;
};
