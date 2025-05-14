import { mockDatabase } from "@/components/IssueV1";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName, unknownDatabase } from "@/types";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";

export const targetOfSpec = (spec: Plan_Spec) => {
  if (spec.changeDatabaseConfig) {
    return spec.changeDatabaseConfig.target;
  } else if (spec.exportDataConfig) {
    return spec.exportDataConfig.target;
  }
  return undefined;
};

export const databaseOfSpec = (plan: ComposedPlan, spec: Plan_Spec) => {
  if (spec.createDatabaseConfig) {
    return mockDatabase(
      plan.projectEntity,
      `${spec.createDatabaseConfig.target}/databases/${spec.createDatabaseConfig.database}`
    );
  }
  const target = targetOfSpec(spec);
  if (!target) {
    return unknownDatabase();
  }
  const db = useDatabaseV1Store().getDatabaseByName(target);
  if (!isValidDatabaseName(db.name)) {
    return mockDatabase(plan.projectEntity, target);
  }
  return db;
};
