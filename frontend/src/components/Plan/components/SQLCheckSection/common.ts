import { DatabaseChangeType } from "@/types/proto-es/v1/common_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export const getSpecChangeType = (spec?: Plan_Spec): boolean => {
  const changeDatabaseConfig =
    spec?.config?.case === "changeDatabaseConfig"
      ? spec.config.value
      : undefined;
  if (changeDatabaseConfig?.type === DatabaseChangeType.MIGRATE) {
    return changeDatabaseConfig.enableGhost === true;
  }
  return false;
};
