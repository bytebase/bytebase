import slug from "slug";
import { UNKNOWN_ID } from "@/types";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName } from "./database";
import { extractInstanceResourceName } from "./instance";

export const extractChangeHistoryUID = (changeHistorySlug: string) => {
  const parts = changeHistorySlug.split("-");
  return parts[parts.length - 1] ?? String(UNKNOWN_ID);
};

export const changeHistorySlug = (uid: string, version: string): string => {
  return [slug(version), uid].join("-");
};

export const changeHistoryLink = (changeHistory: ChangeHistory): string => {
  const { name, uid, version } = changeHistory;
  const { database } = extractDatabaseResourceName(name);
  const instance = extractInstanceResourceName(name);
  const parent = `instances/${instance}/databases/${database}`;
  return `/${parent}/changeHistories/${changeHistorySlug(uid, version)}`;
};
