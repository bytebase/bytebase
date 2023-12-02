import { useChangeHistoryStore, useSheetV1Store } from "@/store";
import { Changelist_Change_Source, ComposedDatabase } from "@/types";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { getHistoryChangeType } from "./changeHistory";
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

export const guessChangelistChangeType = (
  change: Change
): "DML" | "DDL" | "-" => {
  const type = getChangelistChangeSourceType(change);
  if (type === "CHANGE_HISTORY") {
    const history = useChangeHistoryStore().getChangeHistoryByName(
      change.source
    );
    if (!history) {
      return "-";
    }
    return getHistoryChangeType(history.type);
  }
  if (type === "BRANCH") {
    return "DDL";
  }
  if (type === "RAW_SQL") {
    return "-";
  }

  console.error("Should never reach this line");
  return "-";
};

export const generateSQLForChangeToDatabase = async (
  change: Change,
  database: ComposedDatabase
) => {
  const sheet = await useSheetV1Store().fetchSheetByName(
    change.sheet,
    true /* raw */
  );
  if (!sheet) {
    return "";
  }

  return getSheetStatement(sheet);
};
