import slug from "slug";
import { ComposedDatabase, ComposedInstance, UNKNOWN_ID } from "@/types";

const databaseV1Slug = (db: ComposedDatabase) => {
  return [slug(db.databaseName), db.uid].join("-");
};

const instanceV1Slug = (instance: ComposedInstance): string => {
  return [slug(instance.title), instance.uid].join("-");
};

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
