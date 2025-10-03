import {
  DatabaseChangeType,
  MigrationType,
} from "@/types/proto-es/v1/common_pb";
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
  if (changeDatabaseConfig?.type === DatabaseChangeType.MIGRATE) {
    if (changeDatabaseConfig.migrationType === MigrationType.DML) {
      return Release_File_ChangeType.DML;
    } else if (changeDatabaseConfig.migrationType === MigrationType.GHOST) {
      return Release_File_ChangeType.DDL_GHOST;
    }
    return Release_File_ChangeType.DDL;
  }
  if (changeDatabaseConfig?.type === DatabaseChangeType.SDL) {
    return Release_File_ChangeType.DDL;
  }
  // Default to DDL if no type is specified.
  return Release_File_ChangeType.DDL;
};
