import { useSheetV1Store } from "@/store";
import type { Changelist_Change_Source, ComposedDatabase } from "@/types";
import type { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { getSheetStatement } from "./sheet";

export const extractChangelistResourceName = (name: string) => {
  const pattern = /(?:^|\/)changelists\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isChangeHistoryChangeSource = (change: Change) => {
  return change.source.match(/(^|\/)changeHistories\//);
};
export const isBranchChangeSource = (change: Change) => {
  return change.source.match(/(^|\/)branches\//);
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

export const generateSQLForChangeToDatabase = async (
  change: Change,
  _database: ComposedDatabase
) => {
  const sheet = await useSheetV1Store().getOrFetchSheetByName(
    change.sheet,
    "FULL"
  );
  if (!sheet) {
    return "";
  }

  return getSheetStatement(sheet);
};
