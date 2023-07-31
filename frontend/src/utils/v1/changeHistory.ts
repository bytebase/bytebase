import slug from "slug";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName } from "./database";

export const extractChangeHistoryUID = (name: string) => {
  const pattern = /(?:^|\/)(?:changeHistories|migrations)\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const changeHistorySlug = (uid: string, version: string): string => {
  return [slug(version), uid].join("-");
};

export const changeHistoryLink = (changeHistory: ChangeHistory): string => {
  const { name, uid, version } = changeHistory;
  const { instance, database } = extractDatabaseResourceName(name);
  const parent = `instances/${instance}/databases/${database}`;
  return `/${parent}/changeHistories/${changeHistorySlug(uid, version)}`;
};
