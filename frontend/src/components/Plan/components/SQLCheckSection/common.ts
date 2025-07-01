import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
} from "@/types/proto/v1/plan_service";
import { Release_File_ChangeType } from "@/types/proto-es/v1/release_service_pb";

export const getSpecChangeType = (
  spec?: Plan_Spec
): Release_File_ChangeType => {
  switch (spec?.changeDatabaseConfig?.type) {
    case Plan_ChangeDatabaseConfig_Type.MIGRATE:
      return Release_File_ChangeType.DDL;
    case Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
      return Release_File_ChangeType.DDL_GHOST;
    case Plan_ChangeDatabaseConfig_Type.DATA:
      return Release_File_ChangeType.DML;
  }
  // Default to DDL if no type is specified.
  return Release_File_ChangeType.DDL;
};
