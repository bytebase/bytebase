import { DataSourceId, InstanceId, TaskId } from "../types";
import slug from "slug";

export function taskSlug(taskName: string, taskId: TaskId) {
  return [slug(taskName), taskId].join("-");
}

// On the other hand, it's not possible to de-slug due to slug's one-way algorithm
export function instanceSlug(
  environmentName: string,
  instanceName: string,
  instanceId: InstanceId
): string {
  return [slug(environmentName), slug(instanceName), instanceId].join("-");
}

export function dataSourceSlug(
  dataSourceName: string,
  dataSourceId: DataSourceId
): string {
  return [slug(dataSourceName), dataSourceId].join("-");
}

export function idFromSlug(slug: string) {
  const parts = slug.split("-");
  return parts[parts.length - 1];
}
