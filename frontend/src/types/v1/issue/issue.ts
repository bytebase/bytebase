import type { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";

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
