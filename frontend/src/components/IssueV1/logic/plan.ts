import { Plan_Spec } from "@/types/proto/v1/rollout_service";

export const sheetNameForSpec = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig?.sheet ?? "";
};

export const targetForSpec = (spec: Plan_Spec | undefined) => {
  if (!spec) return undefined;
  return (
    spec.changeDatabaseConfig?.target ??
    spec.createDatabaseConfig?.target ??
    spec.restoreDatabaseConfig?.target ??
    undefined
  );
};
