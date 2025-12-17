import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export const sheetNameOfSpec = (spec: Plan_Spec): string => {
  if (spec.config?.case === "changeDatabaseConfig") {
    return spec.config.value.sheet ?? "";
  }
  if (spec.config?.case === "exportDataConfig") {
    return spec.config.value.sheet ?? "";
  }
  return "";
};

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
  return Boolean(
    planUID && planUID !== String(EMPTY_ID) && planUID !== String(UNKNOWN_ID)
  );
};

export const extractPlanCheckRunUID = (name: string) => {
  const pattern = /(?:^|\/)planCheckRuns\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractSpecId = (name: string) => {
  const pattern = /(?:^|\/)specs\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

/**
 * Get spec display info from spec resource name.
 * Returns specId and 1-based display index, or null if not found.
 */
export const getSpecDisplayInfo = (
  specs: Plan_Spec[],
  specResourceName: string
): { specId: string; displayIndex: number } | null => {
  const specId = extractSpecId(specResourceName);
  if (!specId) return null;

  const index = specs.findIndex((spec) => spec.id === specId);
  if (index >= 0) {
    return { specId, displayIndex: index + 1 };
  }
  return null;
};
