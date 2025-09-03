import { UNKNOWN_ID } from "@/types";
import { extractDatabaseGroupName } from "@/utils";

export const isValidDatabaseGroupName = (name: string): boolean => {
  if (typeof name !== "string") return false;
  const dbGroupName = extractDatabaseGroupName(name);
  return Boolean(dbGroupName) && dbGroupName !== String(UNKNOWN_ID);
};
