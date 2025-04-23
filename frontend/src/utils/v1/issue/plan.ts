import slug from "slug";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { Plan, Plan_Spec } from "@/types/proto/api/v1alpha/plan_service";

export const sheetNameOfSpec = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig?.sheet ?? spec.exportDataConfig?.sheet ?? "";
};

export function planV1Slug(plan: Plan): string {
  return [slug(plan.title), extractPlanUID(plan.name)].join("-");
}

export const extractPlanUID = (name: string) => {
  const pattern = /(?:^|\/)plans\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isValidPlanName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const planUID = extractPlanUID(name);
  return (
    planUID && planUID !== String(EMPTY_ID) && planUID !== String(UNKNOWN_ID)
  );
};

export const extractPlanCheckRunUID = (name: string) => {
  const pattern = /(?:^|\/)planCheckRuns\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
