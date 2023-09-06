import { Issue, IssueType, StageId } from "@/types";

export function stageName(issue: Issue, stageId: StageId): string {
  for (const stage of issue.pipeline.stageList) {
    if (stage.id == stageId) {
      return stage.name;
    }
  }
  return "<<Unknown stage>>";
}

export function isGrantRequestIssueType(issueType: IssueType): boolean {
  return issueType === "bb.issue.grant.request";
}

export function isDatabaseRelatedIssueType(issueType: IssueType): boolean {
  return [
    "bb.issue.database.general",
    "bb.issue.database.create",
    "bb.issue.database.grant",
    "bb.issue.database.schema.update",
    "bb.issue.database.data.update",
    "bb.issue.database.rollback",
    "bb.issue.database.schema.update.ghost",
    "bb.issue.database.restore.pitr",
  ].includes(issueType);
}
