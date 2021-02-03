import { InstanceId } from "../types";
import slug from "slug";

export function instanceSlug(
  environmentName: string,
  instanceName: string,
  instanceId: InstanceId
) {
  return [slug(environmentName), slug(instanceName), instanceId].join("-");
}

export function idFromSlug(slug: string) {
  const parts = slug.split("-");
  return parts[parts.length - 1];
}
