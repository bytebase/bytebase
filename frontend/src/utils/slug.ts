import {
  getReviewConfigId,
  reviewConfigNamePrefix,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicy } from "../types";

export function sqlReviewPolicySlug(reviewPolicy: SQLReviewPolicy): string {
  return getReviewConfigId(reviewPolicy.id);
}

export function sqlReviewNameFromSlug(slug: string): string {
  return `${reviewConfigNamePrefix}${slug}`;
}
