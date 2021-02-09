import { InstanceId, TaskId } from "../types";
import slug from "slug";

export function taskSlug(taskName: string, taskId: TaskId) {
  return [slug(taskName), taskId].join("-");
}

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
