import { ComposedIssue } from "@/types";

export const extractIssueId = (name: string) => {
  const pattern = /(?:^|\/)issues\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isGrantRequestIssue = (issue: ComposedIssue): boolean => {
  return false; // todo
  // return issueType === "bb.issue.grant.request";
};

export const isDatabaseRelatedIssue = (issue: ComposedIssue): boolean => {
  return false; // todo
  // return [
  //   "bb.issue.database.create",
  //   "bb.issue.database.grant",
  //   "bb.issue.database.schema.update",
  //   "bb.issue.database.data.update",
  //   "bb.issue.database.rollback",
  //   "bb.issue.database.schema.update.ghost",
  //   "bb.issue.database.restore.pitr",
  // ].includes(issueType);
};
