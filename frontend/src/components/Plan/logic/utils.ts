import { t } from "@/plugins/i18n";
import { useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName } from "@/utils";

export const projectOfPlan = (plan: Plan): ComposedProject => {
  const project = `projects/${extractProjectResourceName(plan.name)}`;
  return useProjectV1Store().getProjectByName(project);
};

export const getSpecTitle = (spec: Plan_Spec): string => {
  let title = "";
  if (spec.config?.case === "createDatabaseConfig") {
    title = t("plan.spec.type.create-database");
  } else if (spec.config?.case === "changeDatabaseConfig") {
    const changeType = spec.config.value.type;
    switch (changeType) {
      case Plan_ChangeDatabaseConfig_Type.MIGRATE:
        title = t("plan.spec.type.schema-change");
        break;
      case Plan_ChangeDatabaseConfig_Type.MIGRATE_SDL:
        title = "SDL";
        break;
      case Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
        title = t("plan.spec.type.ghost-migration");
        break;
      case Plan_ChangeDatabaseConfig_Type.DATA:
        title = t("plan.spec.type.data-change");
        break;
      default:
        title = t("plan.spec.type.database-change");
    }
  } else if (spec.config?.case === "exportDataConfig") {
    title = t("plan.spec.type.export-data");
  } else {
    title = t("plan.spec.type.unknown");
  }
  return title;
};
