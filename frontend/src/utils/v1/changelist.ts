import { Changelist_Change_Source } from "@/types";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";

export const extractChangelistResourceName = (name: string) => {
  const pattern = /(?:^|\/)changelists\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isChangeHistoryChangeSource = (change: Change) => {
  return change.source.match(/(^|\/)changeHistories\//);
};
export const isBranchChangeSource = (change: Change) => {
  return change.source.match(/(^|\/)schemaDesigns\//);
};

export const getChangelistChangeSourceType = (
  change: Change
): Changelist_Change_Source => {
  if (isChangeHistoryChangeSource(change)) {
    return "CHANGE_HISTORY";
  } else if (isBranchChangeSource(change)) {
    return "BRANCH";
  } else {
    return "RAW_SQL";
  }
};
