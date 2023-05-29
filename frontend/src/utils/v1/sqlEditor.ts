import { ComposedDatabase, ComposedInstance, UNKNOWN_ID } from "@/types";
import { databaseV1Slug } from "./database";
import { instanceV1Slug } from "./instance";

export function connectionV1Slug(
  instance: ComposedInstance,
  database?: ComposedDatabase
): string {
  const parts = [instanceV1Slug(instance)];
  if (database && database.uid !== String(UNKNOWN_ID)) {
    parts.push(databaseV1Slug(database));
  }
  return parts.join("_");
}
