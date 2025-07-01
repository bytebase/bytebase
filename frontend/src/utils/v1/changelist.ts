import { useSheetV1Store } from "@/store";
import type { Changelist_Change_Source } from "@/types";
import type { Changelist_Change as Change } from "@/types/proto-es/v1/changelist_service_pb";
import { getSheetStatement } from "./sheet";

export const extractChangelistResourceName = (name: string) => {
  const pattern = /(?:^|\/)changelists\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isChangelogChangeSource = (change: Change) => {
  return change.source.match(/(^|\/)changelogs\//);
};

export const getChangelistChangeSourceType = (
  change: Change
): Changelist_Change_Source => {
  if (isChangelogChangeSource(change)) {
    return "CHANGELOG";
  } else {
    return "RAW_SQL";
  }
};

export const generateSQLForChangeToDatabase = async (change: Change) => {
  const sheet = await useSheetV1Store().getOrFetchSheetByName(
    change.sheet,
    "FULL"
  );
  if (!sheet) {
    return "";
  }

  return getSheetStatement(sheet);
};
