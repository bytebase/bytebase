import { getTimeForPbTimestampProtoEs } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

export type ActiveProjectIssue = {
  name: string;
  relation: "review" | "created";
  updatedAt: number;
  issue: Issue;
};

export function selectActiveProjectIssues({
  projectName,
  memberName,
  awaitingReview,
  createdIssues,
}: {
  projectName: string;
  memberName: string;
  awaitingReview: Issue[];
  createdIssues: Issue[];
}): ActiveProjectIssue[] {
  const issues = new Map<string, ActiveProjectIssue>();
  for (const issue of createdIssues) {
    if (
      issue.name.startsWith(`${projectName}/issues/`) &&
      issue.creator === memberName &&
      issue.status !== IssueStatus.DONE &&
      issue.status !== IssueStatus.CANCELED
    ) {
      issues.set(issue.name, {
        name: issue.name,
        relation: "created",
        updatedAt:
          getTimeForPbTimestampProtoEs(issue.updateTime ?? issue.createTime) ??
          0,
        issue,
      });
    }
  }
  for (const issue of awaitingReview) {
    if (
      issue.name.startsWith(`${projectName}/issues/`) &&
      issue.status !== IssueStatus.DONE &&
      issue.status !== IssueStatus.CANCELED
    ) {
      issues.set(issue.name, {
        name: issue.name,
        relation: "review",
        updatedAt:
          getTimeForPbTimestampProtoEs(issue.updateTime ?? issue.createTime) ??
          0,
        issue,
      });
    }
  }

  return [...issues.values()].sort((a, b) => b.updatedAt - a.updatedAt);
}
