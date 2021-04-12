import { Database, DataSource, Environment, Instance, TaskId } from "../types";
import slug from "slug";

export function taskSlug(taskName: string, taskId: TaskId) {
  return [taskId, slug(taskName)].join("-");
}

// On the other hand, it's not possible to de-slug due to slug's one-way algorithm
export function environmentSlug(environment: Environment): string {
  return [environment.id, slug(environment.name)].join("-");
}

export function instanceSlug(instance: Instance): string {
  return [
    instance.id,
    slug(instance.environment.name),
    slug(instance.name),
  ].join("-");
}

export function databaseSlug(database: Database): string {
  return [database.id, slug(database.name)].join("-");
}

export function fullDatabasePath(database: Database): string {
  return `/db/${databaseSlug(database)}`;
}

export function dataSourceSlug(dataSource: DataSource): string {
  return [slug(dataSource.name), dataSource.id].join("-");
}

export function fullDataSourcePath(dataSource: DataSource): string {
  return `/db/${databaseSlug(dataSource.database)}/datasource/${dataSourceSlug(
    dataSource
  )}`;
}

export function idFromSlug(slug: string) {
  const parts = slug.split("-");
  return parts[0];
}
