import { t } from "@/plugins/i18n";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";

export const semanticTaskType = (type: Task_Type) => {
  switch (type) {
    case Task_Type.DATABASE_CREATE:
      return t("task.type.database-create");
    case Task_Type.DATABASE_MIGRATE:
      return t("task.type.migrate");
    case Task_Type.DATABASE_EXPORT:
      return t("task.type.database-export");
    case Task_Type.GENERAL:
      return t("task.type.general");
    default:
      return "";
  }
};
