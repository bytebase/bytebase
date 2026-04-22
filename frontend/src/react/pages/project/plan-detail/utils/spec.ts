import type { TFunction } from "i18next";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";

export const getSelectedSpec = ({
  selectedSpecId,
  specs,
}: {
  selectedSpecId: string;
  specs: Plan_Spec[];
}): Plan_Spec | undefined => {
  return specs.find((spec) => spec.id === selectedSpecId) ?? specs[0];
};

export const getSpecTitle = (spec: Plan_Spec, t: TFunction): string => {
  if (spec.config.case === "createDatabaseConfig") {
    return t("common.database");
  }
  if (spec.config.case === "changeDatabaseConfig") {
    return t("plan.spec.type.database-change");
  }
  if (spec.config.case === "exportDataConfig") {
    return t("common.export");
  }
  return t("common.unknown");
};
