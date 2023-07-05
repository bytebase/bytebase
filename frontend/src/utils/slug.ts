import slug from "slug";
import {
  Database,
  DataSource,
  Environment,
  Instance,
  IssueId,
  Project,
  SQLReviewPolicy,
  UNKNOWN_ID,
} from "../types";
import { IdType } from "../types/id";
import { Sheet as SheetV1 } from "@/types/proto/v1/sheet_service";
import { Project as ProjectV1 } from "@/types/proto/v1/project_service";
import { ExternalVersionControl as VCSV1 } from "@/types/proto/v1/externalvs_service";
import {
  getProjectAndSheetId,
  projectNamePrefix,
  sheetNamePrefix,
  getVCSUid,
} from "@/store/modules/v1/common";

export const indexOrUIDFromSlug = (slug: string): number => {
  const parts = slug.split("-");
  const indexOrUID = parseInt(parts[parts.length - 1], 10);
  if (Number.isNaN(indexOrUID) || indexOrUID < 0) {
    return -1;
  }
  return indexOrUID;
};

export function idFromSlug(slug: string): IdType {
  const parts = slug.split("-");
  return parseInt(parts[parts.length - 1]);
}

export function sheetNameFromSlug(slug: string): string {
  const parts = slug.split("-");
  return `${projectNamePrefix}${parts
    .slice(0, -1)
    .join("-")}/${sheetNamePrefix}${parts[parts.length - 1]}`;
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

export function projectSlugV1(project: ProjectV1): string {
  return [slug(project.title), project.uid].join("-");
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

export function taskSlug(name: string, id: IdType): string {
  return [slug(name), id].join("-");
}

export function databaseSlug(database: Database): string {
  return [slug(database.name), database.id].join("-");
}

export function dataSourceSlug(dataSource: DataSource): string {
  return [slug(dataSource.name), dataSource.id].join("-");
}

export function fullDatabasePath(database: Database): string {
  return `/db/${databaseSlug(database)}`;
}

export function vcsSlugV1(vcs: VCSV1): string {
  return [slug(vcs.title), getVCSUid(vcs.name)].join("-");
}

export function sqlReviewPolicySlug(reviewPolicy: SQLReviewPolicy): string {
  return [slug(reviewPolicy.name), reviewPolicy.environment.uid].join("-");
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

export function sheetSlugV1(sheet: SheetV1): string {
  const [projectName, uid] = getProjectAndSheetId(sheet.name);
  return [projectName, uid].join("-");
}
