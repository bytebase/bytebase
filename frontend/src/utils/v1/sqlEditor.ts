import slug from "slug";
import type { ComposedDatabase } from "@/types";
import { UNKNOWN_ID } from "@/types";
import type { InstanceResource } from "@/types/proto/v1/instance_service";

const databaseV1Slug = (db: ComposedDatabase) => {
  return [slug(db.databaseName), db.uid].join("-");
};

const instanceV1Slug = (instance: InstanceResource): string => {
  return [slug(instance.title), instance.uid].join("-");
};

export function connectionV1Slug(
  instance: InstanceResource,
  database?: ComposedDatabase
): string {
  const parts = [instanceV1Slug(instance)];
  if (database && database.uid !== String(UNKNOWN_ID)) {
    parts.push(databaseV1Slug(database));
  }
  return parts.join("_");
}
