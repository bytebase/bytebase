import { DataSourceId, Instance, TaskId } from "../types";
import slug from "slug";

export function taskSlug(taskName: string, taskId: TaskId) {
  return [slug(taskName), taskId].join("-");
}

// On the other hand, it's not possible to de-slug due to slug's one-way algorithm
export function instanceSlug(instance: Instance): string {
  return [
    slug(instance.environment.name),
    slug(instance.name),
    instance.id,
  ].join("-");
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
