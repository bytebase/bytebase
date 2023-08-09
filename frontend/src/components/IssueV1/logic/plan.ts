import { Plan_Spec } from "@/types/proto/v1/rollout_service";

export const sheetNameForSpec = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig?.sheet ?? "";
};
