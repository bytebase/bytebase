import slug from "slug";
import {
  getProjectName,
  getWorksheetId,
  worksheetNamePrefix,
  getVCSUid,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { ExternalVersionControl as VCSV1 } from "@/types/proto/v1/externalvs_service";
import { Worksheet } from "@/types/proto/v1/worksheet_service";
import { IssueId, SQLReviewPolicy } from "../types";
import { IdType } from "../types/id";
import { extractEnvironmentResourceName } from "./v1";

export const indexOrUIDFromSlug = (slug: string): number => {
  const parts = slug.split("-");
  const indexOrUID = parseInt(parts[parts.length - 1], 10);
  if (Number.isNaN(indexOrUID) || indexOrUID < 0) {
    return -1;
  }
  return indexOrUID;
};

export function uidFromSlug(slug: string): IdType {
  const parts = slug.split("-");
  return parseInt(parts[parts.length - 1]);
}

export const idFromSlug = (slug: string): string => {
  const parts = slug.split("-");
  return parts[parts.length - 1];
};

export function projectNameFromSheetSlug(slug: string): string {
  const parts = slug.split("-");
  parts.pop();
  return `${projectNamePrefix}${parts.join("-")}`;
}

export function worksheetNameFromSlug(slug: string): string {
  const parts = slug.split("-");
  return `${worksheetNamePrefix}${parts[parts.length - 1]}`;
}

export function indexFromSlug(slug: string): number {
  const parts = slug.split("-");
  return parseInt(parts[parts.length - 1]) - 1;
}

export function issueSlug(issueName: string, issueId: IssueId): string {
  return [slug(issueName), issueId].join("-");
}

export function stageSlug(stageName: string, stageIndex: number): string {
  return [slug(stageName), stageIndex + 1].join("-");
}

export function taskSlug(name: string, id: IdType): string {
  return [slug(name), id].join("-");
}

export function vcsSlugV1(vcs: VCSV1): string {
  return [slug(vcs.title), getVCSUid(vcs.name)].join("-");
}

export function sqlReviewPolicySlug(reviewPolicy: SQLReviewPolicy): string {
  return [
    slug(reviewPolicy.name),
    extractEnvironmentResourceName(reviewPolicy.environment.name),
  ].join("-");
}

export function worksheetSlugV1(sheet: Worksheet): string {
  const uid = getWorksheetId(sheet.name);
  const projectName = getProjectName(sheet.project);
  return [projectName, uid].join("-");
}
