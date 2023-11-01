import dayjs from "dayjs";
import { Issue, IssueType, StageId } from "@/types";

export function stageName(issue: Issue, stageId: StageId): string {
  for (const stage of issue.pipeline?.stageList ?? []) {
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

export const generateIssueName = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update",
  databaseNameList: string[],
  isOnlineMode = false
) => {
  // Create a user friendly default issue name
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  if (isOnlineMode) {
    issueNameParts.push("Online schema change");
  } else {
    issueNameParts.push(
      type === "bb.issue.database.schema.update" ? `Edit schema` : `Change data`
    );
  }
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  return issueNameParts.join(" ");
};
