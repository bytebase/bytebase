import { head } from "lodash-es";
import { mockDatabase } from "@/components/IssueV1";
import { useDatabaseV1Store, useProjectV1Store } from "@/store";
import { isValidDatabaseName, unknownDatabase } from "@/types";
import type { ComposedProject } from "@/types";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import { targetsForSpec } from "./plan";

export const projectOfPlan = (plan: ComposedPlan): ComposedProject =>
  useProjectV1Store().getProjectByName(plan.project);

export const targetOfSpec = (spec: Plan_Spec) => {
  const targets = targetsForSpec(spec);
  return head(targets);
};

export const databaseOfSpec = (project: ComposedProject, spec: Plan_Spec) => {
  if (spec.createDatabaseConfig) {
    return mockDatabase(
      project,
      `${spec.createDatabaseConfig.target}/databases/${spec.createDatabaseConfig.database}`
    );
  }
  const target = targetOfSpec(spec);
  if (!target) {
    return unknownDatabase();
  }
  const db = useDatabaseV1Store().getDatabaseByName(target);
  if (!isValidDatabaseName(db.name)) {
    return mockDatabase(project, target);
  }
  return db;
};
