import { DatabaseChangeType } from "@/types/proto-es/v1/common_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { Release_File_ChangeType } from "@/types/proto-es/v1/release_service_pb";

export const getSpecChangeType = (
  spec?: Plan_Spec
): Release_File_ChangeType => {
  // For export data config, we treat it as DML.
  if (spec?.config.case === "exportDataConfig") {
    return Release_File_ChangeType.DML;
  }
  const changeDatabaseConfig =
    spec?.config?.case === "changeDatabaseConfig"
      ? spec.config.value
      : undefined;
  switch (changeDatabaseConfig?.type) {
    case DatabaseChangeType.MIGRATE:
      return Release_File_ChangeType.DDL;
    case DatabaseChangeType.MIGRATE_GHOST:
      return Release_File_ChangeType.DDL_GHOST;
    case DatabaseChangeType.DATA:
      return Release_File_ChangeType.DML;
  }
  // Default to DDL if no type is specified.
  return Release_File_ChangeType.DDL;
};
