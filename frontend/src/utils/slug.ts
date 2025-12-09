import {
  getReviewConfigId,
  reviewConfigNamePrefix,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "../types";
import type { IdType } from "../types/id";

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

export function sqlReviewPolicySlug(reviewPolicy: SQLReviewPolicy): string {
  return getReviewConfigId(reviewPolicy.id);
}

export function sqlReviewNameFromSlug(slug: string): string {
  return `${reviewConfigNamePrefix}${slug}`;
}
