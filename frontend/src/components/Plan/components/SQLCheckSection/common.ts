import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export const getSpecChangeType = (spec?: Plan_Spec): boolean => {
  const changeDatabaseConfig =
    spec?.config?.case === "changeDatabaseConfig"
      ? spec.config.value
      : undefined;
  // Ghost is only available for sheet-based migrations (not release-based).
  if (changeDatabaseConfig && !changeDatabaseConfig.release) {
    return changeDatabaseConfig.enableGhost === true;
  }
  return false;
};
