import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";

export interface IssueFilter {
  project: string;
  query: string;
  creator?: string;
  currentApprover?: string;
  approvalStatus?: Issue_ApprovalStatus;
  statusList?: IssueStatus[];
  riskLevelList?: RiskLevel[];
  createdTsAfter?: number;
  createdTsBefore?: number;
  // typeList is the issue types to filter by, for example: ROLE_GRANT, DATABASE_EXPORT
  typeList?: Issue_Type[];
  // filter by labels, for example: labels = "feature & bug"
  labels?: string[];
  // order by, for example: "create_time desc", "update_time asc"
  orderBy?: string;
}
