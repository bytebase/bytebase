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
  SQLReviewPolicy,
  Sheet,
  UNKNOWN_ID,
} from "../types";
import { IdType } from "../types/id";

export function idFromSlug(slug: string): IdType {
  const parts = slug.split("-");
  return parseInt(parts[parts.length - 1]);
}

export function migrationHistoryIdFromSlug(slug: string): MigrationHistoryId {
  const parts = slug.split("-").slice(1);
  return parts.join("-");
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

export function vcsSlug(vcs: VCS): string {
  return [slug(vcs.name), vcs.id].join("-");
}

export function sqlReviewPolicySlug(reviewPolicy: SQLReviewPolicy): string {
  return [slug(reviewPolicy.name), reviewPolicy.environment.id].join("-");
}

export function connectionSlug(
  instance: Instance,
  database?: Database
): string {
  const parts = [instanceSlug(instance)];
  if (database && database.id !== UNKNOWN_ID) {
    parts.push(databaseSlug(database));
  }
  return parts.join("_");
}

export function sheetSlug(sheet: Sheet): string {
  return [slug(sheet.name), sheet.id].join("-");
}
