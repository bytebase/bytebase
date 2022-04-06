import slug from "slug";
import {
  Database,
  DataSource,
  Environment,
  Instance,
  IssueId,
  MigrationHistoryId,
  Project,
  ProjectWebhook,
  VCS,
} from "../types";
import { IdType } from "../types/id";

export function idFromSlug(slug: string): IdType {
  const parts = slug.split("-");
  return parseInt(parts[parts.length - 1]);
}

export function indexFromSlug(slug: string): number {
  const parts = slug.split("-");
  return parseInt(parts[parts.length - 1]) - 1;
}

export function issueSlug(issueName: string, issueId: IssueId): string {
  return [slug(issueName), issueId].join("-");
}

// On the other hand, it's not possible to de-slug due to slug's one-way algorithm
export function environmentSlug(environment: Environment): string {
  return [slug(environment.name), environment.id].join("-");
}

export function projectSlug(project: Project): string {
  return [slug(project.name), project.id].join("-");
}

export function projectWebhookSlug(projectWebhook: ProjectWebhook): string {
  return [slug(projectWebhook.name), projectWebhook.id].join("-");
}

export function instanceSlug(instance: Instance): string {
  return [
    slug(instance.environment.name),
    slug(instance.name),
    instance.id,
  ].join("-");
}

export function stageSlug(stageName: string, stageIndex: number): string {
  return [slug(stageName), stageIndex + 1].join("-");
}

export function taskSlug(name: string, id: number): string {
  return [slug(name), id].join("-");
}

export function databaseSlug(database: Database): string {
  return [slug(database.name), database.id].join("-");
}

export function dataSourceSlug(dataSource: DataSource): string {
  return [slug(dataSource.name), dataSource.id].join("-");
}

export function migrationHistorySlug(
  migrationHistoryId: MigrationHistoryId,
  version: string
): string {
  return [slug(version), migrationHistoryId].join("-");
}

export function fullDatabasePath(database: Database): string {
  return `/db/${databaseSlug(database)}`;
}

export function fullDataSourcePath(dataSource: DataSource): string {
  return `/db/${databaseSlug(dataSource.database)}/data-source/${dataSourceSlug(
    dataSource
  )}`;
}

export function vcsSlug(vcs: VCS): string {
  return [slug(vcs.name), vcs.id].join("-");
}

export function connectionSlug(database: Database): string {
  return [
    slug(database.instance.name),
    database.instance.id,
    slug(database.name),
    database.id,
  ].join("_");
}
