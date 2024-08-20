import slug from "slug";
import type { Plan, Plan_Spec } from "@/types/proto/v1/plan_service";

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
